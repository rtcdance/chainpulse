import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import React from 'react'
import ErrorBoundary from '../ErrorBoundary'

function Throwable({ shouldThrow }: { shouldThrow: boolean }) {
  if (shouldThrow) {
    throw new Error('Test explosion')
  }
  return <div>All good</div>
}

function suppressError(fn: () => void) {
  const original = console.error
  console.error = vi.fn()
  try {
    fn()
  } finally {
    console.error = original
  }
}

describe('ErrorBoundary', () => {
  it('renders children normally', () => {
    render(
      <ErrorBoundary>
        <Throwable shouldThrow={false} />
      </ErrorBoundary>
    )
    expect(screen.getByText('All good')).toBeInTheDocument()
  })

  it('renders fallback UI on error', () => {
    suppressError(() => {
      render(
        <ErrorBoundary>
          <Throwable shouldThrow={true} />
        </ErrorBoundary>
      )
    })
    expect(screen.getByText('Something went wrong')).toBeInTheDocument()
    expect(screen.getByText('Test explosion')).toBeInTheDocument()
  })

  it('renders generic message when error has no message', () => {
    const Broken = (): React.ReactNode => {
      throw new Error()
    }
    suppressError(() => {
      render(
        <ErrorBoundary>
          <Broken />
        </ErrorBoundary>
      )
    })
    expect(screen.getByText('An unexpected error occurred')).toBeInTheDocument()
  })

  it('renders reload button', () => {
    suppressError(() => {
      render(
        <ErrorBoundary>
          <Throwable shouldThrow={true} />
        </ErrorBoundary>
      )
    })
    expect(screen.getByText('Reload Page')).toBeInTheDocument()
  })
})