import { AlertCircle, ArrowRight, CheckCircle2, Globe, Loader2, Wallet, Zap, Shield, Code2, Layers, BarChart3, Radio } from 'lucide-react'
import { useAuth } from '../lib/auth'

const features = [
  {
    icon: <Globe className="h-6 w-6" />,
    title: 'Multi-Chain Indexing',
    description: '14+ blockchains supported — Ethereum, Polygon, BSC, Arbitrum, Optimism, Solana, Cosmos and more. One API, all chains.',
  },
  {
    icon: <Zap className="h-6 w-6" />,
    title: 'Real-Time WebSocket',
    description: 'Subscribe to on-chain events as they happen. Filter by chain, contract address, or event name with sub-second latency.',
  },
  {
    icon: <BarChart3 className="h-6 w-6" />,
    title: 'GraphQL & REST API',
    description: 'Query with precision. GraphQL for flexible data fetching, REST for simplicity. Pagination, filtering, and aggregation built in.',
  },
  {
    icon: <Shield className="h-6 w-6" />,
    title: 'Self-Hosted & Private',
    description: 'Deploy on your own infrastructure. No third-party data sharing. Full control over your blockchain data pipeline.',
  },
  {
    icon: <Layers className="h-6 w-6" />,
    title: 'Rich Event Coverage',
    description: 'ERC-20/721/1155 transfers, DeFi protocol events (Uniswap, Aave, Compound), governance, L2 bridges, MEV tracking, and custom ABI support.',
  },
  {
    icon: <Radio className="h-6 w-6" />,
    title: 'Production Ready',
    description: 'Reorg detection, dead-letter queues, idempotent processing, Prometheus metrics, Grafana dashboards, and automated health checks.',
  },
]

function formatAddress(address: string): string {
  return `${address.slice(0, 6)}...${address.slice(-4)}`
}

