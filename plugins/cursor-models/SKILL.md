---
name: cursor-models
description: 透過 Cursor 提供的高階模型（Claude 3.7 Sonnet、Claude Sonnet 4.6、Claude Opus 4.8、GPT-5.5 Preview）進行進階程式碼編寫與架構推理。
tools:
  - cursor_bridge_status
  - cursor_sync_config
---

# Cursor Models Provider 技能

此外掛為 Helper 注入來自 Cursor 的頂尖大語言模型，提供極致的推理與編碼能力。

## 🎯 支援的模型

1. **Cursor: Claude Sonnet 4.6** (`claude-sonnet-4-6`)
   - 兼具極速響應與頂尖軟體架構分析能力。
2. **Cursor: Claude Opus 4.8** (`claude-opus-4-8`)
   - 深度數學證明、超長程式碼重構與嚴謹邏輯審查。
3. **Cursor: GPT-5.5 Preview** (`gpt-5-5-preview`)
   - 跨語言生成與複雜多任務規劃。
4. **Cursor: Claude 3.7 Sonnet** (`claude-3-7-sonnet`)
   - 混合架構思考模型。

## 🛠️ 提供之 MCP 工具

- `cursor_bridge_status`: 檢測 Cursor 橋接服務的連線狀態、端點健康狀況與 API 憑證偵測。
- `cursor_sync_config`: 強制將 Cursor 模型列表寫入 `~/.helper/api-config.json`，以便在 Helper 介面立即選取。

## 💡 使用範例

- 「請檢查 Cursor 橋接服務是否連線正常。」
- 「幫我把 Cursor 模型同步到 Helper 配置檔。」
