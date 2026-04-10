#!/bin/bash
set -e

TASK="${1:-}"
if [ "$TASK" = "-h" ] || [ "$TASK" = "--help" ]; then
    echo "Usage: $0 <task>"
    echo "Task prompt packs were removed with historical docs cleanup."
    echo "Use docs/specs/ and docs/project/ for current execution context."
    exit 0
fi

if [ -z "$TASK" ]; then
    echo "Usage: $0 <task>"
    echo "Task prompt packs were removed with historical docs cleanup."
    echo "Use docs/specs/ and docs/project/ for current execution context."
    exit 1
fi

echo "❌ Deprecated workflow: historical prompt packs are no longer available."
echo "   Requested task: $TASK"
echo "   Suggested next step: open docs/specs/ for current design records."
exit 1
