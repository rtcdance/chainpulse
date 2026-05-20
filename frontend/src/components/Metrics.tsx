import { useEffect, useRef, useState } from 'react'
import { BarChart3, ChevronDown, ChevronUp, Download, Layers, Loader2, RefreshCw, X } from 'lucide-react'
import { AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer, LineChart, Line, Legend } from 'recharts'
import { fetchMetrics, exportToCSV, exportToJSON, summarizeMetricGroups, type MetricSample, type MetricsPayload } from '../lib/chainpulse'

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
  { key: 'events', label: 'Event Throughput', color: '#f4a261', prefix: 'chainpulse_events_', unit: 'count' },
  { key: 'cache', label: 'Cache Hit Ratio', color: '#2dd4bf', prefix: 'chainpulse_cache_', unit: '%' },
  { key: 'api', label: 'API Latency', color: '#818cf8', prefix: 'chainpulse_api_', unit: 'ms' },
  { key: 'reorg', label: 'Reorg Count', color: '#fbbf24', prefix: 'chainpulse_reorg_', unit: 'count' },
]

const timeRangePoints: Record<string, number> = { '1m': 6, '5m': 30, '15m': 90 }

function extractChartValue(samples: MetricSample[], prefix: string): number {
  const matching = samples.filter((s) => s.name.startsWith(prefix))
  if (matching.length === 0) return 0
  const sum = matching.reduce((acc, s) => {
    const v = parseFloat(s.value)
    return acc + (Number.isFinite(v) ? v : 0)
  }, 0)
  return Math.round(sum * 100) / 100
}

function Sparkline({ data, color }: { data: MetricDataPoint[]; color: string }) {
  const values = data.slice(-6)
  const max = Math.max(...values.map((d) => d.value), 1)
  return (
    <div className="flex items-end gap-px" style={{ height: 24 }}>
      {values.map((d, i) => (
        <div
          key={i}
          className="w-1.5 rounded-t-sm transition-all"
          style={{
            height: `${Math.max((d.value / max) * 100, 4)}%`,
            backgroundColor: color,
            opacity: 0.55,
          }}
        />
      ))}
    </div>
  )
}

function CustomTooltip({ active, payload, label }: { active?: boolean; payload?: Array<{ payload: Record<string, unknown>; dataKey: string; name: string; value: number; color: string; stroke: string }>; label?: string }) {
  if (!active || !payload?.length) return null
  return (
    <div className="rounded-xl border border-white/10 bg-[#07111f] p-3 text-xs shadow-lg">
      <p className="text-sand/60 mb-1.5">{label}</p>
      {payload.map((entry, idx) => {
        const data = entry.payload as Record<string, unknown>
        const dataKey = entry.dataKey
        const prevKey = `_prev_${dataKey}`
        const current = Number(data[dataKey])
        const prev = prevKey in data ? Number(data[prevKey]) : null
        const delta = prev !== null && Number.isFinite(prev) ? current - prev : null
        return (
          <div key={idx} className="flex items-center gap-2 py-0.5">
            <span className="inline-block h-2 w-2 rounded-full" style={{ backgroundColor: entry.color || entry.stroke }} />
            <span className="text-sand/60">{entry.name}:</span>
            <span className="font-mono text-white">{typeof entry.value === 'number' ? entry.value.toFixed(2) : entry.value}</span>
            {delta !== null && (
              <span className={`font-mono text-[11px] ${delta >= 0 ? 'text-emerald-400' : 'text-rose-400'}`}>
                {delta >= 0 ? '↑' : '↓'}{Math.abs(delta).toFixed(2)}
              </span>
            )}
          </div>
        )
      })}
    </div>
  )
}

