import type { Anchor, Annotation, AnnotationComment, Identity, SessionView, SiteConfig, User } from './events'
import type { Filter } from './state'

export class ApiError extends Error {
  status: number
  code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

export type CreateAnnotationInput = {
  pageKey: string
  pageTitle: string
  anchor: Anchor
  content: string
}

export class AnnotationClient {
  readonly base: string
  readonly siteKey: string
  private readonly tokenKey: string

  constructor(base: string, siteKey: string) {
    this.base = base.replace(/\/+$/, '')
    this.siteKey = siteKey
    this.tokenKey = `annotation-sdk:${this.base}:${siteKey}`
  }

  getToken(): string | null {
    try {
      return localStorage.getItem(this.tokenKey)
    } catch {
      return null
    }
  }

  setToken(token: string | null): void {
    try {
      if (token) localStorage.setItem(this.tokenKey, token)
      else localStorage.removeItem(this.tokenKey)
    } catch {
      /* ignore */
    }
  }

  /** 读取登录回调 fragment，并清理地址栏。 */
  consumeAuthFromHash(): { loggedIn: boolean; bound: boolean } {
    if (typeof window === 'undefined') return { loggedIn: false, bound: false }
    const hash = window.location.hash
    if (!hash) return { loggedIn: false, bound: false }
    const params = new URLSearchParams(hash.replace(/^#/, ''))
    const token = params.get('annot_token')
    const bound = params.get('annot_bind') === 'ok'
    if (!token && !bound) return { loggedIn: false, bound: false }
    if (token) this.setToken(token)
    params.delete('annot_token')
    params.delete('annot_bind')
    const rest = params.toString()
    const url = window.location.pathname + window.location.search + (rest ? `#${rest}` : '')
    window.history.replaceState(null, '', url)
    return { loggedIn: !!token, bound }
  }

  private async request<T>(path: string, init: RequestInit = {}): Promise<T> {
    const headers = new Headers(init.headers)
    headers.set('X-Site-Key', this.siteKey)
    if (init.body && !headers.has('Content-Type')) headers.set('Content-Type', 'application/json')
    const token = this.getToken()
    if (token) headers.set('Authorization', `Bearer ${token}`)

    const res = await fetch(`${this.base}${path}`, { ...init, headers })
    if (res.status === 401) this.setToken(null)
    if (!res.ok) {
      let code = `http_${res.status}`
      let message = `Request failed (${res.status})`
      try {
        const data = await res.json()
        if (data?.error?.code) code = data.error.code
        if (data?.error?.message) message = data.error.message
      } catch {
        /* ignore */
      }
      throw new ApiError(res.status, code, message)
    }
    if (res.status === 204) return undefined as T
    return (await res.json()) as T
  }

  siteConfig(): Promise<SiteConfig> {
    return this.request<SiteConfig>('/api/v1/site-config')
  }

  me(): Promise<User> {
    return this.request<User>('/api/v1/me')
  }

  listAnnotations(
    pageKey: string,
    filter: Filter = 'all',
    cursor = 0,
    limit = 50,
  ): Promise<{ annotations: Annotation[]; nextCursor: number; total: number }> {
    return this.request(
      `/api/v1/annotations?page=${encodeURIComponent(pageKey)}&filter=${filter}&cursor=${cursor}&limit=${limit}`,
    )
  }

  createAnnotation(input: CreateAnnotationInput): Promise<Annotation> {
    return this.request<Annotation>('/api/v1/annotations', { method: 'POST', body: JSON.stringify(input) })
  }

  updateAnnotation(
    id: number,
    input: { status?: 'open' | 'resolved'; anchor?: Anchor },
  ): Promise<Annotation> {
    return this.request<Annotation>(`/api/v1/annotations/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(input),
    })
  }

  patchAnnotation(id: number, status: 'open' | 'resolved'): Promise<Annotation> {
    return this.updateAnnotation(id, { status })
  }

  deleteAnnotation(id: number): Promise<void> {
    return this.request<void>(`/api/v1/annotations/${id}`, { method: 'DELETE' })
  }

  addComment(id: number, content: string): Promise<AnnotationComment> {
    return this.request<AnnotationComment>(`/api/v1/annotations/${id}/comments`, {
      method: 'POST',
      body: JSON.stringify({ content }),
    })
  }

  deleteComment(id: number): Promise<void> {
    return this.request<void>(`/api/v1/comments/${id}`, { method: 'DELETE' })
  }

  identities(): Promise<{ identities: Identity[] }> {
    return this.request('/api/v1/me/identities')
  }

  sessions(): Promise<{ sessions: SessionView[]; currentSessionId: number }> {
    return this.request('/api/v1/me/sessions')
  }

  revokeSession(id: number): Promise<void> {
    return this.request<void>(`/api/v1/me/sessions/${id}`, { method: 'DELETE' })
  }

  revokeAllSessions(): Promise<void> {
    return this.request<void>('/api/v1/me/sessions', { method: 'DELETE' })
  }

  unbindIdentity(id: number): Promise<void> {
    return this.request<void>(`/api/v1/me/identities/${id}`, { method: 'DELETE' })
  }

  async bindStart(providerId: number, returnTo: string): Promise<string> {
    const path = providerId === 0
      ? `/api/v1/me/identities/github/start?redirect=${encodeURIComponent(returnTo)}`
      : `/api/v1/me/identities/oidc/${providerId}/start?redirect=${encodeURIComponent(returnTo)}`
    const result = await this.request<{ redirectUrl: string }>(path, { method: 'POST' })
    return result.redirectUrl
  }

  devLogin(name: string): Promise<{ token: string; user: User }> {
    return fetch(`${this.base}/api/v1/auth/dev/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-Site-Key': this.siteKey },
      body: JSON.stringify({ name }),
    }).then(async (res) => {
      if (!res.ok) throw new ApiError(res.status, 'dev_login_failed', '本地调试登录失败')
      const data = await res.json()
      this.setToken(data.token)
      return data
    })
  }

  loginUrl(providerId: number, returnTo: string): string {
    const redirect = encodeURIComponent(returnTo)
    if (providerId === 0) {
      return `${this.base}/api/v1/auth/github/start?siteKey=${encodeURIComponent(this.siteKey)}&redirect=${redirect}`
    }
    return `${this.base}/api/v1/auth/oidc/${providerId}/start?siteKey=${encodeURIComponent(this.siteKey)}&redirect=${redirect}`
  }

  async logout(): Promise<void> {
    try {
      await this.request<void>('/api/v1/auth/logout', { method: 'POST' })
    } catch {
      /* 忽略退出失败，本地仍清理 token */
    }
    this.setToken(null)
  }
}
