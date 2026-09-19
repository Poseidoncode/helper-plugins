#!/usr/bin/env node
import readline from 'node:readline';
import http from 'node:http';
import { startBridge, getCursorApiKey, getOrCreateBridgeToken } from './cursor-bridge.mjs';
import { syncApiConfig } from './config-sync.mjs';

const PORT = parseInt(process.env.PORT || '4646', 10);
const args = process.argv.slice(2);

// 1. Health check helper to see if a bridge is already running
function checkBridgeHealth(port) {
  return new Promise((resolve) => {
    const req = http.get(`http://127.0.0.1:${port}/health`, { timeout: 1500 }, (res) => {
      let data = '';
      res.on('data', (c) => (data += c));
      res.on('end', () => {
        try {
          const json = JSON.parse(data);
          resolve({ online: true, data: json });
        } catch {
          resolve({ online: true, raw: data });
        }
      });
    });
    req.on('error', () => resolve({ online: false }));
    req.on('timeout', () => {
      req.destroy();
      resolve({ online: false });
    });
  });
}

// 2. CLI Mode Handling
if (args.includes('--status')) {
  const health = await checkBridgeHealth(PORT);
  const key = getCursorApiKey();
  const token = getOrCreateBridgeToken();
  const mask = (s) => (s && s.length > 8 ? '...' + s.slice(-4) : 'CONFIGURED');
  console.log('=== Helper Cursor Models Provider Status ===');
  console.log(`Port: ${PORT}`);
  console.log(`Bridge Running: ${health.online ? 'YES (online)' : 'NO (stopped)'}`);
  console.log(`Cursor API Key Detected: ${key ? 'YES (' + mask(key) + ')' : 'NO'}`);
  console.log(`Local Bridge Token: ${token ? mask(token) : 'NONE'}`);
  process.exit(0);
}

if (args.includes('--sync')) {
  const result = syncApiConfig(PORT);
  console.log('Sync api-config.json result:', JSON.stringify(result, null, 2));
  process.exit(result.success ? 0 : 1);
}

if (args.includes('--daemon')) {
  // Pure HTTP Bridge Mode
  const existing = await checkBridgeHealth(PORT);
  if (existing.online) {
    console.log(`[cursor-bridge] Bridge is already active on http://127.0.0.1:${PORT}`);
  } else {
    try {
      const bridge = await startBridge({ port: PORT });
      console.log(`[cursor-bridge] Started daemon on http://127.0.0.1:${bridge.port}`);
    } catch (err) {
      console.error('[cursor-bridge] Failed to start daemon:', err.message);
      process.exit(1);
    }
  }
  syncApiConfig(PORT);
  // Keep alive with clean signal exit
  const exitHandler = () => process.exit(0);
  process.on('SIGTERM', exitHandler);
  process.on('SIGINT', exitHandler);
} else {
  // 3. MCP Server Mode (Default)
  runMcpServer().catch((err) => {
    console.error('[MCP Error]', err);
    process.exit(1);
  });
}

async function runMcpServer() {
  // Check or spawn background HTTP bridge
  let bridgeInstance = null;
  const health = await checkBridgeHealth(PORT);
  if (!health.online) {
    try {
      bridgeInstance = await startBridge({
        port: PORT,
        logger: {
          error: (...a) => console.error('[bridge-err]', ...a),
          log: () => {}
        }
      });
    } catch (e) {
      console.error('[cursor-models] Warning: Could not bind bridge port:', e.message);
    }
  }

  // Sync models into ~/.helper/api-config.json
  syncApiConfig(PORT);

  // Setup Standard JSON-RPC 2.0 MCP Interface over stdio (clean input-only)
  const rl = readline.createInterface({
    input: process.stdin,
    terminal: false
  });

  function sendResponse(response) {
    process.stdout.write(JSON.stringify(response) + '\n');
  }

  let pendingOps = 0;
  let isClosing = false;

  const tryExit = async () => {
    if (!isClosing || pendingOps > 0) return;
    const forceKill = setTimeout(() => process.exit(0), 3000);
    forceKill.unref();
    if (bridgeInstance) {
      try {
        await bridgeInstance.close();
      } catch {}
    }
    process.exit(0);
  };

  rl.on('line', async (line) => {
    const trimmed = line.trim();
    if (!trimmed) return;

    let msg;
    try {
      msg = JSON.parse(trimmed);
    } catch {
      return;
    }

    pendingOps++;
    try {
      const { id, method, params } = msg;

      // Standard MCP Handshake
      if (method === 'initialize') {
        sendResponse({
          jsonrpc: '2.0',
          id,
          result: {
            protocolVersion: '2024-11-05',
            capabilities: {
              tools: {}
            },
            serverInfo: {
              name: 'cursor-models',
              version: '1.0.0'
            }
          }
        });
        return;
      }

      if (method === 'notifications/initialized') {
        // Handshake ack, no response required
        return;
      }

      if (method === 'ping') {
        sendResponse({ jsonrpc: '2.0', id, result: {} });
        return;
      }

      // List MCP Tools
      if (method === 'tools/list') {
        sendResponse({
          jsonrpc: '2.0',
          id,
          result: {
            tools: [
              {
                name: 'cursor_bridge_status',
                description: 'Check status of the Cursor Bridge HTTP service, credentials, and connectivity.',
                inputSchema: {
                  type: 'object',
                  properties: {}
                }
              },
              {
                name: 'cursor_sync_config',
                description: 'Force sync Cursor model configurations (Claude Sonnet 4.6, Opus 4.8, GPT-5.5) into ~/.helper/api-config.json.',
                inputSchema: {
                  type: 'object',
                  properties: {}
                }
              }
            ]
          }
        });
        return;
      }

      // Call MCP Tools
      if (method === 'tools/call') {
        const toolName = params?.name;
        if (toolName === 'cursor_bridge_status') {
          const h = await checkBridgeHealth(PORT);
          const hasKey = !!getCursorApiKey();
          const info = [
            `Cursor Bridge Status: ${h.online ? 'Online' : 'Offline'}`,
            `Endpoint: http://127.0.0.1:${PORT}/v1`,
            `Cursor API Key Configured: ${hasKey ? 'Yes' : 'No'}`,
            `Available Models in Helper: Claude Sonnet 4.6, Claude Opus 4.8, GPT-5.5 Preview, Claude 3.7 Sonnet`
          ].join('\n');

          sendResponse({
            jsonrpc: '2.0',
            id,
            result: {
              content: [{ type: 'text', text: info }]
            }
          });
          return;
        }

        if (toolName === 'cursor_sync_config') {
          const syncRes = syncApiConfig(PORT);
          sendResponse({
            jsonrpc: '2.0',
            id,
            result: {
              content: [{ type: 'text', text: `Sync complete: ${JSON.stringify(syncRes)}` }]
            }
          });
          return;
        }

        sendResponse({
          jsonrpc: '2.0',
          id,
          error: {
            code: -32601,
            message: `Tool not found: ${toolName}`
          }
        });
        return;
      }

      if (id !== undefined) {
        sendResponse({
          jsonrpc: '2.0',
          id,
          error: {
            code: -32601,
            message: `Method not implemented: ${method}`
          }
        });
      }
    } finally {
      pendingOps--;
      await tryExit();
    }
  });

  const cleanup = () => {
    isClosing = true;
    tryExit();
  };

  process.on('SIGTERM', cleanup);
  process.on('SIGINT', cleanup);
  rl.on('close', cleanup);
}
