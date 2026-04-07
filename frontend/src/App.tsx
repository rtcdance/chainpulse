import { useState } from 'react'
import Sidebar from './components/Sidebar'
import Dashboard from './components/Dashboard'
import Events from './components/Events'
import GraphQL from './components/GraphQL'
import Metrics from './components/Metrics'
import WebSocket from './components/WebSocket'
import Runtime from './components/Runtime'
import { getHttpBaseLabel, getWebSocketBaseLabel } from './lib/chainpulse'

type View = 'dashboard' | 'events' | 'graphql' | 'metrics' | 'websocket' | 'runtime'

function App() {
  const [view, setView] = useState<View>('dashboard')

  const renderView = () => {
    switch (view) {
      case 'dashboard': return <Dashboard />
      case 'events': return <Events />
      case 'graphql': return <GraphQL />
      case 'metrics': return <Metrics />
      case 'websocket': return <WebSocket />
      case 'runtime': return <Runtime />
      default: return <Dashboard />
    }
  }

  return (
    <div className="min-h-screen bg-ink text-sand">
      <div className="absolute inset-0 bg-grid opacity-60" />
      <div className="absolute inset-x-0 top-0 h-[28rem] bg-[radial-gradient(circle_at_top,rgba(244,162,97,0.28),transparent_55%)]" />
      <div className="relative mx-auto flex min-h-screen max-w-[1600px] flex-col lg:flex-row">
        <Sidebar currentView={view} onNavigate={setView} />
        <main className="flex-1 px-4 pb-10 pt-4 sm:px-6 lg:px-8 lg:pt-8">
          <section className="mb-6 overflow-hidden rounded-[28px] border border-white/10 bg-white/5 p-5 shadow-[0_30px_120px_rgba(8,12,19,0.45)] backdrop-blur">
            <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
              <div>
                <p className="text-xs uppercase tracking-[0.35em] text-mist">ChainPulse Acceptance Console</p>
                <h1 className="mt-3 text-3xl font-semibold text-white sm:text-4xl">
                  One H5 console for health, query, subscription, metrics, and runtime acceptance
                </h1>
                <p className="mt-3 max-w-3xl text-sm leading-6 text-sand/80 sm:text-base">
                  This demo is built for acceptance, not for decoration. Every page executes real backend actions and preserves endpoint evidence so the team can verify the current ChainPulse slice live.
                </p>
              </div>
              <div className="grid gap-3 sm:grid-cols-2">
                <div className="rounded-2xl border border-white/10 bg-black/20 px-4 py-3">
                  <div className="text-xs uppercase tracking-[0.25em] text-mist">HTTP Base</div>
                  <div className="mt-2 break-all font-mono text-sm text-white">{getHttpBaseLabel()}</div>
                </div>
                <div className="rounded-2xl border border-white/10 bg-black/20 px-4 py-3">
                  <div className="text-xs uppercase tracking-[0.25em] text-mist">WebSocket Base</div>
                  <div className="mt-2 break-all font-mono text-sm text-white">{getWebSocketBaseLabel()}</div>
                </div>
              </div>
            </div>
          </section>

          {renderView()}
        </main>
      </div>
    </div>
  )
}

export default App
