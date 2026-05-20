import { Database, Loader2 } from 'lucide-react'
import { formatTimestamp, type NormalizedEvent } from '../lib/chainpulse'

interface EventDetailProps {
  event: NormalizedEvent | null
  loading: boolean
  source: string
}

export default function EventDetail({ event, loading, source }: EventDetailProps) {
  return (
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

      {loading ? (
        <div className="flex h-64 items-center justify-center">
          <Loader2 className="h-8 w-8 animate-spin text-glow" />
        </div>
      ) : event ? (
        <div className="mt-6 space-y-4">
          <div className="rounded-2xl border border-white/10 bg-black/15 p-4">
            <div className="text-xs uppercase tracking-[0.2em] text-mist">Detail Path</div>
            <div className="mt-2 font-mono text-sm text-white">{source}</div>
          </div>
          <div className="grid gap-3 sm:grid-cols-2">
            <Field label="Event Name" value={event.eventName} />
            <Field label="Chain ID" value={event.chainId} />
            <Field label="Block" value={event.blockNumber ?? '-'} />
            <Field label="Timestamp" value={formatTimestamp(event.timestamp)} />
          </div>
          <pre className="max-h-[340px] overflow-auto rounded-2xl border border-white/10 bg-black/25 p-4 text-xs text-sand/85">
            {JSON.stringify(event.raw, null, 2)}
          </pre>
        </div>
      ) : (
        <div className="mt-6 rounded-2xl border border-dashed border-white/15 bg-black/10 p-6 text-sm leading-6 text-sand/65">
          No event detail has been loaded yet. Each row first tries `/events/:id` and falls back to `/api/v1/events/:id` if needed.
        </div>
      )}
    </article>
  )
}

function Field({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="rounded-2xl border border-white/10 bg-black/15 p-4">
      <div className="text-xs uppercase tracking-[0.2em] text-mist">{label}</div>
      <div className="mt-2 text-white">{value}</div>
    </div>
  )
}