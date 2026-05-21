import { http } from '../http'
import type { RuntimePayload, ControlResult } from './types'
import { getField, toRecord, requestFirstMatch, getHttpBaseUrl, trimTrailingSlash } from './internal'

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

export async function postRuntimeControl(
  serviceId: 'puller' | 'event-processor',
  action: 'pause' | 'resume' | 'pause-intake' | 'resume-intake',
): Promise<ControlResult> {
  const { getServiceDefinitions } = await import('./services')
  const services = getServiceDefinitions()
  const service = services.find((s) => s.id === serviceId)
  const baseUrl = service?.baseUrl || getHttpBaseUrl()

  try {
    const response = await http.request<Record<string, unknown>>({
      method: 'POST',
      url: `${trimTrailingSlash(baseUrl)}/runtime/control/${action}`,
      timeout: 8000,
      validateStatus: () => true,
    })

    const ok = response.status >= 200 && response.status < 300
    const body = toRecord(response.data)
    return {
      success: ok,
      message: String(getField(body, ['message', 'status'], ok ? `${action} succeeded` : `${action} failed (${response.status})`)),
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