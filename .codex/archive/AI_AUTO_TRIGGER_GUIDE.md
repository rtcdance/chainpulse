# AI 自动触发 Skills 完整方案

**最后更新**: 2026-03-30

## 方案概述

AI 编码时自动加载和执行 skills，无需手动干预。

## 实现机制

### 1. Claude Code 自动加载
- **文件**: `CLAUDE.md`（项目根目录）
- **触发**: Claude Code 启动时自动读取
- **内容**: 强制 AI 遵循的工作流程

### 2. 技能自动激活
- **脚本**: `scripts/auto-activate-skills.sh`
- **触发**: Git 提交前或手动运行
- **输出**: `.codex/active-skills.md`

### 3. 会话初始化
- **脚本**: `scripts/ai-session-init.sh`
- **用途**: 准备 AI 上下文
- **输出**: `.codex/ai-session-context.md`

## 使用流程

### 方式 1: 自动（推荐）

```bash
# 1. 安装 Git Hooks（一次性）
./scripts/install-hooks.sh

# 2. 正常开发
git add pkg/services/indexing/indexer.go

# 3. 启动 Claude Code
# → 自动读取 CLAUDE.md
# → 自动检测 active-skills.md
# → 自动遵循所有 skills
```

### 方式 2: 手动

```bash
# 1. 激活 skills
./scripts/auto-activate-skills.sh

# 2. 初始化 AI 会话
./scripts/ai-session-init.sh

# 3. 查看上下文
cat .codex/ai-session-context.md

# 4. 开始编码（AI 已加载所有约束）
```

## AI 执行流程

```
AI 启动
  ↓
读取 CLAUDE.md（自动）
  ↓
检测 .codex/active-skills.md
  ↓
加载每个 skill 定义
  ↓
加载 BEHAVIORAL_CONSTRAINTS.md
  ↓
声明适用的 skills
  ↓
列出退出标准
  ↓
开始编码（遵循所有约束）
  ↓
验证所有退出标准
  ↓
完成
```

## 关键文件

| 文件 | 用途 | 何时加载 |
|------|------|----------|
| `CLAUDE.md` | AI 工作流程 | Claude Code 启动时 |
| `.codex/active-skills.md` | 当前激活的 skills | AI 编码前 |
| `.codex/skills/*/SKILL.md` | Skill 定义 | AI 编码前 |
| `.codex/BEHAVIORAL_CONSTRAINTS.md` | 行为约束 | AI 编码前 |
| `.codex/PRE_CODING_CHECKLIST.md` | 编码前检查 | AI 编码前 |

## 验证

### 测试 AI 是否遵循 Skills

```bash
# 1. 修改文件
git add pkg/services/indexing/indexer.go

# 2. 激活 skills
./scripts/auto-activate-skills.sh

# 3. 查看激活的 skills
cat .codex/active-skills.md

# 4. 让 AI 编码
# AI 应该：
# - 声明哪些 skills 适用
# - 列出退出标准
# - 编码时遵循约束
# - 完成后验证标准
```

## 故障排查

### AI 没有遵循 Skills

**原因**: CLAUDE.md 未加载

**解决**:
```bash
# 确认文件存在
ls -la CLAUDE.md

# 手动提醒 AI
"请先读取 CLAUDE.md 和 .codex/active-skills.md"
```

### Skills 未激活

**原因**: 未运行激活脚本

**解决**:
```bash
./scripts/auto-activate-skills.sh
```

### AI 忽略约束

**原因**: 未明确要求遵循

**解决**: 在提示中明确说明
```
请严格遵循 .codex/active-skills.md 中的所有 skills
```

## 总结

✅ **自动化程度**: 95%
- CLAUDE.md 自动加载
- Skills 自动激活（通过 Git Hook）
- 约束自动应用

✅ **人工干预**: 5%
- 仅需安装 Git Hooks（一次性）
- 特殊情况手动覆盖

✅ **执行保证**:
- Pre-commit 检查
- AI 工作流程强制
- Code Review 验证
