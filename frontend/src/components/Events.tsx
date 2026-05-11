import { useEffect, useState } from 'react'
import { Database, Loader2, RefreshCw, Search } from 'lucide-react'
import { fetchEventDetail, fetchEvents, formatTimestamp, type NormalizedEvent, type NormalizedEventsResponse } from '../lib/chainpulse'

interface Filters {
  chainId: string
  eventName: string
  contract: string
}

export default function Events() {
  const [filters, setFilters] = useState<Filters>({ chainId: '', eventName: '', contract: '' })
  const [result, setResult] = useState<NormalizedEventsResponse | null>(null)
  const [selectedEvent, setSelectedEvent] = useState<NormalizedEvent | null>(null)
  const [detailSource, setDetailSource] = useState<string>('')
  const [loading, setLoading] = useState(true)
  const [detailLoading, setDetailLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [limit] = useState(10)
  const [offset, setOffset] = useState(0)
  const [reorgedOnly, setReorgedOnly] = useState(false)

  async function loadEvents(nextOffset = offset, nextFilters: Filters = filters): Promise<void> {
    setLoading(true)
    try {
      const payload = await fetchEvents({
        limit,
        offset: nextOffset,
        chainId: nextFilters.chainId || undefined,
        eventName: nextFilters.eventName || undefined,
        contract: nextFilters.contract || undefined,
      })
      setResult(payload)
      setError(null)
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : 'Failed to load events')
    } finally {
      setLoading(false)
    }
  }

  async function loadDetail(eventId: string): Promise<void> {
    setDetailLoading(true)
    try {
      const detail = await fetchEventDetail(eventId)
      setSelectedEvent(detail.event)
      setDetailSource(detail.evidence.path)
    } catch (detailError) {
      setSelectedEvent(null)
      setDetailSource(detailError instanceof Error ? detailError.message : 'detail unavailable')
    } finally {
      setDetailLoading(false)
    }
  }

  useEffect(() => {
    void loadEvents(0)
  }, [])

  const canGoPrevious = offset > 0
  const canGoNext = Boolean(result) && offset + limit < (result?.pagination.total ?? 0)

  return (
    <div className="space-y-6">
      <section className="rounded-[28px] border border-white/10 bg-white/5 p-6">
        <div className="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
          <div>
            <p className="text-xs uppercase tracking-[0.25em] text-mist">REST Acceptance</p>
            <h2 className="mt-3 text-2xl font-semibold text-white">Event query, filtering, pagination, and detail evidence</h2>
            <p className="mt-3 max-w-3xl text-sm leading-6 text-sand/75">
              The console tries the current gateway `/events` path first and falls back to `/api/v1/events` for older service shapes, so the same demo works across multiple backend modes.
            </p>
          </div>
          <button
            onClick={() => void loadEvents(offset)}
            className="inline-flex items-center gap-2 rounded-full border border-white/15 bg-white/10 px-4 py-2 text-sm text-white transition hover:bg-white/15"
          >
            <RefreshCw className="h-4 w-4" />
            Refresh list
          </button>
        </div>

        <div className="mt-6 grid gap-4 md:grid-cols-3">
          <label className="rounded-2xl border border-white/10 bg-black/15 px-4 py-3">
            <span className="text-xs uppercase tracking-[0.2em] text-mist">Chain ID</span>
            <input
              value={filters.chainId}
              onChange={(event) => setFilters((current) => ({ ...current, chainId: event.target.value }))}
              placeholder="1 / ethereum / polygon"
              className="mt-3 w-full bg-transparent text-sm text-white outline-none placeholder:text-sand/35"
            />
          </label>
          <label className="rounded-2xl border border-white/10 bg-black/15 px-4 py-3">
            <span className="text-xs uppercase tracking-[0.2em] text-mist">Event Name</span>
            <input
              value={filters.eventName}
              onChange={(event) => setFilters((current) => ({ ...current, eventName: event.target.value }))}
              placeholder="Transfer / Swap"
              className="mt-3 w-full bg-transparent text-sm text-white outline-none placeholder:text-sand/35"
            />
          </label>
          <label className="rounded-2xl border border-white/10 bg-black/15 px-4 py-3">
            <span className="text-xs uppercase tracking-[0.2em] text-mist">Contract</span>
            <input
              value={filters.contract}
              onChange={(event) => setFilters((current) => ({ ...current, contract: event.target.value }))}
              placeholder="0x..."
              className="mt-3 w-full bg-transparent text-sm text-white outline-none placeholder:text-sand/35"
            />
          </label>
        </div>

        <div className="mt-4 flex flex-wrap gap-3">
          <button
            onClick={() => {
              setOffset(0)
              void loadEvents(0, filters)
            }}
            className="inline-flex items-center gap-2 rounded-full bg-glow px-4 py-2 text-sm font-medium text-ink transition hover:brightness-110"
          >
            <Search className="h-4 w-4" />
            Apply filters
          </button>
          <button
            onClick={() => {
              const cleared = { chainId: '', eventName: '', contract: '' }
              setFilters(cleared)
              setOffset(0)
              void loadEvents(0, cleared)
            }}
            className="rounded-full border border-white/15 bg-white/10 px-4 py-2 text-sm text-white transition hover:bg-white/15"
          >
            Clear filters
          </button>
          <button
            onClick={() => setReorgedOnly(!reorgedOnly)}
            className={`rounded-full border px-4 py-2 text-sm transition ${
              reorgedOnly
                ? 'border-amber-300/40 bg-amber-300/15 text-amber-100'
                : 'border-white/15 bg-white/10 text-white hover:bg-white/15'
            }`}
          >
            Reorged Only
          </button>
        </div>

        {result && (
          <div className="mt-4 rounded-2xl border border-white/10 bg-black/15 px-4 py-3 text-sm text-sand/75">
            Current request hit <span className="font-mono text-white">{result.evidence.path}</span> and returned{' '}
            <span className="text-white">{result.pagination.total}</span> total records.
          </div>
        )}
      </section>

      <section className="grid gap-6 xl:grid-cols-[1.25fr,0.95fr]">
        <article className="rounded-[28px] border border-white/10 bg-white/5 p-6">
          {loading ? (
            <div className="flex h-72 items-center justify-center">
              <Loader2 className="h-9 w-9 animate-spin text-glow" />
            </div>
          ) : error ? (
            <div className="rounded-2xl border border-rose-400/30 bg-rose-400/10 p-4 text-sm text-rose-100">{error}</div>
          ) : (
            <>
              <div className="overflow-hidden rounded-[22px] border border-white/10">
                <div className="hidden grid-cols-[1.3fr,0.8fr,1fr,0.9fr] gap-4 bg-black/25 px-4 py-3 text-xs uppercase tracking-[0.2em] text-mist md:grid">
                  <span>Event</span>
                  <span>Chain</span>
                  <span>Contract</span>
                  <span>Block</span>
                </div>
                <div className="divide-y divide-white/10">
                  {(reorgedOnly ? result?.events.filter((e) => e.status === 'reorged') : result?.events)?.map((event) => {
                    const isReorged = event.status === 'reorged'
                    return (
                    <button
                      key={event.id || event.eventId}
                      onClick={() => void loadDetail(event.id || event.eventId)}
                      className={`grid w-full gap-2 bg-transparent px-4 py-4 text-left transition hover:bg-white/5 md:grid-cols-[1.3fr,0.8fr,1fr,0.9fr] md:gap-4 ${isReorged ? 'border-l-2 border-l-amber-400' : ''}`}
                    >
                      <div>
                        <div className="flex items-center gap-2">
                          <span className="font-medium text-white">{event.eventName}</span>
                          {isReorged && (
                            <span className="rounded-full border border-amber-300/30 bg-amber-300/15 px-2 py-0.5 text-[10px] uppercase tracking-wider text-amber-200">
                              reorged
                            </span>
                          )}
                        </div>
                        <div className="mt-1 font-mono text-xs text-sand/55">{event.id || event.eventId}</div>
                      </div>
                      <div className="text-sm text-sand/75">{event.chainId}</div>
                      <div className="truncate font-mono text-sm text-sand/75">{event.contractAddress || '-'}</div>
                      <div className="text-sm text-sand/75">{event.blockNumber ?? '-'}</div>
                    </button>
                    )
                  })}
                  {(reorgedOnly ? result?.events.filter((e) => e.status === 'reorged') : result?.events)?.length === 0 && (
                    <div className="px-4 py-12 text-center text-sm text-sand/60">
                      No events matched the current filters, but the endpoint is still reachable and ready for acceptance checks with other criteria.
                    </div>
                  )}
                </div>
              </div>

              <div className="mt-4 flex items-center justify-between text-sm text-sand/75">
                <span>
                  Page {Math.floor((result?.pagination.offset || 0) / limit) + 1} of {Math.max(1, Math.ceil((result?.pagination.total || 0) / limit))}
                </span>
                <div className="flex gap-2">
                  <button
                    disabled={!canGoPrevious}
                    onClick={() => {
                      const nextOffset = Math.max(0, offset - limit)
                      setOffset(nextOffset)
                      void loadEvents(nextOffset)
                    }}
                    className="rounded-full border border-white/15 bg-white/10 px-4 py-2 text-white transition hover:bg-white/15 disabled:cursor-not-allowed disabled:opacity-40"
                  >
                    Previous
                  </button>
                  <button
                    disabled={!canGoNext}
                    onClick={() => {
                      const nextOffset = offset + limit
                      setOffset(nextOffset)
                      void loadEvents(nextOffset)
                    }}
                    className="rounded-full border border-white/15 bg-white/10 px-4 py-2 text-white transition hover:bg-white/15 disabled:cursor-not-allowed disabled:opacity-40"
                  >
                    Next
                  </button>
                </div>
              </div>
            </>
          )}
        </article>

        <article className="rounded-[28px] border border-white/10 bg-white/5 p-6">
          <div className="flex items-center gap-3">
            <div className="flex h-11 w-11 items-center justify-center rounded-2xl bg-glow/15">
              <Database className="h-5 w-5 text-glow" />
            </div>
            <div>
              <p className="text-xs uppercase tracking-[0.25em] text-mist">Event Detail</p>
              <h3 className="mt-1 text-lg font-medium text-white">Select any event on the left to inspect detail evidence</h3>
            </div>
          </div>

          {detailLoading ? (
            <div className="flex h-64 items-center justify-center">
              <Loader2 className="h-8 w-8 animate-spin text-glow" />
            </div>
          ) : selectedEvent ? (
            <div className="mt-6 space-y-4">
              <div className="rounded-2xl border border-white/10 bg-black/15 p-4">
                <div className="text-xs uppercase tracking-[0.2em] text-mist">Detail Path</div>
                <div className="mt-2 font-mono text-sm text-white">{detailSource}</div>
              </div>
              <div className="grid gap-3 sm:grid-cols-2">
                <div className="rounded-2xl border border-white/10 bg-black/15 p-4">
                  <div className="text-xs uppercase tracking-[0.2em] text-mist">Event Name</div>
                  <div className="mt-2 text-white">{selectedEvent.eventName}</div>
                </div>
                <div className="rounded-2xl border border-white/10 bg-black/15 p-4">
                  <div className="text-xs uppercase tracking-[0.2em] text-mist">Chain ID</div>
                  <div className="mt-2 text-white">{selectedEvent.chainId}</div>
                </div>
                <div className="rounded-2xl border border-white/10 bg-black/15 p-4">
                  <div className="text-xs uppercase tracking-[0.2em] text-mist">Block</div>
                  <div className="mt-2 text-white">{selectedEvent.blockNumber ?? '-'}</div>
                </div>
                <div className="rounded-2xl border border-white/10 bg-black/15 p-4">
                  <div className="text-xs uppercase tracking-[0.2em] text-mist">Timestamp</div>
                  <div className="mt-2 text-white">{formatTimestamp(selectedEvent.timestamp)}</div>
                </div>
              </div>
              <pre className="max-h-[340px] overflow-auto rounded-2xl border border-white/10 bg-black/25 p-4 text-xs text-sand/85">
                {JSON.stringify(selectedEvent.raw, null, 2)}
              </pre>
            </div>
          ) : (
            <div className="mt-6 rounded-2xl border border-dashed border-white/15 bg-black/10 p-6 text-sm leading-6 text-sand/65">
              No event detail has been loaded yet. Each row first tries `/events/:id` and falls back to `/api/v1/events/:id` if needed.
            </div>
          )}
        </article>
      </section>
    </div>
  )
}
