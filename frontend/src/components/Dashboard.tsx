import { useEffect, useMemo, useState } from 'react'
import { Activity, CheckCircle2, Loader2, RefreshCw, ShieldAlert, Waves, AlertTriangle, BarChart3, Database, Radio, Cpu } from 'lucide-react'
import {
  fetchCurrentSliceReport,
  fetchEvents,
  fetchHealth,
  fetchMetrics,
  fetchRuntimeSummary,
  formatTimestamp,
  type HealthPayload,
  type MetricsPayload,
  type RuntimePayload,
  type ServiceAcceptanceReport,
} from '../lib/chainpulse'
import LearnContext from './LearnContext'
import DataFlow from './DataFlow'

interface DashboardState {
  health: HealthPayload | null
  runtime: RuntimePayload | null
  metrics: MetricsPayload | null
  sampleEvents: Awaited<ReturnType<typeof fetchEvents>> | null
  serviceReports: ServiceAcceptanceReport[]
}

interface LoadWarning {
  label: string
  message: string
}

function tone(ok: boolean): string {
  return ok
    ? 'border-emerald-300/25 bg-emerald-300/10 text-emerald-100'
    : 'border-rose-400/25 bg-rose-400/10 text-rose-100'
}

export default function Dashboard() {
  const [state, setState] = useState<DashboardState>({
    health: null,
    runtime: null,
    metrics: null,
    sampleEvents: null,
    serviceReports: [],
  })
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [warnings, setWarnings] = useState<LoadWarning[]>([])

  async function load(): Promise<void> {
    setLoading(true)

    const requests = [
      { label: 'Gateway health', run: fetchHealth },
      { label: 'Runtime summary', run: fetchRuntimeSummary },
      { label: 'Metrics', run: fetchMetrics },
      { label: 'Sample events', run: () => fetchEvents({ limit: 5, offset: 0 }) },
      { label: 'Service matrix', run: fetchCurrentSliceReport },
    ] as const

    try {
      const results = await Promise.allSettled(requests.map((request) => request.run()))
      const nextState: DashboardState = {
        health: null,
        runtime: null,
        metrics: null,
        sampleEvents: null,
        serviceReports: [],
      }
      const nextWarnings: LoadWarning[] = []
      let successCount = 0

      results.forEach((result, index) => {
        const label = requests[index].label
        if (result.status === 'fulfilled') {
          successCount += 1
          switch (label) {
            case 'Gateway health':
              nextState.health = result.value as HealthPayload
              break
            case 'Runtime summary':
              nextState.runtime = result.value as RuntimePayload
              break
            case 'Metrics':
              nextState.metrics = result.value as MetricsPayload
              break
            case 'Sample events':
              nextState.sampleEvents = result.value as Awaited<ReturnType<typeof fetchEvents>>
              break
            case 'Service matrix':
              nextState.serviceReports = result.value as ServiceAcceptanceReport[]
              break
          }
          return
        }

        nextWarnings.push({
          label,
          message: result.reason instanceof Error ? result.reason.message : 'request failed',
        })
      })

      if (successCount === 0) {
        setError('All dashboard data sources failed to load')
        setWarnings(nextWarnings)
        setState(nextState)
        return
      }

      setState(nextState)
      setWarnings(nextWarnings)
      setError(null)
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : 'Failed to load acceptance dashboard')
      setWarnings([])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
  }, [])

  const availability = useMemo(() => {
    const total = state.serviceReports.reduce((sum, report) => sum + report.probes.length, 0)
    const ok = state.serviceReports.reduce((sum, report) => sum + report.probes.filter((probe) => probe.ok).length, 0)
    return { total, ok }
  }, [state.serviceReports])

  const reorgedCount = useMemo(() => {
    return state.sampleEvents?.events.filter((e) => e.status === 'reorged').length ?? 0
  }, [state.sampleEvents])

  if (loading) {
    return (
      <div className="flex h-72 items-center justify-center rounded-[28px] border border-white/10 bg-white/5">
        <Loader2 className="h-10 w-10 animate-spin text-glow" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="space-y-6">
        <LearnContext
          title="这个页面是做什么的？"
          concept="Dashboard 展示 ChainPulse 的运行概览：是否健康、已索引多少事件、RPC 调用是否正常。绿色 = 正常，红色 = 异常。如果所有面板都是绿色，说明 ChainPulse 正在正常运行。"
          codePath="frontend/src/components/Dashboard.tsx"
          debugTip="在 https_jsonrpc_puller.go:299 设断点，观察 Poll 循环如何定期拉取新区块"
        />
        <div className={`rounded-2xl border p-6 ${tone(false)}`}>
          <div className="flex items-center gap-3">
            <AlertTriangle className="h-6 w-6" />
            <div>
              <p className="font-medium">Failed to load dashboard</p>
              <p className="mt-1 text-sm opacity-80">{error}</p>
            </div>
          </div>
          <button onClick={load} className="mt-4 flex items-center gap-2 rounded-lg border border-current/20 px-4 py-2 text-sm hover:bg-current/10">
            <RefreshCw className="h-4 w-4" /> Retry
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {warnings.length > 0 ? (
        <section className="rounded-[28px] border border-amber-300/30 bg-amber-300/10 p-5 text-amber-50">
          <div className="flex items-start gap-3">
            <ShieldAlert className="mt-0.5 h-5 w-5 shrink-0" />
            <div className="space-y-2">
              <h2 className="text-lg font-semibold text-white">Dashboard loaded with partial live evidence</h2>
              <div className="space-y-1 text-sm leading-6 text-amber-50/90">
                {warnings.map((warning) => (
                  <p key={warning.label}>
                    <span className="font-medium text-white">{warning.label}:</span> {warning.message}
                  </p>
                ))}
              </div>
            </div>
          </div>
        </section>
      ) : null}

      <section className="mb-6">
        <LearnContext
          title="ChainPulse 数据流 — 从 RPC 到 API"
          concept="一笔链上事件的生命周期：Puller 通过 eth_getLogs 从节点拉取 → ABI 解码为 BlockchainEvent → EventBus 分发给 Processor → 持久化存储 → API 查询返回。整个过程可以在 delve 中用 6 个断点完整追踪。"
          codePath="docs/guides/CODE_TRACE.md"
          debugTip="断点顺序: https_jsonrpc_puller.go:299 → eventbus.go:87 → event_processor.go → event_query_handler.go"
        />
        <DataFlow />
        <div className="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">

          <div className={`rounded-2xl border p-4 ${state.health?.status === 'healthy' ? 'border-emerald-300/25 bg-emerald-300/10 text-emerald-100' : 'border-rose-400/25 bg-rose-400/10 text-rose-100'}`}>
            <div className="flex items-center gap-2 text-sm font-medium"> <Activity className="h-4 w-4" /> 服务健康</div>
            <p className="mt-2 text-2xl font-semibold">{state.health?.status ?? '—'}</p>
            <p className="mt-1 text-xs opacity-70">GET /health</p>
          </div>

          <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
            <div className="flex items-center gap-2 text-sm font-medium text-sand/80"><Waves className="h-4 w-4" /> 已索引</div>
            <p className="mt-2 text-2xl font-semibold text-white">
              {(state.runtime?.summary as Record<string, unknown>)?.shadow_owned_events != null
                ? (state.runtime.summary as Record<string, unknown>).shadow_owned_events as number
                : state.sampleEvents?.events.length ?? '—'}
            </p>
            <p className="mt-1 text-xs opacity-70">Shared Runtime 事件</p>
          </div>

          <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
            <div className="flex items-center gap-2 text-sm font-medium text-sand/80"><BarChart3 className="h-4 w-4" /> 拉取请求</div>
            <p className="mt-2 text-2xl font-semibold text-white">{state.metrics?.samples.length || 0}</p>
            <p className="mt-1 text-xs opacity-70">/metrics 采样数</p>
          </div>

          <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
            <div className="flex items-center gap-2 text-sm font-medium text-sand/80"><Cpu className="h-4 w-4" /> 重组</div>
            <p className="mt-2 text-2xl font-semibold text-white">{(state.runtime?.summary as Record<string, unknown>)?.reorg_posture === 'monolithic-reorg-armed' ? '✓ 已就绪' : '—'}</p>
            <p className="mt-1 text-xs opacity-70">reorg handler</p>
          </div>

        </div>
      </section>

      <section className="grid gap-6 xl:grid-cols-[1.1fr,0.9fr]">
        <article className="rounded-[28px] border border-white/10 bg-white/5 p-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-xs uppercase tracking-[0.25em] text-mist">Acceptance Scope</p>
              <h2 className="mt-3 text-2xl font-semibold text-white">Current slice coverage in one place</h2>
            </div>
            <button
              onClick={() => void load()}
              className="inline-flex items-center gap-2 rounded-full border border-white/15 bg-white/10 px-4 py-2 text-sm text-white transition hover:bg-white/15"
            >
              <RefreshCw className="h-4 w-4" />
              Refresh
            </button>
          </div>

          <div className="mt-6 grid gap-3 md:grid-cols-2">
            {[
              { title: 'REST Event Query', description: 'List, filter, pagination, event detail' },
              { title: 'GraphQL', description: 'Schema, event connection, block query' },
              { title: 'WebSocket', description: 'Connection, messages, manual subscriptions' },
              { title: 'Metrics', description: 'Prometheus text and parsed samples' },
              { title: 'Health Surfaces', description: 'health, ready, live, components, rollout' },
              { title: 'Execution Control', description: 'runtime summary and control endpoints for execution services' },
            ].map((item) => (
              <div key={item.title} className="rounded-2xl border border-white/10 bg-black/15 p-4">
                <h3 className="text-base font-medium text-white">{item.title}</h3>
                <p className="mt-2 text-sm leading-6 text-sand/75">{item.description}</p>
              </div>
            ))}
          </div>
        </article>

        <article className="rounded-[28px] border border-white/10 bg-white/5 p-6">
          <p className="text-xs uppercase tracking-[0.25em] text-mist">Gateway Evidence</p>
          <div className="mt-5 space-y-3">
            <div className="rounded-2xl border border-white/10 bg-black/15 p-4">
              <div className="text-xs uppercase tracking-[0.2em] text-mist">Deployment Mode</div>
              <div className="mt-2 text-white">{state.runtime?.deploymentMode || 'unknown'}</div>
            </div>
            <div className="rounded-2xl border border-white/10 bg-black/15 p-4">
              <div className="text-xs uppercase tracking-[0.2em] text-mist">Runtime Mode</div>
              <div className="mt-2 text-white">{state.runtime?.runtimeMode || 'unknown'}</div>
            </div>
            <div className="rounded-2xl border border-white/10 bg-black/15 p-4">
              <div className="text-xs uppercase tracking-[0.2em] text-mist">Health Timestamp</div>
              <div className="mt-2 text-white">
                {typeof state.health?.timestamp === 'number'
                  ? formatTimestamp(state.health.timestamp)
                  : state.health?.timestamp || '-'}
              </div>
            </div>
            <div className="rounded-2xl border border-white/10 bg-black/15 p-4">
              <div className="text-xs uppercase tracking-[0.2em] text-mist">Summary Path</div>
              <div className="mt-2 font-mono text-sm text-white">{state.runtime?.evidence.path || '/runtime/summary'}</div>
            </div>
          </div>
        </article>
      </section>

      <section className="rounded-[28px] border border-white/10 bg-white/5 p-6">
        <p className="text-xs uppercase tracking-[0.25em] text-mist">Current Slice Service Matrix</p>
        <h2 className="mt-3 text-2xl font-semibold text-white">Gateway, API service, event processor, and puller</h2>
        {state.serviceReports.length === 0 ? (
          <div className="mt-6 rounded-[24px] border border-white/10 bg-black/15 p-5 text-sm leading-6 text-sand/80">
            Service matrix evidence is currently unavailable.
          </div>
        ) : (
          <div className="mt-6 grid gap-4 xl:grid-cols-2">
            {state.serviceReports.map((report) => (
            <article key={report.service.id} className="min-w-0 rounded-[24px] border border-white/10 bg-black/15 p-5">
              <div className="flex items-start justify-between gap-4">
                <div className="min-w-0 flex-1">
                  <h3 className="text-lg font-medium text-white">{report.service.name}</h3>
                  <p className="mt-2 text-sm leading-6 text-sand/70">{report.service.role}</p>
                  <p className="mt-2 break-words font-mono text-xs text-white/75">{report.service.baseUrl}</p>
                </div>
                <div className={`shrink-0 rounded-full border px-3 py-1 text-xs ${tone(report.probes.every((probe) => probe.ok))}`}>
                  {report.probes.filter((probe) => probe.ok).length}/{report.probes.length} ready
                </div>
              </div>

              <div className="mt-5 grid gap-3">
                {report.probes.map((probe) => (
                  <div key={`${report.service.id}-${probe.path}`} className={`min-w-0 rounded-2xl border p-4 ${tone(probe.ok)}`}>
                    <div className="flex flex-wrap items-start justify-between gap-3">
                      <span className="min-w-0 flex-1 break-words font-mono text-sm text-white">{probe.path}</span>
                      <span className="shrink-0 text-xs uppercase tracking-[0.18em]">
                        {probe.status ?? 'ERR'}
                      </span>
                    </div>
                    <pre className="mt-2 max-h-28 overflow-auto whitespace-pre-wrap break-words rounded-xl bg-black/20 p-3 font-mono text-[11px] leading-5 opacity-90">
                      {probe.summary || 'No body preview'}
                    </pre>
                  </div>
                ))}
              </div>
            </article>
            ))}
          </div>
        )}
      </section>
    </div>
  )
}
