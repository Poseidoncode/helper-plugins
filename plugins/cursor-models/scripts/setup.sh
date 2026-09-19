#!/bin/bash
# ==============================================================================
# Helper Cursor Bridge - 跨機器一鍵安裝與部署腳本 (macOS)
# ==============================================================================
set -euo pipefail

COLOR_GREEN="\033[0;32m"
COLOR_BLUE="\033[0;34m"
COLOR_YELLOW="\033[1;33m"
COLOR_RED="\033[0;31m"
COLOR_RESET="\033[0m"

echo -e "${COLOR_BLUE}=== 開始安裝 Helper Cursor 模型外掛服務 (Sidecar Bridge) ===${COLOR_RESET}"

# 1. 檢查 Node.js 環境
if ! command -v node >/dev/null 2>&1; then
  echo -e "${COLOR_RED}[錯誤] 系統未安裝 Node.js。請先透過 brew install node 或 nvm 安裝 Node.js (>= 20)。${COLOR_RESET}"
  exit 1
fi

NODE_BIN=$(command -v node)
NODE_VER=$(node -v)
echo -e "${COLOR_GREEN}✓ 偵測到 Node.js:${COLOR_RESET} $NODE_BIN ($NODE_VER)"

# 2. 建立目錄
HELPER_DIR="$HOME/.helper"
mkdir -p "$HELPER_DIR"
mkdir -p "$HOME/Library/LaunchAgents"

# 3. 解析或安裝 @cursor/sdk
PI_SDK_PATH="$HOME/.pi/agent/npm/node_modules/@cursor/sdk"
LOCAL_SDK_PATH="$HELPER_DIR/node_modules/@cursor/sdk"

if [ -d "$PI_SDK_PATH" ]; then
  echo -e "${COLOR_GREEN}✓ 偵測到本機已存在 Pi Agent 之 @cursor/sdk，直接複用${COLOR_RESET}"
elif [ -d "$LOCAL_SDK_PATH" ]; then
  echo -e "${COLOR_GREEN}✓ 偵測到 ~/.helper/node_modules/@cursor/sdk 已就緒${COLOR_RESET}"
else
  echo -e "${COLOR_YELLOW}正在為 Helper 獨立安裝 @cursor/sdk（此動作不影響全域環境）...${COLOR_RESET}"
  npm install --prefix "$HELPER_DIR" @cursor/sdk --silent
  echo -e "${COLOR_GREEN}✓ @cursor/sdk 安裝完成${COLOR_RESET}"
fi

# 4. 解析或獲取 Cursor API Key
PI_AUTH_FILE="$HOME/.pi/agent/auth.json"
KEY_FILE="$HELPER_DIR/cursor-api-key"
CURSOR_KEY=""

if [ -n "${CURSOR_API_KEY:-}" ]; then
  CURSOR_KEY="$CURSOR_API_KEY"
  echo -e "${COLOR_GREEN}✓ 使用環境變數 CURSOR_API_KEY${COLOR_RESET}"
elif [ -f "$KEY_FILE" ]; then
  CURSOR_KEY=$(cat "$KEY_FILE" | tr -d '[:space:]')
  echo -e "${COLOR_GREEN}✓ 使用已有設定檔 ~/.helper/cursor-api-key${COLOR_RESET}"
