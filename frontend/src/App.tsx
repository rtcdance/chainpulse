import { lazy, Suspense, useEffect } from 'react'
import { Routes, Route, Navigate, useNavigate, useLocation } from 'react-router-dom'
import Sidebar from './components/Sidebar'
import { PageSkeleton } from './components/Skeleton'
import { useAuth } from './lib/auth'
import { ToastProvider } from './lib/toast'
import type { View } from './components/Sidebar'

const Landing = lazy(() => import('./components/Landing'))
const Dashboard = lazy(() => import('./components/Dashboard'))
const Events = lazy(() => import('./components/Events'))
const AdminDashboard = lazy(() => import('./components/AdminDashboard'))
const GraphQL = lazy(() => import('./components/GraphQL'))
const Metrics = lazy(() => import('./components/Metrics'))
const WebSocket = lazy(() => import('./components/WebSocket'))
const Runtime = lazy(() => import('./components/Runtime'))
const DLQ = lazy(() => import('./components/DLQ'))

function resolveView(pathname: string): View {
  if (pathname === '/events') return 'events'
  if (pathname.startsWith('/admin')) return 'admin'
  return 'dashboard'
}

function AppShell() {
  const navigate = useNavigate()
  const location = useLocation()
  const currentView = resolveView(location.pathname)

  useEffect(() => {
    function handleKeyDown(event: KeyboardEvent): void {
      if (event.target instanceof HTMLInputElement || event.target instanceof HTMLTextAreaElement) return
      const map: Record<string, string> = {
        '1': '/dashboard', '2': '/events', '3': '/admin',
      }
      const next = map[event.key]
      if (next) navigate(next)
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [navigate])

  return (
    <div className="relative mx-auto flex min-h-screen max-w-[1600px] flex-col lg:flex-row">
      <Sidebar currentView={currentView} />
      <main className="flex-1 px-4 pb-10 pt-4 sm:px-6 lg:px-8 lg:pt-8">
        <Suspense fallback={<PageSkeleton />}>
          <ToastProvider>
            <Routes>
              <Route path="/" element={<Navigate to="/dashboard" replace />} />
              <Route path="/dashboard" element={<Dashboard />} />
              <Route path="/events" element={<Events />} />
              <Route path="/admin" element={<AdminDashboard />} />
              <Route path="/admin/graphql" element={<GraphQL />} />
              <Route path="/admin/metrics" element={<Metrics />} />
              <Route path="/admin/websocket" element={<WebSocket />} />
              <Route path="/admin/runtime" element={<Runtime />} />
              <Route path="/admin/dlq" element={<DLQ />} />
            </Routes>
          </ToastProvider>
        </Suspense>
      </main>
    </div>
  )
}

export default function App() {
  const { isAuthenticated } = useAuth()

  return (
    <div className="min-h-screen bg-ink text-sand">
      <div className="absolute inset-0 bg-grid opacity-60" />
      <div className="absolute inset-x-0 top-0 h-[28rem] bg-[radial-gradient(circle_at_top,rgba(244,162,97,0.22),transparent_55%)]" />

      <Suspense fallback={<PageSkeleton />}>
        <Routes>
          <Route path="/" element={<Landing />} />
          <Route path="/dashboard" element={
            isAuthenticated ? <AppShell /> : <Navigate to="/" replace />
          } />
          <Route path="/events" element={
            isAuthenticated ? <AppShell /> : <Navigate to="/" replace />
          } />
          <Route path="/admin/*" element={
            isAuthenticated ? <AppShell /> : <Navigate to="/" replace />
          } />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </Suspense>
    </div>
  )
}