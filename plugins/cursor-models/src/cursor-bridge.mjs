import http from 'node:http';
import fs from 'node:fs';
import path from 'node:path';
import os from 'node:os';
import crypto from 'node:crypto';

const HOME = os.homedir();
const HELPER_DIR = path.join(HOME, '.helper');
const TOKEN_FILE = path.join(HELPER_DIR, 'cursor-bridge-token');
const KEY_FILE = path.join(HELPER_DIR, 'cursor-api-key');
const AUTH_FILE = path.join(HOME, '.pi/agent/auth.json');

export function getCursorApiKey() {
  if (process.env.CURSOR_API_KEY) {
    return process.env.CURSOR_API_KEY.trim();
  }
  if (process.env.CURSOR_AUTH_TOKEN) {
    return process.env.CURSOR_AUTH_TOKEN.trim();
  }
  if (fs.existsSync(KEY_FILE)) {
    try {
      const key = fs.readFileSync(KEY_FILE, 'utf8').trim();
      if (key) return key;
    } catch {}
  }
  if (fs.existsSync(AUTH_FILE)) {
    try {
      const auth = JSON.parse(fs.readFileSync(AUTH_FILE, 'utf8'));
      if (auth.cursor?.key) return auth.cursor.key;
    } catch {}
  }
  return null;
}

export function getOrCreateBridgeToken() {
  let token = process.env.CURSOR_BRIDGE_TOKEN;
  if (!token && fs.existsSync(TOKEN_FILE)) {
    try {
      token = fs.readFileSync(TOKEN_FILE, 'utf8').trim();
    } catch {}
  }
  if (!token) {
    token = 'cbr_' + crypto.randomBytes(16).toString('hex');
    try {
      if (!fs.existsSync(HELPER_DIR)) {
        fs.mkdirSync(HELPER_DIR, { recursive: true, mode: 0o700 });
      }
      fs.writeFileSync(TOKEN_FILE, token, { mode: 0o600 });
    } catch (e) {
      console.error('[cursor-bridge] Warning: Failed to persist token:', e.message);
    }
  }
  if (fs.existsSync(TOKEN_FILE)) {
    try { fs.chmodSync(TOKEN_FILE, 0o600); } catch {}
  }
  return token;
}

export async function resolveCursorSdk() {
  const candidates = [
    '@cursor/sdk',
    path.join(HELPER_DIR, 'node_modules', '@cursor', 'sdk', 'dist', 'esm', 'index.js'),
    path.join(HOME, '.pi', 'agent', 'npm', 'node_modules', '@cursor', 'sdk', 'dist', 'esm', 'index.js'),
    path.join(HOME, '.pi', 'agent', 'node_modules', '@cursor', 'sdk', 'dist', 'esm', 'index.js')
  ];

  for (const item of candidates) {
    try {
      if (item.startsWith('@')) {
        const mod = await import(item);
        return mod;
      }
      if (fs.existsSync(item)) {
        const mod = await import(`file://${item}`);
        return mod;
      }
    } catch {}
  }

  throw new Error(
    '@cursor/sdk not found. Please install via: npm install -g @cursor/sdk or npm install --prefix ~/.helper @cursor/sdk'
  );
}

function formatMessages(messages) {
  if (!Array.isArray(messages) || messages.length === 0) return '';
  if (messages.length === 1 && messages[0] && messages[0].role === 'user') {
    const c = messages[0].content;
    return typeof c === 'string' ? c : JSON.stringify(c);
  }

  const parts = [];
  for (const m of messages) {
    if (!m) continue;
    let content = m.content;
    if (content === null || content === undefined) {
      content = '';
    } else if (typeof content !== 'string') {
      content = JSON.stringify(content);
    }

    if (m.role === 'assistant') {
      let text = `[Assistant]`;
      if (content) text += `\n${content}`;
      if (Array.isArray(m.tool_calls) && m.tool_calls.length > 0) {
        text += `\n[Tool Calls Requested]\n${JSON.stringify(m.tool_calls, null, 2)}`;
      }
      parts.push(text);
    } else if (m.role === 'tool') {
      const callId = m.tool_call_id ? ` (call_id: ${m.tool_call_id})` : '';
      parts.push(`[Tool Result${callId}]\n${content}`);
    } else if (m.role === 'system') {
      parts.push(`[System]\n${content}`);
    } else {
      parts.push(`[User]\n${content}`);
    }
  }

  return parts.join('\n\n');
}

