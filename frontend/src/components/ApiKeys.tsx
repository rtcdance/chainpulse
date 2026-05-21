import { useEffect, useState } from 'react'
import { Copy, Key, Plus, Trash2, X } from 'lucide-react'

interface ApiKeyEntry {
  id: string
  name: string
  clientId: string
  keyPrefix: string
  permissions: string
  enabled: boolean
  expiresAt: string
  lastUsed: string | null
  createdAt: string
}

interface GeneratedKey {
  entry: ApiKeyEntry
  fullKey: string
}

const STORAGE_KEY = 'chainpulse_api_keys'

// WARNING: API keys are stored in localStorage for demo purposes.
// This is insecure – any XSS attack can steal all keys.
// Production use requires backend-managed key storage with hashed secrets
// (only the prefix shown here, full key shown once during generation).
function loadKeys(): ApiKeyEntry[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw) {
      return JSON.parse(raw) as ApiKeyEntry[]
    }
  } catch {
    // ignore
  }
  return []
}

function saveKeys(keys: ApiKeyEntry[]): void {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(keys))
}

function generateKeyPrefix(): string {
  const chars = 'abcdefghijklmnopqrstuvwxyz0123456789'
  const random = crypto.getRandomValues(new Uint8Array(12))
  let result = 'cp_'
  for (let i = 0; i < 12; i++) {
    result += chars.charAt(random[i] % chars.length)
  }
  return result
}

function generateFullKey(): string {
  const chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789'
  const random = crypto.getRandomValues(new Uint8Array(40))
  let result = 'cp_sk_'
  for (let i = 0; i < 40; i++) {
    result += chars.charAt(random[i] % chars.length)
  }
  return result
}

