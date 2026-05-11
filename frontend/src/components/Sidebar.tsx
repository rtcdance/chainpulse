import type { ReactNode } from 'react'
import { Activity, Database, FileJson, BarChart3, Zap, Server, AlertTriangle } from 'lucide-react'
import { getHttpBaseLabel } from '../lib/chainpulse'

type View = 'dashboard' | 'events' | 'graphql' | 'metrics' | 'websocket' | 'runtime' | 'dlq'

interface SidebarProps {
  currentView: View
  onNavigate: (view: View) => void
}

const menuItems: { id: View; label: string; icon: ReactNode; shortcut: string }[] = [
  { id: 'dashboard', label: 'Dashboard', icon: <Activity size={20} />, shortcut: '1' },
  { id: 'events', label: 'Events', icon: <Database size={20} />, shortcut: '2' },
  { id: 'graphql', label: 'GraphQL', icon: <FileJson size={20} />, shortcut: '3' },
  { id: 'metrics', label: 'Metrics', icon: <BarChart3 size={20} />, shortcut: '4' },
  { id: 'websocket', label: 'WebSocket', icon: <Zap size={20} />, shortcut: '5' },
  { id: 'runtime', label: 'Runtime', icon: <Server size={20} />, shortcut: '6' },
  { id: 'dlq', label: 'DLQ', icon: <AlertTriangle size={20} />, shortcut: '7' },
]

export default function Sidebar({ currentView, onNavigate }: SidebarProps) {
  return (
    <aside className="border-b border-white/10 bg-black/15 px-4 py-4 backdrop-blur lg:sticky lg:top-0 lg:h-screen lg:w-80 lg:border-b-0 lg:border-r lg:px-6 lg:py-6">
      <div className="flex items-center gap-3">
        <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-[linear-gradient(135deg,rgba(244,162,97,0.95),rgba(233,196,106,0.9))] shadow-[0_12px_30px_rgba(244,162,97,0.28)]">
          <Activity className="text-ink" size={24} />
        </div>
        <div>
          <h1 className="text-lg font-semibold text-white">ChainPulse</h1>
          <p className="text-xs uppercase tracking-[0.25em] text-mist">Acceptance H5 Console</p>
        </div>
      </div>

      <div className="mt-6 rounded-[24px] border border-white/10 bg-white/5 p-4">
        <p className="text-sm leading-6 text-sand/80">
          The goal is not a flashy dashboard. The goal is a clickable, provable, reviewable acceptance surface for the current runtime.
        </p>
      </div>

      <nav className="mt-6 grid gap-2 sm:grid-cols-2 lg:grid-cols-1">
        {menuItems.map((item) => (
          <button
            key={item.id}
            onClick={() => onNavigate(item.id)}
            className={`w-full rounded-2xl border px-4 py-3 text-left transition-all ${
              currentView === item.id
                ? 'border-glow/50 bg-glow/15 text-white shadow-[0_12px_30px_rgba(244,162,97,0.15)]'
                : 'border-white/5 bg-black/10 text-sand/70 hover:border-white/10 hover:bg-white/5 hover:text-white'
            }`}
          >
            <div className="flex items-center gap-3">
              <span className={currentView === item.id ? 'text-glow' : 'text-mist'}>{item.icon}</span>
              <span className="font-medium">{item.label}</span>
              <span className={`ml-auto text-xs ${currentView === item.id ? 'text-glow/60' : 'text-sand/30'}`}>{item.shortcut}</span>
            </div>
          </button>
        ))}
      </nav>

      <div className="mt-6 rounded-[24px] border border-white/10 bg-black/20 p-4">
        <div className="text-xs uppercase tracking-[0.25em] text-mist">Current Target</div>
        <p className="mt-2 break-all font-mono text-xs leading-5 text-white/85">
          {getHttpBaseLabel()}
        </p>
        <div className="mt-4 text-xs leading-5 text-sand/70">
          Navigate with sidebar or keys 1-7. Start with Dashboard, then walk through each surface.
        </div>
      </div>
    </aside>
  )
}
