import { RefreshCw, Search, ArrowDown, ArrowUp, X, Layers, Download } from 'lucide-react'
import { exportToCSV, exportToJSON, type NormalizedEventsResponse } from '../lib/chainpulse'

export interface Filters {
  chainId: string
  eventName: string
  contract: string
  fromBlock: string
  toBlock: string
  search: string
}

export type SortOrder = 'desc' | 'asc'
export type ViewDensity = 'normal' | 'compact'

interface EventFiltersProps {
  filters: Filters
  reorgedOnly: boolean
  sortOrder: SortOrder
  density: ViewDensity
  result: NormalizedEventsResponse | null
  onFiltersChange: (filters: Filters) => void
  onApply: () => void
  onClear: () => void
  onToggleReorged: () => void
  onToggleSort: () => void
  onToggleDensity: () => void
  onRefresh: () => void
}

export default function EventFilters({
  filters,
  reorgedOnly,
  sortOrder,
  density,
  result,
  onFiltersChange,
  onApply,
  onClear,
  onToggleReorged,
  onToggleSort,
  onToggleDensity,
  onRefresh,
}: EventFiltersProps) {
  const updateField = (field: keyof Filters, value: string) => {
    onFiltersChange({ ...filters, [field]: value })
  }

  const clearField = (field: keyof Filters) => {
    onFiltersChange({ ...filters, [field]: '' })
  }

  return (
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
          onClick={onRefresh}
          className="inline-flex items-center gap-2 rounded-full border border-white/15 bg-white/10 px-4 py-2 text-sm text-white transition hover:bg-white/15"
        >
          <RefreshCw className="h-4 w-4" />
          Refresh list
        </button>
      </div>

      <div className="mt-6 grid gap-4 md:grid-cols-3 lg:grid-cols-5">
        <FilterInput
          label="Chain ID"
          value={filters.chainId}
          placeholder="1 / ethereum / solana"
          onChange={(v) => updateField('chainId', v)}
          onClear={() => clearField('chainId')}
        />
        <FilterInput
          label="Event Name"
          value={filters.eventName}
          placeholder="Transfer / Swap"
          onChange={(v) => updateField('eventName', v)}
          onClear={() => clearField('eventName')}
        />
        <FilterInput
          label="Contract"
          value={filters.contract}
          placeholder="0x..."
          onChange={(v) => updateField('contract', v)}
          onClear={() => clearField('contract')}
        />
        <FilterInput
          label="From Block"
          type="number"
          value={filters.fromBlock}
          placeholder="e.g. 19000000"
          onChange={(v) => updateField('fromBlock', v)}
          onClear={() => clearField('fromBlock')}
        />
        <FilterInput
          label="To Block"
          type="number"
          value={filters.toBlock}
          placeholder="e.g. 20000000"
          onChange={(v) => updateField('toBlock', v)}
          onClear={() => clearField('toBlock')}
        />
        <label className="relative rounded-2xl border border-white/10 bg-black/15 px-4 py-3">
          <div className="flex items-center justify-between gap-2">
            <span className="text-xs uppercase tracking-[0.2em] text-mist">Full Text Search</span>
            <Search className="h-4 w-4 text-mist" />
          </div>
          <input
            value={filters.search}
            onChange={(e) => updateField('search', e.target.value)}
            placeholder="Transfer / 0x... / ethereum"
            className="mt-3 w-full bg-transparent pr-8 text-sm text-white outline-none placeholder:text-sand/35"
          />
          {filters.search && (
            <button
              onClick={() => clearField('search')}
              className="absolute bottom-[14px] right-3 rounded-full p-0.5 text-sand/50 transition hover:text-white hover:bg-white/10"
              aria-label="Clear Search"
            >
              <X className="h-4 w-4" />
            </button>
          )}
        </label>
      </div>

      <div className="mt-4 flex flex-wrap gap-3">
        <button
          onClick={onApply}
          className="inline-flex items-center gap-2 rounded-full bg-glow px-4 py-2 text-sm font-medium text-ink transition hover:brightness-110"
        >
          <Search className="h-4 w-4" />
          Apply filters
        </button>
        <button
          onClick={onClear}
          className="rounded-full border border-white/15 bg-white/10 px-4 py-2 text-sm text-white transition hover:bg-white/15"
        >
          Clear filters
        </button>
        <button
          onClick={onToggleReorged}
          className={`rounded-full border px-4 py-2 text-sm transition ${
            reorgedOnly
              ? 'border-amber-300/40 bg-amber-300/15 text-amber-100'
              : 'border-white/15 bg-white/10 text-white hover:bg-white/15'
          }`}
        >
          Reorged Only
        </button>
        <button
          onClick={onToggleSort}
          className="inline-flex items-center gap-1.5 rounded-full border border-white/15 bg-white/10 px-4 py-2 text-sm text-white transition hover:bg-white/15"
        >
          {sortOrder === 'desc' ? <ArrowDown className="h-4 w-4" /> : <ArrowUp className="h-4 w-4" />}
          {sortOrder === 'desc' ? 'Newest' : 'Oldest'}
        </button>
        <button
          onClick={onToggleDensity}
          className="inline-flex items-center gap-1.5 rounded-full border border-white/15 bg-white/10 px-4 py-2 text-sm text-white transition hover:bg-white/15"
        >
          <Layers className="h-4 w-4" />
          {density === 'normal' ? 'Compact' : 'Normal'}
        </button>
        <ExportButtons result={result} />
      </div>

      {result && (
        <div className="mt-4 rounded-2xl border border-white/10 bg-black/15 px-4 py-3 text-sm text-sand/75">
          Current request hit <span className="font-mono text-white">{result.evidence.path}</span> and returned{' '}
          <span className="text-white">{result.pagination.total}</span> total records.
        </div>
      )}
    </section>
  )
}

