import { useEffect, useState } from 'react'
import { Loader2, Pause, Play, RefreshCw, ServerCog } from 'lucide-react'
import { fetchCurrentSliceReport, fetchRuntimeSummary, postRuntimeControl, type RuntimePayload, type ServiceAcceptanceReport } from '../lib/chainpulse'
import { useToast } from '../lib/toast'

interface ActionDef {
  serviceId: 'puller' | 'event-processor'
  action: 'pause' | 'resume' | 'pause-intake' | 'resume-intake'
  label: string
  icon: 'pause' | 'play'
  variant: 'warning' | 'success'
}

const controlActions: ActionDef[] = [
  { serviceId: 'puller', action: 'pause', label: 'Pause Puller', icon: 'pause', variant: 'warning' },
  { serviceId: 'puller', action: 'resume', label: 'Resume Puller', icon: 'play', variant: 'success' },
  { serviceId: 'event-processor', action: 'pause-intake', label: 'Pause Intake', icon: 'pause', variant: 'warning' },
  { serviceId: 'event-processor', action: 'resume-intake', label: 'Resume Intake', icon: 'play', variant: 'success' },
]

export default function Runtime() {
  const [gatewaySummary, setGatewaySummary] = useState<RuntimePayload | null>(null)
  const [reports, setReports] = useState<ServiceAcceptanceReport[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [confirmAction, setConfirmAction] = useState<ActionDef | null>(null)
  const [actionLoading, setActionLoading] = useState(false)
  const { addToast } = useToast()

  async function loadRuntime(): Promise<void> {
    setLoading(true)
    try {
      const [summary, nextReports] = await Promise.all([
        fetchRuntimeSummary(),
        fetchCurrentSliceReport(),
      ])
      setGatewaySummary(summary)
      setReports(nextReports)
      setError(null)
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : 'Failed to load runtime state')
    } finally {
      setLoading(false)
    }
  }

  async function executeAction(action: ActionDef): Promise<void> {
    setActionLoading(true)
    try {
      const result = await postRuntimeControl(action.serviceId, action.action)
      if (result.success) {
        addToast('success', `${action.label} succeeded — ${result.message}`)
        await loadRuntime()
      } else {
        addToast('error', `${action.label} failed — ${result.message}`)
      }
    } catch (err) {
      addToast('error', err instanceof Error ? err.message : 'action failed')
    } finally {
      setActionLoading(false)
      setConfirmAction(null)
    }
  }

  useEffect(() => {
    void loadRuntime()
  }, [])

  return (
    <div className="space-y-6">
      <section className="rounded-[28px] border border-white/10 bg-white/5 p-6">
        <div className="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
          <div>
            <p className="text-xs uppercase tracking-[0.25em] text-mist">Runtime Acceptance</p>
            <h2 className="mt-3 text-2xl font-semibold text-white">Cross-service runtime, rollout, and control evidence</h2>
            <p className="mt-3 max-w-3xl text-sm leading-6 text-sand/75">
              This page surfaces the runtime identity, rollout state, and operator control actions. Use the control panel below to pause/resume services and observe the state change in real time.
            </p>
          </div>
          <button
            onClick={() => void loadRuntime()}
            className="inline-flex items-center gap-2 rounded-full border border-white/15 bg-white/10 px-4 py-2 text-sm text-white transition hover:bg-white/15"
          >
            <RefreshCw className="h-4 w-4" />
            Refresh runtime
          </button>
        </div>
      </section>

      {loading ? (
        <div className="flex h-72 items-center justify-center rounded-[28px] border border-white/10 bg-white/5">
          <Loader2 className="h-9 w-9 animate-spin text-glow" />
        </div>
      ) : error ? (
        <div className="rounded-[28px] border border-rose-400/30 bg-rose-400/10 p-6 text-sm text-rose-100">{error}</div>
      ) : (
        <>
          <section className="grid gap-6 xl:grid-cols-[0.8fr,1.2fr]">
            <article className="rounded-[28px] border border-white/10 bg-white/5 p-6">
              <div className="flex items-center gap-3">
                <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-glow/15">
                  <ServerCog className="h-5 w-5 text-glow" />
                </div>
                <div>
                  <p className="text-xs uppercase tracking-[0.25em] text-mist">Primary Runtime Identity</p>
                  <h3 className="mt-1 text-lg font-medium text-white">{gatewaySummary?.service || 'unknown'}</h3>
                </div>
              </div>

              <div className="mt-6 grid gap-3">
                <div className="rounded-2xl border border-white/10 bg-black/15 p-4">
                  <div className="text-xs uppercase tracking-[0.2em] text-mist">Runtime Mode</div>
                  <div className="mt-2 text-white">{gatewaySummary?.runtimeMode || 'unknown'}</div>
                </div>
                <div className="rounded-2xl border border-white/10 bg-black/15 p-4">
                  <div className="text-xs uppercase tracking-[0.2em] text-mist">Deployment Mode</div>
                  <div className="mt-2 text-white">{gatewaySummary?.deploymentMode || 'unknown'}</div>
                </div>
                <div className="rounded-2xl border border-white/10 bg-black/15 p-4">
                  <div className="text-xs uppercase tracking-[0.2em] text-mist">Evidence Path</div>
                  <div className="mt-2 font-mono text-sm text-white">{gatewaySummary?.evidence.path || '/runtime/summary'}</div>
                </div>
              </div>
            </article>

            <article className="rounded-[28px] border border-white/10 bg-white/5 p-6">
              <p className="text-xs uppercase tracking-[0.25em] text-mist">Gateway Summary JSON</p>
              <pre className="mt-5 max-h-[28rem] overflow-auto rounded-[24px] border border-white/10 bg-black/25 p-4 text-xs leading-6 text-sand/90">
                {JSON.stringify(gatewaySummary?.summary || {}, null, 2)}
              </pre>
            </article>
          </section>

          <section className="rounded-[28px] border border-white/10 bg-white/5 p-6">
            <p className="text-xs uppercase tracking-[0.25em] text-mist">Control Panel</p>
            <p className="mt-2 text-sm text-sand/75">Execute runtime control actions. Each action requires confirmation before it is sent to the backend.</p>

            <div className="mt-5 flex flex-wrap gap-3">
              {controlActions.map((action) => (
                <button
                  key={`${action.serviceId}-${action.action}`}
                  onClick={() => setConfirmAction(action)}
                  disabled={actionLoading}
                  className={`inline-flex items-center gap-2 rounded-full border px-4 py-2 text-sm font-medium transition disabled:cursor-not-allowed disabled:opacity-50 ${
                    action.variant === 'warning'
                      ? 'border-amber-300/30 bg-amber-300/10 text-amber-100 hover:bg-amber-300/20'
                      : 'border-emerald-300/30 bg-emerald-300/10 text-emerald-100 hover:bg-emerald-300/20'
                  }`}
                >
                  {action.icon === 'pause' ? <Pause className="h-4 w-4" /> : <Play className="h-4 w-4" />}
                  {action.label}
                </button>
              ))}
            </div>

            {confirmAction && (
              <div className="mt-4 rounded-2xl border border-amber-300/30 bg-amber-300/10 p-5">
                <p className="text-sm text-amber-50">
                  Confirm: <span className="font-semibold text-white">{confirmAction.label}</span> on <span className="font-semibold text-white">{confirmAction.serviceId}</span>?
                </p>
                <div className="mt-3 flex gap-3">
                  <button
                    onClick={() => void executeAction(confirmAction)}
                    disabled={actionLoading}
                    className="inline-flex items-center gap-2 rounded-full bg-glow px-4 py-2 text-sm font-medium text-ink transition hover:brightness-110 disabled:cursor-not-allowed disabled:opacity-60"
                  >
                    {actionLoading ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
                    Confirm
                  </button>
                  <button
                    onClick={() => setConfirmAction(null)}
                    className="rounded-full border border-white/15 bg-white/10 px-4 py-2 text-sm text-white transition hover:bg-white/15"
                  >
                    Cancel
                  </button>
                </div>
              </div>
            )}
          </section>

          <section className="rounded-[28px] border border-white/10 bg-white/5 p-6">
            <p className="text-xs uppercase tracking-[0.25em] text-mist">Rollout & Control Surfaces</p>
            <div className="mt-5 grid gap-4 xl:grid-cols-2">
              {reports.map((report) => {
                const rollout = report.probes.find((probe) => probe.path === '/health/rollout')
                const control = report.probes.find((probe) => probe.path === '/runtime/control')
                const summary = report.probes.find((probe) => probe.path === '/runtime/summary')

                return (
                  <article key={report.service.id} className="min-w-0 rounded-[24px] border border-white/10 bg-black/15 p-5">
                    <h3 className="text-lg font-medium text-white">{report.service.name}</h3>
                    <p className="mt-2 text-sm leading-6 text-sand/70">{report.service.role}</p>

                    <div className="mt-5 grid gap-3">
                      {[summary, rollout, control].filter(Boolean).map((probe) => (
                        <div
                          key={probe?.path}
                          className={`min-w-0 rounded-2xl border p-4 ${
                            probe?.ok
                              ? 'border-emerald-300/25 bg-emerald-300/10 text-emerald-100'
                              : 'border-rose-400/25 bg-rose-400/10 text-rose-100'
                          }`}
                        >
                          <div className="flex flex-wrap items-start justify-between gap-3">
                            <span className="min-w-0 flex-1 break-words font-mono text-sm text-white">{probe?.path}</span>
                            <span className="shrink-0 text-xs uppercase tracking-[0.2em]">{probe?.status ?? 'ERR'}</span>
                          </div>
                          <pre className="mt-2 max-h-28 overflow-auto whitespace-pre-wrap break-words rounded-xl bg-black/20 p-3 font-mono text-[11px] leading-5 opacity-90">
                            {probe?.summary || 'No preview'}
                          </pre>
                        </div>
                      ))}
                    </div>
                  </article>
                )
              })}
            </div>
          </section>
        </>
      )}
    </div>
  )
}
