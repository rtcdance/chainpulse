import { type AxiosRequestConfig, type AxiosResponse } from 'axios'
import { http } from '../http'
import type { NormalizedEvent } from './types'

const DEFAULT_HTTP_BASE = 'http://localhost:8080'
const DIRECT_SERVICE_PORTS = new Set(['8080', '8081', '8082', '8083'])

export function trimTrailingSlash(value: string): string {
  return value.replace(/\/+$/, '')
}

function isViteProxyOrigin(): boolean {
  if (typeof window === 'undefined') {
    return false
  }

  const { hostname, port } = window.location
  if (!['localhost', '127.0.0.1'].includes(hostname)) {
    return false
  }

  return port !== '' && !DIRECT_SERVICE_PORTS.has(port)
}

export function getHttpBaseUrl(): string {
  const explicit = import.meta.env.VITE_CHAINPULSE_BASE_URL
  if (explicit) {
    return trimTrailingSlash(explicit)
  }

  if (typeof window !== 'undefined') {
    const { origin } = window.location
    if (isViteProxyOrigin()) {
      return ''
    }
    return trimTrailingSlash(origin)
  }

  return DEFAULT_HTTP_BASE
}

export function getHttpBaseLabel(): string {
  return getHttpBaseUrl() || 'vite-proxy -> http://localhost:8080'
}

export function getField<T>(source: Record<string, unknown>, keys: string[], fallback: T): T {
  for (const key of keys) {
    const value = source[key]
    if (value !== undefined && value !== null) {
      return value as T
    }
  }
  return fallback
}

export function toRecord(value: unknown): Record<string, unknown> {
  if (typeof value === 'object' && value !== null && !Array.isArray(value)) {
    return value as Record<string, unknown>
  }
  return {}
}

export function toRecordMap(value: unknown): Record<string, Record<string, unknown>> {
  const source = toRecord(value)
  return Object.fromEntries(
    Object.entries(source).map(([key, entry]) => [key, toRecord(entry)]),
  )
}

export function toNumber(value: unknown): number | null {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return value
  }
  if (typeof value === 'string' && value.trim() !== '') {
    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : null
  }
  return null
}

export function normalizeEvent(value: unknown): NormalizedEvent {
  const raw = toRecord(value)
  const id = String(getField(raw, ['id', 'eventId', 'event_id'], ''))

  return {
    id,
    eventId: String(getField(raw, ['eventId', 'event_id', 'id'], id)),
    eventName: String(getField(raw, ['eventName', 'event_name'], 'unknown')),
    chainId: String(getField(raw, ['chainId', 'chain_id'], 'unknown')),
    contractAddress: String(getField(raw, ['contractAddress', 'contract_address'], '')),
    blockNumber: toNumber(getField(raw, ['blockNumber', 'block_number'], null)),
    timestamp: toNumber(getField(raw, ['timestamp', 'blockTimestamp', 'block_timestamp', 'processedAt'], null)),
    transactionHash: String(getField(raw, ['transactionHash', 'transaction_hash'], '')),
    status: String(getField(raw, ['status'], '')),
    raw,
  }
}

export async function requestFirstMatch<TResponse, TResult>(
  candidates: string[],
  config: AxiosRequestConfig,
  transform: (response: AxiosResponse<TResponse>, candidate: string) => TResult,
  signal?: AbortSignal,
): Promise<TResult> {
  let lastError: unknown = null

  for (const candidate of candidates) {
    try {
      const response = await http.request<TResponse>({
        ...config,
        url: `${getHttpBaseUrl()}${candidate}`,
        validateStatus: () => true,
        signal,
      })

      if (response.status >= 200 && response.status < 300) {
        return transform(response, candidate)
      }

      if (response.status === 404 || response.status === 405) {
        lastError = new Error(`endpoint unavailable: ${candidate}`)
        continue
      }

      throw new Error(`request failed ${response.status} on ${candidate}`)
    } catch (error) {
      lastError = error
    }
  }

  throw lastError instanceof Error ? lastError : new Error('request failed')
}