import { useMemo, useState } from 'react'
import { BookOpen, Check, Copy, Loader2, Play, Search, X } from 'lucide-react'
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
    variables: '',
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
    variables: '',
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
    variables: '',
  },
  {
    label: 'Events by Chain',
    value: `query EventsByChain($chainId: String!) {
  events(first: 10, chainId: $chainId) {
    total
    edges {
      node {
        id
        eventName
        chainId
        blockNumber
      }
    }
  }
}`,
    variables: `{
  "chainId": "ethereum"
}`,
  },
]

const schemaDocs = [
  { name: 'Query.events', description: 'Paginated query for blockchain events. Accepts: first, after, chainId, eventName, contract, startTime, endTime.', args: 'first: Int, after: String, chainId: String, eventName: String, contract: String, startTime: Int, endTime: Int' },
  { name: 'Query.event', description: 'Fetch a single event by ID.', args: 'id: ID!' },
  { name: 'Query.block', description: 'Query a block by number or hash.', args: 'number: Int, hash: String' },
  { name: 'Event', description: 'Blockchain event type. Fields: id, eventName, chainId, blockNumber, transactionHash, contractAddress, status, timestamp, decodedData.', args: '-' },
  { name: 'Block', description: 'Block chain block type. Fields: number, hash, parentHash, timestamp, gasLimit, gasUsed, baseFee.', args: '-' },
  { name: 'EventConnection', description: 'Relay-style connection for events with pagination cursors.', args: '-' },
]

