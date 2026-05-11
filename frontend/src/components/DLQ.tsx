import { useEffect, useState } from 'react'
import { AlertTriangle, Loader2, RefreshCw, RotateCcw } from 'lucide-react'
import { fetchDLQEvents, formatTimestamp, replayDLQEvents, type ControlResult, type DLQEventList } from '../lib/chainpulse'

export default function DLQ() {
  const [result, setResult] = useState<DLQEventList | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [replayResult, setReplayResult] = useState<ControlResult | null>(null)
  const [confirmReplayAll, setConfirmReplayAll] = useState(false)
  const [replaying, setReplaying] = useState(false)
  const [replayingIds, setReplayingIds] = useState<Set<string>>(new Set())

  async function loadDLQ(): Promise<void> {
    setLoading(true)
    try {
      const data = await fetchDLQEvents({ limit: 50, offset: 0 })
      setResult(data)
      setError(null)
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : 'Failed to load DLQ events')
    } finally {
      setLoading(false)
    }
  }

  async function replayOne(eventId: string): Promise<void> {
    setReplayingIds((prev) => new Set(prev).add(eventId))
    try {
      const res = await replayDLQEvents([eventId])
      setReplayResult(res)
      if (res.success) {
        await loadDLQ()
      }
    } catch (err) {
      setReplayResult({
        success: false,
        message: err instanceof Error ? err.message : 'replay failed',
        evidence: { label: 'DLQ Replay', path: '/dlq/replay' },
      })
    } finally {
      setReplayingIds((prev) => {
        const next = new Set(prev)
        next.delete(eventId)
        return next
      })
    }
  }

  async function replayAll(): Promise<void> {
    setReplaying(true)
    setConfirmReplayAll(false)
    try {
      const res = await replayDLQEvents()
      setReplayResult(res)
      if (res.success) {
        await loadDLQ()
      }
    } catch (err) {
      setReplayResult({
        success: false,
        message: err instanceof Error ? err.message : 'replay failed',
        evidence: { label: 'DLQ Replay', path: '/dlq/replay' },
      })
    } finally {
      setReplaying(false)
    }
  }

  useEffect(() => {
    void loadDLQ()
  }, [])

  const events = result?.events || []

  return (
    <div className="space-y-6">
      <section className="rounded-[28px] border border-white/10 bg-white/5 p-6">
        <div className="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
          <div>
            <p className="text-xs uppercase tracking-[0.25em] text-mist">Dead Letter Queue</p>
            <h2 className="mt-3 text-2xl font-semibold text-white">Inspect and replay failed events</h2>
            <p className="mt-3 max-w-3xl text-sm leading-6 text-sand/75">
              Events that failed processing land in the DLQ. Inspect the failure reason, then replay individual events or the entire queue. Each replay sends the event back through the processing pipeline.
            </p>
          </div>
          <div className="flex gap-3">
            <button
              onClick={() => void loadDLQ()}
              className="inline-flex items-center gap-2 rounded-full border border-white/15 bg-white/10 px-4 py-2 text-sm text-white transition hover:bg-white/15"
            >
              <RefreshCw className="h-4 w-4" />
              Refresh
            </button>
            {events.length > 0 && (
              <button
                onClick={() => setConfirmReplayAll(true)}
                disabled={replaying}
                className="inline-flex items-center gap-2 rounded-full bg-glow px-4 py-2 text-sm font-medium text-ink transition hover:brightness-110 disabled:cursor-not-allowed disabled:opacity-60"
              >
                <RotateCcw className="h-4 w-4" />
                Replay All
              </button>
            )}
          </div>
        </div>

        <div className="mt-5 grid gap-4 md:grid-cols-2">
          <div className="rounded-2xl border border-white/10 bg-black/15 p-4">
            <div className="text-xs uppercase tracking-[0.2em] text-mist">DLQ Size</div>
            <div className="mt-2 text-2xl font-semibold text-white">{result?.total ?? '-'}</div>
          </div>
          <div className="rounded-2xl border border-white/10 bg-black/15 p-4">
            <div className="text-xs uppercase tracking-[0.2em] text-mist">Source</div>
            <div className="mt-2 font-mono text-sm text-white">{result?.evidence.path || '/dlq/events'}</div>
          </div>
        </div>
      </section>

      {confirmReplayAll && (
        <div className="rounded-2xl border border-amber-300/30 bg-amber-300/10 p-5">
          <p className="text-sm text-amber-50">
            Confirm: <span className="font-semibold text-white">Replay all {events.length} dead-lettered events</span>?
          </p>
          <div className="mt-3 flex gap-3">
            <button
              onClick={() => void replayAll()}
              disabled={replaying}
              className="inline-flex items-center gap-2 rounded-full bg-glow px-4 py-2 text-sm font-medium text-ink transition hover:brightness-110 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {replaying ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
              Confirm Replay All
            </button>
            <button
              onClick={() => setConfirmReplayAll(false)}
              className="rounded-full border border-white/15 bg-white/10 px-4 py-2 text-sm text-white transition hover:bg-white/15"
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      {replayResult && (
        <div className={`rounded-2xl border p-5 ${
          replayResult.success
            ? 'border-emerald-300/30 bg-emerald-300/10 text-emerald-100'
            : 'border-rose-400/30 bg-rose-400/10 text-rose-100'
        }`}>
          <p className="text-sm font-medium text-white">{replayResult.success ? 'Replay succeeded' : 'Replay failed'}</p>
          <p className="mt-1 text-sm">{replayResult.message}</p>
          <p className="mt-2 font-mono text-xs text-sand/70">Endpoint: {replayResult.evidence.path}</p>
        </div>
      )}

      {loading ? (
        <div className="flex h-72 items-center justify-center rounded-[28px] border border-white/10 bg-white/5">
          <Loader2 className="h-9 w-9 animate-spin text-glow" />
        </div>
      ) : error ? (
        <div className="rounded-[28px] border border-rose-400/30 bg-rose-400/10 p-6 text-sm text-rose-100">{error}</div>
      ) : events.length === 0 ? (
        <div className="rounded-[28px] border border-dashed border-white/15 bg-black/10 p-12 text-center">
          <AlertTriangle className="mx-auto h-10 w-10 text-emerald-300" />
          <h3 className="mt-4 text-lg font-medium text-white">DLQ is clean</h3>
          <p className="mt-2 text-sm text-sand/70">No dead-lettered events found. All events are processing successfully.</p>
        </div>
      ) : (
        <section className="rounded-[28px] border border-white/10 bg-white/5 p-6">
          <div className="overflow-hidden rounded-[22px] border border-white/10">
            <div className="hidden grid-cols-[1.2fr,0.7fr,0.6fr,0.8fr,1.3fr] gap-4 bg-black/25 px-4 py-3 text-xs uppercase tracking-[0.2em] text-mist md:grid">
              <span>Event</span>
              <span>Chain</span>
              <span>Retries</span>
              <span>Timestamp</span>
              <span>Reason</span>
            </div>
            <div className="divide-y divide-white/10">
              {events.map((event) => (
                <div
                  key={event.id}
                  className="grid w-full gap-2 bg-transparent px-4 py-4 text-left transition hover:bg-white/5 md:grid-cols-[1.2fr,0.7fr,0.6fr,0.8fr,1.3fr] md:gap-4"
                >
                  <div>
                    <div className="font-medium text-white">{event.eventName}</div>
                    <div className="mt-1 font-mono text-xs text-sand/55">{event.id}</div>
                  </div>
                  <div className="text-sm text-sand/75">{event.chainId}</div>
                  <div className="text-sm text-sand/75">{event.retryCount}</div>
                  <div className="text-sm text-sand/75">{formatTimestamp(event.timestamp)}</div>
                  <div className="flex items-center gap-3">
                    <span className="flex-1 truncate text-sm text-rose-200/80">{event.reason || 'unknown'}</span>
                    <button
                      onClick={() => void replayOne(event.id)}
                      disabled={replayingIds.has(event.id)}
                      className="shrink-0 inline-flex items-center gap-1 rounded-full border border-amber-300/30 bg-amber-300/10 px-3 py-1 text-xs text-amber-100 transition hover:bg-amber-300/20 disabled:cursor-not-allowed disabled:opacity-50"
                    >
                      {replayingIds.has(event.id) ? <Loader2 className="h-3 w-3 animate-spin" /> : <RotateCcw className="h-3 w-3" />}
                      Replay
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </section>
      )}
    </div>
  )
}
