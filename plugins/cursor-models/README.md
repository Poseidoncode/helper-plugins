# Cursor Models Provider for Helper (`cursor-models`)

本外掛讓 [Helper](https://github.com/Poseidoncode/Helper) 能夠直接使用 Cursor 的旗艦模型（包括 **Claude 3.7 Sonnet**, **Claude Sonnet 4.6**, **Claude Opus 4.8**, **GPT-5.5 Preview** 等），**100% 零侵入、不改動 Helper 任何原始碼**。

---

## 🌟 特色

1. **Marketplace 原生整合**：直接在 Helper「外掛市集 (Marketplace)」點擊安裝即可啟用。
2. **雙軌模式架構**：
   - **MCP Server (stdio)**：符合 Model Context Protocol 規範，向 Helper 提供 `cursor_bridge_status` 與 `cursor_sync_config` 工具。
   - **HTTP Bridge (127.0.0.1:4646)**：背景啟動高相容性 OpenAI `/v1/chat/completions` 流式推播服務（基於 `@cursor/sdk` HTTP/2 ConnectRPC）。
3. **零手動設定**：外掛啟動時自動偵測 `~/.pi/agent/auth.json` 或 `~/.helper/cursor-api-key`，並自動在 `~/.helper/api-config.json` 登錄所有 Cursor 模型。
4. **安全加固**：
   - 本地 Bearer Token 存取控制，防止惡意網頁跨域探測（Drive-by Attack）。
   - 請求中斷即時取消雲端 Agent，杜絕資源洩漏與配額浪費。

---

## 🚀 安裝方式

### 方式 A：Helper 市集一鍵安裝（推薦）
1. 打開 Helper 桌面應用程式。
2. 進入 **設定 -> 外掛市集 (Marketplace)**。
3. 找到 **Cursor Models Provider** 並點擊 **「安裝 (Install)」**。
4. 完成！在對話視窗的模型下拉選單中即可選擇 `Cursor: Claude Sonnet 4.6`。

### 方式 B：命令列本機開發 / 獨立守護進程
```bash
# 檢查狀態
node plugins/cursor-models/src/index.mjs --status

# 單獨作為 HTTP Bridge 守護進程啟動
node plugins/cursor-models/src/index.mjs --daemon

# 執行標準 MCP stdio 模式
node plugins/cursor-models/src/index.mjs --mcp
```

### 方式 C：終端一鍵部署腳本（純 CLI / 新 Mac / 無 UI 環境）
若您在另一台電腦或無 UI 環境下，可直接執行外掛內建的一鍵安裝腳本：
```bash
./plugins/cursor-models/scripts/setup.sh
```
更多架構設計、LaunchAgent 背景服務細節與手動維運指南，請參閱 [docs/integration-guide.md](./docs/integration-guide.md)。


---

## 🛠️ MCP 工具清單

| 工具名稱 | 說明 |
|---|---|
| `cursor_bridge_status` | 檢查 HTTP 橋接器健康狀態、端口與 API 憑證偵測狀況 |
| `cursor_sync_config` | 強制將 Cursor 最新模型列表寫入 Helper 配置檔 |

---

## 📄 授權條款
Apache-2.0
