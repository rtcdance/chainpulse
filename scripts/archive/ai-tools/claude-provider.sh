#!/bin/bash

CLAUDE_DIR="$HOME/.claude"
SETTINGS_FILE="$CLAUDE_DIR/settings.json"

switch_anthropic() {
    mkdir -p "$CLAUDE_DIR"
    cat > "$SETTINGS_FILE" << 'EOF'
{
  "env": {
    "ANTHROPIC_AUTH_TOKEN": "YOUR_ANTHROPIC_KEY",
    "ANTHROPIC_BASE_URL": "https://api.anthropic.com"
  },
  "permissions": {
    "allow": [],
    "deny": []
  },
  "effortLevel": "medium"
}
EOF
    echo "✅ 已切换到 Anthropic 官方配置"
    echo "⚠️  请替换 YOUR_ANTHROPIC_KEY 为你的 Anthropic API Key"
}

switch_xidao() {
    mkdir -p "$CLAUDE_DIR"
    cat > "$SETTINGS_FILE" << 'EOF'
{
  "env": {
    "ANTHROPIC_AUTH_TOKEN": "sk-IN53qrgIBvIMPhFmdZHaBG6bcGyO92pjqs2J6wlMel6E8Y7h",
    "ANTHROPIC_BASE_URL": "https://api.xidao.online"
  },
  "permissions": {
    "allow": [],
    "deny": []
  },
  "effortLevel": "medium"
}
EOF
    echo "✅ 已切换到西道代理配置"
}

switch_fakestcode() {
    mkdir -p "$CLAUDE_DIR"
    cat > "$SETTINGS_FILE" << 'EOF'
{
  "env": {
    "ANTHROPIC_AUTH_TOKEN": "s4yFvu9oVYOFxOKaWd0M71xu0kL1opK4",
    "ANTHROPIC_BASE_URL": "http://proxy.fakestcode.xin/back"
  },
  "permissions": {
    "allow": [],
    "deny": []
  },
  "effortLevel": "medium"
}
EOF
    echo "✅ 已切换到 FakestCode 代理配置"
}

switch_openai_proxy() {
    mkdir -p "$CLAUDE_DIR"
    cat > "$SETTINGS_FILE" << 'EOF'
{
  "env": {
    "ANTHROPIC_AUTH_TOKEN": "YOUR_OPENAI_KEY",
    "ANTHROPIC_BASE_URL": "https://api.openai.com/v1"
  },
  "permissions": {
    "allow": [],
    "deny": []
  },
  "effortLevel": "medium"
}
EOF
    echo "✅ 已切换到 OpenAI 代理配置"
    echo "⚠️  请替换 YOUR_OPENAI_KEY 为你的 OpenAI API Key"
}

set_api_key() {
    local api_key=$1
    if [ -z "$api_key" ]; then
        echo "❌ 请提供 API Key"
        exit 1
    fi
    
    if [ ! -f "$SETTINGS_FILE" ]; then
        echo "❌ 配置文件不存在，请先选择一个供应商"
        exit 1
    fi
    
    local temp_file=$(mktemp)
    if command -v jq &> /dev/null; then
        jq ".env.ANTHROPIC_AUTH_TOKEN = \"$api_key\"" "$SETTINGS_FILE" > "$temp_file" && mv "$temp_file" "$SETTINGS_FILE"
        echo "✅ 已更新 API Key"
    else
        sed -i '' "s/\"ANTHROPIC_AUTH_TOKEN\": \"[^\"]*\"/\"ANTHROPIC_AUTH_TOKEN\": \"$api_key\"/" "$SETTINGS_FILE"
        echo "✅ 已更新 API Key"
    fi
}

show_status() {
    echo "=== Claude CLI 当前配置 ==="
    echo ""
    if [ -f "$SETTINGS_FILE" ]; then
        echo "【settings.json】"
        local token=$(grep 'ANTHROPIC_AUTH_TOKEN' "$SETTINGS_FILE" | cut -d':' -f2 | tr -d ' ",' | head -1)
        local base_url=$(grep 'ANTHROPIC_BASE_URL' "$SETTINGS_FILE" | cut -d':' -f2 | tr -d ' ",' | head -1)
        
        if [ ${#token} -gt 20 ]; then
            echo "  API Key: ${token:0:10}...${token: -4}"
        else
            echo "  API Key: $token"
        fi
        echo "  Base URL: $base_url"
    else
        echo "  ❌ settings.json 不存在"
    fi
}

show_help() {
    echo "Claude CLI 供应商管理工具"
    echo ""
    echo "用法: $0 <命令> [参数]"
    echo ""
    echo "命令:"
    echo "  anthropic    切换到 Anthropic 官方"
    echo "  xidao        切换到西道代理"
    echo "  fakestcode   切换到 FakestCode 代理"
    echo "  openai       切换到 OpenAI 代理"
    echo "  set-key <key>   设置当前供应商的 API Key"
    echo "  status       显示当前配置"
    echo "  help         显示帮助信息"
    echo ""
    echo "供应商详情:"
    echo "  ┌─────────────┬─────────────────────────────────────┬─────────────────┐"
    echo "  │ 供应商      │ Base URL                            │ 认证方式        │"
    echo "  ├─────────────┼─────────────────────────────────────┼─────────────────┤"
    echo "  │ anthropic   │ https://api.anthropic.com           │ Anthropic Key   │"
    echo "  │ xidao       │ https://api.xidao.online            │ API Key         │"
    echo "  │ fakestcode  │ http://proxy.fakestcode.xin/back    │ API Key         │"
    echo "  │ openai      │ https://api.openai.com/v1           │ OpenAI Key      │"
    echo "  └─────────────┴─────────────────────────────────────┴─────────────────┘"
    echo ""
    echo "示例:"
    echo "  $0 xidao                    # 切换到西道代理"
    echo "  $0 set-key sk-xxxxx         # 设置 API Key"
    echo "  $0 status                   # 查看当前配置"
}

case "$1" in
    anthropic|official)
        switch_anthropic
        ;;
    xidao)
        switch_xidao
        ;;
    fakestcode)
        switch_fakestcode
        ;;
    openai)
        switch_openai_proxy
        ;;
    set-key)
        if [ -z "$2" ]; then
            echo "❌ 请提供 API Key"
            show_help
            exit 1
        fi
        set_api_key "$2"
        ;;
    status)
        show_status
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        show_help
        ;;
esac
