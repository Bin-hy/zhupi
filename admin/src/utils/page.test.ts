import { describe, expect, it } from 'vitest'
import { buildPageLink, preferredSiteOrigin } from './page'

describe('page links', () => {
  it('prefers a non-loopback HTTPS origin', () => {
    expect(
      preferredSiteOrigin(['http://localhost:4321', 'https://zhupi.pages.dev']),
    ).toBe('https://zhupi.pages.dev')
    expect(
      preferredSiteOrigin(['http://127.0.0.1:4321', 'http://blog.example.com']),
    ).toBe('http://blog.example.com')
  })

  it('falls back to loopback only when it is the only origin', () => {
    expect(preferredSiteOrigin(['http://localhost:4321'])).toBe(
      'http://localhost:4321',
    )
    expect(preferredSiteOrigin(['not-a-url'])).toBe('')
  })

  it('builds links from page keys and keeps absolute URLs', () => {
    const origins = ['http://localhost:4321', 'https://zhupi.pages.dev']
    expect(buildPageLink('/playground/', origins)).toBe(
      'https://zhupi.pages.dev/playground/',
    )
    expect(buildPageLink('guide/start', origins)).toBe(
      'https://zhupi.pages.dev/guide/start',
    )
    expect(buildPageLink('https://docs.example.com/post/1', origins)).toBe(
      'https://docs.example.com/post/1',
    )
    expect(buildPageLink('', origins)).toBe('')
  })
})
