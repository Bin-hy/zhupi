import { describe, expect, it } from 'vitest'
import { buildDeepLink, parseDeepLink } from './deeplink'

describe('deeplink', () => {
  it('parses annotation id', () => {
    expect(parseDeepLink('#annot-12')).toBe(12)
    expect(parseDeepLink('#foo=1&annot-3')).toBe(3)
    expect(parseDeepLink('#annot-x')).toBeNull()
    expect(parseDeepLink('')).toBeNull()
  })

  it('builds link without token and keeps other params', () => {
    const link = buildDeepLink('https://blog.test/doc?x=1#annot_token=secret&foo=bar', 9)
    expect(link).not.toContain('annot_token')
    expect(link).toContain('annot-9')
    expect(link).toContain('foo=bar')
    expect(link).toContain('x=1')
  })
})
