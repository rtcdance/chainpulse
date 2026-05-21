import { createContext, useCallback, useContext, useState, type ReactNode } from 'react'
import { getHttpBaseUrl } from './chainpulse'

interface AuthState {
  address: string
  token: string | null
  isAuthenticated: boolean
  step: 'idle' | 'connecting' | 'connected' | 'signing' | 'authenticated'
}

interface AuthContextValue extends AuthState {
  connect: () => Promise<void>
  signIn: () => Promise<void>
  signOut: () => void
  error: string | null
}

const AUTH_TOKEN_KEY = 'chainpulse_auth_token'
const AUTH_ADDRESS_KEY = 'chainpulse_auth_address'

const AuthContext = createContext<AuthContextValue | null>(null)

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}

export function getStoredToken(): string | null {
  return localStorage.getItem(AUTH_TOKEN_KEY)
}

export function getStoredAddress(): string | null {
  return localStorage.getItem(AUTH_ADDRESS_KEY)
}

async function fetchSIWEChallenge(address: string): Promise<{ message: string; nonce: string }> {
  const baseUrl = getHttpBaseUrl()
  const resp = await fetch(`${baseUrl}/auth/siwe/challenge`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ address }),
  })
  if (!resp.ok) {
    const err = await resp.json().catch(() => ({ message: 'Challenge request failed' }))
    throw new Error((err as Record<string, unknown>).message as string || 'Challenge request failed')
  }
  const data = await resp.json()
  return { message: data.data.message, nonce: data.data.nonce }
}

async function verifySIWE(message: string, signature: string): Promise<{ token: string; address: string }> {
  const baseUrl = getHttpBaseUrl()
  const resp = await fetch(`${baseUrl}/auth/siwe/verify`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ message, signature }),
  })
  if (!resp.ok) {
    const err = await resp.json().catch(() => ({ message: 'Verification failed' }))
    throw new Error((err as Record<string, unknown>).message as string || 'Verification failed')
  }
  const data = await resp.json()
  return { token: data.data.token, address: data.data.address }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const savedToken = localStorage.getItem(AUTH_TOKEN_KEY)
  const savedAddress = localStorage.getItem(AUTH_ADDRESS_KEY)

  const [state, setState] = useState<AuthState>(() => {
    if (savedToken && savedAddress) {
      return { address: savedAddress, token: savedToken, isAuthenticated: true, step: 'authenticated' }
    }
    return { address: '', token: null, isAuthenticated: false, step: 'idle' }
  })
  const [error, setError] = useState<string | null>(null)

  const connect = useCallback(async () => {
    setError(null)
    setState((prev) => ({ ...prev, step: 'connecting' }))

    try {
      if (typeof window === 'undefined' || !(window as unknown as Record<string, unknown>).ethereum) {
        setError('MetaMask not detected. Please install MetaMask to continue.')
        setState((prev) => ({ ...prev, step: 'idle' }))
        return
      }

      const ethereum = (window as unknown as Record<string, unknown>).ethereum as {
        request: (args: { method: string; params?: unknown[] }) => Promise<unknown>
      }

      const accounts = (await ethereum.request({ method: 'eth_requestAccounts' })) as string[]
      if (!accounts || accounts.length === 0) {
        setError('No accounts found. Please unlock MetaMask.')
        setState((prev) => ({ ...prev, step: 'idle' }))
        return
      }

      setState({ address: accounts[0], token: null, isAuthenticated: false, step: 'connected' })
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to connect wallet'
      setError(message)
      setState((prev) => ({ ...prev, step: 'idle' }))
    }
  }, [])

  const signIn = useCallback(async () => {
    setError(null)
    setState((prev) => ({ ...prev, step: 'signing' }))

    try {
      const ethereum = (window as unknown as Record<string, unknown>).ethereum as {
        request: (args: { method: string; params?: unknown[] }) => Promise<unknown>
      }

      const { message } = await fetchSIWEChallenge(state.address)

      const sig = (await ethereum.request({
        method: 'personal_sign',
        params: [message, state.address],
      })) as string

      if (!sig || sig.length < 10) {
        setError('Signature rejected or invalid')
        setState((prev) => ({ ...prev, step: 'connected' }))
        return
      }

      const { token } = await verifySIWE(message, sig)

      localStorage.setItem(AUTH_TOKEN_KEY, token)
      localStorage.setItem(AUTH_ADDRESS_KEY, state.address)

      setState({ address: state.address, token, isAuthenticated: true, step: 'authenticated' })
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Sign-in failed'
      setError(message)
      setState((prev) => ({ ...prev, step: 'connected' }))
    }
  }, [state.address])

  const signOut = useCallback(() => {
    localStorage.removeItem(AUTH_TOKEN_KEY)
    localStorage.removeItem(AUTH_ADDRESS_KEY)
    setState({ address: '', token: null, isAuthenticated: false, step: 'idle' })
    setError(null)
  }, [])

  return (
    <AuthContext.Provider value={{ ...state, connect, signIn, signOut, error }}>
      {children}
    </AuthContext.Provider>
  )
}