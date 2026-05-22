# 发布流程和变更日志管理

## 概述

本指南介绍ChainPulse Web3 Indexer的发布流程和变更日志管理。

## 版本控制

### 语义版本控制 (SemVer)

格式: `MAJOR.MINOR.PATCH`

- **MAJOR**: 不兼容的API变更
- **MINOR**: 向后兼容的功能添加
- **PATCH**: 向后兼容的bug修复

示例:
- `1.0.0` - 初始发布
- `1.1.0` - 新功能
- `1.1.1` - Bug修复
- `2.0.0` - 重大变更

## 变更日志管理

### CHANGELOG.md格式

```markdown
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- New features

### Changed
- Changes in existing functionality

### Deprecated
- Soon-to-be removed features

### Removed
- Now removed features

### Fixed
- Any bug fixes

### Security
- In case of vulnerabilities

## [1.0.0] - 2026-01-14

### Added
- Initial release
- E2E testing framework
- CI/CD pipeline
- Deployment automation
- Monitoring and alerting

### Changed
- Improved performance

### Fixed
- Initial bug fixes

[Unreleased]: https://github.com/rtcdance/chainpulse/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/rtcdance/chainpulse/releases/tag/v1.0.0
```

### 自动化变更日志更新

**脚本**: `scripts/update-changelog.sh`

```bash
#!/bin/bash

# 获取最后一个标签
LAST_TAG=$(git describe --tags --abbrev=0)
CURRENT_VERSION=$1

if [ -z "$CURRENT_VERSION" ]; then
    echo "Usage: $0 <version>"
    exit 1
fi

# 获取提交日志
COMMITS=$(git log $LAST_TAG..HEAD --pretty=format:"- %s (%h)")

# 生成变更日志条目
CHANGELOG_ENTRY="## [$CURRENT_VERSION] - $(date +%Y-%m-%d)

### Added
$(echo "$COMMITS" | grep -i "feat:" | sed 's/feat: //')

### Changed
$(echo "$COMMITS" | grep -i "change:" | sed 's/change: //')

### Fixed
$(echo "$COMMITS" | grep -i "fix:" | sed 's/fix: //')

### Security
$(echo "$COMMITS" | grep -i "security:" | sed 's/security: //')

"

# 更新CHANGELOG.md
sed -i "s/## \[Unreleased\]/## [Unreleased]\n\n$CHANGELOG_ENTRY/" CHANGELOG.md

echo "Changelog updated for version $CURRENT_VERSION"
```

## 发布说明模板

### release-notes-template.md

```markdown
# Release Notes: ChainPulse Indexer v{VERSION}

**Release Date**: {DATE}

## Overview

{Brief description of the release}

## New Features

### Feature 1: {Feature Name}
- Description
- Benefits
- Usage example

### Feature 2: {Feature Name}
- Description
- Benefits
- Usage example

## Improvements

### Performance
- {Performance improvement 1}
- {Performance improvement 2}

### Reliability
- {Reliability improvement 1}
- {Reliability improvement 2}

### User Experience
- {UX improvement 1}
- {UX improvement 2}

## Bug Fixes

- {Bug fix 1}
- {Bug fix 2}
- {Bug fix 3}

## Breaking Changes

⚠️ **Important**: This release contains breaking changes.

### Change 1: {Breaking change}
- **Impact**: {What breaks}
- **Migration**: {How to migrate}

### Change 2: {Breaking change}
- **Impact**: {What breaks}
- **Migration**: {How to migrate}

## Migration Guide

### For Users Upgrading from v{PREVIOUS_VERSION}

#### Step 1: Backup Data
```bash
# Backup database
pg_dump chainpulse > backup_v{PREVIOUS_VERSION}.sql

# Backup configuration
cp -r config config_backup_v{PREVIOUS_VERSION}
```

#### Step 2: Update Application
```bash
# Download new version
wget https://github.com/rtcdance/chainpulse/releases/download/v{VERSION}/chainpulse-v{VERSION}.tar.gz

# Extract
tar -xzf chainpulse-v{VERSION}.tar.gz

# Install
cd chainpulse-v{VERSION}
make install
```

#### Step 3: Run Migrations
```bash
# Run database migrations
chainpulse migrate --version {VERSION}

# Verify migrations
chainpulse migrate --status
```

#### Step 4: Restart Services
```bash
# Restart indexer
systemctl restart chainpulse-indexer

