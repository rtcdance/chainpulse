import { useEffect, useRef, useState } from 'react'
import { BarChart3, ChevronDown, ChevronUp, Loader2, RefreshCw } from 'lucide-react'
import { AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts'
import { fetchMetrics, summarizeMetricGroups, type MetricSample, type MetricsPayload } from '../lib/chainpulse'

interface MetricDataPoint {
  time: string
  value: number
}

interface MetricChartState {
  key: string
  label: string
  color: string
  data: MetricDataPoint[]
  current: number
}

const chartConfigs = [
  { key: 'events', label: 'Event Throughput', color: '#f4a261', prefix: 'chainpulse_events_' },
  { key: 'cache', label: 'Cache Hit Ratio', color: '#2dd4bf', prefix: 'chainpulse_cache_' },
  { key: 'api', label: 'API Latency', color: '#818cf8', prefix: 'chainpulse_api_' },
  { key: 'reorg', label: 'Reorg Count', color: '#fbbf24', prefix: 'chainpulse_reorg_' },
]

function extractChartValue(samples: MetricSample[], prefix: string): number {
  const matching = samples.filter((s) => s.name.startsWith(prefix))
  if (matching.length === 0) return 0
  const sum = matching.reduce((acc, s) => {
    const v = parseFloat(s.value)
    return acc + (Number.isFinite(v) ? v : 0)
  }, 0)
  return Math.round(sum * 100) / 100
}

export default function Metrics() {
  const [payload, setPayload] = useState<MetricsPayload | null>(null)
  const [filter, setFilter] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [charts, setCharts] = useState<MetricChartState[]>([])
  const [showAdvanced, setShowAdvanced] = useState(false)
  const historyRef = useRef<Map<string, MetricDataPoint[]>>(new Map())
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)

  function buildCharts(samples: MetricSample[]): MetricChartState[] {
    const now = new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })
    return chartConfigs.map((config) => {
      const value = extractChartValue(samples, config.prefix)
      const history = historyRef.current.get(config.key) || []
      const next = [...history, { time: now, value }].slice(-20)
      historyRef.current.set(config.key, next)
      return { key: config.key, label: config.label, color: config.color, data: next, current: value }
    })
  }

  async function loadMetrics(): Promise<void> {
    setLoading(true)
    try {
      const next = await fetchMetrics()
      setPayload(next)
      setCharts(buildCharts(next.samples))
      setError(null)
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : 'Failed to load metrics')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void loadMetrics()
    intervalRef.current = setInterval(() => {
      if (document.visibilityState === 'visible') {
        void (async () => {
          try {
            const next = await fetchMetrics()
            setPayload(next)
            setCharts(buildCharts(next.samples))
          } catch { /* auto-refresh failures are silent */ }
        })()
      }
    }, 10_000)
    return () => { if (intervalRef.current) clearInterval(intervalRef.current) }
  }, [])

  const samples = payload?.samples.filter((sample) => sample.name.toLowerCase().includes(filter.toLowerCase())) || []
  const groups = summarizeMetricGroups(samples)

  return (
    <div className="space-y-6">
      <section className="rounded-[28px] border border-white/10 bg-white/5 p-6">
        <div className="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
          <div>
            <p className="text-xs uppercase tracking-[0.25em] text-mist">Metrics Acceptance</p>
            <h2 className="mt-3 text-2xl font-semibold text-white">Prometheus samples, time-series charts, and raw text evidence</h2>
            <p className="mt-3 max-w-3xl text-sm leading-6 text-sand/75">
              This view reads the raw `/metrics` payload and auto-refreshes every 10 seconds. Charts show the last 20 data points for key metric groups.
            </p>
          </div>
          <button
            onClick={() => void loadMetrics()}
            className="inline-flex items-center gap-2 rounded-full border border-white/15 bg-white/10 px-4 py-2 text-sm text-white transition hover:bg-white/15"
          >
            <RefreshCw className="h-4 w-4" />
            Refresh metrics
          </button>
        </div>

        <div className="mt-5 grid gap-4 md:grid-cols-3">
          <div className="rounded-2xl border border-white/10 bg-black/15 p-4">
            <div className="text-xs uppercase tracking-[0.2em] text-mist">Source</div>
            <div className="mt-2 font-mono text-sm text-white">{payload?.evidence.path || '/metrics'}</div>
          </div>
          <div className="rounded-2xl border border-white/10 bg-black/15 p-4">
            <div className="text-xs uppercase tracking-[0.2em] text-mist">Metric Samples</div>
            <div className="mt-2 text-2xl font-semibold text-white">{payload?.samples.length || 0}</div>
          </div>
          <label className="rounded-2xl border border-white/10 bg-black/15 px-4 py-3">
            <div className="text-xs uppercase tracking-[0.2em] text-mist">Filter</div>
            <input
              value={filter}
              onChange={(event) => setFilter(event.target.value)}
              placeholder="events / api / cache"
              className="mt-3 w-full bg-transparent text-sm text-white outline-none placeholder:text-sand/35"
            />
          </label>
        </div>
      </section>

      {loading && !payload ? (
        <div className="flex h-72 items-center justify-center rounded-[28px] border border-white/10 bg-white/5">
          <Loader2 className="h-9 w-9 animate-spin text-glow" />
        </div>
      ) : error && !payload ? (
        <div className="rounded-[28px] border border-rose-400/30 bg-rose-400/10 p-6 text-sm text-rose-100">{error}</div>
      ) : (
        <>
          <section className="grid gap-4 xl:grid-cols-2">
            {charts.map((chart) => (
              <article key={chart.key} className="rounded-[28px] border border-white/10 bg-white/5 p-5">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <div className="flex h-10 w-10 items-center justify-center rounded-2xl" style={{ backgroundColor: `${chart.color}20` }}>
                      <BarChart3 className="h-4 w-4" style={{ color: chart.color }} />
                    </div>
                    <div>
                      <p className="text-sm font-medium text-white">{chart.label}</p>
                      <p className="text-xs text-sand/60">chainpulse_{chart.key}_*</p>
                    </div>
                  </div>
                  <div className="text-2xl font-semibold text-white">{chart.current}</div>
                </div>
                <div className="mt-4 h-32">
                  <ResponsiveContainer width="100%" height="100%">
                    <AreaChart data={chart.data} margin={{ top: 5, right: 5, bottom: 5, left: 5 }}>
                      <defs>
                        <linearGradient id={`gradient-${chart.key}`} x1="0" y1="0" x2="0" y2="1">
                          <stop offset="5%" stopColor={chart.color} stopOpacity={0.3} />
                          <stop offset="95%" stopColor={chart.color} stopOpacity={0} />
                        </linearGradient>
                      </defs>
                      <XAxis dataKey="time" tick={{ fontSize: 10, fill: '#a89f91' }} interval="preserveStartEnd" />
                      <YAxis tick={{ fontSize: 10, fill: '#a89f91' }} width={40} />
                      <Tooltip
                        contentStyle={{ backgroundColor: '#07111f', border: '1px solid rgba(255,255,255,0.1)', borderRadius: '12px', fontSize: '12px' }}
                        labelStyle={{ color: '#a89f91' }}
                        itemStyle={{ color: chart.color }}
                      />
                      <Area type="monotone" dataKey="value" stroke={chart.color} fill={`url(#gradient-${chart.key})`} strokeWidth={2} />
                    </AreaChart>
                  </ResponsiveContainer>
                </div>
              </article>
            ))}
          </section>

          <section>
            <button
              onClick={() => setShowAdvanced(!showAdvanced)}
              className="inline-flex items-center gap-2 rounded-full border border-white/15 bg-white/10 px-4 py-2 text-sm text-white transition hover:bg-white/15"
            >
              {showAdvanced ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
              {showAdvanced ? 'Hide' : 'Show'} advanced — groups, samples, raw text
            </button>

            {showAdvanced && (
              <div className="mt-4 grid gap-6 xl:grid-cols-[0.85fr,1.15fr]">
                <article className="rounded-[28px] border border-white/10 bg-white/5 p-6">
                  <p className="text-xs uppercase tracking-[0.25em] text-mist">Metric Groups</p>
                  <div className="mt-5 space-y-3">
                    {groups.map((group) => (
                      <div key={group.name} className="flex items-center justify-between rounded-2xl border border-white/10 bg-black/15 p-4">
                        <div className="flex items-center gap-3">
                          <div className="flex h-10 w-10 items-center justify-center rounded-2xl bg-glow/15">
                            <BarChart3 className="h-4 w-4 text-glow" />
                          </div>
                          <span className="font-medium capitalize text-white">{group.name}</span>
                        </div>
                        <span className="rounded-full border border-white/10 px-3 py-1 text-sm text-sand/75">
                          {group.count}
                        </span>
                      </div>
                    ))}
                  </div>
                </article>

                <article className="rounded-[28px] border border-white/10 bg-white/5 p-6">
                  <p className="text-xs uppercase tracking-[0.25em] text-mist">Samples & Raw Text</p>
                  <div className="mt-5 grid gap-5 lg:grid-cols-[0.9fr,1.1fr]">
                    <div className="max-h-[28rem] overflow-auto rounded-[24px] border border-white/10 bg-black/20 p-4">
                      <div className="space-y-3">
                        {samples.map((sample) => (
                          <div key={`${sample.name}-${sample.labels}`} className="rounded-2xl border border-white/10 bg-black/15 p-4">
                            <div className="font-mono text-sm text-white">{sample.name}</div>
                            {sample.labels ? <div className="mt-1 text-xs text-sand/60">{sample.labels}</div> : null}
                            <div className="mt-2 text-sm text-glow">{sample.value}</div>
                          </div>
                        ))}
                      </div>
                    </div>
                    <pre className="max-h-[28rem] overflow-auto rounded-[24px] border border-white/10 bg-black/25 p-4 text-xs leading-6 text-sand/85">
                      {payload?.raw || ''}
                    </pre>
                  </div>
                </article>
              </div>
            )}
          </section>
        </>
      )}
    </div>
  )
}
