import { useEffect, useState } from 'react'
import { BarChart3, Loader2, RefreshCw } from 'lucide-react'
import { fetchMetrics, summarizeMetricGroups, type MetricsPayload } from '../lib/chainpulse'

export default function Metrics() {
  const [payload, setPayload] = useState<MetricsPayload | null>(null)
  const [filter, setFilter] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  async function loadMetrics(): Promise<void> {
    setLoading(true)
    try {
      const next = await fetchMetrics()
      setPayload(next)
      setError(null)
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : 'Failed to load metrics')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void loadMetrics()
  }, [])

  const samples = payload?.samples.filter((sample) => sample.name.toLowerCase().includes(filter.toLowerCase())) || []
  const groups = summarizeMetricGroups(samples)

  return (
    <div className="space-y-6">
      <section className="rounded-[28px] border border-white/10 bg-white/5 p-6">
        <div className="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
          <div>
            <p className="text-xs uppercase tracking-[0.25em] text-mist">Metrics Acceptance</p>
            <h2 className="mt-3 text-2xl font-semibold text-white">Prometheus samples, grouping, and raw text evidence</h2>
            <p className="mt-3 max-w-3xl text-sm leading-6 text-sand/75">
              This view reads the raw `/metrics` payload and parses the `chainpulse_*` samples directly, so the team can verify both the human summary and the exact source text.
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

      {loading ? (
        <div className="flex h-72 items-center justify-center rounded-[28px] border border-white/10 bg-white/5">
          <Loader2 className="h-9 w-9 animate-spin text-glow" />
        </div>
      ) : error ? (
        <div className="rounded-[28px] border border-rose-400/30 bg-rose-400/10 p-6 text-sm text-rose-100">{error}</div>
      ) : (
        <section className="grid gap-6 xl:grid-cols-[0.85fr,1.15fr]">
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
        </section>
      )}
    </div>
  )
}
