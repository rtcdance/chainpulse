import axios, { type AxiosError, type AxiosInstance, type AxiosRequestConfig, type AxiosResponse, type InternalAxiosRequestConfig } from 'axios'

const DEFAULT_TIMEOUT = 12_000

function createInstance(): AxiosInstance {
  const instance = axios.create({
    timeout: DEFAULT_TIMEOUT,
    headers: { 'Accept': 'application/json' },
  })

  instance.interceptors.request.use(
    (config: InternalAxiosRequestConfig) => {
      const token = localStorage.getItem('chainpulse_auth_token')
      if (token) {
        config.headers.Authorization = `Bearer ${token}`
      }
      return config
    },
    (error: AxiosError) => Promise.reject(error),
  )

  instance.interceptors.response.use(
    (response: AxiosResponse) => response,
    (error: AxiosError) => {
      if (error.code === 'ERR_CANCELED') {
        return Promise.reject(error)
      }
      if (error.response) {
        const { status, data } = error.response
        const msg = typeof data === 'object' && data !== null
          ? String((data as Record<string, unknown>).message || JSON.stringify(data).slice(0, 120))
          : String(data || '').slice(0, 120)
        return Promise.reject(new Error(`HTTP ${status}: ${msg}`))
      }
      if (error.request) {
        return Promise.reject(new Error(`Network error: ${error.message}`))
      }
      return Promise.reject(error)
    },
  )

  return instance
}

export const http: AxiosInstance = createInstance()

export function httpRequest<T = unknown>(config: AxiosRequestConfig): Promise<AxiosResponse<T>> {
  return http.request<T>(config)
}