export default function GraphQL() {
  const [query, setQuery] = useState<string>(presets[1].value)
  const [variables, setVariables] = useState<string>(presets[1].variables)
  const [result, setResult] = useState<string>('No query has been executed yet.')
  const [path, setPath] = useState('/graphql')
  const [loading, setLoading] = useState(false)
  const [copied, setCopied] = useState(false)
  const [variablesError, setVariablesError] = useState<string | null>(null)
  const [showSchema, setShowSchema] = useState(false)
  const [schemaFilter, setSchemaFilter] = useState('')

  const parsedTypes = useMemo(() => {
    try {
      const parsed = JSON.parse(result) as Record<string, unknown>
      const schema = (parsed?.data as Record<string, unknown> | undefined)?.__schema as { types?: Array<{ name: string; kind: string; description?: string; fields?: Array<{ name: string }> }> } | undefined
      if (!schema?.types) return null
      return schema.types
        .filter((t) => !t.name.startsWith('__'))
        .sort((a, b) => a.name.localeCompare(b.name))
    } catch {
      return null
    }
  }, [result])

  const schemaTypes = parsedTypes?.filter((t) =>
    !schemaFilter || t.name.toLowerCase().includes(schemaFilter.toLowerCase())
  ) ?? []

  async function runQuery(): Promise<void> {
    setLoading(true)
    setVariablesError(null)
    try {
      let parsedVariables: Record<string, unknown> | undefined
      if (variables.trim()) {
        try {
          parsedVariables = JSON.parse(variables)
        } catch {
          setVariablesError('Invalid JSON in variables editor')
          setLoading(false)
          return
        }
      }
      const payload = await executeGraphQL(query, parsedVariables)
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
          The console ships with introspection, event connection, and block query presets. Use the variables editor to test parameterized queries.
        </p>

        <div className="mt-5 flex flex-wrap gap-3">
          {presets.map((preset) => (
            <button
              key={preset.label}
              onClick={() => { setQuery(preset.value); setVariables(preset.variables) }}
              className="rounded-full border border-white/15 bg-white/10 px-4 py-2 text-sm text-white transition hover:bg-white/15"
            >
              {preset.label}
            </button>
          ))}
          <button
            onClick={() => setShowSchema(!showSchema)}
            className={`rounded-full border px-4 py-2 text-sm transition ${
              showSchema
                ? 'border-glow/50 bg-glow/15 text-glow'
                : 'border-white/15 bg-white/10 text-white hover:bg-white/15'
            }`}
          >
            <span className="flex items-center gap-2">
              <BookOpen className="h-4 w-4" />
              Schema Docs
            </span>
          </button>
        </div>

        {showSchema && (
          <div className="mt-5 rounded-[24px] border border-white/10 bg-black/15 p-5">
            <div className="flex items-center justify-between mb-4">
              <p className="text-xs uppercase tracking-[0.25em] text-mist">
                {parsedTypes ? `Dynamic Schema (${parsedTypes.length} types)` : 'GraphQL Schema Reference'}
              </p>
              {parsedTypes && (
                <label className="flex items-center gap-2 rounded-full border border-white/10 bg-black/20 px-3 py-1.5">
                  <Search className="h-3 w-3 text-mist" />
                  <input
                    value={schemaFilter}
                    onChange={(e) => setSchemaFilter(e.target.value)}
                    placeholder="Filter types..."
                    className="w-32 bg-transparent text-xs text-white outline-none placeholder:text-sand/35"
                  />
                  {schemaFilter && (
                    <button
                      onClick={() => setSchemaFilter('')}
                      className="rounded-full p-0.5 text-sand/50 transition hover:text-white hover:bg-white/10"
                      aria-label="Clear type filter"
                    >
                      <X className="h-3 w-3" />
                    </button>
                  )}
                </label>
              )}
            </div>
            {parsedTypes ? (
              <div className="grid gap-3 md:grid-cols-2">
                {schemaTypes.map((type) => (
                  <div key={type.name} className="rounded-2xl border border-white/10 bg-black/20 p-4">
                    <div className="flex items-center gap-2">
                      <h4 className="font-mono text-sm font-medium text-glow">{type.name}</h4>
                      <span className="rounded-full border border-white/10 px-2 py-0.5 text-[10px] text-sand/50">{type.kind}</span>
                    </div>
                    {type.description && (
                      <p className="mt-2 text-xs leading-5 text-sand/80">{type.description}</p>
                    )}
                    {type.fields && type.fields.length > 0 && (
                      <div className="mt-2 font-mono text-[10px] text-sand/50">
                        Fields: {type.fields.map((f) => f.name).join(', ')}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            ) : (
              <div className="grid gap-3 md:grid-cols-2">
                {schemaDocs.map((doc) => (
                  <div key={doc.name} className="rounded-2xl border border-white/10 bg-black/20 p-4">
                    <h4 className="font-mono text-sm font-medium text-glow">{doc.name}</h4>
                    <p className="mt-2 text-xs leading-5 text-sand/80">{doc.description}</p>
                    {doc.args !== '-' && (
                      <p className="mt-2 font-mono text-[10px] text-sand/50">Args: {doc.args}</p>
                    )}
                  </div>
                ))}
              </div>
            )}
            {parsedTypes && schemaTypes.length === 0 && schemaFilter && (
              <p className="text-sm text-sand/50">No types matching &ldquo;{schemaFilter}&rdquo;</p>
            )}
          </div>
        )}
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
            className="mt-5 h-[22rem] w-full rounded-[24px] border border-white/10 bg-black/25 p-5 font-mono text-sm leading-6 text-sand outline-none placeholder:text-sand/35"
            spellCheck={false}
          />

          <div className="mt-4">
            <div className="flex items-center justify-between">
              <p className="text-xs uppercase tracking-[0.25em] text-mist">Variables (JSON)</p>
              {variablesError && <p className="text-xs text-rose-300">{variablesError}</p>}
            </div>
            <textarea
              value={variables}
              onChange={(event) => { setVariables(event.target.value); setVariablesError(null) }}
              placeholder='{"chainId": "ethereum"}'
              className="mt-2 h-32 w-full rounded-[24px] border border-white/10 bg-black/25 p-5 font-mono text-sm leading-6 text-sand outline-none placeholder:text-sand/35"
              spellCheck={false}
            />
          </div>
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
