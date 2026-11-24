import { describe, expect, it } from 'vitest'
import { formatTime, shortKey } from './format'

describe('format', () => {
  it('formats time safely', () => {
    expect(formatTime('')).toBe('-')
    expect(formatTime('not-a-date')).toBe('-')
    expect(formatTime('2026-01-01T12:00:00Z')).toContain('2026')
  })

  it('shortens keys', () => {
    expect(shortKey('abcdefghijklmnop')).toBe('abcdef…mnop')
    expect(shortKey('short')).toBe('short')
  })
})
