import type { DLQEventList, ControlResult } from './types'
import { getField, toRecord, toNumber, requestFirstMatch } from './internal'

export async function fetchDLQEvents(params?: { limit?: number; offset?: number }): Promise<DLQEventList> {
  const search = new URLSearchParams()
  if (params?.limit) search.set('limit', String(params.limit))
  if (params?.offset) search.set('offset', String(params.offset))
  const qs = search.toString()

  return requestFirstMatch<DLQEventList>(
    [`/dlq/events${qs ? `?${qs}` : ''}`, `/api/v1/dlq/events${qs ? `?${qs}` : ''}`],
    { method: 'GET' },
    (response, candidate) => {
      const body = toRecord(response.data)
      const list = Array.isArray(body.events) ? body.events : Array.isArray(body.data) ? body.data : []
      const total = toNumber(getField(body, ['total'], list.length)) ?? list.length

      return {
        events: list.map((item: unknown) => {
          const raw = toRecord(item)
          return {
            id: String(getField(raw, ['id', 'eventId', 'event_id'], '')),
            eventName: String(getField(raw, ['eventName', 'event_name'], 'unknown')),
            chainId: String(getField(raw, ['chainId', 'chain_id'], 'unknown')),
            reason: String(getField(raw, ['reason', 'deadLetterReason', 'dead_letter_reason'], '')),
            retryCount: toNumber(getField(raw, ['retryCount', 'retry_count'], 0)) ?? 0,
            timestamp: toNumber(getField(raw, ['timestamp', 'processedAt'], null)),
            raw,
          }
        }),
        total,
        evidence: { label: 'DLQ Events', path: candidate },
      }
    },
  )
}

export async function replayDLQEvents(eventIDs?: string[]): Promise<ControlResult> {
  const body = eventIDs?.length ? { event_ids: eventIDs } : {}

  return requestFirstMatch<ControlResult>(
    ['/dlq/replay', '/api/v1/dlq/replay', '/runtime/indexing/dlq/replay'],
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body,
    },
    (response, candidate) => {
      const resBody = toRecord(response.data)
      return {
        success: true,
        message: String(getField(resBody, ['message', 'status'], 'replay initiated')),
        evidence: { label: 'DLQ Replay', path: candidate },
      }
    },
  )
}