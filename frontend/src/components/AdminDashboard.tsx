import { useEffect, useRef, useState } from 'react'
import { Activity, Bell, ChevronDown, ChevronUp, Loader2, Pause, Play, RefreshCw, ShieldAlert, Waves, AlertTriangle, BarChart3, Cpu } from 'lucide-react'
import {
  fetchCurrentSliceReport,
  fetchEvents,
  fetchHealth,
  fetchMetrics,
  fetchRuntimeSummary,
  fetchEventStats,
  formatTimestamp,
  type HealthPayload,
  type MetricsPayload,
  type RuntimePayload,
  type ServiceAcceptanceReport,
  type EventStats,
} from '../lib/chainpulse'
import LearnContext from './LearnContext'
import DataFlow from './DataFlow'

interface DashboardState {
  health: HealthPayload | null
  runtime: RuntimePayload | null
  metrics: MetricsPayload | null
  sampleEvents: Awaited<ReturnType<typeof fetchEvents>> | null
  serviceReports: ServiceAcceptanceReport[]
  eventStats: EventStats | null
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

export default function AdminDashboard() {
  const [state, setState] = useState<DashboardState>({
    health: null,
    runtime: null,
    metrics: null,
    sampleEvents: null,
    serviceReports: [],
    eventStats: null,
  })
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [warnings, setWarnings] = useState<LoadWarning[]>([])
  const [autoRefresh, setAutoRefresh] = useState(false)
  const [lastRefresh, setLastRefresh] = useState<Date | null>(null)
  const [collapsedServices, setCollapsedServices] = useState<Set<string>>(new Set())
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)

