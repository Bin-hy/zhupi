import { describe, expect, it } from 'vitest'
import { absoluteTime, initials, relativeTime } from './format'

describe('format', () => {
  it('initials', () => {
    expect(initials('Alice Wang')).toBe('AW')
    expect(initials('张三')).toBe('张')
    expect(initials('')).toBe('?')
  })

  it('relative time', () => {
    const now = Date.parse('2026-01-01T12:00:00Z')
    expect(relativeTime('2026-01-01T11:59:30Z', 'zh-CN', now)).toContain('30')
    expect(relativeTime('2026-01-01T11:00:00Z', 'zh-CN', now)).toContain('1')
    expect(relativeTime('2026-01-01T12:00:00Z', 'en', now).length).toBeGreaterThan(0)
  })

  it('absolute time', () => {
    expect(absoluteTime('2026-01-01T12:00:00Z', 'zh-CN')).toContain('2026')
  })
})
