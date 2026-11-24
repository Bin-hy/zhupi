const TOKEN_KEY = 'annotation-admin:token'

export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly code: string,
    message: string,
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

class ApiClient {
  readonly base = '/api/v1'

  getToken(): string | null {
    try {
      return localStorage.getItem(TOKEN_KEY)
    } catch {
      return null
    }
  }

  setToken(token: string | null): void {
    try {
      if (token) localStorage.setItem(TOKEN_KEY, token)
      else localStorage.removeItem(TOKEN_KEY)
    } catch {
      /* ignore */
    }
  }

  /** 消费管理端登录回调的 token，并清理地址栏。 */
  consumeHashToken(): boolean {
    if (typeof window === 'undefined') return false
    const hash = window.location.hash
    if (!hash) return false
    const params = new URLSearchParams(hash.replace(/^#/, ''))
    const token = params.get('annot_token')
    if (!token) return false
    this.setToken(token)
    params.delete('annot_token')
    const rest = params.toString()
    const url = window.location.pathname + window.location.search + (rest ? `#${rest}` : '')
    window.history.replaceState(null, '', url)
    return true
  }

  async request<T>(path: string, init: RequestInit = {}): Promise<T> {
    const headers = new Headers(init.headers)
    if (init.body && !headers.has('Content-Type')) headers.set('Content-Type', 'application/json')
    const token = this.getToken()
    if (token) headers.set('Authorization', `Bearer ${token}`)

    const res = await fetch(`${this.base}${path}`, { ...init, headers })
    if (res.status === 401) {
      this.setToken(null)
      window.dispatchEvent(new CustomEvent('admin:auth-expired'))
    }
    if (!res.ok) {
      let code = `http_${res.status}`
      let message = `请求失败 (${res.status})`
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
}

export const http = new ApiClient()
export { TOKEN_KEY }
