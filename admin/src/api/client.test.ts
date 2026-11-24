import { beforeEach, describe, expect, it, vi } from 'vitest'
import { http, TOKEN_KEY } from './client'

beforeEach(() => {
  localStorage.clear()
  vi.restoreAllMocks()
  window.history.replaceState(null, '', '/admin/')
})

describe('admin client', () => {
  it('consumes token from hash and cleans url', () => {
    window.history.replaceState(null, '', '/admin/#annot_token=abc')
    expect(http.consumeHashToken()).toBe(true)
    expect(localStorage.getItem(TOKEN_KEY)).toBe('abc')
    expect(window.location.hash).toBe('')
  })

  it('clears token and emits event on 401', async () => {
    const listener = vi.fn()
    window.addEventListener('admin:auth-expired', listener)
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ error: { code: 'unauthorized', message: 'no' } }), {
          status: 401,
          headers: { 'Content-Type': 'application/json' },
        }),
      ),
    )
    http.setToken('abc')
    await expect(http.request('/me')).rejects.toMatchObject({ status: 401, code: 'unauthorized' })
    expect(localStorage.getItem(TOKEN_KEY)).toBeNull()
    expect(listener).toHaveBeenCalled()
    window.removeEventListener('admin:auth-expired', listener)
  })

  it('attaches bearer token', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ ok: true }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)
    http.setToken('tok')
    await http.request('/me')
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('/api/v1/me')
    expect((init.headers as Headers).get('Authorization')).toBe('Bearer tok')
  })
})
