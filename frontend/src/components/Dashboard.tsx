import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Activity, ArrowRight, BarChart3, Bell, Globe, Layers, Loader2, RefreshCw, Search } from 'lucide-react'
import { useAuth } from '../lib/auth'
import {
  fetchEvents,
  fetchEventStats,
  formatTimestamp,
  type NormalizedEventsResponse,
  type EventStats,
} from '../lib/chainpulse'

function formatAddress(address: string): string {
  return `${address.slice(0, 6)}...${address.slice(-4)}`
}

export default function Dashboard() {
  console.log('[DASH] Dashboard render start')
  const { address } = useAuth()
  const navigate = useNavigate()

  const [events, setEvents] = useState<NormalizedEventsResponse | null>(null)
  const [stats, setStats] = useState<EventStats | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  async function tryFetchEvents() {
    try {
      const res = await fetchEvents({ limit: 10, offset: 0 })
      setEvents(res)
      setError(null)
    } catch (err) {
      // single failure is OK if stats succeeds
    }
  }

  async function tryFetchStats() {
    try {
      const res = await fetchEventStats()
      setStats(res)
      setError(null)
    } catch (err) {
      // single failure is OK if events succeeds
    }
  }

  async function load(): Promise<void> {
    try {
      await Promise.all([tryFetchEvents(), tryFetchStats()])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load')
    } finally {
      setLoading(false)
    }
  }

  if (loading && !events && !error) {
    load().catch(() => {/* handled in load */})
  }

  const chainCount = stats ? Object.keys(stats.byChain).length : 0
  const eventTypeCount = stats ? Object.keys(stats.byEventName).length : 0

  return (
    <div className="space-y-8">
      <section className="rounded-[28px] border border-white/10 bg-white/5 p-8 backdrop-blur">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <p className="text-xs uppercase tracking-[0.35em] text-mist">Welcome back</p>
            <h1 className="mt-2 text-2xl font-semibold text-white sm:text-3xl">
              {formatAddress(address)}
            </h1>
            <p className="mt-1 font-mono text-xs text-sand/40">{address}</p>
          </div>
          <button
            onClick={() => void load()}
            className="inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/5 px-4 py-2 text-sm text-sand/70 transition hover:border-white/20 hover:text-white"
          >
            <RefreshCw className="h-3.5 w-3.5" />
            Refresh
          </button>
        </div>
      </section>

      {loading ? (
        <div className="flex h-48 items-center justify-center rounded-[28px] border border-white/10 bg-white/5">
          <Loader2 className="h-8 w-8 animate-spin text-glow" />
        </div>
      ) : error ? (
        <div className="rounded-2xl border border-rose-400/25 bg-rose-400/10 p-6 text-rose-100">
          <p className="font-medium">Error: {error}</p>
          <button onClick={() => { setLoading(true); void load() }} className="mt-3 flex items-center gap-2 rounded-lg border border-current/20 px-4 py-2 text-sm hover:bg-current/10">
            <RefreshCw className="h-4 w-4" /> Retry
          </button>
        </div>
      ) : (
        <>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <div className="rounded-2xl border border-white/10 bg-white/5 p-5">
              <div className="flex items-center gap-2 text-sm text-sand/60"><Layers className="h-4 w-4" /> Total Events</div>
              <p className="mt-3 text-3xl font-semibold text-white">{stats?.total ?? '—'}</p>
            </div>
            <div className="rounded-2xl border border-white/10 bg-white/5 p-5">
              <div className="flex items-center gap-2 text-sm text-sand/60"><Globe className="h-4 w-4" /> Active Chains</div>
              <p className="mt-3 text-3xl font-semibold text-white">{chainCount || '—'}</p>
            </div>
            <div className="rounded-2xl border border-white/10 bg-white/5 p-5">
              <div className="flex items-center gap-2 text-sm text-sand/60"><Activity className="h-4 w-4" /> Event Types</div>
              <p className="mt-3 text-3xl font-semibold text-white">{eventTypeCount || '—'}</p>
            </div>
            <div className={`rounded-2xl border p-5 ${(stats?.reorged ?? 0) > 0 ? 'border-amber-300/25 bg-amber-300/10' : 'border-white/10 bg-white/5'}`}>
              <div className="flex items-center gap-2 text-sm text-sand/60"><Bell className="h-4 w-4" /> Reorgs</div>
              <p className={`mt-3 text-3xl font-semibold ${(stats?.reorged ?? 0) > 0 ? 'text-amber-300' : 'text-white'}`}>
                {stats?.reorged ?? '—'}
              </p>
            </div>
          </div>

          <section className="grid gap-6 lg:grid-cols-[1.4fr,1fr]">
            <div className="rounded-[28px] border border-white/10 bg-white/5 p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-xs uppercase tracking-[0.25em] text-mist">Recent Events</p>
                  <h2 className="mt-2 text-xl font-semibold text-white">Latest indexed events</h2>
                </div>
                <button
                  onClick={() => navigate('/events')}
                  className="inline-flex items-center gap-1.5 rounded-full border border-white/10 px-4 py-2 text-sm text-sand/60 transition hover:border-white/20 hover:text-white"
                >
                  View All <ArrowRight className="h-3.5 w-3.5" />
                </button>
              </div>

              <div className="mt-5 overflow-x-auto">
                {events && events.events.length > 0 ? (
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b border-white/5 text-left text-xs uppercase tracking-[0.2em] text-mist">
                        <th className="pb-3 pr-4 font-medium">Event</th>
                        <th className="pb-3 pr-4 font-medium">Chain</th>
                        <th className="pb-3 pr-4 font-medium">Contract</th>
                        <th className="pb-3 pr-4 font-medium">Block</th>
                        <th className="pb-3 font-medium">Time</th>
                      </tr>
                    </thead>
                    <tbody>
                      {events.events.slice(0, 8).map((event) => (
                        <tr key={event.id} className="border-b border-white/[0.03] hover:bg-white/[0.02]">
                          <td className="py-3 pr-4 font-medium text-white">{event.eventName}</td>
                          <td className="py-3 pr-4 text-sand/60">{event.chainId}</td>
                          <td className="py-3 pr-4 font-mono text-xs text-sand/50">{event.contractAddress.slice(0, 8)}...</td>
                          <td className="py-3 pr-4 font-mono text-xs text-sand/50">{event.blockNumber ?? '—'}</td>
                          <td className="py-3 text-xs text-sand/45">{formatTimestamp(event.timestamp)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                ) : (
                  <div className="py-10 text-center text-sm text-sand/40">
                    No events indexed yet. Start the ChainPulse puller to begin indexing.
                  </div>
                )}
              </div>
            </div>

            <div className="space-y-4">
              <div className="rounded-[28px] border border-white/10 bg-white/5 p-6">
                <p className="text-xs uppercase tracking-[0.25em] text-mist">Quick Search</p>
                <h3 className="mt-2 text-lg font-semibold text-white">Find events</h3>
                <p className="mt-2 text-sm leading-6 text-sand/60">
                  Search by event name, contract address, or transaction hash.
                </p>
                <button
                  onClick={() => navigate('/events')}
                  className="mt-4 inline-flex w-full items-center justify-center gap-2 rounded-full border border-glow/30 bg-glow/10 px-5 py-2.5 text-sm text-glow transition hover:bg-glow/20"
                >
                  <Search className="h-4 w-4" />
                  Open Event Explorer
                  <ArrowRight className="h-4 w-4" />
                </button>
              </div>

              <div className="rounded-[28px] border border-white/10 bg-white/5 p-6">
                <p className="text-xs uppercase tracking-[0.25em] text-mist">API Access</p>
                <h3 className="mt-2 text-lg font-semibold text-white">Integration</h3>
                <p className="mt-2 text-sm leading-6 text-sand/60">
                  Use REST, GraphQL, or WebSocket to query events from your applications.
                </p>
                <div className="mt-4 space-y-2 font-mono text-xs text-sand/40">
                  <div className="rounded-lg bg-black/20 px-3 py-2">GET /events</div>
                  <div className="rounded-lg bg-black/20 px-3 py-2">POST /graphql</div>
                  <div className="rounded-lg bg-black/20 px-3 py-2">WS /events/subscribe</div>
                </div>
              </div>

              {stats && Object.keys(stats.byChain).length > 0 && (
                <div className="rounded-[28px] border border-white/10 bg-white/5 p-6">
                  <div className="flex items-center gap-2 text-xs uppercase tracking-[0.25em] text-mist">
                    <BarChart3 className="h-3.5 w-3.5" />
                    Chain Distribution
                  </div>
                  <div className="mt-4 space-y-3">
                    {Object.entries(stats.byChain).slice(0, 5).map(([chain, count]) => (
                      <div key={chain} className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                          <div className="h-2 w-2 rounded-full bg-glow/50" />
                          <span className="text-sm text-white">{chain}</span>
                        </div>
                        <span className="font-mono text-xs text-sand/50">{count}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </section>
        </>
      )}
    </div>
  )
}