import type { Annotation, AnnotationComment, Identity, SessionView, SiteConfig, User } from './events'

export type Filter = 'all' | 'mine' | 'resolved' | 'pending'
export type ThemeMode = 'auto' | 'light' | 'dark'
export type SortMode = 'position' | 'newest'

export type FilterCounts = {
  all: number
  mine: number
  resolved: number
  pending: number
}

export type SDKState = {
  siteConfig: SiteConfig | null
  user: User | null
  identities: Identity[]
  sessions: SessionView[]
  currentSessionId: number | null
  sessionsLoading: boolean
  sessionsLoaded: boolean
  annotations: Annotation[]
  total: number
  counts: FilterCounts
  orphans: Set<number>
  activeId: number | null
  filter: Filter
  sort: SortMode
  search: string
  unreadIds: Set<number>
  lastSeenAt: number
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
    total: 0,
    counts: { all: 0, mine: 0, resolved: 0, pending: 0 },
    orphans: new Set(),
    activeId: null,
    filter: 'all',
    sort: 'position',
    search: '',
    unreadIds: new Set(),
    lastSeenAt: 0,
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

/** 把新回复追加到对应批注，避免整页重新加载导致滚动位置丢失。 */
export function appendComment(
  current: Annotation[],
  annotationId: number,
  comment: AnnotationComment,
): Annotation[] {
  return current.map((item) => {
    if (item.id !== annotationId) return item
    if (item.comments.some((c) => c.id === comment.id)) return item
    return { ...item, comments: [...item.comments, comment], updatedAt: comment.createdAt }
  })
}

/** 面板展示顺序：position 保持文档顺序，newest 按发布时间倒序。 */
export function sortForDisplay(list: Annotation[], mode: SortMode): Annotation[] {
  if (mode !== 'newest') return list
  return [...list].sort((a, b) => {
    const diff = Date.parse(b.createdAt) - Date.parse(a.createdAt)
    return diff !== 0 ? diff : b.id - a.id
  })
}

/** 在已加载的批注中做多关键词搜索（引用、正文、作者、页面标题）。 */
export function searchAnnotations(list: Annotation[], query: string): Annotation[] {
  const terms = query.trim().toLowerCase().split(/\s+/).filter(Boolean)
  if (terms.length === 0) return list
  return list.filter((a) => {
    const haystack = [
      a.quoteExact,
      a.pageTitle,
      a.section,
      a.author.name,
      ...a.comments.map((c) => `${c.author.name} ${c.content}`),
    ]
      .join('\n')
      .toLowerCase()
    return terms.every((term) => haystack.includes(term))
  })
}

/** 最近一次讨论时间，用于「新回复」判断。 */
export function lastActivityAt(a: Annotation): number {
  const times = [a.createdAt, a.updatedAt, ...a.comments.map((c) => c.createdAt)]
    .map((value) => Date.parse(value))
    .filter((value) => Number.isFinite(value))
  return times.length ? Math.max(...times) : 0
}
