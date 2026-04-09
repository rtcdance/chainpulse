import { useState } from 'react'
import { Check, Copy, Loader2, Play } from 'lucide-react'
import { executeGraphQL } from '../lib/chainpulse'

const presets = [
  {
    label: 'Schema Introspection',
    value: `query {
  __schema {
    queryType { name }
    types { name }
  }
}`,
  },
  {
    label: 'Events Connection',
    value: `query {
  events(first: 5) {
    total
    edges {
      cursor
      node {
        id
        eventName
        chainId
        blockNumber
      }
    }
  }
}`,
  },
  {
    label: 'Block Surface',
    value: `query {
  block(number: 1) {
    number
    hash
    parentHash
    timestamp
  }
}`,
  },
]

export default function GraphQL() {
  const [query, setQuery] = useState<string>(presets[1].value)
  const [result, setResult] = useState<string>('No query has been executed yet.')
  const [path, setPath] = useState('/graphql')
  const [loading, setLoading] = useState(false)
  const [copied, setCopied] = useState(false)

  async function runQuery(): Promise<void> {
    setLoading(true)
    try {
      const payload = await executeGraphQL(query)
      setResult(JSON.stringify(payload.body, null, 2))
      setPath(payload.evidence.path)
    } catch (error) {
      setResult(JSON.stringify({ error: error instanceof Error ? error.message : 'request failed' }, null, 2))
    } finally {
      setLoading(false)
    }
  }

  async function copyResult(): Promise<void> {
    await navigator.clipboard.writeText(result)
    setCopied(true)
    window.setTimeout(() => setCopied(false), 1600)
  }

  return (
    <div className="space-y-6">
      <section className="rounded-[28px] border border-white/10 bg-white/5 p-6">
        <p className="text-xs uppercase tracking-[0.25em] text-mist">GraphQL Acceptance</p>
        <h2 className="mt-3 text-2xl font-semibold text-white">Explorer, schema, and query evidence</h2>
        <p className="mt-3 max-w-3xl text-sm leading-6 text-sand/75">
          The console ships with introspection, event connection, and block query presets so the team can prove the GraphQL surface quickly during a live review.
        </p>

        <div className="mt-5 flex flex-wrap gap-3">
          {presets.map((preset) => (
            <button
              key={preset.label}
              onClick={() => setQuery(preset.value)}
              className="rounded-full border border-white/15 bg-white/10 px-4 py-2 text-sm text-white transition hover:bg-white/15"
            >
              {preset.label}
            </button>
          ))}
        </div>
      </section>

      <section className="grid gap-6 xl:grid-cols-2">
        <article className="rounded-[28px] border border-white/10 bg-white/5 p-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-xs uppercase tracking-[0.25em] text-mist">Query Editor</p>
              <p className="mt-2 text-sm text-sand/75">Request target: <span className="font-mono text-white">{path}</span></p>
            </div>
            <button
              onClick={() => void runQuery()}
              disabled={loading}
              className="inline-flex items-center gap-2 rounded-full bg-glow px-4 py-2 text-sm font-medium text-ink transition hover:brightness-110 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Play className="h-4 w-4" />}
              Execute
            </button>
          </div>

          <textarea
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            className="mt-5 h-[28rem] w-full rounded-[24px] border border-white/10 bg-black/25 p-5 font-mono text-sm leading-6 text-sand outline-none placeholder:text-sand/35"
            spellCheck={false}
          />
        </article>

        <article className="rounded-[28px] border border-white/10 bg-white/5 p-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-xs uppercase tracking-[0.25em] text-mist">Response Evidence</p>
              <p className="mt-2 text-sm text-sand/75">The raw JSON response is preserved for screenshots, copy-paste, and review evidence.</p>
            </div>
            <button
              onClick={() => void copyResult()}
              className="inline-flex items-center gap-2 rounded-full border border-white/15 bg-white/10 px-4 py-2 text-sm text-white transition hover:bg-white/15"
            >
              {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
              {copied ? 'Copied' : 'Copy result'}
            </button>
          </div>

          <pre className="mt-5 h-[28rem] overflow-auto rounded-[24px] border border-white/10 bg-black/25 p-5 text-xs leading-6 text-sand/90">
            {result}
          </pre>
        </article>
      </section>
    </div>
  )
}
