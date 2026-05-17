#!/usr/bin/env bash
# claude-plugin-profile.sh - 快速切换 Claude CLI 插件 Profile
# 兼容 macOS 默认 bash 3.2
# 用法:
#   ./claude-plugin-profile.sh list              # 列出所有 Profile
#   ./claude-plugin-profile.sh apply <name>       # 应用指定 Profile 到当前项目
#   ./claude-plugin-profile.sh current            # 显示当前项目的插件配置
#   ./claude-plugin-profile.sh save <name>        # 保存当前项目插件配置为 Profile

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROFILES_DIR="${SCRIPT_DIR}/.claude-plugin-profiles"
CLAUDE_DIR="$(pwd)/.claude"

# Profile 定义：每个 Profile 对应一个函数返回插件列表
profile_go_backend() {
    cat <<'EOF'
security-guidance@claude-plugins-official
code-review@claude-plugins-official
commit-commands@claude-plugins-official
code-simplifier@claude-plugins-official
hookify@claude-plugins-official
feature-dev@claude-plugins-official
pr-review-toolkit@claude-plugins-official
skill-creator@claude-plugins-official
session-report@claude-plugins-official
claude-md-management@claude-plugins-official
gopls-lsp@claude-plugins-official
superpowers@superpowers-marketplace
EOF
}

profile_frontend() {
    cat <<'EOF'
frontend-design@claude-plugins-official
commit-commands@claude-plugins-official
code-simplifier@claude-plugins-official
skill-creator@claude-plugins-official
code-review@claude-plugins-official
session-report@claude-plugins-official
claude-md-management@claude-plugins-official
typescript-lsp@claude-plugins-official
superpowers@superpowers-marketplace
EOF
}

profile_fullstack() {
    cat <<'EOF'
security-guidance@claude-plugins-official
code-review@claude-plugins-official
pr-review-toolkit@claude-plugins-official
commit-commands@claude-plugins-official
code-simplifier@claude-plugins-official
hookify@claude-plugins-official
feature-dev@claude-plugins-official
frontend-design@claude-plugins-official
skill-creator@claude-plugins-official
claude-md-management@claude-plugins-official
session-report@claude-plugins-official
superpowers@superpowers-marketplace
EOF
}

profile_minimal() {
    cat <<'EOF'
security-guidance@claude-plugins-official
commit-commands@claude-plugins-official
claude-md-management@claude-plugins-official
EOF
}

profile_lightweight() {
    cat <<'EOF'
security-guidance@claude-plugins-official
code-review@claude-plugins-official
commit-commands@claude-plugins-official
code-simplifier@claude-plugins-official
skill-creator@claude-plugins-official
claude-md-management@claude-plugins-official
EOF
}

# Profile 描述
profile_desc_go_backend="Go/后端项目（12个插件，含 Go LSP）"
profile_desc_frontend="前端项目（TypeScript LSP + 前端设计）"
profile_desc_fullstack="全栈项目（Go/后端 + 前端完整套件）"
profile_desc_minimal="最低配置（3个核心插件，最低 Token）"
profile_desc_lightweight="轻量配置（6个常用插件，快速启动）"

# MCP 配置
mcp_go_backend="context7"
mcp_frontend="context7 playwright"
mcp_fullstack="context7 playwright"
mcp_minimal=""
mcp_lightweight="context7"

# 获取插件列表函数
get_plugins() {
    local name="$1"
    eval "profile_${name}" 2>/dev/null || return 1
}

# 获取 MCP 配置
get_mcp() {
    local name="$1"
    eval "echo \"\$mcp_${name}\"" 2>/dev/null
}

# 获取描述
get_desc() {
    local name="$1"
    eval "echo \"\$profile_desc_${name}\"" 2>/dev/null
}

# 所有 Profile 名称
PROFILES_LIST="go-backend frontend fullstack minimal lightweight"

cmd_list() {
    echo "📋 可用 Profile:"
    echo "----------------------------------------------"
    for name in $PROFILES_LIST; do
        # Convert hyphens to underscores for function lookup
        local func_name="${name//-/_}"
        local desc
        desc="$(get_desc "$func_name")"
        printf "  %-15s  %s\n" "$name" "$desc"
    done
    echo ""
    echo "用法: $0 apply <profile-name>"
}

