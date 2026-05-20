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
  status: string
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

export interface DLQEvent {
  id: string
  eventName: string
  chainId: string
  reason: string
  retryCount: number
  timestamp: number | null
  raw: Record<string, unknown>
}

export interface DLQEventList {
  events: DLQEvent[]
  total: number
  evidence: EndpointEvidence
}

export interface ControlResult {
  success: boolean
  message: string
  evidence: EndpointEvidence
}

export interface EventStats {
  total: number
  byChain: Record<string, number>
  byEventName: Record<string, number>
  reorged: number
  evidence: EndpointEvidence
}