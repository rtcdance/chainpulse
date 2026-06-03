import type { ServiceDefinition, ServiceAcceptanceReport, EndpointProbe } from './types'
import { trimTrailingSlash } from './internal'

const DIRECT_SERVICE_PORTS = new Set(['8080', '8081', '8082', '8083'])

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

export function getServiceBaseUrl(
  serviceId: ServiceDefinition['id'],
  variable: string,
  fallback: string,
): string {
  const explicit = import.meta.env[variable]
  if (explicit) {
    return trimTrailingSlash(explicit)
  }

  if (typeof window !== 'undefined' && isViteProxyOrigin()) {
    return `${trimTrailingSlash(window.location.origin)}/__proxy/${serviceId}`
  }

  return fallback
}

export function getServiceDefinitions(): ServiceDefinition[] {
  return [
    {
      id: 'api-gateway',
      name: 'API Gateway',
      role: 'External entrypoint, query bridge, GraphQL, WebSocket',
      baseUrl: getServiceBaseUrl('api-gateway', 'VITE_API_GATEWAY_BASE_URL', 'http://localhost:8080'),
    },
    {
      id: 'api-service',
      name: 'API Service',
      role: 'Event query backend and runtime summary source',
      baseUrl: getServiceBaseUrl('api-service', 'VITE_API_SERVICE_BASE_URL', 'http://localhost:8081'),
    },
    {
      id: 'event-processor',
      name: 'Event Processor',
      role: 'Execution path, runtime summary, runtime control',
      baseUrl: getServiceBaseUrl('event-processor', 'VITE_EVENT_PROCESSOR_BASE_URL', 'http://localhost:8082'),
    },
    {
      id: 'puller',
      name: 'Puller',
      role: 'Execution path, polling/runtime control, chain ingestion',
      baseUrl: getServiceBaseUrl('puller', 'VITE_PULLER_BASE_URL', 'http://localhost:8083'),
    },
  ]
}

export function buildFilteredSubscribeUrl(filters: { chainId?: string; contract?: string; eventName?: string }): string {
  if (filters.eventName) return `/events/subscribe/name/${encodeURIComponent(filters.eventName)}`
  if (filters.contract) return `/events/subscribe/contract/${encodeURIComponent(filters.contract)}`
  if (filters.chainId) return `/events/subscribe/chain/${encodeURIComponent(filters.chainId)}`
  return '/events/subscribe'
}

export async function probeEndpoint(baseUrl: string, path: string): Promise<EndpointProbe> {
  try {
    const url = `${trimTrailingSlash(baseUrl)}${path}`
    const token = localStorage.getItem('chainpulse_auth_token')
    const headers: Record<string, string> = {}
    if (token) headers['Authorization'] = `Bearer ${token}`

    const abortController = new AbortController()
    const timeoutId = setTimeout(() => abortController.abort(), 8000)

    const response = await fetch(url, { headers, signal: abortController.signal })
    clearTimeout(timeoutId)

    const text = await response.text()
    return {
      path,
      ok: response.ok,
      status: response.status,
      summary: text.slice(0, 180),
      contentType: response.headers.get('content-type') || '',
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
    'api-service': ['/health', '/health/ready', '/health/live', '/health/components', '/health/rollout', '/runtime/summary', '/metrics', '/api/v1/events?limit=3', '/graphql'],
    'event-processor': ['/health', '/health/ready', '/health/live', '/health/components', '/health/rollout', '/runtime/summary', '/runtime/control', '/metrics'],
    'puller': ['/health', '/health/ready', '/health/live', '/health/components', '/health/rollout', '/runtime/summary', '/runtime/control', '/metrics'],
  }

  return Promise.all(
    services.map(async (service) => ({
      service,
      probes: await Promise.all(endpointMap[service.id].map((path) => probeEndpoint(service.baseUrl, path))),
    })),
  )
}