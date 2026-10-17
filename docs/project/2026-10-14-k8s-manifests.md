# K8s manifests：39 个文件，没人引用过；以及矩阵自身的一处错

- 日期：2026-10-14
- 对应 §10 行：*"README、支持矩阵、录屏、架构图、ADR与最终tag一致；**无未经验证的能力/生产规模声明**"*
- 对应 §7.5（CD）：*"缺 kubeconfig 时不能报告 K8s 部署成功；不采用 K8s 的版本应明确调用已选 Docker 部署路径"*
- 证据：`build/acceptance/<sha>/k8s-manifests/SUMMARY.txt`
- 来源：**支持矩阵的 `unverified` 行**（把它当待办清单）

## 一、39 个 manifest，没有任何东西引用它们

```
$ grep -rln "k8s/\|kustomize" --include="*.yml" --include="*.sh" --include=Makefile .
（空）
```

`k8s/` 下 **39 个文件**，而**没有任何 workflow、Makefile 目标或脚本**碰过它们。
⇒ 一个 manifest 如果哪天不再能构建，**只有到部署时才会暴露**。

## 二、它们确实能构建（已实测）

`kubectl kustomize` 是**纯客户端**的，不需要集群：

```
k8s/base                     56 resources
k8s/overlays/monolithic      58 resources
k8s/overlays/microservice    59 resources
```

并且 ConfigMap 里的取值**全部通过校验器**：`MQ_TYPE: kafka`、`CACHE_TYPE: redis`、
`DATABASE_TYPE: postgres`、`LOG_LEVEL: info`；`DATA_PULLER_TYPE` 与 `API_TYPE` **未设置**，
所以走默认值 `https-jsonrpc` / `rest`——两个都合法。
（这也顺带回答了"我上一轮把 `grpc` 从校验器里去掉，会不会打断 k8s 部署"：**不会**，
manifest 从来没设过 `grpc`。）

## 三、接线：让它可以红

矩阵那一行现在写着"**没有任何 lane 检查这个，所以它可以静默退化**"。已补上：

```yaml
      - name: Kubernetes manifests build
        run: |
          set -euo pipefail
          command -v kubectl >/dev/null || {
            echo "::error::kubectl is not available, so the manifests cannot be validated"
            exit 1
          }
          for overlay in k8s/base k8s/overlays/monolithic k8s/overlays/microservice; do
            out=$(kubectl kustomize "$overlay")
            count=$(printf '%s\n' "$out" | grep -c '^kind:' || true)
            echo "  $overlay: ${count} resources"
            if [ "${count:-0}" -lt 1 ]; then
              echo "::error::$overlay built no resources"
              exit 1
            fi
          done
```

**本地逐字验证**：三个 overlay 分别输出 56/58/59，全过；
指向一个不存在的目录时 `kubectl kustomize` 返回非零（`must build at directory: not a valid directory`），
所以这一步**真的会红**，不是永远绿。8 个 workflow YAML 全部可解析。

## 四、写这一轮时，发现**我自己的矩阵**有一处是错的

矩阵第 1 节我写过：

> | Seven-chain configuration | **unverified** | `pkg/chainid/chain_ids.go` 列出这些链；**只有两条链曾经一起启动过**，而且那个测试并不索引它们 |

**这是错的。** 实跑 `TestMultiChainRealEvents`：

```
=== RUN   TestMultiChainRealEvents/ethereum
=== RUN   TestMultiChainRealEvents/polygon
=== RUN   TestMultiChainRealEvents/bsc
=== RUN   TestMultiChainRealEvents/arbitrum
=== RUN   TestMultiChainRealEvents/optimism
=== RUN   TestMultiChainRealEvents/base
=== RUN   TestMultiChainRealEvents/avalanche
--- PASS: TestMultiChainRealEvents (17.08s)
```

**七条 Anvil 链确实同时启动**（各自独立端口），并且每条都连上、读到区块高度与账户余额。
测试**没有 build tag**，所以它在 `e2e` job 里会跑。

已把这一行拆成三行，把"配置"与"索引"分开：

| 行 | 状态 | 依据 |
| --- | --- | --- |
| Seven-chain **configuration** | **verified** | 七条 Anvil 同时启动且各自可 RPC；实跑 7/7 通过（17.08s） |
| Seven-chain **indexing** | **unverified** | 那个测试读链状态，**不索引任何东西**；没有任何测试跨超过两条链索引日志 |
| Multi-chain production operation | **out of scope** | 没有测试在 production scale 下并发索引 |

⇒ 这是**连续第三轮**在我自己写的矩阵里发现错判。前两轮是 GraphQL / WebSocket / WS-采集位置，
这一轮是七链。**`unverified` 行必须在两个方向上都核**：有些其实已经验证过，只是我写的时候没查。

## 诚实边界

1. **`kubectl kustomize` 只验证 kustomize 引用能解析**，**不验证 k8s API schema**。
   `kubectl apply --dry-run=client` 需要集群连接才能取到 OpenAPI schema，本地没有集群，
   所以**没有做 schema 校验**。字段名写错、类型不匹配这类问题这一步抓不到。
2. **`kubectl` 在 ubuntu-latest 上是预装的**，我没有本地验证 GitHub runner 的镜像内容；
   脚本里加了 `command -v kubectl || exit 1`，缺了就红而不是跳过。
3. **没有集群、也没有 apply 过任何 manifest**：矩阵里"Kubernetes deployment **applied** to a cluster"
   仍是 **unverified**，与本轮无关。
4. **七链只验到"能启动并能读状态"**：没有验证七链并发索引、链间隔离（E04 覆盖的是两条链）、
   或者七链下 checkpoint 的行为。
5. **矩阵仍靠人维护**，没有自动一致性检查——这一轮的错判就是这么来的。

## 复现

```bash
# 没有任何东西引用它们
grep -rln "k8s/\|kustomize" --include="*.yml" --include="*.sh" --include=Makefile .

# 它们确实能构建
for o in k8s/base k8s/overlays/monolithic k8s/overlays/microservice; do
  echo "$o: $(kubectl kustomize $o | grep -c '^kind:') resources"
done

# 七链确实同时启动
PATH="$HOME/.foundry/bin:$PATH" go test -count=1 -v -run TestMultiChainRealEvents ./test/e2e/
```
