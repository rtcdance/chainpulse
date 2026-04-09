#!/bin/bash
set -e

TASK="${1:-}"
if [ "$TASK" = "-h" ] || [ "$TASK" = "--help" ]; then
    echo "Usage: $0 <task>"
    echo "Available: M1-1a M1-1b M1-1c M2 M3a M3b M3c"
    echo "Prompt directory: docs/archive/planning/"
    exit 0
fi

if [ -z "$TASK" ]; then
    echo "Usage: $0 <task>"
    echo "Available: M1-1a M1-1b M1-1c M2 M3a M3b M3c"
    echo "Prompt directory: docs/archive/planning/"
    exit 1
fi

PROMPT_FILE="docs/archive/planning/${TASK}_PROMPT.md"
if [ ! -f "$PROMPT_FILE" ]; then
    echo "❌ Missing: $PROMPT_FILE"
    exit 1
fi

echo "📋 Task: $TASK"
echo "📄 Prompt: $PROMPT_FILE"
echo ""

CONTEXT_FILES=(
    "docs/archive/ARCHITECTURE_v1.md"
    "docs/IMPLEMENTATION_STATUS.md"
    "docs/DEPENDENCY_GRAPH.md"
    "docs/project/ARCHITECTURE_RULES.md"
    "$PROMPT_FILE"
)

echo "📦 Context:"
for f in "${CONTEXT_FILES[@]}"; do
    if [ -f "$f" ]; then
        echo "  ✓ $f ($(wc -l < "$f") lines)"
    else
        echo "  ✗ $f (missing)"
    fi
done
echo ""

MISSING=0
for f in "${CONTEXT_FILES[@]}"; do
    if [ ! -f "$f" ]; then
        MISSING=1
    fi
done

if [ $MISSING -eq 1 ]; then
    echo "⚠️  Missing context files — GPT may drift"
    read -p "Continue? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

echo "🚀 Starting Codex..."
echo ""

PROMPT="Execute the following task strictly. All context files are in the project directory.

$(cat "$PROMPT_FILE")"

codex "$PROMPT"
