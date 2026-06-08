#!/usr/bin/env bash
# claude-plugin-enable.sh - 按需启用 Claude CLI 插件
# 用法:
#   ./claude-plugin-enable.sh code-review    # 启用代码审查
#   ./claude-plugin-enable.sh review         # 启用审查套件（code-review + pr-review-toolkit）
#   ./claude-plugin-enable.sh code-simplify  # 启用代码简化
#   ./claude-plugin-enable.sh feature        # 启用新功能开发套件
#   ./claude-plugin-enable.sh go             # 启用 Go 开发套件
#   ./claude-plugin-enable.sh all            # 启用所有插件
#   ./claude-plugin-enable.sh --reset        # 重置到最小配置

set -euo pipefail

CLAUDE_DIR="$(pwd)/.claude"
SETTINGS="${CLAUDE_DIR}/settings.json"

# 确保目录存在
mkdir -p "${CLAUDE_DIR}"

# 读取当前配置
current_plugins=""
if [ -f "$SETTINGS" ]; then
    current_plugins=$(jq -r '.enabledPlugins | to_entries[] | select(.value == true) | .key' "$SETTINGS" 2>/dev/null || true)
fi

enable_plugin() {
    local plugin="$1"
    if ! echo "$current_plugins" | grep -q "$plugin"; then
        echo "  ✅ 启用: ${plugin}"
        current_plugins="${current_plugins}"$'\n'"${plugin}"
    else
        echo "  ℹ️  已启用: ${plugin}"
    fi
}

# 场景包
case "${1:?用法: $0 <功能> | 功能: code-review, review, code-simplify, feature, go, all, --reset}" in
    code-review)
        enable_plugin "code-review@claude-plugins-official"
        ;;
    review)
        enable_plugin "code-review@claude-plugins-official"
        enable_plugin "pr-review-toolkit@claude-plugins-official"
        ;;
    code-simplify)
        enable_plugin "code-simplifier@claude-plugins-official"
        enable_plugin "code-review@claude-plugins-official"
        ;;
    feature)
        enable_plugin "feature-dev@claude-plugins-official"
        enable_plugin "code-review@claude-plugins-official"
        enable_plugin "pr-review-toolkit@claude-plugins-official"
        enable_plugin "skill-creator@claude-plugins-official"
        ;;
    go)
        enable_plugin "gopls-lsp@claude-plugins-official"
        enable_plugin "code-review@claude-plugins-official"
        enable_plugin "code-simplifier@claude-plugins-official"
        ;;
    all)
        enable_plugin "security-guidance@claude-plugins-official"
        enable_plugin "commit-commands@claude-plugins-official"
        enable_plugin "code-review@claude-plugins-official"
        enable_plugin "code-simplifier@claude-plugins-official"
        enable_plugin "hookify@claude-plugins-official"
        enable_plugin "feature-dev@claude-plugins-official"
        enable_plugin "pr-review-toolkit@claude-plugins-official"
        enable_plugin "skill-creator@claude-plugins-official"
        enable_plugin "session-report@claude-plugins-official"
        enable_plugin "claude-md-management@claude-plugins-official"
        enable_plugin "gopls-lsp@claude-plugins-official"
        enable_plugin "superpowers@superpowers-marketplace"
        ;;
    --reset)
        echo "🔄 重置到最小配置..."
        current_plugins=""
        enable_plugin "security-guidance@claude-plugins-official"
        enable_plugin "commit-commands@claude-plugins-official"
        ;;
    *)
        echo "❌ 未知功能: ${1}"
        echo ""
        echo "可用功能:"
        echo "  code-review    - 启用代码审查 (~3,000 tokens)"
        echo "  review         - 启用审查套件 (~7,000 tokens)"
        echo "  code-simplify  - 启用代码简化 (~5,500 tokens)"
        echo "  feature        - 启用新功能开发 (~10,000 tokens)"
        echo "  go             - 启用 Go 开发套件 (~13,500 tokens)"
        echo "  all            - 启用所有插件 (~35,000 tokens)"
        echo "  --reset        - 重置到最小配置 (~4,000 tokens)"
        exit 1
        ;;
esac

# 写入配置
mkdir -p "${CLAUDE_DIR}"
settings_json='{"enabledPlugins":{'
first=true

while IFS= read -r plugin; do
    plugin="$(echo "$plugin" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')"
    [ -z "$plugin" ] && continue
    if [ "$first" = true ]; then
        first=false
    else
        settings_json+=","
    fi
    settings_json+="\"${plugin}\":true"
done <<< "$current_plugins"

settings_json+='}}'
echo "$settings_json" | jq '.' > "${SETTINGS}"

echo ""
echo "✅ 已更新配置: ${SETTINGS}"
