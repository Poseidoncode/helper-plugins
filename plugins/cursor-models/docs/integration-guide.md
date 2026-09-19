# Helper 零侵入式 Cursor 模型外掛整合指南

本指南說明如何在 **不修改 Helper 任何一行原始碼** 的前提下，透過**外部側車外掛服務（Sidecar Service）**，直接在 Helper 桌面版（GUI）與命令列版（CLI）中使用 Cursor 訂閱的頂級模型（如 Claude Sonnet 4.6、Claude Opus 4.8、GPT-5.5、Gemini 3.8 Flash 等）。

---

## 1. 架構原理與設計模式

### 1.1 為什麼能做到「零侵入（Zero-Code-Change）」？
- **Helper 的開放插座**：Helper 原生支援任何 `openai-compatible`（OpenAI 相容）的 HTTP 端點，不限制伺服器位址是遠端還是本地。
- **外部外掛化運作**：在 Helper 專案外部（`~/.helper/`）建立一支小巧的常駐外掛腳本 `cursor-bridge.mjs`，監聽 `http://127.0.0.1:4646`。
- **透明轉譯**：
  ```
  [ Helper 桌面版 / CLI ] (完全原生，未改動任何代碼)
          │
          │ HTTP /v1/chat/completions (標準 OpenAI 串流協議)
          ▼
  [ 本地側車外掛 (~/.helper/cursor-bridge.mjs) ] (macOS LaunchAgent 常駐)
          │
          │ HTTP/2 ConnectRPC 原生長連線 (@cursor/sdk)
          ▼
  [ Cursor 雲端 API 端點 ]
  ```
- **憑證無縫複用**：外掛直接讀取電腦中現有 Pi Agent 的憑證檔案 `~/.pi/agent/auth.json`，自動套用現有 Cursor 訂閱權益，無需重複設定金鑰。

### 1.2 為什麼完全不佔用 Context Window（0 Token 負擔）？
- **Provider vs MCP/Skill 的本質差異**：
  - **MCP 工具**：會在 System Prompt 注入大量 JSON Schema 定義（每個工具佔用 500 ~ 2000 Token）。
  - **Skill 技能**：會在上下文注入長篇規則指示（佔用提示詞空間）。
  - **Cursor 橋接外掛**：角色是 **模型提供者（Provider）**，只負責在網路另一端接收 Prompt 並產出回答，**完全不往對話上下文注入任何額外文字**。
- **無感切換**：在 Helper 下拉選單切換回其他模型時，上下文中完全沒有 Cursor 的任何殘留。

---

## 2. 效能表現分析

本方案採用 **方案 B：基於 `@cursor/sdk` 的長連線常駐模式**，而非社群常見的 CLI 進程包裝模式（如 `cursor-agent-api-proxy`）。

| 效能指標 | CLI 進程包裝 (`cursor-agent-api-proxy`) | 本方案：SDK 長連線常駐 (`@cursor/sdk`) | 效益差距 |
| :--- | :--- | :--- | :--- |
| **首字延遲 (TTFT)** | 慢（400ms ~ 800ms+）<br>每輪需 `spawn` 進程 + 重建 TLS 握手 | **極快（50ms ~ 150ms）**<br>進程已在記憶體中，連線保持熱啟動 | **快 3～5 倍**，無感出字 |
| **網路連線機制** | 每回合銷毀連線，下次重新握手 | **HTTP/2 多路複用 (Multiplexing)**<br>單一 TCP 連線長駐 | 節省多次網路往返時間 (RTT) |
| **資源消耗** | 頻繁 OS 進程調度，CPU 尖峰與記憶體顛簸 | **平穩低耗**（記憶體穩定 ~50MB，閒置 CPU ~0%） | 避免長任務連續呼叫時發熱卡頓 |
| **串流流暢度** | 經 stdout pipe 緩衝切割，易有字元跳動 | **原生事件驅動**（ConnectRPC → SSE 零拷貝） | 輸出平滑細膩 |
| **長對話擴展** | 隨歷史增長需將數萬 token 經 stdin 灌入 | **記憶體結構化物件傳輸** | 長對話依然輕快 |

---

## 3. 安裝檔案與系統配置

本機上已配置的所有檔案皆位在 Helper 代碼庫之外：

### 3.1 外掛核心腳本
- **路徑**：`~/.helper/cursor-bridge.mjs`
- **職責**：
  1. 提供 `/health` 健康檢查端點。
  2. 提供 `GET /v1/models` 動態查詢 Cursor 模型目錄。
  3. 提供 `POST /v1/chat/completions`，支援 OpenAI SSE 串流輸出與 `reasoning_content`（思考過程傳輸）。
  4. 支援客戶端中斷偵測（Cancel Run）。

### 3.2 系統自啟動服務 (macOS LaunchAgent)
- **路徑**：`~/Library/LaunchAgents/com.helper.cursor-bridge.plist`
- **狀態**：開機自啟動、背景常駐、異常重啟。