# Verify
chainpulse health
```

## Performance Metrics

### Latency Improvements
- Event collection: {X}% faster
- Event processing: {X}% faster
- API queries: {X}% faster

### Throughput Improvements
- Events per second: {X}% increase
- Transactions per second: {X}% increase

### Resource Usage
- Memory: {X}% reduction
- CPU: {X}% reduction

## Known Issues

### Issue 1: {Issue Description}
- **Workaround**: {Workaround}
- **Fix**: {When will be fixed}

### Issue 2: {Issue Description}
- **Workaround**: {Workaround}
- **Fix**: {When will be fixed}

## Deprecations

### Deprecated Features
- {Feature 1} - Will be removed in v{X.Y.Z}
- {Feature 2} - Will be removed in v{X.Y.Z}

### Migration Path
- Use {New Feature} instead of {Old Feature}

## Security Updates

### Vulnerabilities Fixed
- {CVE-XXXX-XXXXX}: {Description}
- {CVE-XXXX-XXXXX}: {Description}

### Security Recommendations
- Update to this version immediately
- Review security advisories

## Contributors

Thank you to all contributors:
- {Contributor 1}
- {Contributor 2}
- {Contributor 3}

## Download

- [Source Code](https://github.com/rtcdance/chainpulse/archive/v{VERSION}.tar.gz)
- [Docker Image](https://hub.docker.com/r/chainpulse/indexer:v{VERSION})
- [Binary Releases](https://github.com/rtcdance/chainpulse/releases/tag/v{VERSION})

## Support

- [Documentation](https://docs.chainpulse.io)
- [Issue Tracker](https://github.com/rtcdance/chainpulse/issues)
- [Discussions](https://github.com/rtcdance/chainpulse/discussions)
- [Discord](https://discord.gg/chainpulse)

## What's Next

- {Upcoming feature 1}
- {Upcoming feature 2}
- {Upcoming feature 3}
```

## 发布流程

### 1. 准备发布

```bash
# 更新版本号
VERSION="1.1.0"

# 更新CHANGELOG
./scripts/update-changelog.sh $VERSION

# 更新版本文件
sed -i "s/VERSION=.*/VERSION=$VERSION/" version.sh

# 提交变更
git add CHANGELOG.md version.sh
git commit -m "chore: prepare release v$VERSION"
```

### 2. 创建发布标签

```bash
# 创建标签
git tag -a v$VERSION -m "Release v$VERSION"

# 推送标签
git push origin v$VERSION
```

### 3. 构建发布物件

```bash
# 构建Docker镜像
docker build -t chainpulse/indexer:v$VERSION .
docker push chainpulse/indexer:v$VERSION

# 构建二进制文件
make build VERSION=$VERSION

# 创建发布包
tar -czf chainpulse-v$VERSION.tar.gz bin/
```

### 4. 生成发布说明

```bash
# 使用模板生成发布说明
./scripts/generate-release-notes.sh $VERSION > RELEASE_NOTES.md

# 创建GitHub Release
gh release create v$VERSION \
  --title "ChainPulse Indexer v$VERSION" \
  --notes-file RELEASE_NOTES.md \
  chainpulse-v$VERSION.tar.gz
```

### 5. 发布通知

```bash
# 发送通知
./scripts/notify-release.sh $VERSION

# 更新文档
./scripts/update-docs.sh $VERSION

# 发送邮件通知
./scripts/send-release-email.sh $VERSION
```

## 发布检查清单

- [ ] 所有测试通过
- [ ] 代码审查完成
- [ ] 文档已更新
- [ ] CHANGELOG已更新
- [ ] 版本号已更新
- [ ] 发布说明已生成
- [ ] Docker镜像已构建
- [ ] 二进制文件已构建
- [ ] GitHub Release已创建
- [ ] 通知已发送
- [ ] 文档已发布
- [ ] 监控已配置

## 回滚流程

### 如果发布有问题

```bash
# 1. 识别问题
# 检查日志和监控

# 2. 决定回滚
# 评估影响范围

# 3. 执行回滚
git revert v$VERSION
git tag v$VERSION-rollback
git push origin v$VERSION-rollback

# 4. 恢复服务
docker pull chainpulse/indexer:v$PREVIOUS_VERSION
docker run -d chainpulse/indexer:v$PREVIOUS_VERSION

# 5. 验证
chainpulse health

# 6. 通知
./scripts/notify-rollback.sh $VERSION $PREVIOUS_VERSION
```

## 最佳实践

1. **频繁发布**: 小的、频繁的发布比大的、不频繁的发布更好
2. **自动化**: 尽可能自动化发布流程
3. **测试**: 在发布前充分测试
4. **文档**: 清晰的发布说明和迁移指南
5. **通知**: 及时通知用户
6. **监控**: 发布后密切监控
7. **回滚**: 准备好快速回滚计划