function resolveModelId(modelInput, availableModels) {
  if (!modelInput) return 'default';
  let clean = String(modelInput).trim().replace(/^cursor[\/-]/, '');
  if (clean === 'auto') clean = 'default';

  const matchId = availableModels.find((m) => m.id === clean);
  if (matchId) return matchId.id;

  const matchAlias = availableModels.find((m) => m.aliases && m.aliases.includes(clean));
  if (matchAlias) return matchAlias.id;

  const matchPrefix = availableModels.find((m) => m.id.startsWith(clean));
  if (matchPrefix) return matchPrefix.id;

  return clean || 'default';
}

export async function startBridge(options = {}) {
  const port = parseInt(options.port || process.env.PORT || '4646', 10);
  const bridgeToken = options.token || getOrCreateBridgeToken();
  const logger = options.logger || console;

  const sdk = await resolveCursorSdk();
  const { Agent, Cursor } = sdk;

  let cachedModels = null;
  let lastModelFetch = 0;

  async function getAvailableModels() {
    const now = Date.now();
    if (cachedModels && now - lastModelFetch < 300000) {
      return cachedModels;
    }
    const currentKey = getCursorApiKey();
    if (!currentKey) return cachedModels || [];
    try {
      const list = await Cursor.models.list({ apiKey: currentKey });
      cachedModels = list;
      lastModelFetch = now;
      return list;
    } catch (e) {
      logger.error?.('[cursor-bridge] Error fetching model catalog:', e.message);
      return cachedModels || [];
    }
  }

  const server = http.createServer(async (req, res) => {
    req.on('error', (err) => logger.error?.('Request socket error:', err.message));
    res.on('error', (err) => logger.error?.('Response socket error:', err.message));

    // Strict CORS / Hostname Protection
    const origin = req.headers.origin;
    if (origin) {
      let allowed = false;
      if (origin.startsWith('vscode-webview://') || origin.startsWith('wails://')) {
        allowed = true;
      } else {
        try {
          const u = new URL(origin);
          allowed = (
            u.hostname === 'localhost' ||
            u.hostname === '127.0.0.1' ||
            u.hostname === '[::1]'
          );
        } catch {
          allowed = false;
        }
      }

      if (allowed) {
        res.setHeader('Access-Control-Allow-Origin', origin);
        res.setHeader('Access-Control-Allow-Methods', 'GET, POST, OPTIONS');
        res.setHeader('Access-Control-Allow-Headers', 'Content-Type, Authorization');
      } else {
        res.writeHead(403, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ error: { message: 'Forbidden origin', type: 'access_denied' } }));
        return;
      }
    }

    if (req.method === 'OPTIONS') {
      res.writeHead(204);
      res.end();
      return;
    }

    // Health
    if (req.url === '/health' && req.method === 'GET') {
      const hasKey = !!getCursorApiKey();
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(
        JSON.stringify({
          status: 'ok',
          service: 'helper-cursor-bridge',
          hasCursorKey: hasKey,
          version: '1.0.0'
        })
      );
      return;
    }

    // Token Auth
    const authHeader = req.headers.authorization || '';
    const tokenMatch = authHeader.match(/^Bearer\s+(.+)$/i);
    const providedToken = tokenMatch ? tokenMatch[1].trim() : '';

    if (!providedToken || providedToken !== bridgeToken) {
      res.writeHead(401, { 'Content-Type': 'application/json' });
      res.end(
        JSON.stringify({
          error: {
            message: 'Unauthorized: Valid Bearer token required.',
            type: 'authentication_error'
          }
        })
      );
      return;
    }

    // Models
    if (req.url === '/v1/models' && req.method === 'GET') {
      const models = await getAvailableModels();
      const openAiModels = models.map((m) => ({
        id: m.id,
        object: 'model',
        created: Math.floor(Date.now() / 1000),
        owned_by: 'cursor'
      }));
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ object: 'list', data: openAiModels }));
      return;
    }

    // Chat completions
    if (req.url === '/v1/chat/completions' && req.method === 'POST') {
      const chunks = [];
      let totalBytes = 0;
      const MAX_PAYLOAD_BYTES = 10 * 1024 * 1024;
      let sizeExceeded = false;

      req.on('data', (chunk) => {
        if (sizeExceeded) return;
        totalBytes += chunk.length;
        if (totalBytes > MAX_PAYLOAD_BYTES) {
          sizeExceeded = true;
          return;
        }
        chunks.push(chunk);
      });

      req.on('end', async () => {
        if (sizeExceeded) {
          if (!res.headersSent) {
            res.writeHead(413, { 'Content-Type': 'application/json' });
            res.end(JSON.stringify({ error: { message: 'Payload Too Large (10MB limit)', type: 'invalid_request_error' } }));
          }
          return;
        }

        const rawData = Buffer.concat(chunks).toString('utf8');
        let body;
        try {
          body = JSON.parse(rawData);
        } catch {
          if (!res.headersSent) {
            res.writeHead(400, { 'Content-Type': 'application/json' });
            res.end(JSON.stringify({ error: { message: 'Malformed JSON', type: 'invalid_request_error' } }));
          }
          return;
        }

        const isStream = body.stream === true;
        const requestId = 'chatcmpl-' + crypto.randomBytes(8).toString('hex');
        let prompt, availableModels, modelId;

        try {
          prompt = formatMessages(body.messages);
          availableModels = await getAvailableModels();
          modelId = resolveModelId(body.model, availableModels);
        } catch (prepErr) {
          if (!res.headersSent) {
            res.writeHead(400, { 'Content-Type': 'application/json' });
            res.end(JSON.stringify({ error: { message: prepErr.message, type: 'invalid_request_error' } }));
          }
          return;
        }

        let agent = null;
        let run = null;
        let aborted = false;
        const activeCursorKey = getCursorApiKey();

        if (!activeCursorKey) {
          res.writeHead(500, { 'Content-Type': 'application/json' });
          res.end(JSON.stringify({ error: { message: 'No Cursor API key detected.', type: 'api_error' } }));
          return;
        }

        req.on('close', () => {
          if (!res.writableEnded) {
            aborted = true;
            if (run && typeof run.cancel === 'function') {
              run.cancel().catch(() => {});
            }
          }
        });

        try {
          agent = await Agent.create({
            apiKey: activeCursorKey,
            model: { id: modelId },
            mode: 'plan'
          });

          if (aborted) return;
          run = await agent.send(prompt);
          if (aborted) {
            if (run && typeof run.cancel === 'function') run.cancel().catch(() => {});
            return;
          }

          if (isStream) {
            res.writeHead(200, {
              'Content-Type': 'text/event-stream',
              'Cache-Control': 'no-cache',
              'Connection': 'keep-alive'
            });

            const firstChunk = {
              id: requestId,
              object: 'chat.completion.chunk',
              created: Math.floor(Date.now() / 1000),
              model: modelId,
              choices: [{ index: 0, delta: { role: 'assistant' }, finish_reason: null }]
            };
            res.write(`data: ${JSON.stringify(firstChunk)}\n\n`);

            if (run.supports('stream')) {
              for await (const msg of run.stream()) {
                if (aborted) break;
                if (msg.type === 'thinking' && msg.text) {
                  const chunk = {
                    id: requestId,
                    object: 'chat.completion.chunk',
                    created: Math.floor(Date.now() / 1000),
                    model: modelId,
                    choices: [{ index: 0, delta: { reasoning_content: msg.text }, finish_reason: null }]
                  };
                  res.write(`data: ${JSON.stringify(chunk)}\n\n`);
                } else if (msg.type === 'assistant' && msg.message?.content) {
                  const text = msg.message.content
                    .filter((c) => c.type === 'text')
                    .map((c) => c.text)
                    .join('');
                  if (text) {
                    const chunk = {
                      id: requestId,
                      object: 'chat.completion.chunk',
                      created: Math.floor(Date.now() / 1000),
                      model: modelId,
                      choices: [{ index: 0, delta: { content: text }, finish_reason: null }]
                    };
                    res.write(`data: ${JSON.stringify(chunk)}\n\n`);
                  }
                }
              }
            } else {
              const result = await run.wait();
              if (result.result && !aborted) {
                const chunk = {
                  id: requestId,
                  object: 'chat.completion.chunk',
                  created: Math.floor(Date.now() / 1000),
                  model: modelId,
                  choices: [{ index: 0, delta: { content: result.result }, finish_reason: null }]
                };
                res.write(`data: ${JSON.stringify(chunk)}\n\n`);
              }
            }

            if (!aborted) {
              const doneChunk = {
                id: requestId,
                object: 'chat.completion.chunk',
                created: Math.floor(Date.now() / 1000),
                model: modelId,
                choices: [{ index: 0, delta: {}, finish_reason: 'stop' }]
              };
              res.write(`data: ${JSON.stringify(doneChunk)}\n\n`);
              res.write('data: [DONE]\n\n');
              res.end();
            }
          } else {
            let fullContent = '';
            if (run.supports('stream')) {
              for await (const msg of run.stream()) {
                if (aborted) break;
                if (msg.type === 'assistant' && msg.message?.content) {
                  fullContent += msg.message.content
                    .filter((c) => c.type === 'text')
                    .map((c) => c.text)
                    .join('');
                }
              }
            } else {
              const result = await run.wait();
              fullContent = result.result || '';
            }

            if (!aborted) {
              res.writeHead(200, { 'Content-Type': 'application/json' });
              res.end(
                JSON.stringify({
                  id: requestId,
                  object: 'chat.completion',
                  created: Math.floor(Date.now() / 1000),
                  model: modelId,
                  choices: [
                    {
                      index: 0,
                      message: { role: 'assistant', content: fullContent },
                      finish_reason: 'stop'
                    }
                  ],
                  usage: { prompt_tokens: 0, completion_tokens: 0, total_tokens: 0 }
                })
              );
            }
          }
        } catch (err) {
          logger.error?.(`Execution error: ${err.message}`);
          if (!res.headersSent) {
            res.writeHead(500, { 'Content-Type': 'application/json' });
            res.end(JSON.stringify({ error: { message: err.message, type: 'api_error' } }));
          } else {
            try { res.end(); } catch {}
          }
        } finally {
          if (agent) {
            const agentId = agent.agentId;
            try { await agent[Symbol.asyncDispose](); } catch {}
            if (aborted && run && typeof run.cancel === 'function') {
              try { await run.cancel(); } catch {}
            }
            try {
              await Agent.delete(agentId, { apiKey: activeCursorKey });
            } catch (delErr) {
              logger.error?.(`Warning: Failed to delete cloud agent ${agentId}:`, delErr.message);
            }
          }
        }
      });
      return;
    }

    res.writeHead(404, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ error: { message: 'Not Found', type: 'not_found' } }));
  });

  return new Promise((resolve, reject) => {
    server.once('error', (err) => {
      reject(err);
    });

    server.listen(port, '127.0.0.1', () => {
      resolve({
        server,
        port,
        token: bridgeToken,
        close: () =>
          new Promise((r) => {
            const forceTimer = setTimeout(() => r(), 3000);
            forceTimer.unref();
            if (typeof server.closeAllConnections === 'function') {
              server.closeAllConnections();
            }
            server.close(() => {
              clearTimeout(forceTimer);
              r();
            });
          })
      });
    });
  });
}
