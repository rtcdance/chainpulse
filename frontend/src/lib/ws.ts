const DIRECT_SERVICE_PORTS = new Set(['8080', '8081', '8082', '8083'])

function trimTrailingSlash(value: string): string {
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
    if (isViteProxyOrigin()) {
      const serviceId = 'api-gateway'
      return `${protocol}//${window.location.host}/__ws/${serviceId}${path}`
    }

    return `${protocol}//${window.location.host}${path}`
  }

  return `ws://localhost:8080${path}`
}