export default function ApiKeys() {
  const [keys, setKeys] = useState<ApiKeyEntry[]>(loadKeys)
  const [generatedKey, setGeneratedKey] = useState<GeneratedKey | null>(null)
  const [copiedId, setCopiedId] = useState<string | null>(null)

  const [form, setForm] = useState({
    name: '',
    clientId: '',
    permissions: 'read',
    expiryDays: '90',
  })

  useEffect(() => {
    saveKeys(keys)
  }, [keys])

  function handleGenerate(): void {
    if (!form.name.trim() || !form.clientId.trim()) return

    const now = new Date()
    const expiresAt = new Date(now.getTime() + parseInt(form.expiryDays || '90', 10) * 86400000)

    const entry: ApiKeyEntry = {
      id: crypto.randomUUID(),
      name: form.name.trim(),
      clientId: form.clientId.trim(),
      keyPrefix: generateKeyPrefix(),
      permissions: form.permissions,
      enabled: true,
      expiresAt: expiresAt.toISOString(),
      lastUsed: null,
      createdAt: now.toISOString(),
    }

    const fullKey = generateFullKey()

    setKeys((prev) => [entry, ...prev])
    setGeneratedKey({ entry, fullKey })
    setForm({ name: '', clientId: '', permissions: 'read', expiryDays: '90' })
  }

  function handleRevoke(id: string): void {
    setKeys((prev) => prev.filter((k) => k.id !== id))
  }

  function handleToggle(id: string): void {
    setKeys((prev) =>
      prev.map((k) => (k.id === id ? { ...k, enabled: !k.enabled } : k))
    )
  }

  function handleCopyFullKey(fullKey: string): void {
    void navigator.clipboard.writeText(fullKey)
    setCopiedId(fullKey)
    setTimeout(() => setCopiedId(null), 2000)
  }

  function formatDate(iso: string): string {
    try {
      return new Date(iso).toLocaleDateString('en-US', {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
      })
    } catch {
      return iso
    }
  }

  function isExpired(iso: string): boolean {
    try {
      return new Date(iso).getTime() < Date.now()
    } catch {
      return false
    }
  }

  return (
    <div className="space-y-6">
      {generatedKey && (
        <section className="rounded-[28px] border border-glow/40 bg-glow/10 p-6">
          <div className="flex items-start justify-between">
            <div>
              <div className="flex items-center gap-3">
                <Key className="h-5 w-5 text-glow" />
                <h2 className="text-lg font-semibold text-white">API Key Generated</h2>
              </div>
              <p className="mt-2 text-sm text-amber-100/80">
                Store this key safely — it won't be shown again.
              </p>
            </div>
            <button
              onClick={() => setGeneratedKey(null)}
              className="rounded-full p-2 text-white/50 transition hover:bg-white/10 hover:text-white"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
          <div className="mt-4 rounded-2xl border border-glow/30 bg-black/30 p-4">
            <div className="flex items-center gap-3">
              <code className="flex-1 break-all font-mono text-sm text-white">{generatedKey.fullKey}</code>
              <button
                onClick={() => handleCopyFullKey(generatedKey.fullKey)}
                className="shrink-0 rounded-full border border-white/15 bg-white/10 px-3 py-2 text-sm text-white transition hover:bg-white/15"
              >
                {copiedId === generatedKey.fullKey ? (
                  <span className="text-glow">Copied</span>
                ) : (
                  <Copy className="h-4 w-4" />
                )}
              </button>
            </div>
          </div>
        </section>
      )}

      <section data-tour="api-keys" className="rounded-[28px] border border-white/10 bg-white/5 p-6">
        <div className="flex items-center gap-3">
          <div className="flex h-11 w-11 items-center justify-center rounded-2xl bg-glow/15">
            <Plus className="h-5 w-5 text-glow" />
          </div>
          <div>
            <p className="text-xs uppercase tracking-[0.25em] text-mist">Create API Key</p>
            <p className="mt-1 text-sm text-sand/75">Generate new credentials for API access</p>
          </div>
        </div>

        <div className="mt-6 grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <label className="rounded-2xl border border-white/10 bg-black/15 px-4 py-3">
            <span className="text-xs uppercase tracking-[0.2em] text-mist">Name</span>
            <input
              value={form.name}
              onChange={(e) => setForm((prev) => ({ ...prev, name: e.target.value }))}
              placeholder="Production Key"
              className="mt-3 w-full bg-transparent text-sm text-white outline-none placeholder:text-sand/35"
            />
          </label>
          <label className="rounded-2xl border border-white/10 bg-black/15 px-4 py-3">
            <span className="text-xs uppercase tracking-[0.2em] text-mist">Client ID</span>
            <input
              value={form.clientId}
              onChange={(e) => setForm((prev) => ({ ...prev, clientId: e.target.value }))}
              placeholder="my-app"
              className="mt-3 w-full bg-transparent text-sm text-white outline-none placeholder:text-sand/35"
            />
          </label>
          <label className="rounded-2xl border border-white/10 bg-black/15 px-4 py-3">
            <span className="text-xs uppercase tracking-[0.2em] text-mist">Permissions</span>
            <select
              value={form.permissions}
              onChange={(e) => setForm((prev) => ({ ...prev, permissions: e.target.value }))}
              className="mt-3 w-full bg-transparent text-sm text-white outline-none"
            >
              <option value="read" className="bg-black">Read</option>
              <option value="read_write" className="bg-black">Read & Write</option>
              <option value="admin" className="bg-black">Admin</option>
            </select>
          </label>
          <label className="rounded-2xl border border-white/10 bg-black/15 px-4 py-3">
            <span className="text-xs uppercase tracking-[0.2em] text-mist">Expiry (days)</span>
            <input
              value={form.expiryDays}
              onChange={(e) => setForm((prev) => ({ ...prev, expiryDays: e.target.value }))}
              placeholder="90"
              className="mt-3 w-full bg-transparent text-sm text-white outline-none placeholder:text-sand/35"
            />
          </label>
        </div>

        <button
          onClick={handleGenerate}
          disabled={!form.name.trim() || !form.clientId.trim()}
          className="mt-4 inline-flex items-center gap-2 rounded-full bg-glow px-5 py-2.5 text-sm font-medium text-ink transition hover:brightness-110 disabled:cursor-not-allowed disabled:opacity-40"
        >
          <Key className="h-4 w-4" />
          Generate
        </button>
      </section>

      <section className="rounded-[28px] border border-white/10 bg-white/5 p-6">
        <div className="flex items-center gap-3">
          <div className="flex h-11 w-11 items-center justify-center rounded-2xl bg-white/10">
            <Key className="h-5 w-5 text-sand/80" />
          </div>
          <div>
            <p className="text-xs uppercase tracking-[0.25em] text-mist">API Keys List</p>
            <p className="mt-1 text-sm text-sand/75">{keys.length} key{keys.length !== 1 ? 's' : ''} registered</p>
          </div>
        </div>

        {keys.length === 0 ? (
          <div className="mt-6 rounded-2xl border border-dashed border-white/15 bg-black/10 p-6 text-center text-sm text-sand/65">
            No API keys yet. Create one above to get started.
          </div>
        ) : (
          <div className="mt-6 overflow-hidden rounded-[22px] border border-white/10">
            <div className="hidden grid-cols-[1.2fr,1fr,0.8fr,0.8fr,0.6fr,0.8fr,0.6fr,0.5fr] gap-3 bg-black/25 px-4 py-3 text-[10px] uppercase tracking-[0.2em] text-mist lg:grid">
              <span>Name</span>
              <span>Client ID</span>
              <span>Key Prefix</span>
              <span>Permissions</span>
              <span>Status</span>
              <span>Expires</span>
              <span>Last Used</span>
              <span>Actions</span>
            </div>
            <div className="divide-y divide-white/10">
              {keys.map((key) => {
                const expired = isExpired(key.expiresAt)
                return (
                  <div
                    key={key.id}
                    className={`grid gap-2 px-4 py-3 text-sm lg:grid-cols-[1.2fr,1fr,0.8fr,0.8fr,0.6fr,0.8fr,0.6fr,0.5fr] lg:gap-3 ${expired ? 'bg-rose-400/5' : ''}`}
                  >
                    <div>
                      <span className="font-medium text-white">{key.name}</span>
                      <div className="mt-0.5 text-[10px] text-sand/55 lg:hidden">
                        Created {formatDate(key.createdAt)}
                      </div>
                    </div>
                    <div className="font-mono text-xs text-sand/75">{key.clientId}</div>
                    <div className="font-mono text-xs text-sand/75">{key.keyPrefix}</div>
                    <div>
                      <span className={`rounded-full border px-2 py-0.5 text-[10px] uppercase tracking-wider ${
                        key.permissions === 'admin'
                          ? 'border-rose-300/30 bg-rose-300/15 text-rose-200'
                          : key.permissions === 'read_write'
                            ? 'border-amber-300/30 bg-amber-300/15 text-amber-200'
                            : 'border-emerald-300/30 bg-emerald-300/15 text-emerald-200'
                      }`}>
                        {key.permissions.replace('_', ' ')}
                      </span>
                    </div>
                    <div>
                      <button
                        onClick={() => handleToggle(key.id)}
                        className={`rounded-full border px-2 py-0.5 text-[10px] uppercase tracking-wider transition ${
                          key.enabled
                            ? 'border-emerald-300/30 bg-emerald-300/15 text-emerald-200'
                            : 'border-white/15 bg-white/5 text-sand/55'
                        }`}
                      >
                        {key.enabled ? 'Enabled' : 'Disabled'}
                      </button>
                    </div>
                    <div>
                      <span className={`text-xs ${expired ? 'text-rose-300' : 'text-sand/75'}`}>
                        {formatDate(key.expiresAt)}
                        {expired && <span className="ml-1 text-[10px] uppercase text-rose-400">Expired</span>}
                      </span>
                    </div>
                    <div className="text-xs text-sand/55">
                      {key.lastUsed ? formatDate(key.lastUsed) : '—'}
                    </div>
                    <div>
                      <button
                        onClick={() => handleRevoke(key.id)}
                        className="inline-flex items-center gap-1 rounded-full border border-rose-400/30 bg-rose-400/10 px-2 py-0.5 text-[10px] uppercase tracking-wider text-rose-200 transition hover:bg-rose-400/20"
                      >
                        <Trash2 className="h-3 w-3" />
                        Revoke
                      </button>
                    </div>
                  </div>
                )
              })}
            </div>
          </div>
        )}
      </section>
    </div>
  )
}