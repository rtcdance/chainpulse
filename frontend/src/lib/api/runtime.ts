import type { RuntimePayload, ControlResult } from './types'
import { getField, toRecord, requestFirstMatch, getHttpBaseUrl, trimTrailingSlash } from './internal'

export async function fetchRuntimeSummary(): Promise<RuntimePayload> {
  return requestFirstMatch<RuntimePayload>(
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

export async function postRuntimeControl(
  serviceId: 'puller' | 'event-processor',
  action: 'pause' | 'resume' | 'pause-intake' | 'resume-intake',
): Promise<ControlResult> {
  const { getServiceDefinitions } = await import('./services')
  const services = getServiceDefinitions()
  const service = services.find((s) => s.id === serviceId)
  const baseUrl = service?.baseUrl || getHttpBaseUrl()

  try {
    const url = `${trimTrailingSlash(baseUrl)}/runtime/control/${action}`
    const token = localStorage.getItem('chainpulse_auth_token')
    const headers: Record<string, string> = {}
    if (token) headers['Authorization'] = `Bearer ${token}`

    const response = await fetch(url, { method: 'POST', headers })
    const body = toRecord(await response.json())
    return {
      success: response.ok,
      message: String(getField(body, ['message', 'status'], response.ok ? `${action} succeeded` : `${action} failed (${response.status})`)),
      evidence: { label: 'Runtime Control', path: `/runtime/control/${action}` },
    }
  } catch (error) {
    return {
      success: false,
      message: error instanceof Error ? error.message : 'request failed',
      evidence: { label: 'Runtime Control', path: `/runtime/control/${action}` },
    }
  }
}