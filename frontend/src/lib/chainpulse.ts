import axios, { type AxiosRequestConfig, type AxiosResponse } from 'axios'

export interface EndpointEvidence {
  label: string
  path: string
}

export interface NormalizedEvent {
  id: string
  eventId: string
  eventName: string
  chainId: string
  contractAddress: string
  blockNumber: number | null
  timestamp: number | null
  transactionHash: string
  raw: Record<string, unknown>
}

export interface NormalizedEventsResponse {
  events: NormalizedEvent[]
  pagination: {
    limit: number
    offset: number
    total: number
  }
  evidence: EndpointEvidence
}

export interface HealthPayload {
  status: string
  timestamp?: string | number
  version?: string
  components: Record<string, Record<string, unknown>>
  evidence: EndpointEvidence
}

export interface RuntimePayload {
  service?: string
  runtimeMode?: string
  deploymentMode?: string
  summary: Record<string, unknown>
  evidence: EndpointEvidence
}

export interface GraphQLPayload {
  body: Record<string, unknown>
  evidence: EndpointEvidence
}

export interface MetricSample {
  name: string
  labels: string
  value: string
}

export interface MetricsPayload {
  raw: string
  samples: MetricSample[]
  evidence: EndpointEvidence
}

export interface ServiceDefinition {
  id: 'api-gateway' | 'api-service' | 'event-processor' | 'puller'
  name: string
  role: string
  baseUrl: string
}

export interface EndpointProbe {
  path: string
  ok: boolean
  status: number | null
  summary: string
  contentType: string
}

export interface ServiceAcceptanceReport {
  service: ServiceDefinition
  probes: EndpointProbe[]
}

const DEFAULT_HTTP_BASE = 'http://localhost:8080'

function trimTrailingSlash(value: string): string {
  return value.replace(/\/+$/, '')
}

function getHttpBaseUrl(): string {
  const explicit = import.meta.env.VITE_CHAINPULSE_BASE_URL
  if (explicit) {
    return trimTrailingSlash(explicit)
  }

  if (typeof window !== 'undefined') {
    const { origin, port } = window.location
    if (port === '3000' || port === '5173' || port === '4173') {
      return ''
    }
    return trimTrailingSlash(origin)
  }

  return DEFAULT_HTTP_BASE
}

export function getHttpBaseLabel(): string {
  return getHttpBaseUrl() || 'vite-proxy -> http://localhost:8080'
}

function getEnvBaseUrl(variable: string, fallback: string): string {
  const value = import.meta.env[variable]
  return value ? trimTrailingSlash(value) : fallback
}

export function getServiceDefinitions(): ServiceDefinition[] {
  return [
    {
      id: 'api-gateway',
      name: 'API Gateway',
      role: 'External entrypoint, query bridge, GraphQL, WebSocket',
      baseUrl: getEnvBaseUrl('VITE_API_GATEWAY_BASE_URL', 'http://localhost:8080'),
    },
    {
      id: 'api-service',
      name: 'API Service',
      role: 'Event query backend and runtime summary source',
      baseUrl: getEnvBaseUrl('VITE_API_SERVICE_BASE_URL', 'http://localhost:8081'),
    },
    {
      id: 'event-processor',
      name: 'Event Processor',
      role: 'Execution path, runtime summary, runtime control',
      baseUrl: getEnvBaseUrl('VITE_EVENT_PROCESSOR_BASE_URL', 'http://localhost:8082'),
    },
    {
      id: 'puller',
      name: 'Puller',
      role: 'Execution path, polling/runtime control, chain ingestion',
      baseUrl: getEnvBaseUrl('VITE_PULLER_BASE_URL', 'http://localhost:8083'),
    },
  ]
}

export function getWebSocketBaseLabel(): string {
  return buildWebSocketUrl('/ws')
}

export function buildWebSocketUrl(path: string): string {
  const explicit = import.meta.env.VITE_CHAINPULSE_WS_URL
  if (explicit) {
    return `${trimTrailingSlash(explicit)}${path}`
  }

  if (typeof window !== 'undefined') {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.port === '3000' || window.location.port === '5173' || window.location.port === '4173'
      ? 'localhost:8080'
      : window.location.host
    return `${protocol}//${host}${path}`
  }

  return `ws://localhost:8080${path}`
}

function getField<T>(source: Record<string, unknown>, keys: string[], fallback: T): T {
  for (const key of keys) {
    const value = source[key]
    if (value !== undefined && value !== null) {
      return value as T
    }
  }
  return fallback
}

function toRecord(value: unknown): Record<string, unknown> {
  if (typeof value === 'object' && value !== null && !Array.isArray(value)) {
    return value as Record<string, unknown>
  }
  return {}
}

