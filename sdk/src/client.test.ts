import { beforeEach, describe, expect, it, vi } from 'vitest'
import { AnnotationClient } from './client'

beforeEach(() => {
  localStorage.clear()
  vi.restoreAllMocks()
})

describe('AnnotationClient', () => {
  it('attaches site key and bearer token', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ id: 1, displayName: 'alice', avatarUrl: '', isSuperadmin: false }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)

    const client = new AnnotationClient('http://api.test/', 'site-key')
    client.setToken('token-123')
    await client.me()

    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('http://api.test/api/v1/me')
    const headers = init.headers as Headers
    expect(headers.get('X-Site-Key')).toBe('site-key')
    expect(headers.get('Authorization')).toBe('Bearer token-123')
  })

  it('clears token and throws ApiError on 401', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ error: { code: 'unauthorized', message: '请先登录' } }), {
        status: 401,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)

    const client = new AnnotationClient('http://api.test', 'site-key')
    client.setToken('token-123')

    await expect(client.me()).rejects.toMatchObject({ status: 401, code: 'unauthorized' })
    expect(client.getToken()).toBeNull()
  })

  it('consumes auth token from hash and cleans the URL', () => {
    window.history.replaceState(null, '', '/doc#annot_token=abc')
    const client = new AnnotationClient('http://api.test', 'site-key')
    const result = client.consumeAuthFromHash()
    expect(result.loggedIn).toBe(true)
    expect(client.getToken()).toBe('abc')
    expect(window.location.hash).toBe('')
  })
})

describe('AnnotationClient extensions', () => {
  it('builds filtered pagination query', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ annotations: [], nextCursor: 0, total: 5 }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)
    const client = new AnnotationClient('http://api.test', 'site-key')
    const result = await client.listAnnotations('/a b', 'resolved', 10, 20)
    expect(result.total).toBe(5)
    const [url] = fetchMock.mock.calls[0]
    expect(url).toContain('page=%2Fa%20b')
    expect(url).toContain('filter=resolved')
    expect(url).toContain('cursor=10')
    expect(url).toContain('limit=20')
  })

  it('sends anchor update body', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ id: 1 }), { status: 200, headers: { 'Content-Type': 'application/json' } }),
    )
    vi.stubGlobal('fetch', fetchMock)
    const client = new AnnotationClient('http://api.test', 'site-key')
    const anchor = { quote: { type: 'TextQuoteSelector', exact: 'x', prefix: '', suffix: '' } }
    await client.updateAnnotation(1, { anchor })
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('http://api.test/api/v1/annotations/1')
    expect(init.method).toBe('PATCH')
    expect(JSON.parse(init.body)).toEqual({ anchor })
  })

  it('lists sessions', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ sessions: [], currentSessionId: 1 }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)
    const client = new AnnotationClient('http://api.test', 'site-key')
    await client.sessions()
    expect(fetchMock.mock.calls[0][0]).toBe('http://api.test/api/v1/me/sessions')
  })
})
