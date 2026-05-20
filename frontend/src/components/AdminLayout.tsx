import type { ReactNode } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { Activity, FileJson, BarChart3, Zap, Server, AlertTriangle } from 'lucide-react'

const adminTabs: { path: string; label: string; icon: ReactNode }[] = [
  { path: '/admin', label: 'Overview', icon: <Activity size={16} /> },
  { path: '/admin/graphql', label: 'GraphQL', icon: <FileJson size={16} /> },
  { path: '/admin/metrics', label: 'Metrics', icon: <BarChart3 size={16} /> },
  { path: '/admin/websocket', label: 'WebSocket', icon: <Zap size={16} /> },
  { path: '/admin/runtime', label: 'Runtime', icon: <Server size={16} /> },
  { path: '/admin/dlq', label: 'DLQ', icon: <AlertTriangle size={16} /> },
]

export default function AdminLayout({ children }: { children: ReactNode }) {
  const navigate = useNavigate()
  const location = useLocation()

  return (
    <div className="space-y-6">
      <nav className="flex flex-wrap gap-1.5 rounded-2xl border border-white/10 bg-white/5 p-1.5">
        {adminTabs.map((tab) => {
          const isActive = location.pathname === tab.path
          return (
            <button
              key={tab.path}
              onClick={() => navigate(tab.path)}
              className={`inline-flex items-center gap-2 rounded-xl px-4 py-2 text-sm font-medium transition-all ${
                isActive
                  ? 'bg-glow/15 text-glow shadow-[0_4px_12px_rgba(244,162,97,0.12)]'
                  : 'text-sand/60 hover:bg-white/5 hover:text-sand/90'
              }`}
            >
              <span className={isActive ? 'text-glow' : 'text-mist'}>{tab.icon}</span>
              {tab.label}
            </button>
          )
        })}
      </nav>
      {children}
    </div>
  )
}