  async function load(silent = false): Promise<void> {
    if (!silent) setLoading(true)

    const requests = [
      { label: 'Gateway health', run: fetchHealth },
      { label: 'Runtime summary', run: fetchRuntimeSummary },
      { label: 'Metrics', run: fetchMetrics },
      { label: 'Sample events', run: () => fetchEvents({ limit: 5, offset: 0 }) },
      { label: 'Event statistics', run: fetchEventStats },
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
        eventStats: null,
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
            case 'Event statistics':
              nextState.eventStats = result.value as EventStats
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
      setLastRefresh(new Date())
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : 'Failed to load acceptance dashboard')
      setWarnings([])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load(false)
  }, [])

  useEffect(() => {
    if (autoRefresh) {
      intervalRef.current = setInterval(() => {
        if (document.visibilityState === 'visible') {
          void load(true)
        }
      }, 15_000)
    }
    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current)
        intervalRef.current = null
      }
    }
  }, [autoRefresh])

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
        <div className={`rounded-2xl border p-6 ${tone(false)}`}>
          <div className="flex items-center gap-3">
            <AlertTriangle className="h-6 w-6" />
            <div>
              <p className="font-medium">Failed to load admin dashboard</p>
              <p className="mt-1 text-sm opacity-80">{error}</p>
            </div>
          </div>
          <button onClick={() => void load(false)} className="mt-4 flex items-center gap-2 rounded-lg border border-current/20 px-4 py-2 text-sm hover:bg-current/10">
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
                ? (state.runtime?.summary as Record<string, unknown>).shadow_owned_events as number
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
              onClick={() => void load(false)}
              className="inline-flex items-center gap-2 rounded-full border border-white/15 bg-white/10 px-4 py-2 text-sm text-white transition hover:bg-white/15"
            >
              <RefreshCw className="h-4 w-4" />
              Refresh
            </button>
            <button
              onClick={() => setAutoRefresh(!autoRefresh)}
              className={`inline-flex items-center gap-2 rounded-full border px-4 py-2 text-sm transition ${
                autoRefresh
                  ? 'border-emerald-300/30 bg-emerald-300/10 text-emerald-100 hover:bg-emerald-300/20'
                  : 'border-white/15 bg-white/10 text-white hover:bg-white/15'
              }`}
            >
              {autoRefresh ? <Pause className="h-4 w-4" /> : <Play className="h-4 w-4" />}
              {autoRefresh ? 'Auto 15s' : 'Auto'}
            </button>
            {lastRefresh && (
              <span className="text-xs text-sand/50 self-center">
                Updated {lastRefresh.toLocaleTimeString()}
              </span>
            )}
          </div>

          <div className="mt-6 grid gap-3 md:grid-cols-2">
            {[ 
              { title: 'REST Event Query', description: 'List, filter, pagination, event detail' },
              { title: 'GraphQL', description: 'Schema, event connection, block query' },
              { title: 'WebSocket', description: 'Connection, messages, manual subscriptions' },
              { title: 'Webhooks', description: 'Register, test, and manage event notification endpoints' },
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
          <>
            <div className="mt-5 flex flex-wrap items-center gap-4 rounded-2xl border border-white/10 bg-black/15 px-5 py-3">
              <span className="text-sm text-sand/75">
                总计{' '}<span className="font-semibold text-white">{state.serviceReports.reduce((sum, r) => sum + r.probes.filter((p) => p.ok).length, 0)}/{state.serviceReports.reduce((sum, r) => sum + r.probes.length, 0)}</span>{' '}探头通过
              </span>
              {state.serviceReports.map((report) => {
                const ok = report.probes.filter((p) => p.ok).length
                const total = report.probes.length
                return (
                  <span key={report.service.id} className={`text-sm ${ok === total ? 'text-emerald-300' : 'text-amber-300'}`}>
                    {report.service.name}: {ok}/{total}
                  </span>
                )
              })}
            </div>
            <div className="mt-6 grid gap-4 xl:grid-cols-2">
              {state.serviceReports.map((report) => {
                const isCollapsed = collapsedServices.has(report.service.id)
                return (
                <article key={report.service.id} className="min-w-0 rounded-[24px] border border-white/10 bg-black/15 p-5">
                  <button
                    onClick={() => {
                      setCollapsedServices((prev) => {
                        const next = new Set(prev)
                        if (next.has(report.service.id)) next.delete(report.service.id)
                        else next.add(report.service.id)
                        return next
                      })
                    }}
                    className="flex w-full items-start justify-between gap-4 text-left"
                  >
                    <div className="min-w-0 flex-1">
                      <h3 className="text-lg font-medium text-white">{report.service.name}</h3>
                      <p className="mt-2 text-sm leading-6 text-sand/70">{report.service.role}</p>
                      <p className="mt-2 break-words font-mono text-xs text-white/75">{report.service.baseUrl}</p>
                    </div>
                    <div className="flex items-center gap-2">
                      <div className={`shrink-0 rounded-full border px-3 py-1 text-xs ${tone(report.probes.every((probe) => probe.ok))}`}>
                        {report.probes.filter((probe) => probe.ok).length}/{report.probes.length} ready
                      </div>
                      {isCollapsed ? <ChevronDown className="h-4 w-4 text-sand/50" /> : <ChevronUp className="h-4 w-4 text-sand/50" />}
                    </div>
                  </button>

                  {!isCollapsed && (
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
                  )}
                </article>
                )
              })}
            </div>
          </>
        )}
      </section>

      <section className="rounded-[28px] border border-white/10 bg-white/5 p-6">
        <div className="flex items-center justify-between">
          <div>
            <p className="text-xs uppercase tracking-[0.25em] text-mist">Event Statistics</p>
            <h2 className="mt-3 text-2xl font-semibold text-white">Aggregated event distribution and insights</h2>
          </div>
          <div className="rounded-full border border-white/10 px-3 py-1 text-xs text-mist">
            {state.eventStats ? `${state.eventStats.total} total events` : 'Loading...'}
          </div>
        </div>
        {state.eventStats ? (
          <div className="mt-6 grid gap-4 md:grid-cols-3">
            <div className="rounded-2xl border border-white/10 bg-black/15 p-4">
              <div className="text-xs uppercase tracking-[0.2em] text-mist">By Chain</div>
              <div className="mt-3 space-y-2">
                {Object.entries(state.eventStats.byChain).slice(0, 8).map(([chain, count]) => (
                  <div key={chain} className="flex items-center justify-between">
                    <span className="text-sm text-white">{chain}</span>
                    <span className="font-mono text-xs text-sand/75">{count}</span>
                  </div>
                ))}
                {Object.keys(state.eventStats.byChain).length === 0 && (
                  <div className="text-sm text-sand/55">No data yet</div>
                )}
              </div>
            </div>
            <div className="rounded-2xl border border-white/10 bg-black/15 p-4">
              <div className="text-xs uppercase tracking-[0.2em] text-mist">By Event Name</div>
              <div className="mt-3 space-y-2">
                {Object.entries(state.eventStats.byEventName).slice(0, 8).map(([name, count]) => (
                  <div key={name} className="flex items-center justify-between">
                    <span className="text-sm text-white">{name}</span>
                    <span className="font-mono text-xs text-sand/75">{count}</span>
                  </div>
                ))}
                {Object.keys(state.eventStats.byEventName).length === 0 && (
                  <div className="text-sm text-sand/55">No data yet</div>
                )}
              </div>
            </div>
            <div className="rounded-2xl border border-white/10 bg-black/15 p-4">
              <div className="text-xs uppercase tracking-[0.2em] text-mist">Reorg Alerts</div>
              <div className="mt-3">
                <div className="flex items-baseline gap-2">
                  <span className={`text-2xl font-semibold ${state.eventStats.reorged > 0 ? 'text-amber-300' : 'text-emerald-300'}`}>
                    {state.eventStats.reorged}
                  </span>
                  <span className="text-sm text-sand/75">reorged events</span>
                </div>
                <div className="mt-2 text-xs text-sand/55">
                  {state.eventStats.reorged > 50
                    ? 'High reorg count — investigate chain stability'
                    : state.eventStats.reorged > 0
                      ? 'Minor reorg activity — normal during network fluctuations'
                      : 'No reorgs detected — chains are stable'}
                </div>
              </div>
              <div className="mt-4 rounded-xl border border-amber-300/15 bg-amber-300/5 p-3">
                <div className="flex items-center gap-2 text-xs text-amber-200/80">
                  <Bell className="h-3 w-3" />
                  <span>DLQ alert threshold: &gt;100 events triggers warning</span>
                </div>
              </div>
            </div>
          </div>
        ) : (
          <div className="mt-6 rounded-[24px] border border-white/10 bg-black/15 p-5 text-sm leading-6 text-sand/80">
            Event statistics are currently being collected.
          </div>
        )}
      </section>
    </div>
  )
}