function FilterInput({
  label,
  value,
  placeholder,
  type,
  onChange,
  onClear,
}: {
  label: string
  value: string
  placeholder: string
  type?: string
  onChange: (value: string) => void
  onClear: () => void
}) {
  return (
    <label className="relative rounded-2xl border border-white/10 bg-black/15 px-4 py-3">
      <span className="text-xs uppercase tracking-[0.2em] text-mist">{label}</span>
      <input
        type={type || 'text'}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        className="mt-3 w-full bg-transparent pr-8 text-sm text-white outline-none placeholder:text-sand/35"
      />
      {value && (
        <button
          onClick={onClear}
          className="absolute bottom-[14px] right-3 rounded-full p-0.5 text-sand/50 transition hover:text-white hover:bg-white/10"
          aria-label={`Clear ${label}`}
        >
          <X className="h-4 w-4" />
        </button>
      )}
    </label>
  )
}

function ExportButtons({ result }: { result: NormalizedEventsResponse | null }) {
  const hasEvents = Boolean(result?.events.length)
  return (
    <div className="flex items-center gap-2">
      <button
        onClick={() => {
          if (!result?.events.length) return
          exportToCSV(result.events as unknown as Record<string, unknown>[], [
            { key: 'eventName', label: 'Event' },
            { key: 'chainId', label: 'Chain' },
            { key: 'contractAddress', label: 'Contract' },
            { key: 'blockNumber', label: 'Block' },
            { key: 'status', label: 'Status' },
          ], `events-${new Date().toISOString().slice(0, 10)}.csv`)
        }}
        disabled={!hasEvents}
        className="inline-flex items-center gap-1.5 rounded-full border border-white/15 bg-white/10 px-4 py-2 text-sm text-white transition hover:bg-white/15 disabled:cursor-not-allowed disabled:opacity-40"
      >
        <Download className="h-4 w-4" />
        CSV
      </button>
      <button
        onClick={() => {
          if (!result?.events.length) return
          exportToJSON(result.events, `events-${new Date().toISOString().slice(0, 10)}.json`)
        }}
        disabled={!hasEvents}
        className="inline-flex items-center gap-1.5 rounded-full border border-white/15 bg-white/10 px-4 py-2 text-sm text-white transition hover:bg-white/15 disabled:cursor-not-allowed disabled:opacity-40"
      >
        <Download className="h-4 w-4" />
        JSON
      </button>
    </div>
  )
}