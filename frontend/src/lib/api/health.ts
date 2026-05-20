import type { HealthPayload } from './types'
import { getField, toRecord, toRecordMap, requestFirstMatch } from './internal'

export async function fetchHealth(): Promise<HealthPayload> {
  return requestFirstMatch<Record<string, unknown>, HealthPayload>(
    ['/health'],
    { method: 'GET' },
    (response, candidate) => {
      const body = toRecord(response.data)
      return {
        status: String(getField(body, ['status'], 'unknown')),
        timestamp: getField(body, ['timestamp'], undefined),
        version: getField(body, ['version'], undefined),
        components: toRecordMap(getField(body, ['components'], {})),
        evidence: { label: 'Health', path: candidate },
      }
    },
  )
}