cmd_current() {
    echo "📍 当前项目: $(pwd)"
    echo ""
    if [ -f "${CLAUDE_DIR}/settings.json" ]; then
        echo "已启用插件:"
        jq -r '.enabledPlugins | to_entries[] | select(.value == true) | "  ✓ \(.key)"' "${CLAUDE_DIR}/settings.json" 2>/dev/null || echo "  (无)"
    else
        echo "(项目级配置文件不存在)"
    fi
}

cmd_apply() {
    local profile_name="${1:?用法: $0 apply <profile-name>}"
    local func_name="${profile_name//-/_}"

    if ! get_plugins "$func_name" > /dev/null 2>&1; then
        echo "❌ 未知的 Profile: ${profile_name}"
        echo ""
        cmd_list
        exit 1
    fi

    echo "🔧 应用 Profile: ${profile_name}"
    echo ""

    # 创建新的配置
    mkdir -p "${CLAUDE_DIR}"

    # 构建 JSON
    local settings_json='{"enabledPlugins":{'
    local first=true

    while IFS= read -r plugin; do
        plugin="$(echo "$plugin" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')"
        [ -z "$plugin" ] && continue

        if [ "$first" = true ]; then
            first=false
        else
            settings_json+=","
        fi
        settings_json+="\"${plugin}\":true"
        echo "  ✅ 启用: ${plugin}"
    done < <(get_plugins "$func_name")

    settings_json+='}}'

    echo "$settings_json" | jq '.' > "${CLAUDE_DIR}/settings.json"

    # 处理 MCP 服务器
    local mcp_config
    mcp_config="$(get_mcp "$func_name")"

    if [ -n "$mcp_config" ]; then
        echo ""
        echo "🔌 配置 MCP 服务器..."
        local mcp_json='{"mcpServers":{'
        local mcp_first=true

        for mcp in $mcp_config; do
            if [ "$mcp_first" = true ]; then
                mcp_first=false
            else
                mcp_json+=","
            fi

            case "$mcp" in
                context7)
                    mcp_json+="\"${mcp}\":{\"type\":\"stdio\",\"command\":\"npx\",\"args\":[\"-y\",\"@upstash/context7-mcp\"],\"env\":{}}"
                    ;;
                playwright)
                    mcp_json+="\"${mcp}\":{\"type\":\"stdio\",\"command\":\"npx\",\"args\":[\"-y\",\"@playwright/mcp@latest\"],\"env\":{}}"
                    ;;
            esac
            echo "  ✅ MCP: ${mcp}"
        done

        mcp_json+='}}'
        echo "$mcp_json" | jq '.' > "${CLAUDE_DIR}/.mcp.json"
    elif [ -f "${CLAUDE_DIR}/.mcp.json" ]; then
        echo ""
        echo "🔌 ${profile_name} Profile 不需要 MCP 服务器，移除 .mcp.json"
        rm -f "${CLAUDE_DIR}/.mcp.json"
    fi

    echo ""
    echo "✅ Profile '${profile_name}' 已应用到 $(pwd)"
}

cmd_save() {
    local profile_name="${1:?用法: $0 save <profile-name>}"

    if [ ! -f "${CLAUDE_DIR}/settings.json" ]; then
        echo "❌ 当前项目没有插件配置"
        exit 1
    fi

    mkdir -p "${PROFILES_DIR}"
    cp "${CLAUDE_DIR}/settings.json" "${PROFILES_DIR}/${profile_name}.json"

    # 如果有 MCP 配置也保存
    if [ -f "${CLAUDE_DIR}/.mcp.json" ]; then
        cp "${CLAUDE_DIR}/.mcp.json" "${PROFILES_DIR}/${profile_name}-mcp.json"
    fi

    echo "✅ 已将当前配置保存为 Profile: ${profile_name}"
    echo "   保存在: ${PROFILES_DIR}/${profile_name}.json"
}

# 主入口
case "${1:-help}" in
    list|ls)    cmd_list ;;
    current)    cmd_current ;;
    apply)      cmd_apply "${2:-}" ;;
    save)       cmd_save "${2:-}" ;;
    help|--help|-h)
        echo "Claude CLI 插件 Profile 管理工具"
        echo ""
        echo "用法:"
        echo "  $0 list              列出所有可用 Profile"
        echo "  $0 apply <name>      应用 Profile 到当前项目"
        echo "  $0 current           显示当前项目的插件配置"
        echo "  $0 save <name>       保存当前配置为 Profile"
        echo ""
        echo "内置 Profile: go-backend, frontend, fullstack, minimal, lightweight"
        ;;
    *)          echo "❌ 未知命令: ${1}"; cmd_list; exit 1 ;;
esac
