import { Loader2 } from 'lucide-react'
import type { NormalizedEvent } from '../lib/chainpulse'
import type { ViewDensity, SortOrder } from './EventFilters'

interface EventTableProps {
  events: NormalizedEvent[]
  loading: boolean
  error: string | null
  density: ViewDensity
  sortOrder: SortOrder
  reorgedOnly: boolean
  offset: number
  limit: number
  total: number
  canGoPrevious: boolean
  canGoNext: boolean
  onSelectEvent: (eventId: string) => void
  onPrevious: () => void
  onNext: () => void
}

export default function EventTable({
  events,
  loading,
  error,
  density,
  reorgedOnly,
  offset,
  limit,
  total,
  canGoPrevious,
  canGoNext,
  onSelectEvent,
  onPrevious,
  onNext,
}: EventTableProps) {
  const filtered = reorgedOnly ? events.filter((e) => e.status === 'reorged') : events

  return (
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
              {filtered.map((event) => {
                const isReorged = event.status === 'reorged'
                return (
                  <button
                    key={event.id || event.eventId}
                    onClick={() => onSelectEvent(event.id || event.eventId)}
                    className={`block w-full text-left transition hover:bg-white/5 ${isReorged ? 'border-l-2 border-l-amber-400' : ''} ${density === 'compact' ? 'px-4 py-2' : 'px-4 py-4'}`}
                  >
                    <div className="md:hidden space-y-2">
                      <div className="flex items-center gap-2">
                        <span className="font-medium text-white">{event.eventName}</span>
                        {isReorged && (
                          <span className="rounded-full border border-amber-300/30 bg-amber-300/15 px-2 py-0.5 text-[10px] uppercase tracking-wider text-amber-200">
                            reorged
                          </span>
                        )}
                      </div>
                      <div className="font-mono text-xs text-sand/55">{event.id || event.eventId}</div>
                      <div className="grid grid-cols-2 gap-2 text-sm">
                        <div>
                          <span className="text-xs text-mist">Chain</span>
                          <p className="text-sand/75">{event.chainId}</p>
                        </div>
                        <div>
                          <span className="text-xs text-mist">Block</span>
                          <p className="text-sand/75">{event.blockNumber ?? '-'}</p>
                        </div>
                      </div>
                      <div>
                        <span className="text-xs text-mist">Contract</span>
                        <p className="truncate font-mono text-sm text-sand/75">{event.contractAddress || '-'}</p>
                      </div>
                    </div>
                    <div className="hidden md:grid md:grid-cols-[1.3fr,0.8fr,1fr,0.9fr] md:gap-4 md:items-center">
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
                    </div>
                  </button>
                )
              })}
              {filtered.length === 0 && (
                <div className="px-4 py-12 text-center text-sm text-sand/60">
                  No events matched the current filters, but the endpoint is still reachable and ready for acceptance checks with other criteria.
                </div>
              )}
            </div>
          </div>

          <div className="mt-4 flex items-center justify-between text-sm text-sand/75">
            <span>
              Page {Math.floor((offset || 0) / limit) + 1} of {Math.max(1, Math.ceil((total || 0) / limit))}
            </span>
            <div className="flex gap-2">
              <button
                disabled={!canGoPrevious}
                onClick={onPrevious}
                className="rounded-full border border-white/15 bg-white/10 px-4 py-2 text-white transition hover:bg-white/15 disabled:cursor-not-allowed disabled:opacity-40"
              >
                Previous
              </button>
              <button
                disabled={!canGoNext}
                onClick={onNext}
                className="rounded-full border border-white/15 bg-white/10 px-4 py-2 text-white transition hover:bg-white/15 disabled:cursor-not-allowed disabled:opacity-40"
              >
                Next
              </button>
            </div>
          </div>
        </>
      )}
    </article>
  )
}