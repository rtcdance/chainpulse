import { describe, expect, it } from 'vitest'
import { finalityLabel } from '../finality'

const now = Date.parse('2026-10-15T10:00:00Z')
const evidence = { state: 'finalized', confidence: 'finalized', source: 'rpc:finalized', observed_at: new Date(now).toISOString(), head_number: 12, head_hash: `0x${'ab'.repeat(32)}` }
const event = (finality: Record<string, unknown>, canonicality = 'canonical') => ({ blockNumber: 10, state: { canonicality, processing: 'processed', finality } })

describe('finality evidence display', () => {
  it.each([
    ['legacy status alone', { blockNumber: 10, status: 'finalized' }, 'Unknown'],
    ['complete finalized evidence', event(evidence), 'Finalized'],
    ['safe never means finalized', event({ ...evidence, state: 'safe', confidence: 'safe', source: 'rpc:safe' }), 'Safe'],
    ['provider failure', event({ state: 'unknown', confidence: 'degraded', source: 'rpc:unavailable' }), 'Degraded'],
    ['stale observation', event({ ...evidence, observed_at: new Date(now - 60001).toISOString() }), 'Unknown'],
    ['future observation', event({ ...evidence, observed_at: new Date(now + 5001).toISOString() }), 'Unknown'],
    ['latest fallback', event({ ...evidence, source: 'rpc:latest' }), 'Unknown'],
    ['missing hash', event({ ...evidence, head_hash: '' }), 'Unknown'],
    ['head below event', event({ ...evidence, head_number: 9 }), 'Unknown'],
    ['orphaned branch', event(evidence, 'orphaned'), 'Unknown'],
  ])('%s', (_name, raw, expected) => {
    expect(finalityLabel(raw as Record<string, unknown>, now)).toBe(expected)
  })
})
