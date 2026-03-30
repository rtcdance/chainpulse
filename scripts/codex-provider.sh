#!/bin/bash

CODEX_DIR="$HOME/.codex"
CONFIG_FILE="$CODEX_DIR/config.toml"
AUTH_FILE="$CODEX_DIR/auth.json"

switch_openai() {
    mkdir -p "$CODEX_DIR"
    cat > "$CONFIG_FILE" << 'EOF'
model_provider = "openai"
model = "o4-mini"
model_reasoning_effort = "medium"
disable_response_storage = true

[mcp_servers.figma]
url = "https://mcp.figma.com/mcp"

[mcp_servers.notion]
url = "https://mcp.notion.com/mcp"

[mcp_servers.linear]
url = "https://mcp.linear.app/mcp"

[projects."/Users/mingo/Applications/workspace/web3/project/chainpulse"]
trust_level = "trusted"
EOF
    echo "✅ 已切换到 OpenAI 官方配置"
}

switch_xidao() {
    mkdir -p "$CODEX_DIR"
    cat > "$CONFIG_FILE" << 'EOF'
model_provider = "xidao"
model = "gpt-4o"
model_reasoning_effort = "medium"
disable_response_storage = true

[model_providers.xidao]
name = "xidao"
base_url = "https://api.xidao.online/v1"
wire_api = "responses"
requires_openai_auth = false
api_key = "sk-IN53qrgIBvIMPhFmdZHaBG6bcGyO92pjqs2J6wlMel6E8Y7h"

[mcp_servers.figma]
url = "https://mcp.figma.com/mcp"

[mcp_servers.notion]
url = "https://mcp.notion.com/mcp"

[mcp_servers.linear]
url = "https://mcp.linear.app/mcp"

[projects."/Users/mingo/Applications/workspace/web3/project/chainpulse"]
trust_level = "trusted"
EOF
    echo "✅ 已切换到西道代理配置"
}

switch_fakestcode() {
    mkdir -p "$CODEX_DIR"
    cat > "$CONFIG_FILE" << 'EOF'
model_provider = "fakestcode"
model = "gpt-4o"
model_reasoning_effort = "medium"
disable_response_storage = true

[model_providers.fakestcode]
name = "fakestcode"
base_url = "http://proxy.fakestcode.xin/back/v1"
wire_api = "responses"
requires_openai_auth = false
api_key = "s4yFvu9oVYOFxOKaWd0M71xu0kL1opK4"

[mcp_servers.figma]
url = "https://mcp.figma.com/mcp"

[mcp_servers.notion]
url = "https://mcp.notion.com/mcp"

[mcp_servers.linear]
url = "https://mcp.linear.app/mcp"

[projects."/Users/mingo/Applications/workspace/web3/project/chainpulse"]
trust_level = "trusted"
EOF
    echo "✅ 已切换到 FakestCode 代理配置"
}

switch_anthropic() {
    mkdir -p "$CODEX_DIR"
    cat > "$CONFIG_FILE" << 'EOF'
model_provider = "anthropic"
model = "claude-sonnet-4-20250514"
model_reasoning_effort = "medium"
disable_response_storage = true

[model_providers.anthropic]
name = "anthropic"
base_url = "https://api.anthropic.com"
wire_api = "responses"
requires_openai_auth = false
api_key = "YOUR_ANTHROPIC_KEY"

[mcp_servers.figma]
url = "https://mcp.figma.com/mcp"

[mcp_servers.notion]
url = "https://mcp.notion.com/mcp"

[mcp_servers.linear]
url = "https://mcp.linear.app/mcp"

[projects."/Users/mingo/Applications/workspace/web3/project/chainpulse"]
trust_level = "trusted"
EOF
    echo "✅ 已切换到 Anthropic 官方配置"
}

show_status() {
    echo "=== Codex App 当前配置 ==="
    echo ""
    if [ -f "$CONFIG_FILE" ]; then
        echo "【config.toml】"
        echo "  model_provider: $(grep '^model_provider' "$CONFIG_FILE" | cut -d'=' -f2 | tr -d ' "')"
        echo "  model: $(grep '^model' "$CONFIG_FILE" | head -1 | cut -d'=' -f2 | tr -d ' "')"
        local base_url=$(grep '^base_url' "$CONFIG_FILE" | head -1 | cut -d'=' -f2 | tr -d ' "')
        if [ -n "$base_url" ]; then
            echo "  base_url: $base_url"
        else
            echo "  base_url: (使用内置 OpenAI URL)"
        fi
    else
        echo "  ❌ config.toml 不存在"
    fi
    echo ""
    if [ -f "$AUTH_FILE" ]; then
        echo "【auth.json】"
        echo "  auth_mode: $(grep 'auth_mode' "$AUTH_FILE" | cut -d':' -f2 | tr -d ' ",')"
    fi
}

show_help() {
    echo "Codex App 供应商管理工具"
    echo ""
    echo "用法: $0 <命令>"
    echo ""
    echo "命令:"
    echo "  openai       切换到 OpenAI 官方 (使用 ChatGPT 登录认证)"
    echo "  xidao        切换到西道代理"
    echo "  fakestcode   切换到 FakestCode 代理"
    echo "  anthropic    切换到 Anthropic 官方"
    echo "  status       显示当前配置"
    echo "  help         显示帮助信息"
    echo ""
    echo "注意事项:"
    echo "  - openai 是内置 provider，使用 ChatGPT 账户认证"
    echo "  - 其他 provider 需要配置 api_key"
    echo ""
    echo "示例:"
    echo "  $0 openai       # 切换到 OpenAI 官方"
    echo "  $0 status       # 查看当前配置"
}

case "$1" in
    openai|official)
        switch_openai
        ;;
    xidao)
        switch_xidao
        ;;
    fakestcode)
        switch_fakestcode
        ;;
    anthropic)
        switch_anthropic
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
