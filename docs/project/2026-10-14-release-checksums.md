# release checksum：循环覆盖写文件（§7.5 第 3 条）

- 日期：2026-10-14
- 对应 §10 行：*"README、支持矩阵、录屏、架构图、ADR与最终tag一致"* 之外的发布工件完整性；
  直接对应 **§7.5 第 3 条**：*"生成覆盖所有二进制/镜像的 checksum、SBOM 与扫描报告。**修 `release.yml` checksum 循环覆盖写文件问题；每个文件名唯一且可验证**"*
- 证据：`build/acceptance/<sha>/release-checksums/{before-after.txt,matrix.txt,matrix.json}`
- 工具：`scripts/acceptance/checksums.sh`（新增）；`scripts/acceptance/negative-cases.py`（扩展到 25 例）

## 缺陷：每个平台都覆盖同一个 checksums.txt

`release.yml` 的 `Create checksums` 步骤原文：

```bash
cd dist
for dir in binaries-*; do
  cd "$dir"
  sha256sum * > ../checksums.txt      # ← 每轮都截断同一个文件
  cd ..
done
```

`>` 是**截断写**。循环每转一圈就把 `../checksums.txt` 清空重写，所以**只有最后一轮的结果留下**。

而 build matrix 有 **5 个平台**（linux/darwin × amd64/arm64 + windows/amd64），
每个平台产出 **5 个二进制**（`chainpulse` + `puller`/`event-processor`/`api-service`/`api-gateway`）：

**模拟真实 dist/ 复现**（`build/acceptance/<sha>/release-checksums/before-after.txt`）：

```
=== simulated release dist/ ===
platform dirs: 5
binaries:      25

=== BEFORE (the loop as shipped) ===
loop exit=0
checksums.txt entries: 5   (expected 25)
platforms represented: ['api-gateway-windows-amd64.exe', 'api-service-windows-amd64.exe',
                        'chainpulse-windows-amd64.exe', 'event-processor-windows-amd64.exe',
                        'puller-windows-amd64.exe']
```

⇒ **25 个二进制只登记了 5 个**（只剩按字母序最后的 `binaries-windows-amd64`），
**20 个 checksum 被静默丢弃**，而这份被截断的清单会**当作完整工件随 release 一起发布**。
循环本身 `exit=0`，**没有任何信号**。

## 修复

把判定逻辑抽成 `scripts/acceptance/checksums.sh`（**不在 workflow 里内联**），
理由与仓库既有的 `acceptance-gate.py` 一致：*"判定逻辑放在可被测试的地方，workflow 只负责调用它"*。
`release.yml` 的步骤现在只有两行：

```yaml
      run: |
        set -euo pipefail
        bash scripts/acceptance/checksums.sh dist
```

脚本做四件事：

1. **只截断一次，然后逐目录追加**（`>>`）——这正是缺陷的根因。
2. 排序（`sort -k2`），让清单**确定可复现**。
3. **三道守卫**：
   - `count`：清单行数必须等于实际二进制数 ⇒ 抓住截断，也抓住"某个 artifact 什么都没下载"。
   - `unique`：文件名必须唯一 ⇒ 对不上文件的 checksum 不算 checksum（§7.5"每个文件名唯一"）。
   - `verify`：对每个目录跑 `sha256sum -c --ignore-missing` ⇒ 抓住"哈希之后文件被改动"。
4. 支持 `--verify-only`，**跳过生成只跑守卫**——这样"哈希后改动"这种外部注入不到的故障也能被测。

**修复后**（同一份模拟 dist）：

```
=== AFTER (scripts/acceptance/checksums.sh) ===
exit=0
verified 25 checksums across 5 platforms
checksums.txt entries: 25
unique names: 25
```

## 守卫是否真的会红（并入 §7.4 反例矩阵）

按上一轮建立的纪律，**每条守卫都要证明它会红**，并且**配正向对照**。
四例已并入 `scripts/acceptance/negative-cases.py`，矩阵从 21 例扩到 **25 例**：

| 用例 | 注入 | 期望 | 实际 |
| --- | --- | --- | --- |
| `checksum-manifest-truncated` | 4 个二进制 / 清单只 1 行（**旧循环的产物状态**） | fail | ✅ count 守卫：`records 1 of 4 binaries` |
| `checksum-duplicate-names` | 两个平台目录里同名文件 | fail | ✅ unique 守卫：`duplicate filenames: chainpulse` |
| `checksum-binary-tampered` | 清单生成后改动二进制 | fail | ✅ verify 守卫：`checksum verification failed in binaries-linux-amd64` |
| `control-checksums-clean` | 3 平台 × 2 二进制 | pass | ✅ `verified 6 checksums` |

```
negative-case matrix: 25/25 as expected
✓ every gate fired on its negative input and passed its positive control
```

另外单独验证过两条边界：**空 `dist/`（无 `binaries-*`）→ exit 1**；`bash -n` 语法通过。

## 顺带确认的事

- **二进制文件名本身是唯一的**：build 步骤把 `${goos}-${goarch}` 写进输出名
  （`output="chainpulse-${{ matrix.goos }}-${{ matrix.goarch }}..."`），所以跨平台不会重名——
  这也是为什么旧循环虽然截断，但残留的那 5 行至少还是可归属的。修复后 25 行仍两两不同。
- 7 个 workflow 的 YAML **全部仍可解析**。

## 诚实边界

1. **workflow 本身未端到端跑过**（本机无 GitHub、无 matrix）。我验证的是：
   ① 从 YAML 里**逐字抽出** `run` 块在模拟 dist 上执行通过；
   ② 脚本本身在 5 种状态下行为正确。**未**验证的是 GitHub Actions 的真实 matrix 下载布局。
2. **`checksums.txt` 是否会随 release 发布，我没有本地办法确认。**
   release 步骤用的是 `files: dist/**/*`；按 minimatch 的 `**` 语义（可匹配零个路径段），
   `dist/checksums.txt`（深度 1）应当被匹配——但那是 action 自己的 glob 方言，本地无法验证。
   **我刻意没有改成"再加一行显式 `dist/checksums.txt`"**，因为若该文件本已被匹配，
   重复项可能导致 `already_exists` 资产冲突。这一条**留给有 GitHub 的环境确认**。
3. **镜像 digest 未覆盖**：§7.5 第 3 条同时要求"镜像使用不可变 digest，禁止重新 build 后直接提升"。
   本轮只修了**二进制**的 checksum 循环；`docker` job 的 digest 处理**未审查**。
4. **SBOM 未接**：§7.5 要求 SBOM；`govulncheck-summary.py` 有 `--sbom-dir`，但 release 流程里没有 SBOM 步骤。

## 复现

```bash
# 模拟真实 dist 布局并复现旧行为
mkdir -p /tmp/relsim/dist && cd /tmp/relsim/dist
for p in linux-amd64 linux-arm64 darwin-amd64 darwin-arm64 windows-amd64; do
  mkdir -p "binaries-$p"
  for b in chainpulse puller event-processor api-service api-gateway; do
    echo payload > "binaries-$p/$b-$p"
  done
done
# 旧循环：25 个二进制只留 5 行
for dir in binaries-*; do cd "$dir"; sha256sum * > ../checksums.txt; cd ..; done
wc -l checksums.txt          # 5

# 修复后：25 行且全部校验通过
bash scripts/acceptance/checksums.sh /tmp/relsim/dist
wc -l /tmp/relsim/dist/checksums.txt   # 25

# 反例矩阵（25 例）
python3 scripts/acceptance/negative-cases.py
```
