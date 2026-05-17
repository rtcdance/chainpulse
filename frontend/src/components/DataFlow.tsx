interface DataFlowStep {
  step: number
  label: string
  file: string
  active?: boolean
}

const defaultSteps: DataFlowStep[] = [
  { step: 1, label: 'RPC Poll', file: 'https_jsonrpc_puller.go:299' },
  { step: 2, label: 'eth_getLogs', file: 'https_jsonrpc_puller.go:499' },
  { step: 3, label: 'Event Decode', file: 'chained_decoder.go:44' },
  { step: 4, label: 'EventBus', file: 'eventbus.go:87' },
  { step: 5, label: 'Process & Store', file: 'event_processor.go' },
  { step: 6, label: 'API Query', file: 'event_query_handler.go' },
]

export default function DataFlow({ steps = defaultSteps }: { steps?: DataFlowStep[] }) {
  return (
    <div className="rounded-2xl border border-white/10 bg-white/5 p-5">
      <p className="mb-4 text-xs uppercase tracking-[0.2em] text-mist">数据流路径</p>
      <div className="flex flex-wrap gap-1.5">
        {steps.map((s, i) => (
          <div key={s.step} className="flex items-center gap-1.5">
            <span className="inline-flex h-7 w-7 items-center justify-center rounded-full bg-white/10 text-xs font-medium text-white/80">
              {s.step}
            </span>
            <span className="text-xs text-sand/70">{s.label}</span>
            {i < steps.length - 1 && <span className="text-white/20 mx-1">→</span>}
          </div>
        ))}
      </div>
      <p className="mt-3 text-[11px] text-mist">
        在 delve 中: 按路径顺序设置断点 → continue → 观察每步数据变化
      </p>
    </div>
  )
}