elif [ -f "$PI_AUTH_FILE" ]; then
  # 嘗試從 Pi Agent auth.json 提取
  CURSOR_KEY=$(node -e '
    try {
      const a = JSON.parse(require("fs").readFileSync("'"$PI_AUTH_FILE"'", "utf8"));
      if (a.cursor && a.cursor.key) console.log(a.cursor.key);
    } catch {}
  ' 2>/dev/null || true)
  if [ -n "$CURSOR_KEY" ]; then
    echo -e "${COLOR_GREEN}✓ 自動從 ~/.pi/agent/auth.json 讀取 Cursor 金鑰${COLOR_RESET}"
  fi
fi

# 若皆未找到，引導使用者輸入
if [ -z "$CURSOR_KEY" ]; then
  echo -e "${COLOR_YELLOW}"
  echo "未偵測到 Cursor API Key。"
  echo "請前往 https://cursor.com/settings (或 Cursor Dashboard → API Keys) 獲取 API Key。"
  echo -e "${COLOR_RESET}"
  read -r -s -p "請貼上您的 Cursor API Key: " INPUT_KEY
  echo ""
  if [ -z "$INPUT_KEY" ]; then
    echo -e "${COLOR_RED}[錯誤] 未提供 API Key，安裝終止。${COLOR_RESET}"
    exit 1
  fi
  CURSOR_KEY="$INPUT_KEY"
  echo "$CURSOR_KEY" > "$KEY_FILE"
  chmod 600 "$KEY_FILE"
  echo -e "${COLOR_GREEN}✓ 金鑰已安全儲存至 ~/.helper/cursor-api-key (權限 0600)${COLOR_RESET}"
fi

# 5. 部署橋接服務核心腳本 ~/.helper/cursor-bridge.mjs
SCRIPT_SRC="$(dirname "$0")/../.helper/cursor-bridge.mjs"
if [ ! -f "$SCRIPT_SRC" ]; then
  SCRIPT_SRC="$HOME/.helper/cursor-bridge.mjs"
fi

if [ -f "$SCRIPT_SRC" ] && [ "$SCRIPT_SRC" != "$HELPER_DIR/cursor-bridge.mjs" ]; then
  cp "$SCRIPT_SRC" "$HELPER_DIR/cursor-bridge.mjs"
fi
chmod 700 "$HELPER_DIR/cursor-bridge.mjs"
echo -e "${COLOR_GREEN}✓ 部署核心腳本至 ~/.helper/cursor-bridge.mjs${COLOR_RESET}"

# 6. 生成本機安全權杖 ~/.helper/cursor-bridge-token
TOKEN_FILE="$HELPER_DIR/cursor-bridge-token"
if [ ! -f "$TOKEN_FILE" ]; then
  node -e '
    const crypto = require("crypto");
    const token = "cbr_" + crypto.randomBytes(16).toString("hex");
    require("fs").writeFileSync("'"$TOKEN_FILE"'", token, { mode: 0o600 });
  '
fi
BRIDGE_TOKEN=$(cat "$TOKEN_FILE" | tr -d '[:space:]')
chmod 600 "$TOKEN_FILE"
echo -e "${COLOR_GREEN}✓ 本機通訊安全權杖已就緒 (Token: ${BRIDGE_TOKEN:0:10}...)${COLOR_RESET}"

# 7. 配置並註冊 macOS LaunchAgent
PLIST_PATH="$HOME/Library/LaunchAgents/com.helper.cursor-bridge.plist"
NODE_DIR=$(dirname "$NODE_BIN")

cat <<EOF > "$PLIST_PATH"
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.helper.cursor-bridge</string>
    <key>WorkingDirectory</key>
    <string>$HELPER_DIR</string>
    <key>ProgramArguments</key>
    <array>
        <string>$NODE_BIN</string>
        <string>$HELPER_DIR/cursor-bridge.mjs</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <dict>
        <key>SuccessfulExit</key>
        <false/>
    </dict>
    <key>ThrottleInterval</key>
    <integer>10</integer>
    <key>StandardOutPath</key>
    <string>$HELPER_DIR/cursor-bridge.log</string>
    <key>StandardErrorPath</key>
    <string>$HELPER_DIR/cursor-bridge.error.log</string>
    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>$NODE_DIR:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin</string>
    </dict>
</dict>
</plist>
EOF

launchctl unload "$PLIST_PATH" 2>/dev/null || true
launchctl load "$PLIST_PATH"
echo -e "${COLOR_GREEN}✓ 系統背景常駐服務註冊完成 (com.helper.cursor-bridge)${COLOR_RESET}"

# 8. 更新 ~/.helper/api-config.json
node -e '
  const fs = require("fs");
  const path = require("path");
  const os = require("os");
  const cfgPath = path.join(os.homedir(), ".helper", "api-config.json");
  let cfg = { activeModelId: "model-cursor-sonnet-46", models: [] };
  if (fs.existsSync(cfgPath)) {
    try { cfg = JSON.parse(fs.readFileSync(cfgPath, "utf8")); } catch {}
  }
  if (!Array.isArray(cfg.models)) cfg.models = [];

  const token = "'"$BRIDGE_TOKEN"'";
  const cursorModels = [
    {
      id: "model-cursor-sonnet-46",
      displayName: "Cursor: Claude Sonnet 4.6",
      apiKey: token,
      baseUrl: "http://127.0.0.1:4646/v1",
      model: "claude-sonnet-4-6",
      apiType: "openai-compatible",
      maxTokens: 8192,
      contextTokens: 1048576,
      reasoning: true,
      thinkingLevel: "high",
      isFavorite: true
    },
    {
      id: "model-cursor-opus-48",
      displayName: "Cursor: Claude Opus 4.8",
      apiKey: token,
      baseUrl: "http://127.0.0.1:4646/v1",
      model: "claude-opus-4-8",
      apiType: "openai-compatible",
      maxTokens: 8192,
      contextTokens: 1048576,
      reasoning: true,
      thinkingLevel: "high",
      isFavorite: true
    },
    {
      id: "model-cursor-gpt-55",
      displayName: "Cursor: GPT-5.5",
      apiKey: token,
      baseUrl: "http://127.0.0.1:4646/v1",
      model: "gpt-5.5",
      apiType: "openai-compatible",
      maxTokens: 8192,
      contextTokens: 1048576,
      reasoning: true,
      thinkingLevel: "medium",
      isFavorite: true
    }
  ];

  for (const cm of cursorModels) {
    const idx = cfg.models.findIndex(m => m.id === cm.id);
    if (idx >= 0) {
      cfg.models[idx] = Object.assign({}, cfg.models[idx], cm);
    } else {
      cfg.models.push(cm);
    }
  }
  if (!cfg.activeModelId) cfg.activeModelId = "model-cursor-sonnet-46";
  fs.writeFileSync(cfgPath, JSON.stringify(cfg, null, 2));
'
echo -e "${COLOR_GREEN}✓ Helper 模型配置已自動加入 ~/.helper/api-config.json${COLOR_RESET}"

# 9. 健康檢查
echo -e "${COLOR_BLUE}正在驗證服務連線...${COLOR_RESET}"
sleep 1
HEALTH=$(curl -s http://127.0.0.1:4646/health || true)
if [[ "$HEALTH" == *"\"status\":\"ok\""* ]]; then
  echo -e "${COLOR_GREEN}🎉 安裝成功！Cursor 模型外掛服務正在本機運作。${COLOR_RESET}"
  echo -e "現在您可以直接打開 Helper 桌面版，在模型下拉選單中直接選擇 Cursor 模型開始使用！"
else
  echo -e "${COLOR_YELLOW}服務已啟動，請查看日誌確認狀態：tail -n 20 ~/.helper/cursor-bridge.log${COLOR_RESET}"
fi
