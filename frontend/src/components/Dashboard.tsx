import { useEffect, useMemo, useState } from 'react'
import { Activity, CheckCircle2, Loader2, RefreshCw, ShieldAlert, Waves } from 'lucide-react'
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

interface DashboardState {
  health: HealthPayload | null
  runtime: RuntimePayload | null
  metrics: MetricsPayload | null
  sampleEvents: Awaited<ReturnType<typeof fetchEvents>> | null
  serviceReports: ServiceAcceptanceReport[]
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

  async function load(): Promise<void> {
    setLoading(true)
    try {
      const [health, runtime, metrics, sampleEvents, serviceReports] = await Promise.all([
        fetchHealth(),
        fetchRuntimeSummary(),
        fetchMetrics(),
        fetchEvents({ limit: 5, offset: 0 }),
        fetchCurrentSliceReport(),
      ])
      setState({ health, runtime, metrics, sampleEvents, serviceReports })
      setError(null)
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : 'Failed to load acceptance dashboard')
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

  if (loading) {
    return (
      <div className="flex h-72 items-center justify-center rounded-[28px] border border-white/10 bg-white/5">
        <Loader2 className="h-10 w-10 animate-spin text-glow" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="rounded-[28px] border border-rose-400/30 bg-rose-400/10 p-8">
        <ShieldAlert className="h-10 w-10 text-rose-200" />
        <h2 className="mt-4 text-2xl font-semibold text-white">Acceptance dashboard unavailable</h2>
        <p className="mt-2 max-w-2xl text-sm leading-6 text-rose-100/90">{error}</p>
        <button
          onClick={() => void load()}
          className="mt-5 inline-flex items-center gap-2 rounded-full border border-white/15 bg-white/10 px-4 py-2 text-sm text-white transition hover:bg-white/15"
        >
          <RefreshCw className="h-4 w-4" />
          Retry
        </button>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <section className="grid gap-4 xl:grid-cols-4">
        <article className="rounded-[26px] border border-white/10 bg-white/5 p-5">
          <div className="flex items-center justify-between">
            <span className="text-xs uppercase tracking-[0.25em] text-mist">Gateway Health</span>
            <Activity className="h-5 w-5 text-glow" />
          </div>
          <div className="mt-4 text-3xl font-semibold text-white">{state.health?.status || 'unknown'}</div>
          <p className="mt-3 text-sm text-sand/75">
            Source <span className="font-mono text-white/90">{state.health?.evidence.path}</span>
          </p>
        </article>

        <article className="rounded-[26px] border border-white/10 bg-white/5 p-5">
          <div className="flex items-center justify-between">
            <span className="text-xs uppercase tracking-[0.25em] text-mist">Sample Events</span>
            <Waves className="h-5 w-5 text-glow" />
          </div>
          <div className="mt-4 text-3xl font-semibold text-white">{state.sampleEvents?.events.length || 0}</div>
          <p className="mt-3 text-sm text-sand/75">
            Live sample from {state.sampleEvents?.evidence.path || '/events'}
          </p>
        </article>

        <article className="rounded-[26px] border border-white/10 bg-white/5 p-5">
          <div className="flex items-center justify-between">
            <span className="text-xs uppercase tracking-[0.25em] text-mist">Metrics Samples</span>
            <Activity className="h-5 w-5 text-glow" />
          </div>
          <div className="mt-4 text-3xl font-semibold text-white">{state.metrics?.samples.length || 0}</div>
          <p className="mt-3 text-sm text-sand/75">
            Parsed from {state.metrics?.evidence.path || '/metrics'}
          </p>
        </article>

        <article className="rounded-[26px] border border-white/10 bg-white/5 p-5">
          <div className="flex items-center justify-between">
            <span className="text-xs uppercase tracking-[0.25em] text-mist">Slice Coverage</span>
            <CheckCircle2 className="h-5 w-5 text-glow" />
          </div>
          <div className="mt-4 text-3xl font-semibold text-white">{availability.ok}/{availability.total}</div>
          <p className="mt-3 text-sm text-sand/75">
            Successful probes across the current runnable slice
          </p>
        </article>
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
      </section>
    </div>
  )
}