export default function Metrics() {
  const [payload, setPayload] = useState<MetricsPayload | null>(null)
  const [filter, setFilter] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [charts, setCharts] = useState<MetricChartState[]>([])
  const [showAdvanced, setShowAdvanced] = useState(false)
  const [timeRange, setTimeRange] = useState<'1m' | '5m' | '15m' | 'all'>('15m')
  const [combinedView, setCombinedView] = useState(false)
  const historyRef = useRef<Map<string, MetricDataPoint[]>>(new Map())
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)

  function getFilteredData(data: MetricDataPoint[]): MetricDataPoint[] {
    if (timeRange === 'all') return data
    const count = timeRangePoints[timeRange] || 90
    return data.slice(-count)
  }

  function buildCharts(samples: MetricSample[]): MetricChartState[] {
    const now = new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })
    return chartConfigs.map((config) => {
      const value = extractChartValue(samples, config.prefix)
      const history = historyRef.current.get(config.key) || []
      const next = [...history, { time: now, value }].slice(-90)
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

  const combinedData = (charts[0]?.data || []).map((_, idx) => {
    const entry: Record<string, unknown> = { time: charts[0].data[idx].time }
    chartConfigs.forEach((config) => {
      const val = charts.find((c) => c.key === config.key)?.data[idx]?.value ?? 0
      entry[config.key] = val
      if (idx > 0) {
        const prevVal = charts.find((c) => c.key === config.key)?.data[idx - 1]?.value ?? 0
        entry[`_prev_${config.key}`] = prevVal
      }
    })
    return entry
  })

  const timeButtons: { label: string; value: '1m' | '5m' | '15m' | 'all' }[] = [
    { label: '1m', value: '1m' },
    { label: '5m', value: '5m' },
    { label: '15m', value: '15m' },
    { label: 'all', value: 'all' },
  ]

  return (
    <div className="space-y-6">
      <section className="rounded-[28px] border border-white/10 bg-white/5 p-6">
        <div className="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
          <div>
            <p className="text-xs uppercase tracking-[0.25em] text-mist">Metrics Acceptance</p>
            <h2 className="mt-3 text-2xl font-semibold text-white">Prometheus samples, time-series charts, and raw text evidence</h2>
            <p className="mt-3 max-w-3xl text-sm leading-6 text-sand/75">
              This view reads the raw `/metrics` payload and auto-refreshes every 10 seconds.
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <div className="flex rounded-full border border-white/15 bg-white/5 p-0.5">
              {timeButtons.map((btn) => (
                <button
                  key={btn.value}
                  onClick={() => setTimeRange(btn.value)}
                  className={`rounded-full px-3 py-1 text-xs font-medium transition ${
                    timeRange === btn.value ? 'bg-white/15 text-white' : 'text-sand/50 hover:text-white'
                  }`}
                >
                  {btn.label}
                </button>
              ))}
            </div>
            <button
              onClick={() => setCombinedView(!combinedView)}
              className={`inline-flex items-center gap-1.5 rounded-full border px-3 py-1 text-xs font-medium transition ${
                combinedView ? 'border-glow/50 bg-glow/10 text-glow' : 'border-white/15 bg-white/5 text-sand/50 hover:text-white'
              }`}
            >
              <Layers className="h-3.5 w-3.5" />
              Combined
            </button>
            <button
              onClick={() => void loadMetrics()}
              className="inline-flex items-center gap-2 rounded-full border border-white/15 bg-white/10 px-4 py-2 text-sm text-white transition hover:bg-white/15"
            >
              <RefreshCw className="h-4 w-4" />
              Refresh
            </button>
          </div>
          <div className="flex gap-2">
            <button
              onClick={() => {
                if (!payload?.samples.length) return
                exportToCSV(payload.samples as unknown as Record<string, unknown>[], [
                  { key: 'name', label: 'Name' },
                  { key: 'labels', label: 'Labels' },
                  { key: 'value', label: 'Value' },
                ], `metrics-${new Date().toISOString().slice(0, 10)}.csv`)
              }}
              disabled={!payload?.samples.length}
              className="inline-flex items-center gap-1.5 rounded-full border border-white/15 bg-white/10 px-4 py-2 text-sm text-white transition hover:bg-white/15 disabled:cursor-not-allowed disabled:opacity-40"
            >
              <Download className="h-4 w-4" />
              CSV
            </button>
            <button
              onClick={() => {
                if (!payload) return
                exportToJSON(payload, `metrics-${new Date().toISOString().slice(0, 10)}.json`)
              }}
              disabled={!payload}
              className="inline-flex items-center gap-1.5 rounded-full border border-white/15 bg-white/10 px-4 py-2 text-sm text-white transition hover:bg-white/15 disabled:cursor-not-allowed disabled:opacity-40"
            >
              <Download className="h-4 w-4" />
              JSON
            </button>
          </div>
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
          <label className="relative rounded-2xl border border-white/10 bg-black/15 px-4 py-3">
            <div className="text-xs uppercase tracking-[0.2em] text-mist">Filter</div>
            <input
              value={filter}
              onChange={(event) => setFilter(event.target.value)}
              placeholder="events / api / cache"
              className="mt-3 w-full bg-transparent pr-8 text-sm text-white outline-none placeholder:text-sand/35"
            />
            {filter && (
              <button
                onClick={() => setFilter('')}
                className="absolute bottom-[14px] right-3 rounded-full p-0.5 text-sand/50 transition hover:text-white hover:bg-white/10"
                aria-label="Clear filter"
              >
                <X className="h-4 w-4" />
              </button>
            )}
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
          {combinedView ? (
            <section className="rounded-[28px] border border-white/10 bg-white/5 p-5">
              <p className="mb-4 text-xs uppercase tracking-[0.25em] text-mist">Combined View</p>
              <div className="h-80">
                <ResponsiveContainer width="100%" height="100%">
                  <LineChart data={combinedData} margin={{ top: 5, right: 5, bottom: 5, left: 5 }}>
                    <XAxis dataKey="time" tick={{ fontSize: 10, fill: '#a89f91' }} interval="preserveStartEnd" />
                    <YAxis tick={{ fontSize: 10, fill: '#a89f91' }} width={50} />
                    <Tooltip content={<CustomTooltip />} />
                    <Legend wrapperStyle={{ fontSize: '11px' }} iconType="circle" />
                    {chartConfigs.map((config) => (
                      <Line
                        key={config.key}
                        type="monotone"
                        dataKey={config.key}
                        name={`${config.label} (${config.unit})`}
                        stroke={config.color}
                        strokeWidth={2}
                        dot={false}
                      />
                    ))}
                  </LineChart>
                </ResponsiveContainer>
              </div>
            </section>
          ) : (
            <section className="grid gap-4 xl:grid-cols-2">
              {charts.map((chart) => {
                const filtered = getFilteredData(chart.data).map((d, i, arr) => ({
                  ...d,
                  _prev_value: i > 0 ? arr[i - 1].value : null,
                }))
                return (
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
                      <div className="flex items-center gap-3">
                        <Sparkline data={chart.data} color={chart.color} />
                        <div className="text-2xl font-semibold text-white">{chart.current}<span className="ml-1 text-sm font-normal text-sand/50">{chartConfigs.find((c) => c.key === chart.key)?.unit || ''}</span></div>
                      </div>
                    </div>
                    <div className="mt-4 h-32">
                      <ResponsiveContainer width="100%" height="100%">
                        <AreaChart data={filtered} margin={{ top: 5, right: 5, bottom: 5, left: 5 }}>
                          <defs>
                            <linearGradient id={`gradient-${chart.key}`} x1="0" y1="0" x2="0" y2="1">
                              <stop offset="5%" stopColor={chart.color} stopOpacity={0.3} />
                              <stop offset="95%" stopColor={chart.color} stopOpacity={0} />
                            </linearGradient>
                          </defs>
                          <XAxis dataKey="time" tick={{ fontSize: 10, fill: '#a89f91' }} interval="preserveStartEnd" />
                          <YAxis tick={{ fontSize: 10, fill: '#a89f91' }} width={40} />
                          <Tooltip content={<CustomTooltip />} />
                          <Area type="monotone" dataKey="value" name={chart.label} stroke={chart.color} fill={`url(#gradient-${chart.key})`} strokeWidth={2} />
                        </AreaChart>
                      </ResponsiveContainer>
                    </div>
                  </article>
                )
              })}
            </section>
          )}

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
