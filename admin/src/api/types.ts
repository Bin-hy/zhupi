export type Me = {
  id: number
  displayName: string
  avatarUrl: string
  isSuperadmin: boolean
  username: string
  mustChangePassword: boolean
}

export type Site = {
  id: number
  key: string
  name: string
  displayName: string
  status: 'active' | 'disabled'
  moderationMode: 'post' | 'pre'
  themeColor: string
  defaultLocale: string
  origins: string[]
  createdAt: string
  updatedAt: string
}

export type SiteAdmin = {
  siteId: number
  userId: number
  grantedBy: number
  createdAt: string
}

export type SiteBanView = {
  userId: number
  name: string
  avatarUrl: string
  reason: string
  bannedBy: number
  createdAt: string
}

export type Provider = {
  id: number
  siteId: number
  name: string
  issuer: string
  clientId: string
  scopes: string[]
  allowedEmailDomains: string[]
  enabled: boolean
  createdAt: string
  updatedAt: string
}

export type User = {
  id: number
  displayName: string
  avatarUrl: string
  status: 'active' | 'banned'
  isSuperadmin: boolean
  createdAt: string
}

export type Author = {
  id: number
  name: string
  avatarUrl: string
}

export type Annotation = {
  id: number
  siteId?: number
  pageKey: string
  pageTitle: string
  quoteExact: string
  section?: string
  status: 'open' | 'resolved'
  moderation: 'pending' | 'approved' | 'rejected'
  visibility: string
  author: Author
  createdAt: string
  updatedAt?: string
  comments?: Comment[]
}

export type Comment = {
  id: number
  siteId?: number
  annotationId: number
  content: string
  moderation: 'pending' | 'approved' | 'rejected'
  visibility: string
  author: Author
  createdAt: string
}

export type AnnotationBrief = {
  id: number
  pageKey: string
  pageTitle: string
  quoteExact: string
  body?: string
  section?: string
  status: 'open' | 'resolved'
  moderation: 'pending' | 'approved' | 'rejected'
  visibility: string
  author: Author
}

export type ModerationComment = Comment & {
  parent?: AnnotationBrief
}

export type StatusCounts = {
  pending: number
  approved: number
  rejected: number
}

export type ModerationCounts = {
  annotations: StatusCounts
  comments: StatusCounts
}

export type OverviewPending = {
  id: number
  kind: 'annotation' | 'comment'
  siteId: number
  siteName: string
  author: Author
  quoteExact?: string
  content?: string
  createdAt: string
}

export type Overview = {
  counts: {
    sites: number
    users: number
    annotations: number
    pendingAnnotations: number
    comments: number
    pendingComments: number
  }
  sites: Site[]
  recentAnnotations: OverviewPending[]
  recentComments: OverviewPending[]
}

export type Queue = {
  annotations: Annotation[]
  comments: ModerationComment[]
  annotationCursor: number
  commentCursor: number
  counts: ModerationCounts
}

export type AdminAuthProviders = {
  github: boolean
  local: boolean
  providers: Array<{
    id: number
    siteId: number
    siteName: string
    name: string
    issuer: string
  }>
}

export type CreateSiteInput = {
  name: string
  displayName: string
  origins: string[]
  moderationMode: 'post' | 'pre'
  themeColor: string
  defaultLocale: string
}

export type UpdateSiteInput = {
  displayName?: string
  origins?: string[]
  moderationMode?: 'post' | 'pre'
  themeColor?: string
  defaultLocale?: string
  status?: 'active' | 'disabled'
}

export type ProviderInput = {
  name: string
  issuer: string
  clientId: string
  clientSecret?: string
  scopes: string[]
  allowedEmailDomains: string[]
  enabled: boolean
}
