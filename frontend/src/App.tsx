import { useEffect, useState } from 'react'
import Sidebar from './components/Sidebar'
import Dashboard from './components/Dashboard'
import Events from './components/Events'
import GraphQL from './components/GraphQL'
import Metrics from './components/Metrics'
import WebSocket from './components/WebSocket'
import Runtime from './components/Runtime'
import DLQ from './components/DLQ'
import { getHttpBaseLabel, getWebSocketBaseLabel } from './lib/chainpulse'

type View = 'dashboard' | 'events' | 'graphql' | 'metrics' | 'websocket' | 'runtime' | 'dlq'

function App() {
  const [view, setView] = useState<View>('dashboard')

  useEffect(() => {
    function handleKeyDown(event: KeyboardEvent): void {
      if (event.target instanceof HTMLInputElement || event.target instanceof HTMLTextAreaElement) return
      const map: Record<string, View> = {
        '1': 'dashboard', '2': 'events', '3': 'graphql',
        '4': 'metrics', '5': 'websocket', '6': 'runtime', '7': 'dlq',
      }
      const next = map[event.key]
      if (next) setView(next)
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [])

  const renderView = () => {
    switch (view) {
      case 'dashboard': return <Dashboard />
      case 'events': return <Events />
      case 'graphql': return <GraphQL />
      case 'metrics': return <Metrics />
      case 'websocket': return <WebSocket />
      case 'runtime': return <Runtime />
      case 'dlq': return <DLQ />
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
                <p className="text-xs uppercase tracking-[0.35em] text-mist">ChainPulse Learning Dashboard</p>
                <h1 className="mt-3 text-3xl font-semibold text-white sm:text-4xl">
                  ChainPulse 学习仪表盘 — 可视化理解区块链索引器
                </h1>
                <p className="mt-3 max-w-3xl text-sm leading-6 text-sand/80 sm:text-base">
                  每个页面展示 ChainPulse 的一个核心能力。数据来自运行中的后端，配合调试器断点可以完整追踪一笔事件从 RPC 拉取到 API 查询的全路径。按 <kbd className="rounded border border-white/10 bg-white/5 px-1.5 py-0.5 font-mono text-xs">1</kbd>-<kbd className="rounded border border-white/10 bg-white/5 px-1.5 py-0.5 font-mono text-xs">7</kbd> 切换页面。
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
