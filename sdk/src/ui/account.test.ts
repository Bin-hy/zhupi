import { describe, expect, it } from 'vitest'
import { renderAccount } from './account'
import { messages } from '../i18n'
import { initialState, mergeState } from '../state'

const msg = messages('zh-CN')

describe('renderAccount', () => {
  it('shows login providers when logged out', () => {
    const state = mergeState(initialState(), {
      siteConfig: {
        siteKey: 'k',
        displayName: 'demo',
        themeColor: '#087f5b',
        locale: 'zh-CN',
        moderationMode: 'post',
        providers: [{ id: 0, type: 'github', name: 'GitHub' }],
      },
    })
    const html = renderAccount(state, msg, true)
    expect(html).toContain('data-action="login"')
    expect(html).toContain('data-action="dev-login"')
  })

  it('hides bind for already bound providers and renders sessions', () => {
    const state = mergeState(initialState(), {
      user: { id: 1, displayName: 'alice', avatarUrl: '', isSuperadmin: false },
      siteConfig: {
        siteKey: 'k',
        displayName: 'demo',
        themeColor: '#087f5b',
        locale: 'zh-CN',
        moderationMode: 'post',
        providers: [
          { id: 0, type: 'github', name: 'GitHub' },
          { id: 2, type: 'oidc', name: 'SSO', issuer: 'https://sso.test' },
        ],
      },
      identities: [
        { id: 1, provider: 'github', siteId: 0, issuer: 'https://github.com', handle: 'alice', avatarUrl: '', createdAt: '', lastLoginAt: '' },
      ],
      sessions: [
        { id: 10, userAgent: 'Chrome', ip: '127.0.0.1', createdAt: '', lastSeenAt: new Date().toISOString(), expiresAt: '', current: true },
      ],
      sessionsLoaded: true,
    })
    const html = renderAccount(state, msg, false)
    expect(html).toContain('data-action="bind" data-provider="2"')
    expect(html).not.toContain('data-action="bind" data-provider="0"')
    expect(html).toContain('data-action="revoke-session" data-id="10"')
    expect(html).toContain('当前设备')
  })
})
