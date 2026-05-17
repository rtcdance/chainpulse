#!/usr/bin/env bash
# claude-plugin-stats.sh - 查看 Claude CLI 插件状态与 Token 开销估算
# 用法: ./claude-plugin-stats.sh

set -euo pipefail

CLAUDE_CONFIG="$HOME/.claude.json"

echo "=============================================="
echo "  Claude CLI 插件状态 & Token 开销估算"
echo "=============================================="
echo ""

# 获取已安装插件列表
echo "📦 已安装插件:"
echo "----------------------------------------------"
claude plugin list 2>/dev/null | tail -n +2

echo ""
echo "📊 Token 开销估算（参考值）:"
echo "----------------------------------------------"
printf "%-30s %s\n" "插件" "Token 估算"

# 按类型估算 Token 开销
plugin_estimates() {
    local total=0

    # 审查类
    printf "%-30s ~%s\n" "code-review" "3,000"
    printf "%-30s ~%s\n" "pr-review-toolkit" "4,000"
    printf "%-30s ~%s\n" "security-guidance" "2,500"
    total=$((total + 9500))

    # 工作流类
    printf "%-30s ~%s\n" "superpowers" "5,000"
    printf "%-30s ~%s\n" "feature-dev" "3,500"
    printf "%-30s ~%s\n" "commit-commands" "1,500"
    printf "%-30s ~%s\n" "session-report" "800"
    total=$((total + 10800))

    # 工具类
    printf "%-30s ~%s\n" "hookify" "1,000"
    printf "%-30s ~%s\n" "skill-creator" "2,000"
    printf "%-30s ~%s\n" "claude-md-management" "1,200"
    printf "%-30s ~%s\n" "code-simplifier" "2,500"
    total=$((total + 6700))

    # LSP 类（较重）
    printf "%-30s ~%s\n" "gopls-lsp" "8,000"
    total=$((total + 8000))

    # 已禁用（不计入）
    printf "%-30s %s\n" "frontend-design (已禁用)" "(0)"
    printf "%-30s %s\n" "agent-sdk-dev (已禁用)" "(0)"
    printf "%-30s %s\n" "ralph-loop (已禁用)" "(0)"

    # MCP 服务器（启动时开销）
    echo ""
    echo "🔌 MCP 服务器开销:"
    printf "%-30s ~%s\n" "context7" "5,000"
    printf "%-30s ~%s\n" "playwright" "10,000"
    total=$((total + 15000))

    echo ""
    echo "----------------------------------------------"
    echo "📈 预估总 Token 开销: ~${total} tokens"
    echo "💡 200K 上下文窗口剩余: ~$(( 200000 - total )) tokens"
    echo ""
    if [ "$total" -gt 60000 ]; then
        echo "⚠️  警告: Token 开销超过 60K，建议禁用不常用插件"
    else
        echo "✅ Token 开销在合理范围内"
    fi
}

plugin_estimates

echo ""
echo "=============================================="
