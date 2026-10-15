// Finality requires explicit RPC evidence. The legacy status field describes
// processing and must never be used to infer a finality guarantee.
export function finalityLabel(raw: Record<string, unknown>, now = Date.now()): string {
  const record = (value: unknown): Record<string, unknown> =>
    typeof value === 'object' && value !== null && !Array.isArray(value) ? value as Record<string, unknown> : {}
  const state = record(raw.state)
  const finality = record(state.finality)
  if (state.canonicality !== 'canonical') return 'Unknown'
  if (finality.confidence === 'degraded') return 'Degraded'
  if (finality.state !== 'safe' && finality.state !== 'finalized') return 'Unknown'
  if (finality.confidence !== finality.state || finality.source !== `rpc:${finality.state}`) return 'Unknown'
  const observed = typeof finality.observed_at === 'string' ? Date.parse(finality.observed_at) : NaN
  // This display expires observations after 60 seconds; missing, future or
  // stale evidence cannot become a green finalized label.
  if (!Number.isFinite(observed) || observed > now + 5000 || now - observed > 60000) return 'Unknown'
  const block = raw.blockNumber ?? raw.block_number
  if (typeof finality.head_number !== 'number' || !Number.isSafeInteger(finality.head_number)
    || typeof block !== 'number' || !Number.isSafeInteger(block) || block < 0 || finality.head_number < block) return 'Unknown'
  if (typeof finality.head_hash !== 'string' || !/^0x[0-9a-fA-F]{64}$/.test(finality.head_hash)) return 'Unknown'
  return finality.state === 'safe' ? 'Safe' : 'Finalized'
}