export default function Landing() {
  const { address, step, error, connect, signIn, isAuthenticated } = useAuth()

  return (
    <div className="min-h-screen bg-ink text-sand">
      <div className="absolute inset-0 bg-grid opacity-60" />
      <div className="absolute inset-x-0 top-0 h-[36rem] bg-[radial-gradient(circle_at_top,rgba(244,162,97,0.22),transparent_50%)]" />

      <div className="relative mx-auto max-w-6xl px-4 py-6 sm:px-6 lg:px-8">
        <nav className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-[linear-gradient(135deg,rgba(244,162,97,0.95),rgba(233,196,106,0.9))] shadow-[0_8px_24px_rgba(244,162,97,0.22)]">
              <Zap className="text-ink" size={20} />
            </div>
            <span className="text-lg font-semibold text-white">ChainPulse</span>
          </div>
          <div className="flex items-center gap-4">
            <a href="https://github.com/rtcdance/chainpulse" className="text-sm text-sand/60 transition hover:text-sand/90" target="_blank" rel="noopener noreferrer">
              <Code2 className="inline h-4 w-4 mr-1" />
              GitHub
            </a>
            {isAuthenticated ? (
              <span className="rounded-full border border-emerald-300/25 bg-emerald-300/10 px-3 py-1.5 text-xs text-emerald-200">
                <CheckCircle2 className="inline h-3 w-3 mr-1" />
                {formatAddress(address)}
              </span>
            ) : null}
          </div>
        </nav>

        <section className="mt-24 text-center sm:mt-32">
          <div className="mx-auto max-w-3xl">
            <div className="inline-flex items-center gap-2 rounded-full border border-glow/20 bg-glow/8 px-4 py-1.5 text-sm text-glow/90">
              <Radio className="h-3.5 w-3.5" />
              Web3 Blockchain Event Indexing Platform
            </div>

            <h1 className="mt-8 text-4xl font-semibold leading-tight text-white sm:text-5xl lg:text-6xl">
              Index blockchain events.
              <br />
              <span className="text-glow">Query them anywhere.</span>
            </h1>

            <p className="mt-6 text-lg leading-8 text-sand/70 sm:text-xl">
              ChainPulse is a self-hosted, multi-chain event indexing platform.
              Pull on-chain events from 14+ blockchains, decode them, and query via REST, GraphQL, or real-time WebSocket — all on your own infrastructure.
            </p>
          </div>

          <div className="mt-12">
            {error && (
              <div className="mx-auto mb-6 flex max-w-md items-center gap-2 rounded-2xl border border-rose-400/30 bg-rose-400/10 p-3 text-sm text-rose-100">
                <AlertCircle className="h-4 w-4 shrink-0" />
                <span>{error}</span>
              </div>
            )}

            {step === 'connected' ? (
              <div className="mx-auto max-w-md space-y-4 rounded-[28px] border border-white/10 bg-white/5 p-8 backdrop-blur">
                <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-2xl bg-emerald-300/15">
                  <CheckCircle2 className="h-7 w-7 text-emerald-300" />
                </div>
                <h3 className="text-xl font-semibold text-white">Wallet Connected</h3>
                <p className="font-mono text-sm text-sand/60">{formatAddress(address)}</p>
                <p className="text-sm leading-6 text-sand/75">
                  Sign a message with your wallet to verify ownership and access the dashboard. This uses EIP-4361 (Sign-In with Ethereum).
                </p>
                <button
                  onClick={signIn}
                  className="inline-flex w-full items-center justify-center gap-2 rounded-full bg-glow px-6 py-3 text-sm font-medium text-ink transition hover:brightness-110"
                >
                  <Wallet className="h-4 w-4" />
                  Sign In to Continue
                  <ArrowRight className="h-4 w-4" />
                </button>
              </div>
            ) : step === 'signing' ? (
              <div className="mx-auto flex max-w-md items-center justify-center gap-3 rounded-[28px] border border-white/10 bg-white/5 p-12 backdrop-blur">
                <Loader2 className="h-6 w-6 animate-spin text-glow" />
                <span className="text-lg text-white">Signing in...</span>
              </div>
            ) : step === 'connecting' ? (
              <div className="mx-auto flex max-w-md items-center justify-center gap-3 rounded-[28px] border border-white/10 bg-white/5 p-12 backdrop-blur">
                <Loader2 className="h-6 w-6 animate-spin text-glow" />
                <span className="text-lg text-white">Connecting wallet...</span>
              </div>
            ) : (
              <button
                onClick={connect}
                className="inline-flex items-center gap-3 rounded-full bg-glow px-8 py-4 text-base font-semibold text-ink shadow-[0_12px_40px_rgba(244,162,97,0.25)] transition hover:brightness-110"
              >
                <Wallet className="h-5 w-5" />
                Connect MetaMask to Get Started
              </button>
            )}
          </div>
        </section>

        <section className="mt-28 sm:mt-36">
          <div className="text-center">
            <p className="text-xs uppercase tracking-[0.35em] text-mist">Capabilities</p>
            <h2 className="mt-4 text-3xl font-semibold text-white sm:text-4xl">
              Everything you need to index and query blockchain data
            </h2>
          </div>

          <div className="mt-14 grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
            {features.map((feature) => (
              <div
                key={feature.title}
                className="group rounded-[28px] border border-white/10 bg-white/5 p-6 transition hover:border-glow/20 hover:bg-white/[0.07]"
              >
                <div className="inline-flex h-12 w-12 items-center justify-center rounded-2xl bg-glow/10 text-glow">
                  {feature.icon}
                </div>
                <h3 className="mt-5 text-lg font-medium text-white">{feature.title}</h3>
                <p className="mt-3 text-sm leading-6 text-sand/65">{feature.description}</p>
              </div>
            ))}
          </div>
        </section>

        <section className="mt-28 rounded-[28px] border border-white/10 bg-white/5 p-8 text-center sm:p-12">
          <h2 className="text-2xl font-semibold text-white sm:text-3xl">
            Ready to index your first event?
          </h2>
          <p className="mt-4 text-base leading-7 text-sand/65 sm:text-lg">
            Connect your wallet, deploy ChainPulse on your infrastructure, and start pulling blockchain events in minutes. Self-hosted means your data stays yours.
          </p>
          <button
            onClick={connect}
            className="mt-8 inline-flex items-center gap-3 rounded-full bg-glow px-8 py-4 text-base font-semibold text-ink shadow-[0_12px_40px_rgba(244,162,97,0.25)] transition hover:brightness-110"
          >
            <Wallet className="h-5 w-5" />
            Connect Wallet
          </button>
        </section>

        <footer className="mt-20 border-t border-white/5 py-8 text-center text-sm text-sand/40">
          <p>ChainPulse — Self-hosted Web3 event indexing. Open source. Built with Go + React.</p>
        </footer>
      </div>
    </div>
  )
}