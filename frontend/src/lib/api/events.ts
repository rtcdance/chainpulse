import type { NormalizedEventsResponse, NormalizedEvent, EventStats, EndpointEvidence } from './types'
import { getField, toRecord, toNumber, normalizeEvent, requestFirstMatch, getHttpBaseUrl } from './internal'

export async function fetchEvents(filters: {
  limit: number
  offset: number
  chainId?: string
  eventName?: string
  contract?: string
  fromBlock?: number
  toBlock?: number
  search?: string
  sort?: string
}): Promise<NormalizedEventsResponse> {
  const search = new URLSearchParams()
  search.set('limit', String(filters.limit))
  search.set('offset', String(filters.offset))
  if (filters.chainId) {
    search.set('chain_id', filters.chainId)
  }
  if (filters.eventName) {
    search.set('event_name', filters.eventName)
  }
  if (filters.contract) {
    search.set('contract', filters.contract)
  }
  if (filters.fromBlock !== undefined) {
    search.set('from_block', String(filters.fromBlock))
  }
  if (filters.toBlock !== undefined) {
    search.set('to_block', String(filters.toBlock))
  }
  if (filters.search) {
    search.set('search', filters.search)
  }
  if (filters.sort) {
    search.set('sort', filters.sort)
  }

  return requestFirstMatch<NormalizedEventsResponse>(
    [`/events?${search.toString()}`, `/api/v1/events?${search.toString()}`],
    { method: 'GET' },
    (response, candidate) => {
      const body = toRecord(response.data)
      const list = Array.isArray(body.events)
        ? body.events
        : Array.isArray(body.data)
          ? body.data
          : []
      const paginationRecord = toRecord(getField(body, ['pagination'], {}))
      const total = toNumber(getField(body, ['total'], paginationRecord.total)) ?? list.length
      const limit = toNumber(paginationRecord.limit) ?? filters.limit
      const offset = toNumber(paginationRecord.offset) ?? filters.offset

      return {
        events: list.map(normalizeEvent),
        pagination: {
          limit,
          offset,
          total,
        },
        evidence: { label: 'Events', path: candidate },
      }
    },
  )
}

export async function fetchEventDetail(eventId: string): Promise<{ event: NormalizedEvent; evidence: EndpointEvidence }> {
  return requestFirstMatch<{ event: NormalizedEvent; evidence: EndpointEvidence }>(
    [`/events/${eventId}`, `/api/v1/events/${eventId}`],
    { method: 'GET' },
    (response, candidate) => {
      const body = toRecord(response.data)
      const event = body.data ? normalizeEvent(body.data) : normalizeEvent(body)
      return {
        event,
        evidence: { label: 'Event Detail', path: candidate },
      }
    },
  )
}

export async function fetchEventStats(): Promise<EventStats> {
  return requestFirstMatch<EventStats>(
    ['/events/stats', '/api/v1/events/stats'],
    { method: 'GET' },
    (response, candidate) => {
      const body = toRecord(response.data)
      return {
        total: toNumber(getField(body, ['total'], 0)) ?? 0,
        byChain: getField(body, ['byChain', 'by_chain'], {}) as Record<string, number>,
        byEventName: getField(body, ['byEventName', 'by_event_name'], {}) as Record<string, number>,
        reorged: toNumber(getField(body, ['reorged'], 0)) ?? 0,
        evidence: { label: 'Event Stats', path: candidate },
      }
    },
  )
}

export async function exportEvents(format: 'csv' | 'json', filters: {
  limit?: number
  chainId?: string
  eventName?: string
  contract?: string
  startTime?: number
  endTime?: number
}): Promise<void> {
  const search = new URLSearchParams()
  search.set('format', format)
  if (filters.limit) search.set('limit', String(filters.limit))
  if (filters.chainId) search.set('chain_id', filters.chainId)
  if (filters.eventName) search.set('event_name', filters.eventName)
  if (filters.contract) search.set('contract', filters.contract)
  if (filters.startTime) search.set('start_time', String(filters.startTime))
  if (filters.endTime) search.set('end_time', String(filters.endTime))

  const base = getHttpBaseUrl()
  const url = `${base}/events/export?${search.toString()}`
  const token = localStorage.getItem('chainpulse_auth_token')
  const headers: Record<string, string> = {}
  if (token) headers['Authorization'] = `Bearer ${token}`

  const response = await fetch(url, { headers })

  if (!response.ok) {
    throw new Error(`Export failed with status ${response.status}`)
  }

  const blob = await response.blob()
  const downloadUrl = window.URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = downloadUrl
  link.download = `events.${format}`
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  window.URL.revokeObjectURL(downloadUrl)
}