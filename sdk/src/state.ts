import type { Annotation, Identity, SessionView, SiteConfig, User } from './events'

export type Filter = 'all' | 'mine' | 'resolved' | 'pending'
export type ThemeMode = 'auto' | 'light' | 'dark'

export type SDKState = {
  siteConfig: SiteConfig | null
  user: User | null
  identities: Identity[]
  sessions: SessionView[]
  currentSessionId: number | null
  sessionsLoading: boolean
  sessionsLoaded: boolean
  annotations: Annotation[]
  orphans: Set<number>
  activeId: number | null
  filter: Filter
  nextCursor: number
  loading: boolean
  loadingMore: boolean
  error: string
  drawerOpen: boolean
  accountOpen: boolean
  siteBanned: boolean
  theme: ThemeMode
  locale: 'zh-CN' | 'en'
}

export function initialState(): SDKState {
  return {
    siteConfig: null,
    user: null,
    identities: [],
    sessions: [],
    currentSessionId: null,
    sessionsLoading: false,
    sessionsLoaded: false,
    annotations: [],
    orphans: new Set(),
    activeId: null,
    filter: 'all',
    nextCursor: 0,
    loading: false,
    loadingMore: false,
    error: '',
    drawerOpen: false,
    accountOpen: false,
    siteBanned: false,
    theme: 'auto',
    locale: 'zh-CN',
  }
}

export function mergeState(state: SDKState, patch: Partial<SDKState>): SDKState {
  return { ...state, ...patch }
}

/** 追加分页结果，按 id 去重并保持顺序。 */
export function appendAnnotations(current: Annotation[], incoming: Annotation[]): Annotation[] {
  const seen = new Set(current.map((a) => a.id))
  const out = [...current]
  for (const item of incoming) {
    if (seen.has(item.id)) continue
    seen.add(item.id)
    out.push(item)
  }
  return out
}

export function upsertAnnotation(current: Annotation[], annotation: Annotation): Annotation[] {
  let found = false
  const out = current.map((item) => {
    if (item.id !== annotation.id) return item
    found = true
    return annotation
  })
  if (!found) out.push(annotation)
  return out
}

export function removeAnnotation(current: Annotation[], id: number): Annotation[] {
  return current.filter((item) => item.id !== id)
}
