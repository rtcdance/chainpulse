# §7.3 frontend-api lane：不存在，而且前端测试本来就跑不起来

- 日期：2026-10-14
- 对应 §10 行：*"README、支持矩阵、录屏、架构图、ADR与最终tag一致；**无未经验证的能力/生产规模声明**"*
- 直接对应 **§7.3** 的 `frontend-api` lane：*"根与 frontend `npm ci`；`npm --prefix frontend run build`、`npm --prefix frontend run test`；主流程 Playwright + OpenAPI diff | UI调用真实 API；关键场景实际执行；构建错误不忽略"*
- 证据：`build/acceptance/<sha>/frontend-lane/`

## 结论

§7.3 列出的 `frontend-api` lane **完全不存在**：

```
grep 所有 workflow 里的 "frontend" 或 "npm "   →  空
```

而 `frontend/` 里**有**：`package.json`（`build: tsc && vite build`、`test: vitest run`）、
**3 个单元测试文件**、根目录 `playwright.config.ts`（`testDir: './test/acceptance'`，**2 个 spec**）。
**这些从来没有跑过一次**——`frontend/node_modules` 根本不存在。

而"没人跑"掩盖了更根本的事：**这些测试本来也跑不起来。**

## 根因：三个测试依赖从未被声明

```
$ npm --prefix frontend run test          （修复前）
  MISSING DEPENDENCY  Cannot find dependency 'jsdom'
  Test Files  (3)   Tests  no tests   Errors 3
  exit 1
```

- `frontend/vitest.config.ts` 设 `environment: 'jsdom'`
- `frontend/src/test-setup.ts` `import '@testing-library/jest-dom'`
- `ErrorBoundary.test.tsx` `import { render, screen } from '@testing-library/react'`

**`jsdom`、`@testing-library/react`、`@testing-library/jest-dom` 三个包，`package.json` 里没有，
`package-lock.json` 里也没有。** `npm ci` 严格按 lockfile 装 ⇒ 一个都不装 ⇒ vitest 起不来。

⇒ **不是"测试失败"，是"测试无法启动"**。而因为没有 lane 跑它们，**没有人知道**。

## 修复

**1. 补上三个 devDependency**（只进 devDependencies，不进 bundle）：
```json
"jsdom": "^30.1.2",
"@testing-library/react": "^16.3.3",
"@testing-library/jest-dom": "^7.0.1"
```

**2. 装完依赖后仍有一条失败，也是真问题**：
```
FAIL  exportToCSV > escapes double quotes in CSV
TypeError: window.URL.revokeObjectURL is not a function
  ❯ downloadFile src/lib/export.ts:10:14
```
jsdom **既不实现 `createObjectURL` 也不实现 `revokeObjectURL`**；测试只 stub 了前者，
而 `export.ts` 两个都用。**产品代码对浏览器是正确的**，所以 stub 属于 `test-setup.ts`
（共享环境），不属于被测组件。已加，并用 `typeof` 守卫以免覆盖真实实现。

**3. 记录依赖决策**：F17 门禁监控 `frontend/package.json`，我改它时门禁**正确地失败**了，
直到 `docs/project/DEPENDENCY_APPROVAL.md` 补上条目才通过。**门禁工作正常。**

**4. 新增 `.github/workflows/frontend.yml`**，`frontend-api` job：
`npm ci`（根）→ `npm ci --prefix frontend` → `npm --prefix frontend run build` →
`npm --prefix frontend run test` → `openapi-drift.py` → 上传证据（`if-no-files-found: error`）。

**5. 写完 lane 才发现：`.gitignore` 把两个 lockfile 都忽略了，所以这个 lane 一进去就会红。**

```
$ git check-ignore -v frontend/package-lock.json
.gitignore:86:package-lock.json    frontend/package-lock.json
```

`.gitignore` 第 83 行忽略 `frontend/package-lock.json`，第 86 行忽略 `package-lock.json`，
**两个都没有被 git 跟踪**。而 `npm ci` **必须**有 lockfile：

```
$ mv frontend/package-lock.json /tmp/ && npm ci --prefix frontend
npm error code EUSAGE
npm error The `npm ci` command can only install with an existing package-lock.json
```

⇒ **全新 checkout 上第一条 `npm ci` 就会失败。** 我新加的 lane 会**在第二步红**，
而我在本机跑得通，只是因为**本机的 lockfile 恰好存在（只是没被跟踪）**——
又一个"本机通过 ≠ CI 通过"。

**修复**：从 `.gitignore` 移除这两条，并把两个 lockfile 纳入版本控制。
这也解释了三依赖缺失为什么能长期存在：**lockfile 不被跟踪，就没有任何机制保证它与 `package.json` 同步**。

**本地逐条验证**（就是 workflow 里那几条命令，从干净状态跑）：
```
npm ci (root)                     exit 0
npm ci --prefix frontend          exit 0
npm --prefix frontend run build   exit 0
npm --prefix frontend run test    exit 0   Test Files 3 passed (3)   Tests 21 passed (21)
openapi-drift.py --report ...     exit 0   "openapi drift check passed"
```
8 个 workflow YAML 全部可解析。

## 顺带：又一个"存在但没人调用"

`scripts/acceptance/openapi-drift.py` **有通过的自测、文档写得很清楚、而且被谁都不调用**。
它自己的 docstring 就记录了 spec **曾经漂移过**（`EventResponse` 少文档了 handler 实际返回的
`network` 字段）。今天它**通过**——但"通过"与"从没跑过"在工件层面是一样的，这正是把它接进 lane 的理由。

## 诚实边界

1. **Playwright 那一半没有接**：§7.3 的 lane 明确含"主流程 Playwright"，而
   `test/acceptance/*.spec.ts` 需要**一个已运行的栈**（API gateway :8080、API service :8081、
   Grafana :3000、Prometheus :9090，见 `test/acceptance/README.md`）。在 CI 里拉起整套 compose
   是我**无法本地验证**的改动（本机拉起 7 链全栈代价很高），所以**没有单方面加**，记为 **D23**。
   本轮接的是 lane 里**不需要运行栈**的那三件事。
2. **`npm ci` 的依赖是新装的**：`node_modules` 之前不存在，所以"21 passed"是**首次**结果，
   不代表历史上曾经通过。
3. **前端测试只有 3 个文件 21 个用例**，覆盖 `ErrorBoundary`、`Skeleton` 和 `utils`/`api` 的纯函数。
   这是**很薄的**覆盖，不是"前端已被验证"。§7.3 要求的"关键场景实际执行"要靠 Playwright 那半。
4. **未做**：README 对前端的描述、以及支持矩阵（§10 item 22 要求"支持矩阵"且**它不存在**）
   仍待处理；`frontend/dist` 作为工件上传，但**未**接 SBOM/digest。
5. **workflow 未在 GitHub 上跑过**：验的是 YAML 可解析 + 每条命令本地实跑。

## 复现

```bash
# 修复前（把三个依赖从 package.json 移除即可复现）
npm ci --prefix frontend && npm --prefix frontend run test
# → MISSING DEPENDENCY  Cannot find dependency 'jsdom'

# 修复后
npm ci --no-audit --no-fund
npm ci --prefix frontend --no-audit --no-fund
npm --prefix frontend run build
npm --prefix frontend run test          # 3 passed (3), 21 passed (21)
python3 scripts/acceptance/openapi-drift.py --report /tmp/drift.json

# 依赖审批门禁
git status --porcelain | awk '{print $2}' > /tmp/changed.txt
python3 scripts/acceptance/dep-approval-gate.py --changed-files /tmp/changed.txt
```
