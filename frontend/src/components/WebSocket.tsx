import { useEffect, useRef, useState } from 'react'
import { Loader2, Plug, Radio, Send, Trash2 } from 'lucide-react'
import { buildWebSocketUrl, buildFilteredSubscribeUrl } from '../lib/chainpulse'

interface MessageLog {
  id: number
  direction: 'system' | 'in' | 'out' | 'error'
  text: string
  timestamp: string
}

interface WSFilters {
  chainId: string
  contract: string
  eventName: string
}

const endpointOptions = ['/ws', '/events/subscribe']

export default function WebSocket() {
  const [selectedPath, setSelectedPath] = useState(endpointOptions[0])
  const [status, setStatus] = useState<'idle' | 'connecting' | 'connected'>('idle')
  const [messages, setMessages] = useState<MessageLog[]>([])
  const [draft, setDraft] = useState('{"type":"subscribe","topic":"events"}')
  const [wsFilters, setWsFilters] = useState<WSFilters>({ chainId: '', contract: '', eventName: '' })
  const [activePath, setActivePath] = useState('/ws')
  const socketRef = useRef<WebSocket | null>(null)
  const sequenceRef = useRef(0)

  function pushMessage(direction: MessageLog['direction'], text: string): void {
    sequenceRef.current += 1
    setMessages((current) => [
      {
        id: sequenceRef.current,
        direction,
        text,
        timestamp: new Date().toLocaleTimeString(),
      },
      ...current,
    ].slice(0, 60))
  }

  function disconnect(): void {
    socketRef.current?.close()
    socketRef.current = null
    setStatus('idle')
  }

  function getEffectivePath(): string {
    if (selectedPath === '/events/subscribe') {
      const filtered = buildFilteredSubscribeUrl(wsFilters)
      if (filtered !== '/events/subscribe') return filtered
    }
    return selectedPath
  }

  function connect(): void {
    disconnect()
    setStatus('connecting')
    const path = getEffectivePath()
    setActivePath(path)
    const socket = new window.WebSocket(buildWebSocketUrl(path))
    socketRef.current = socket

    socket.onopen = () => {
      setStatus('connected')
      pushMessage('system', `connected ${path}`)
    }

    socket.onerror = () => {
      pushMessage('error', `websocket error on ${path}`)
    }

    socket.onclose = () => {
      setStatus('idle')
      pushMessage('system', `closed ${path}`)
    }

    socket.onmessage = (event) => {
      pushMessage('in', typeof event.data === 'string' ? event.data : '[binary frame]')
    }

    socket.onerror = () => {
      pushMessage('error', `websocket error on ${selectedPath}`)
    }

    socket.onclose = () => {
      setStatus('idle')
      pushMessage('system', `closed ${selectedPath}`)
    }
  }

  function sendDraft(): void {
    if (!socketRef.current || status !== 'connected') {
      return
    }
    socketRef.current.send(draft)
    pushMessage('out', draft)
  }

  useEffect(() => () => disconnect(), [])

  return (
    <div className="space-y-6">
      <section className="rounded-[28px] border border-white/10 bg-white/5 p-6">
        <p className="text-xs uppercase tracking-[0.25em] text-mist">WebSocket Acceptance</p>
        <h2 className="mt-3 text-2xl font-semibold text-white">Connection, subscriptions, logs, and multi-path compatibility</h2>
        <p className="mt-3 max-w-3xl text-sm leading-6 text-sand/75">
          The default path is the documented `/ws` endpoint, but you can also switch to the gateway subscription path `/events/subscribe`. That keeps the demo useful across mock, monolithic, and gateway-driven runs.
        </p>

        <div className="mt-5 flex flex-wrap gap-3">
          {endpointOptions.map((path) => (
            <button
              key={path}
              onClick={() => setSelectedPath(path)}
              className={`rounded-full border px-4 py-2 text-sm transition ${
                selectedPath === path
                  ? 'border-glow/40 bg-glow/15 text-white'
                  : 'border-white/15 bg-white/10 text-white hover:bg-white/15'
              }`}
            >
              {path}
            </button>
          ))}
        </div>

        <div className="mt-5 flex flex-wrap items-center gap-3">
          <button
            onClick={connect}
            disabled={status === 'connecting'}
            className="inline-flex items-center gap-2 rounded-full bg-glow px-4 py-2 text-sm font-medium text-ink transition hover:brightness-110 disabled:cursor-not-allowed disabled:opacity-60"
          >
            {status === 'connecting' ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plug className="h-4 w-4" />}
            {status === 'connected' ? 'Reconnect' : 'Connect'}
          </button>
          <button
            onClick={disconnect}
            className="rounded-full border border-white/15 bg-white/10 px-4 py-2 text-sm text-white transition hover:bg-white/15"
          >
            Disconnect
          </button>
          <span className={`rounded-full border px-3 py-1 text-sm ${
            status === 'connected'
              ? 'border-emerald-300/30 bg-emerald-300/10 text-emerald-200'
              : status === 'connecting'
                ? 'border-amber-300/30 bg-amber-300/10 text-amber-100'
                : 'border-white/10 bg-black/15 text-sand/70'
          }`}>
            {status}
          </span>
          <span className="font-mono text-sm text-sand/75">{buildWebSocketUrl(activePath)}</span>
        </div>

        {selectedPath === '/events/subscribe' && (
          <div className="mt-5 grid gap-4 md:grid-cols-3">
            <label className="rounded-2xl border border-white/10 bg-black/15 px-4 py-3">
              <span className="text-xs uppercase tracking-[0.2em] text-mist">Chain ID</span>
              <input
                value={wsFilters.chainId}
                onChange={(event) => setWsFilters((current) => ({ ...current, chainId: event.target.value }))}
                placeholder="ethereum / 1"
                className="mt-3 w-full bg-transparent text-sm text-white outline-none placeholder:text-sand/35"
              />
            </label>
            <label className="rounded-2xl border border-white/10 bg-black/15 px-4 py-3">
              <span className="text-xs uppercase tracking-[0.2em] text-mist">Contract</span>
              <input
                value={wsFilters.contract}
                onChange={(event) => setWsFilters((current) => ({ ...current, contract: event.target.value }))}
                placeholder="0x..."
                className="mt-3 w-full bg-transparent text-sm text-white outline-none placeholder:text-sand/35"
              />
            </label>
            <label className="rounded-2xl border border-white/10 bg-black/15 px-4 py-3">
              <span className="text-xs uppercase tracking-[0.2em] text-mist">Event Name</span>
              <input
                value={wsFilters.eventName}
                onChange={(event) => setWsFilters((current) => ({ ...current, eventName: event.target.value }))}
                placeholder="Transfer / Swap"
                className="mt-3 w-full bg-transparent text-sm text-white outline-none placeholder:text-sand/35"
              />
            </label>
          </div>
        )}
      </section>

      <section className="grid gap-6 xl:grid-cols-[0.95fr,1.05fr]">
        <article className="rounded-[28px] border border-white/10 bg-white/5 p-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-xs uppercase tracking-[0.25em] text-mist">Message Composer</p>
              <p className="mt-2 text-sm text-sand/75">Send any JSON payload you need, or start from the built-in actions.</p>
            </div>
            <button
              onClick={sendDraft}
              disabled={status !== 'connected'}
              className="inline-flex items-center gap-2 rounded-full bg-glow px-4 py-2 text-sm font-medium text-ink transition hover:brightness-110 disabled:cursor-not-allowed disabled:opacity-60"
            >
              <Send className="h-4 w-4" />
              Send
            </button>
          </div>

          <textarea
            value={draft}
            onChange={(event) => setDraft(event.target.value)}
            className="mt-5 h-64 w-full rounded-[24px] border border-white/10 bg-black/25 p-5 font-mono text-sm leading-6 text-sand outline-none"
            spellCheck={false}
          />

          <div className="mt-4 flex flex-wrap gap-3">
            <button
              onClick={() => setDraft('{"type":"subscribe","topic":"events"}')}
              className="rounded-full border border-white/15 bg-white/10 px-4 py-2 text-sm text-white transition hover:bg-white/15"
            >
              Subscribe events
            </button>
            <button
              onClick={() => setDraft('{"type":"subscribe","eventName":"Transfer"}')}
              className="rounded-full border border-white/15 bg-white/10 px-4 py-2 text-sm text-white transition hover:bg-white/15"
            >
              Subscribe Transfer
            </button>
            <button
              onClick={() => setDraft('{"type":"unsubscribe"}')}
              className="rounded-full border border-white/15 bg-white/10 px-4 py-2 text-sm text-white transition hover:bg-white/15"
            >
              Unsubscribe
            </button>
          </div>
        </article>

        <article className="rounded-[28px] border border-white/10 bg-white/5 p-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-xs uppercase tracking-[0.25em] text-mist">Message Evidence</p>
              <p className="mt-2 text-sm text-sand/75">Keeps the latest 60 entries, including connect, close, and error messages.</p>
            </div>
            <button
              onClick={() => setMessages([])}
              className="inline-flex items-center gap-2 rounded-full border border-white/15 bg-white/10 px-4 py-2 text-sm text-white transition hover:bg-white/15"
            >
              <Trash2 className="h-4 w-4" />
              Clear
            </button>
          </div>

          <div className="mt-5 max-h-[34rem] overflow-auto rounded-[24px] border border-white/10 bg-black/25 p-4">
            <div className="space-y-3">
              {messages.length === 0 ? (
                <div className="rounded-2xl border border-dashed border-white/15 bg-black/10 p-6 text-sm text-sand/65">
                  No messages yet. Connect first, then send a subscription or wait for the service to push events.
                </div>
              ) : (
                messages.map((message) => (
                  <div
                    key={message.id}
                    className={`rounded-2xl border p-4 ${
                      message.direction === 'in'
                        ? 'border-emerald-300/20 bg-emerald-300/10'
                        : message.direction === 'out'
                          ? 'border-sky-300/20 bg-sky-300/10'
                          : message.direction === 'error'
                            ? 'border-rose-400/25 bg-rose-400/10'
                            : 'border-white/10 bg-black/15'
                    }`}
                  >
                    <div className="flex items-center justify-between text-xs uppercase tracking-[0.2em] text-mist">
                      <span className="inline-flex items-center gap-2">
                        <Radio className="h-3.5 w-3.5" />
                        {message.direction}
                      </span>
                      <span>{message.timestamp}</span>
                    </div>
                    <pre className="mt-3 whitespace-pre-wrap break-words font-mono text-xs leading-6 text-white/90">
                      {message.text}
                    </pre>
                  </div>
                ))
              )}
            </div>
          </div>
        </article>
      </section>
    </div>
  )
}