function toRecordMap(value: unknown): Record<string, Record<string, unknown>> {
  const source = toRecord(value)
  return Object.fromEntries(
    Object.entries(source).map(([key, entry]) => [key, toRecord(entry)]),
  )
}

function toNumber(value: unknown): number | null {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return value
  }
  if (typeof value === 'string' && value.trim() !== '') {
    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : null
  }
  return null
}

function normalizeEvent(value: unknown): NormalizedEvent {
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
    raw,
  }
}

async function requestFirstMatch<TResponse, TResult>(
  candidates: string[],
  config: AxiosRequestConfig,
  transform: (response: AxiosResponse<TResponse>, candidate: string) => TResult,
): Promise<TResult> {
  let lastError: unknown = null

  for (const candidate of candidates) {
    try {
      const response = await axios.request<TResponse>({
        ...config,
        url: `${getHttpBaseUrl()}${candidate}`,
        validateStatus: () => true,
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

async function probeEndpoint(baseUrl: string, path: string): Promise<EndpointProbe> {
  try {
    const response = await axios.request<string | Record<string, unknown>>({
      method: 'GET',
      url: `${trimTrailingSlash(baseUrl)}${path}`,
      responseType: 'text',
      timeout: 4000,
      validateStatus: () => true,
    })
    const text = typeof response.data === 'string' ? response.data : JSON.stringify(response.data)
    return {
      path,
      ok: response.status >= 200 && response.status < 300,
      status: response.status,
      summary: text.slice(0, 180),
      contentType: String(response.headers['content-type'] || ''),
    }
  } catch (error) {
    return {
      path,
      ok: false,
      status: null,
      summary: error instanceof Error ? error.message : 'request failed',
      contentType: '',
    }
  }
}

export async function fetchCurrentSliceReport(): Promise<ServiceAcceptanceReport[]> {
  const services = getServiceDefinitions()
  const endpointMap: Record<ServiceDefinition['id'], string[]> = {
    'api-gateway': ['/health', '/health/ready', '/health/live', '/health/components', '/health/rollout', '/runtime/summary', '/metrics', '/events?limit=3', '/graphql', '/models'],
    'api-service': ['/health', '/health/ready', '/health/live', '/health/components', '/health/rollout', '/runtime/summary', '/metrics', '/api/v1/events?limit=3', '/graphql', '/ws'],
    'event-processor': ['/health', '/health/ready', '/health/live', '/health/components', '/health/rollout', '/runtime/summary', '/runtime/control', '/metrics'],
    'puller': ['/health', '/health/ready', '/health/live', '/health/components', '/health/rollout', '/runtime/summary', '/runtime/control', '/metrics', '/status'],
  }

  return Promise.all(
    services.map(async (service) => ({
      service,
      probes: await Promise.all(endpointMap[service.id].map((path) => probeEndpoint(service.baseUrl, path))),
    })),
  )
}

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

export async function fetchRuntimeSummary(): Promise<RuntimePayload> {
  return requestFirstMatch<Record<string, unknown>, RuntimePayload>(
    ['/runtime/summary'],
    { method: 'GET' },
    (response, candidate) => {
      const body = toRecord(response.data)
      return {
        service: String(getField(body, ['service'], 'unknown')),
        runtimeMode: String(getField(body, ['runtime_mode', 'runtimeMode'], 'unknown')),
        deploymentMode: String(getField(body, ['deployment_mode', 'deploymentMode'], 'unknown')),
        summary: body,
        evidence: { label: 'Runtime Summary', path: candidate },
      }
    },
  )
}

export async function fetchMetrics(): Promise<MetricsPayload> {
  return requestFirstMatch<string, MetricsPayload>(
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

export async function fetchEvents(filters: {
  limit: number
  offset: number
  chainId?: string
  eventName?: string
  contract?: string
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

  return requestFirstMatch<Record<string, unknown>, NormalizedEventsResponse>(
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
  return requestFirstMatch<Record<string, unknown>, { event: NormalizedEvent; evidence: EndpointEvidence }>(
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

export async function executeGraphQL(query: string): Promise<GraphQLPayload> {
  return requestFirstMatch<Record<string, unknown>, GraphQLPayload>(
    ['/graphql'],
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      data: { query },
    },
    (response, candidate) => ({
      body: toRecord(response.data),
      evidence: { label: 'GraphQL', path: candidate },
    }),
  )
}

export function formatTimestamp(value: number | null): string {
  if (!value) {
    return '-'
  }

  const millis = value > 1_000_000_000_000 ? value : value * 1000
  const date = new Date(millis)
  if (Number.isNaN(date.getTime())) {
    return String(value)
  }
  return date.toLocaleString()
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
