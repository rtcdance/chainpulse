import type { ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'
import { Activity, Database, LogOut, Server, Zap } from 'lucide-react'
import { useAuth } from '../lib/auth'
import { getHttpBaseLabel } from '../lib/chainpulse'

export type View = 'dashboard' | 'events' | 'admin'

interface SidebarProps {
  currentView: View
}

const menuItems: { id: View; path: string; label: string; icon: ReactNode; shortcut: string }[] = [
  { id: 'dashboard', path: '/dashboard', label: 'Dashboard', icon: <Activity size={20} />, shortcut: '1' },
  { id: 'events', path: '/events', label: 'Events', icon: <Database size={20} />, shortcut: '2' },
  { id: 'admin', path: '/admin', label: 'Admin', icon: <Server size={20} />, shortcut: '3' },
]

function formatAddress(address: string): string {
  return `${address.slice(0, 6)}...${address.slice(-4)}`
}

export default function Sidebar({ currentView }: SidebarProps) {
  const navigate = useNavigate()
  const { address, signOut } = useAuth()

  return (
    <aside className="border-b border-white/10 bg-black/15 px-4 py-4 backdrop-blur lg:sticky lg:top-0 lg:h-screen lg:w-72 lg:border-b-0 lg:border-r lg:px-6 lg:py-6">
      <div className="flex items-center gap-3">
        <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-[linear-gradient(135deg,rgba(244,162,97,0.95),rgba(233,196,106,0.9))] shadow-[0_8px_20px_rgba(244,162,97,0.22)]">
          <Zap className="text-ink" size={20} />
        </div>
        <div>
          <h1 className="text-base font-semibold text-white">ChainPulse</h1>
          <p className="text-[10px] uppercase tracking-[0.25em] text-mist">Multi-Chain Indexing</p>
        </div>
      </div>

      <div className="mt-5 rounded-2xl border border-white/10 bg-white/5 p-3">
        <p className="text-xs text-sand/50">Connected as</p>
        <p className="mt-1 font-mono text-sm text-white">{formatAddress(address)}</p>
        <p className="mt-0.5 truncate font-mono text-[10px] text-sand/30">{address}</p>
      </div>

      <nav className="mt-5 grid gap-2 sm:grid-cols-2 lg:grid-cols-1">
        {menuItems.map((item) => (
          <button
            key={item.id}
            onClick={() => navigate(item.path)}
            className={`w-full rounded-2xl border px-4 py-3 text-left transition-all ${
              currentView === item.id
                ? 'border-glow/50 bg-glow/15 text-white shadow-[0_8px_24px_rgba(244,162,97,0.12)]'
                : 'border-white/5 bg-black/10 text-sand/70 hover:border-white/10 hover:bg-white/5 hover:text-white'
            }`}
          >
            <div className="flex items-center gap-3">
              <span className={currentView === item.id ? 'text-glow' : 'text-mist'}>{item.icon}</span>
              <span className="font-medium text-sm">{item.label}</span>
              <span className={`ml-auto text-xs ${currentView === item.id ? 'text-glow/60' : 'text-sand/30'}`}>{item.shortcut}</span>
            </div>
          </button>
        ))}
      </nav>

      <div className="mt-5 rounded-2xl border border-white/10 bg-black/20 p-3">
        <div className="text-[10px] uppercase tracking-[0.2em] text-mist">API Endpoint</div>
        <p className="mt-1 truncate font-mono text-[11px] text-white/70">{getHttpBaseLabel()}</p>
        <div className="mt-3 text-[11px] leading-5 text-sand/55">
          Press <kbd className="rounded border border-white/10 bg-white/5 px-1 py-0.5 font-mono text-[10px]">1</kbd>-<kbd className="rounded border border-white/10 bg-white/5 px-1 py-0.5 font-mono text-[10px]">3</kbd> to navigate.
        </div>
      </div>

      <div className="mt-auto pt-4">
        <button
          onClick={signOut}
          className="flex w-full items-center gap-2 rounded-2xl border border-rose-400/15 bg-rose-400/5 px-4 py-2.5 text-sm text-rose-200/70 transition hover:border-rose-400/25 hover:bg-rose-400/10 hover:text-rose-200"
        >
          <LogOut className="h-4 w-4" />
          Sign Out
        </button>
      </div>
    </aside>
  )
}