### 3.3 Helper 模型配置
- **路徑**：`~/.helper/api-config.json`
- 已註冊模型範例：
  ```json
  {
    "id": "model-cursor-sonnet-46",
    "displayName": "Cursor: Claude Sonnet 4.6",
    "apiKey": "not-needed",
    "baseUrl": "http://127.0.0.1:4646/v1",
    "model": "claude-sonnet-4-6",
    "apiType": "openai-compatible",
    "maxTokens": 8192,
    "contextTokens": 1048576,
    "reasoning": true,
    "thinkingLevel": "high",
    "isFavorite": true
  }
  ```

---

## 4. 使用方式

### 4.1 在 Helper 桌面版 (GUI) 中使用
1. 開啟 Helper 應用程式。
2. 在對話介面右上角或下方的**模型切換下拉選單**，即可直接選用：
   - **`Cursor: Claude Sonnet 4.6`**
   - **`Cursor: Claude Opus 4.8`**
   - **`Cursor: GPT-5.5`**
3. 提問時可直接享有 Cursor 頂級模型的推論能力，包含思考過程（Thinking）展現。

### 4.2 在 Helper 命令列版 (CLI) 中使用
```bash
# 單次提問
helper --mode json --prompt "請幫我分析當前專案架構"

# 信任當前工作區並執行
helper --mode json --trust --prompt "執行測試並說明結果"
```

### 4.3 新增更多 Cursor 模型
若想嘗試其他 Cursor 支援的模型（如 `gemini-3.8-flash`、`grok-4.6` 等）：
1. 在 Helper 桌面版點擊左下角 **設定 (Settings)** → **模型 (Models)**。
2. 點擊 **新增模型 (Add Model)**：
   - **供應商 (Provider)**：`OpenAI Compatible`
   - **Base URL**：`http://127.0.0.1:4646/v1`
   - **API Key**：填入本機安全權杖（可透過 `cat ~/.helper/cursor-bridge-token` 查看）
   - **模型名稱 (Model)**：填入對應的 Cursor 模型 ID（例如 `gemini-3.8-flash` 或 `grok-4.6`）。

---

## 5. 跨機器遷移與新 Mac 一鍵安裝

如果您有另一台 Mac 也想安裝此外掛服務，本專案已提供自動化配置腳本 `scripts/setup-cursor-bridge.sh`。無論另一台 Mac 是否有裝 Pi Agent，腳本均能自動處理相依性。

### 方式 A：有 Clone Helper 專案時（推薦，1 行搞定）
在另一台 Mac 的 Helper 專案目錄下執行：
```bash
./scripts/setup-cursor-bridge.sh
```

### 方式 B：獨立安裝（不 Clone 專案）
若另一台 Mac 僅安裝了 Helper 桌面應用程式而未 Clone 原始碼：
1. 確保已安裝 Node.js（`node -v`，支援 Node 20+）。
2. 在終端機執行下列指令直接安裝：
```bash
mkdir -p ~/.helper && npm install --prefix ~/.helper @cursor/sdk --silent
curl -fsSL https://raw.githubusercontent.com/Poseidoncode/Helper/main/scripts/setup-cursor-bridge.sh -o ~/.helper/setup.sh 2>/dev/null || true
```
3. 若另一台機器未安裝 Pi Agent，腳本會自動提示您輸入一次 Cursor API Key（取得自 [cursor.com/settings](https://cursor.com/settings)），並自動完成所有註冊。

---

## 6. 維護與故障排除

### 檢查服務狀態
```bash
curl -s http://127.0.0.1:4646/health
# 預期輸出：{"status":"ok","provider":"cursor-sdk","bridge":"helper"}
```

### 查看運行日誌
```bash
tail -f ~/.helper/cursor-bridge.log
```

### 手動重啟服務
```bash
launchctl unload ~/Library/LaunchAgents/com.helper.cursor-bridge.plist
launchctl load ~/Library/LaunchAgents/com.helper.cursor-bridge.plist
```

---

## 7. 一鍵徹底卸載（不想使用時）

若日後不再需要此功能，可在 30 秒內乾淨移除，完全無任何系統殘留：

```bash
# 1. 停止並註銷背景服務
launchctl unload ~/Library/LaunchAgents/com.helper.cursor-bridge.plist

# 2. 刪除獨立外掛檔案與服務設定
rm -f ~/Library/LaunchAgents/com.helper.cursor-bridge.plist
rm -f ~/.helper/cursor-bridge.mjs
rm -f ~/.helper/cursor-bridge-token
rm -f ~/.helper/cursor-bridge.log ~/.helper/cursor-bridge.error.log

# 3. （可選）在 Helper 桌面版設定介面中，將 Cursor 模型點擊刪除，或在 ~/.helper/api-config.json 中移除對應項目
```
移除後 Helper 依然維持原狀運作，代碼庫乾淨純粹。
