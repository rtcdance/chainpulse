import type { MetricsPayload, MetricSample } from './types'
import { requestFirstMatch } from './internal'

export async function fetchMetrics(): Promise<MetricsPayload> {
  return requestFirstMatch<MetricsPayload>(
    ['/metrics'],
    { method: 'GET', responseType: 'text' },
    (response, candidate) => {
      const raw = String(response.data || '')
      const samples = raw
        .split('\n')
        .map((line) => line.trim())
        .filter((line) => line.startsWith('chainpulse_') && !line.startsWith('#'))
        .map((line) => {
          const match = line.match(/^([^{\s]+)(\{[^}]*\})?\s+(.+)$/)
          return {
            name: match?.[1] || line,
            labels: match?.[2] || '',
            value: match?.[3] || '',
          }
        })

      return {
        raw,
        samples,
        evidence: { label: 'Metrics', path: candidate },
      }
    },
  )
}

export function summarizeMetricGroups(samples: MetricSample[]): Array<{ name: string; count: number }> {
  const groups = new Map<string, number>()

  for (const sample of samples) {
    const parts = sample.name.replace(/^chainpulse_/, '').split('_')
    const key = parts[0] || 'other'
    groups.set(key, (groups.get(key) || 0) + 1)
  }

  return [...groups.entries()]
    .map(([name, count]) => ({ name, count }))
    .sort((left, right) => right.count - left.count)
}