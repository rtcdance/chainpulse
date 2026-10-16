import { act, cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import FinalityStatus from '../FinalityStatus'

const observed = new Date('2026-10-16T00:00:00Z')
const raw = {
  blockNumber: 5,
  state: {
    canonicality: 'canonical',
    processing: 'processed',
    finality: {
      state: 'finalized', confidence: 'finalized', source: 'rpc:finalized',
      observed_at: observed.toISOString(), head_number: 10, head_hash: `0x${'1'.repeat(64)}`,
    },
  },
}

afterEach(() => { cleanup(); vi.useRealTimers() })

describe('FinalityStatus freshness', () => {
  it('expires visible evidence without a list refresh and stops its clock on unmount', () => {
    vi.useFakeTimers()
    vi.setSystemTime(observed)
    const view = render(<FinalityStatus raw={raw} />)
    expect(screen.getByText('Finalized')).toBeInTheDocument()
    act(() => vi.advanceTimersByTime(61000))
    expect(screen.getByText('Unknown')).toBeInTheDocument()
    view.unmount()
    expect(vi.getTimerCount()).toBe(0)
  })
})
