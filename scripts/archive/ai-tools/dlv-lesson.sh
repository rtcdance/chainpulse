#!/bin/bash
# Delve 调试课程启动脚本
# 使用方法: bash scripts/dlv-lesson.sh <lesson_number>
# 示例: bash scripts/dlv-lesson.sh 1

set -e

LESSON=${1:?Usage: $0 <lesson_number 1-5>}

case $LESSON in
  1)
    echo "=== 课程 1: 事件总线 (EventBus) 分发流程 ==="
    echo "学习重点: goroutine 调度、channel 通信、sync.RWMutex"
    echo ""
    dlv debug ./examples/01_event_bus/
    ;;
  2)
    echo "=== 课程 2: ABI 事件签名与解码 ==="
    echo "学习重点: keccak256 哈希、ABI 编码规则"
    echo ""
    dlv debug ./examples/02_event_signature/
    ;;
  3)
    echo "=== 课程 3: 错误处理与重试逻辑 ==="
    echo "学习重点: errors.Is/As、错误包装链、指数退避"
    echo ""
    dlv debug ./examples/03_error_handling/
    ;;
  4)
    echo "=== 课程 4: 链重组 (Reorg) 处理 ==="
    echo "学习重点: 状态回滚、一致性保护、maxRollback"
    echo ""
    dlv debug ./examples/04_reorg_detection/
    ;;
  5)
    echo "=== 课程 5: Context 优雅关闭 ==="
    echo "学习重点: context 传播、goroutine 生命周期管理"
    echo ""
    dlv debug ./examples/05_context_graceful_shutdown/
    ;;
  *)
    echo "错误: 无效的课程编号 '$LESSON'"
    echo "可选: 1, 2, 3, 4, 5"
    echo ""
    echo "教程文档: docs/exercises/delve_debug_tutorial.md"
    exit 1
    ;;
esac
