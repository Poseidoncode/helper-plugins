import fs from "node:fs";
import path from "node:path";
import os from "node:os";
import { getOrCreateBridgeToken } from "./cursor-bridge.mjs";

const HOME = os.homedir();
const HELPER_DIR = path.join(HOME, ".helper");
const CONFIG_FILE = path.join(HELPER_DIR, "api-config.json");

export function syncApiConfig(port = 4646) {
  try {
    if (!fs.existsSync(HELPER_DIR)) {
      fs.mkdirSync(HELPER_DIR, { recursive: true, mode: 0o700 });
    }

    const token = getOrCreateBridgeToken();
    const baseUrl = `http://127.0.0.1:${port}/v1`;

    let config = { activeModelId: "", models: [] };
    if (fs.existsSync(CONFIG_FILE)) {
      try {
        const raw = fs.readFileSync(CONFIG_FILE, "utf8");
        config = JSON.parse(raw);
        if (!Array.isArray(config.models)) {
          config.models = [];
        }
      } catch (err) {
        const corruptBackup = `${CONFIG_FILE}.corrupt.${Date.now()}`;
        try { fs.copyFileSync(CONFIG_FILE, corruptBackup); } catch {}
        console.error(`[cursor-models] Warning: could not parse api-config.json (backed up to ${corruptBackup}):`, err.message);
        return {
          success: false,
          error: `Corrupt api-config.json, backup created at ${corruptBackup}`
        };
      }
    }

    const defaultCursorModels = [
      {
        id: "model-cursor-sonnet-46",
        displayName: "Cursor: Claude Sonnet 4.6",
        apiKey: token,
        baseUrl: baseUrl,
        model: "claude-sonnet-4-6",
        apiType: "openai-compatible",
        maxTokens: 8192,
        reasoning: true,
        thinkingLevel: "high",
        isFavorite: true
      },
      {
        id: "model-cursor-opus-48",
        displayName: "Cursor: Claude Opus 4.8",
        apiKey: token,
        baseUrl: baseUrl,
        model: "claude-opus-4-8",
        apiType: "openai-compatible",
        maxTokens: 8192,
        reasoning: true,
        thinkingLevel: "high",
        isFavorite: true
      },
      {
        id: "model-cursor-gpt-55",
        displayName: "Cursor: GPT-5.5 Preview",
        apiKey: token,
        baseUrl: baseUrl,
        model: "gpt-5-5-preview",
        apiType: "openai-compatible",
        maxTokens: 8192,
        reasoning: true,
        thinkingLevel: "high",
        isFavorite: true
      },
      {
        id: "model-cursor-37-sonnet",
        displayName: "Cursor: Claude 3.7 Sonnet",
        apiKey: token,
        baseUrl: baseUrl,
        model: "claude-3-7-sonnet",
        apiType: "openai-compatible",
        maxTokens: 8192,
        reasoning: true,
        thinkingLevel: "high",
        isFavorite: false
      }
    ];

    let updated = false;
    for (const cm of defaultCursorModels) {
      const idx = config.models.findIndex((m) => m && m.id === cm.id);
      if (idx === -1) {
        config.models.push(cm);
        updated = true;
      } else {
        const existing = config.models[idx];
        if (existing.baseUrl !== cm.baseUrl || (token && existing.apiKey !== token)) {
          config.models[idx] = {
            ...existing,
            baseUrl: cm.baseUrl,
            apiKey: token || existing.apiKey
          };
          updated = true;
        }
      }
    }

    if (updated) {
      const tmpFile = `${CONFIG_FILE}.tmp.${Date.now()}.${Math.random().toString(36).slice(2, 8)}`;
      fs.writeFileSync(tmpFile, JSON.stringify(config, null, 2), { mode: 0o600 });
      fs.renameSync(tmpFile, CONFIG_FILE);
    }

    return {
      success: true,
      updated,
      modelsCount: config.models.length,
      cursorModels: defaultCursorModels.map((m) => m.id)
    };
  } catch (err) {
    return {
      success: false,
      error: err.message
    };
  }
}
