import { describe, it, expect } from 'vitest'
import { render } from '@testing-library/react'
import { Skeleton, PageSkeleton } from '../Skeleton'

describe('Skeleton', () => {
  it('renders with default class', () => {
    const { container } = render(<Skeleton />)
    const div = container.firstChild as HTMLElement
    expect(div.className).toContain('animate-pulse')
    expect(div.className).toContain('rounded-2xl')
    expect(div.className).toContain('bg-white/5')
  })

  it('appends custom className', () => {
    const { container } = render(<Skeleton className="h-8 w-40" />)
    const div = container.firstChild as HTMLElement
    expect(div.className).toContain('h-8')
    expect(div.className).toContain('w-40')
  })
})

describe('PageSkeleton', () => {
  it('renders multiple skeleton placeholders', () => {
    const { container } = render(<PageSkeleton />)
    const skeletons = container.querySelectorAll('.animate-pulse')
    expect(skeletons.length).toBeGreaterThanOrEqual(5)
  })
})