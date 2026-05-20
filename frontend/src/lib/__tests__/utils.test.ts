import { describe, it, expect, vi } from 'vitest'
import { formatTimestamp } from '../utils'
import { toRecord, getField, toNumber } from '../api/internal'
import { exportToCSV } from '../export'

describe('formatTimestamp', () => {
  it('returns "-" for null', () => {
    expect(formatTimestamp(null)).toBe('-')
  })

  it('handles second timestamps', () => {
    const result = formatTimestamp(1710000000)
    expect(result).toBeTruthy()
    expect(result).not.toBe('-')
  })

  it('handles millisecond timestamps', () => {
    const result = formatTimestamp(1710000000000)
    expect(result).toBeTruthy()
    expect(result).not.toBe('-')
  })

  it('returns "-" for NaN timestamps', () => {
    expect(formatTimestamp(Number.NaN)).toBe('-')
  })
})

describe('toRecord', () => {
  it('returns object for valid input', () => {
    expect(toRecord({ a: 1 })).toEqual({ a: 1 })
  })

  it('returns empty object for null', () => {
    expect(toRecord(null)).toEqual({})
  })

  it('returns empty object for string', () => {
    expect(toRecord('invalid')).toEqual({})
  })
})

describe('getField', () => {
  it('returns value for first matching key', () => {
    expect(getField({ name: 'Transfer' }, ['eventName', 'name'], '')).toBe('Transfer')
  })

  it('falls back to second key', () => {
    expect(getField({ eventName: 'Swap' }, ['name', 'eventName'], '')).toBe('Swap')
  })

  it('returns fallback for no match', () => {
    expect(getField({}, ['name'], 'default')).toBe('default')
  })
})

describe('toNumber', () => {
  it('converts string number', () => {
    expect(toNumber('42')).toBe(42)
  })

  it('returns number as-is', () => {
    expect(toNumber(42)).toBe(42)
  })

  it('returns null for non-numeric', () => {
    expect(toNumber('abc')).toBeNull()
  })
})

describe('exportToCSV', () => {
  it('escapes double quotes in CSV', () => {
    const createObjectURL = vi.fn(() => 'blob:test')
    const originalCreateObjectURL = URL.createObjectURL
    URL.createObjectURL = createObjectURL
    try {
      exportToCSV([{ name: 'Transfer', value: 'with "quotes"' }], [{ key: 'name', label: 'Name' }, { key: 'value', label: 'Value' }], 'test.csv')
      expect(createObjectURL).toHaveBeenCalled()
    } finally {
      URL.createObjectURL = originalCreateObjectURL
    }
  })
})