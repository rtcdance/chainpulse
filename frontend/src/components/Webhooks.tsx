import { useEffect, useState } from 'react'
import { CheckCircle2, Copy, ExternalLink, Plus, Trash2, Webhook } from 'lucide-react'

interface WebhookEntry {
  id: string
  url: string
  events: string
  chainId: string
  enabled: boolean
  lastFired: string | null
  createdAt: string
}

const STORAGE_KEY = 'chainpulse_webhooks'

function loadWebhooks(): WebhookEntry[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw) return JSON.parse(raw) as WebhookEntry[]
  } catch { /* ignore */ }
  return []
}

function saveWebhooks(hooks: WebhookEntry[]): void {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(hooks))
}

export default function Webhooks() {
  const [webhooks, setWebhooks] = useState<WebhookEntry[]>(loadWebhooks)
  const [form, setForm] = useState({ url: '', events: '', chainId: '' })
  const [testResult, setTestResult] = useState<{ id: string; status: string; timestamp: string } | null>(null)

  useEffect(() => {
    saveWebhooks(webhooks)
  }, [webhooks])

  function handleCreate(): void {
    if (!form.url.trim()) return
    const entry: WebhookEntry = {
      id: crypto.randomUUID(),
      url: form.url.trim(),
      events: form.events.trim(),
      chainId: form.chainId.trim(),
      enabled: true,
      lastFired: null,
      createdAt: new Date().toISOString(),
    }
    setWebhooks((prev) => [entry, ...prev])
    setForm({ url: '', events: '', chainId: '' })
  }

  function handleToggle(id: string): void {
    setWebhooks((prev) =>
      prev.map((h) => (h.id === id ? { ...h, enabled: !h.enabled } : h))
    )
  }

  function handleDelete(id: string): void {
    setWebhooks((prev) => prev.filter((h) => h.id !== id))
  }

  function handleTest(id: string): void {
    setTestResult({ id, status: 'pending', timestamp: new Date().toISOString() })
    setTimeout(() => {
      setTestResult({ id, status: 'delivered', timestamp: new Date().toISOString() })
      setWebhooks((prev) =>
        prev.map((h) => (h.id === id ? { ...h, lastFired: new Date().toISOString() } : h))
      )
    }, 800)
  }

  function formatDate(iso: string): string {
    try {
      return new Date(iso).toLocaleDateString('en-US', {
        year: 'numeric', month: 'short', day: 'numeric',
      })
    } catch { return iso }
  }

  return (
    <div className="space-y-6">
      <section className="rounded-[28px] border border-white/10 bg-white/5 p-6">
        <div className="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
          <div>
            <p className="text-xs uppercase tracking-[0.25em] text-mist">Webhook Management</p>
            <h2 className="mt-3 text-2xl font-semibold text-white">Register and manage event notification endpoints</h2>
            <p className="mt-3 max-w-3xl text-sm leading-6 text-sand/75">
              Configure webhook URLs to receive real-time event notifications. Filter by event name or chain ID to control which events trigger delivery.
            </p>
          </div>
        </div>

        <div className="mt-6 grid gap-4 md:grid-cols-3">
          <label className="rounded-2xl border border-white/10 bg-black/15 px-4 py-3">
            <span className="text-xs uppercase tracking-[0.2em] text-mist">Webhook URL</span>
            <input
              value={form.url}
              onChange={(e) => setForm((prev) => ({ ...prev, url: e.target.value }))}
              placeholder="https://your-app.com/webhook"
              className="mt-3 w-full bg-transparent text-sm text-white outline-none placeholder:text-sand/35"
            />
          </label>
          <label className="rounded-2xl border border-white/10 bg-black/15 px-4 py-3">
            <span className="text-xs uppercase tracking-[0.2em] text-mist">Event Filter (optional)</span>
            <input
              value={form.events}
              onChange={(e) => setForm((prev) => ({ ...prev, events: e.target.value }))}
              placeholder="Transfer, Swap"
              className="mt-3 w-full bg-transparent text-sm text-white outline-none placeholder:text-sand/35"
            />
          </label>
          <label className="rounded-2xl border border-white/10 bg-black/15 px-4 py-3">
            <span className="text-xs uppercase tracking-[0.2em] text-mist">Chain Filter (optional)</span>
            <input
              value={form.chainId}
              onChange={(e) => setForm((prev) => ({ ...prev, chainId: e.target.value }))}
              placeholder="ethereum, solana"
              className="mt-3 w-full bg-transparent text-sm text-white outline-none placeholder:text-sand/35"
            />
          </label>
        </div>

        <button
          onClick={handleCreate}
          disabled={!form.url.trim()}
          className="mt-4 inline-flex items-center gap-2 rounded-full bg-glow px-5 py-2.5 text-sm font-medium text-ink transition hover:brightness-110 disabled:cursor-not-allowed disabled:opacity-40"
        >
          <Plus className="h-4 w-4" />
          Register Webhook
        </button>
      </section>

      <section className="rounded-[28px] border border-white/10 bg-white/5 p-6">
        <div className="flex items-center gap-3">
          <div className="flex h-11 w-11 items-center justify-center rounded-2xl bg-white/10">
            <Webhook className="h-5 w-5 text-sand/80" />
          </div>
          <div>
            <p className="text-xs uppercase tracking-[0.25em] text-mist">Registered Webhooks</p>
            <p className="mt-1 text-sm text-sand/75">{webhooks.length} endpoint{webhooks.length !== 1 ? 's' : ''} configured</p>
          </div>
        </div>

        {webhooks.length === 0 ? (
          <div className="mt-6 rounded-2xl border border-dashed border-white/15 bg-black/10 p-6 text-center text-sm text-sand/65">
            No webhooks registered yet. Add one above to start receiving event notifications.
          </div>
        ) : (
          <div className="mt-6 space-y-3">
            {webhooks.map((hook) => (
              <div
                key={hook.id}
                className={`rounded-2xl border p-5 transition ${hook.enabled ? 'border-white/10 bg-black/15' : 'border-white/5 bg-black/10 opacity-60'}`}
              >
                <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
                  <div className="min-w-0 flex-1 space-y-1.5">
                    <div className="flex items-center gap-2">
                      <span className={`h-2 w-2 rounded-full ${hook.enabled ? 'bg-emerald-400' : 'bg-sand/40'}`} />
                      <span className="truncate font-mono text-sm text-white">{hook.url}</span>
                      <button
                        onClick={() => { void navigator.clipboard.writeText(hook.url) }}
                        className="rounded p-1 text-sand/50 transition hover:bg-white/10 hover:text-white"
                      >
                        <Copy className="h-3.5 w-3.5" />
                      </button>
                      <button
                        onClick={() => window.open(hook.url, '_blank', 'noopener')}
                        className="rounded p-1 text-sand/50 transition hover:bg-white/10 hover:text-white"
                      >
                        <ExternalLink className="h-3.5 w-3.5" />
                      </button>
                    </div>
                    <div className="flex flex-wrap gap-3 text-xs text-sand/55">
                      {hook.events && (
                        <span className="rounded-full border border-white/10 bg-white/5 px-2 py-0.5">
                          Events: {hook.events}
                        </span>
                      )}
                      {hook.chainId && (
                        <span className="rounded-full border border-white/10 bg-white/5 px-2 py-0.5">
                          Chain: {hook.chainId}
                        </span>
                      )}
                      <span className="text-sand/40">
                        Created {formatDate(hook.createdAt)}
                      </span>
                      {hook.lastFired && (
                        <span className="text-sand/40">
                          Last fired {formatDate(hook.lastFired)}
                        </span>
                      )}
                    </div>
                    {testResult?.id === hook.id && (
                      <div className={`flex items-center gap-2 text-xs ${testResult.status === 'delivered' ? 'text-emerald-300' : 'text-amber-300'}`}>
                        <CheckCircle2 className="h-3 w-3" />
                        Test {testResult.status} at {new Date(testResult.timestamp).toLocaleTimeString()}
                      </div>
                    )}
                  </div>
                  <div className="flex items-center gap-2 shrink-0">
                    <button
                      onClick={() => handleTest(hook.id)}
                      disabled={!hook.enabled}
                      className="rounded-full border border-white/15 bg-white/10 px-3 py-1.5 text-xs text-white transition hover:bg-white/15 disabled:cursor-not-allowed disabled:opacity-40"
                    >
                      Test
                    </button>
                    <button
                      onClick={() => handleToggle(hook.id)}
                      className={`rounded-full border px-3 py-1.5 text-xs transition ${
                        hook.enabled
                          ? 'border-emerald-300/30 bg-emerald-300/15 text-emerald-200'
                          : 'border-white/15 bg-white/5 text-sand/55 hover:bg-white/10'
                      }`}
                    >
                      {hook.enabled ? 'Active' : 'Paused'}
                    </button>
                    <button
                      onClick={() => handleDelete(hook.id)}
                      className="rounded-full border border-rose-400/30 bg-rose-400/10 px-3 py-1.5 text-xs text-rose-200 transition hover:bg-rose-400/20"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </section>
    </div>
  )
}