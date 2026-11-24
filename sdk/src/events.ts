export type User = {
  id: number
  displayName: string
  avatarUrl: string
  isSuperadmin: boolean
}

export type Author = {
  id: number
  name: string
  avatarUrl: string
}

export type QuoteSelector = {
  type: string
  exact: string
  prefix: string
  suffix: string
}

export type Anchor = {
  quote: QuoteSelector
  position?: { start: number; end: number }
  section?: string
}

export type AnnotationComment = {
  id: number
  annotationId: number
  content: string
  moderation: string
  visibility: string
  createdAt: string
  author: Author
}

export type Annotation = {
  id: number
  pageKey: string
  pageTitle: string
  anchor: Anchor
  quoteExact: string
  section: string
  status: 'open' | 'resolved'
  moderation: string
  visibility: string
  createdAt: string
  updatedAt: string
  author: Author
  comments: AnnotationComment[]
}

export type AuthProvider = {
  id: number
  type: 'github' | 'oidc'
  name: string
  issuer?: string
}

export type SiteConfig = {
  siteKey: string
  displayName: string
  themeColor: string
  locale: string
  moderationMode: 'post' | 'pre'
  providers: AuthProvider[]
}

export type Identity = {
  id: number
  provider: string
  siteId: number
  issuer: string
  handle: string
  email?: string
  avatarUrl: string
  createdAt: string
  lastLoginAt: string
}

export type SessionView = {
  id: number
  userAgent: string
  ip: string
  createdAt: string
  lastSeenAt: string
  expiresAt: string
  current: boolean
}

export type EventMap = {
  ready: { siteKey: string }
  login: User
  logout: undefined
  created: Annotation
  updated: Annotation
  deleted: { id: number }
  error: Error
  pagechange: { pageKey: string }
}

type Handler<T> = (payload: T) => void

export class Emitter {
  private handlers = new Map<keyof EventMap, Set<Handler<any>>>()

  on<K extends keyof EventMap>(event: K, handler: Handler<EventMap[K]>): () => void {
    let set = this.handlers.get(event)
    if (!set) {
      set = new Set()
      this.handlers.set(event, set)
    }
    set.add(handler)
    return () => set!.delete(handler)
  }

  emit<K extends keyof EventMap>(event: K, payload: EventMap[K]): void {
    const set = this.handlers.get(event)
    if (!set) return
    for (const handler of [...set]) {
      try {
        handler(payload)
      } catch (err) {
        console.error('[annotation-sdk] event handler failed', event, err)
      }
    }
  }
}
