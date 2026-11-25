import { http } from './client'
import type {
  AdminAuthProviders,
  AdminIdentities,
  Annotation,
  Comment,
  CreateLocalAdminInput,
  CreateSiteInput,
  LocalAdminAccount,
  LocalAdminResult,
  Me,
  OpenLocalCredentialInput,
  Overview,
  Provider,
  ProviderInput,
  Queue,
  Site,
  SiteAdmin,
  SiteBanView,
  UpdateSiteInput,
  User,
} from './types'

const q = (params: Record<string, string | number | undefined>) => {
  const search = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== '') search.set(key, String(value))
  }
  const raw = search.toString()
  return raw ? `?${raw}` : ''
}

export const api = {
  // 认证
  providers: () => http.request<AdminAuthProviders>('/admin/auth/providers'),
  localLogin: (username: string, password: string) =>
    http.request<{ token: string; expiresAt: string; user: User; mustChangePassword: boolean }>(
      '/admin/auth/local/login',
      { method: 'POST', body: JSON.stringify({ username, password }) },
    ),
  me: () => http.request<Me>('/me'),
  mySites: () => http.request<{ sites: Site[] }>('/me/sites'),
  adminIdentities: () => http.request<AdminIdentities>('/admin/auth/identities'),
  bindGitHub: () =>
    http.request<{ redirectUrl: string }>('/admin/auth/identities/github/start', { method: 'POST' }),
  bindOIDC: (providerId: number) =>
    http.request<{ redirectUrl: string }>(`/admin/auth/identities/oidc/${providerId}/start`, {
      method: 'POST',
    }),
  unbindIdentity: (id: number) =>
    http.request<void>(`/me/identities/${id}`, { method: 'DELETE' }),
  changePassword: (currentPassword: string, newPassword: string) =>
    http.request<void>('/me/password', {
      method: 'POST',
      body: JSON.stringify({ currentPassword, newPassword }),
    }),
  logout: () => http.request<void>('/auth/logout', { method: 'POST' }),

  // 管理员账号
  listLocalAdmins: () => http.request<{ admins: LocalAdminAccount[] }>('/admin/local-admins'),
  createLocalAdmin: (input: CreateLocalAdminInput) =>
    http.request<LocalAdminResult>('/admin/local-admins', {
      method: 'POST',
      body: JSON.stringify(input),
    }),
  openLocalCredential: (input: OpenLocalCredentialInput) =>
    http.request<LocalAdminResult>('/admin/local-admins/open', {
      method: 'POST',
      body: JSON.stringify(input),
    }),
  resetLocalPassword: (userId: number, password: string) =>
    http.request<{ generatedPassword: string }>(`/admin/local-admins/${userId}/password/reset`, {
      method: 'POST',
      body: JSON.stringify({ password }),
    }),
  setSuperadmin: (userId: number, isSuperadmin: boolean) =>
    http.request<void>(`/admin/users/${userId}/superadmin`, {
      method: 'PUT',
      body: JSON.stringify({ isSuperadmin }),
    }),

  // 概览
  overview: () => http.request<Overview>('/admin/overview'),

  // 站点
  listSites: (cursor = 0, limit = 50) =>
    http.request<{ sites: Site[]; nextCursor: number }>(`/admin/sites${q({ cursor, limit })}`),
  createSite: (input: CreateSiteInput) =>
    http.request<Site>('/admin/sites', { method: 'POST', body: JSON.stringify(input) }),
  updateSite: (id: number, input: UpdateSiteInput) =>
    http.request<Site>(`/admin/sites/${id}`, { method: 'PATCH', body: JSON.stringify(input) }),
  rotateSiteKey: (id: number) =>
    http.request<{ siteKey: string }>(`/admin/sites/${id}/rotate-key`, { method: 'POST' }),

  // 站点管理员
  listSiteAdmins: (siteId: number) =>
    http.request<{ admins: SiteAdmin[] }>(`/admin/sites/${siteId}/admins`),
  addSiteAdmin: (siteId: number, userId: number) =>
    http.request<void>(`/admin/sites/${siteId}/admins`, {
      method: 'POST',
      body: JSON.stringify({ userId }),
    }),
  removeSiteAdmin: (siteId: number, userId: number) =>
    http.request<void>(`/admin/sites/${siteId}/admins/${userId}`, { method: 'DELETE' }),

  // 封禁
  listSiteBans: (siteId: number) =>
    http.request<{ bans: SiteBanView[] }>(`/admin/sites/${siteId}/bans`),
  banSiteUser: (siteId: number, userId: number, reason: string) =>
    http.request<void>(`/admin/sites/${siteId}/bans/${userId}`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    }),
  unbanSiteUser: (siteId: number, userId: number) =>
    http.request<void>(`/admin/sites/${siteId}/bans/${userId}`, { method: 'DELETE' }),

  // OIDC 提供商
  listProviders: (siteId: number) =>
    http.request<{ providers: Provider[] }>(`/admin/sites/${siteId}/providers`),
  createProvider: (siteId: number, input: ProviderInput) =>
    http.request<Provider>(`/admin/sites/${siteId}/providers`, {
      method: 'POST',
      body: JSON.stringify(input),
    }),
  updateProvider: (siteId: number, providerId: number, input: ProviderInput) =>
    http.request<Provider>(`/admin/sites/${siteId}/providers/${providerId}`, {
      method: 'PATCH',
      body: JSON.stringify(input),
    }),
  deleteProvider: (siteId: number, providerId: number) =>
    http.request<void>(`/admin/sites/${siteId}/providers/${providerId}`, { method: 'DELETE' }),

  // 审核
  queue: (
    siteId: number,
    status: string,
    keyword = '',
    type = 'all',
    annotationCursor = 0,
    commentCursor = 0,
    limit = 50,
  ) =>
    http.request<Queue>(
      `/admin/sites/${siteId}/queue${q({ status, q: keyword, type, annotationCursor, commentCursor, limit })}`,
    ),
  moderateAnnotation: (siteId: number, id: number, action: string) =>
    http.request<void>(`/admin/sites/${siteId}/annotations/${id}/moderate`, {
      method: 'POST',
      body: JSON.stringify({ action }),
    }),
  annotationDetail: (siteId: number, id: number) =>
    http.request<Annotation>(`/admin/sites/${siteId}/annotations/${id}`),
  moderateComment: (siteId: number, id: number, action: string) =>
    http.request<void>(`/admin/sites/${siteId}/comments/${id}/moderate`, {
      method: 'POST',
      body: JSON.stringify({ action }),
    }),

  // 用户
  listUsers: (cursor = 0, limit = 50) =>
    http.request<{ users: User[]; nextCursor: number }>(`/admin/users${q({ cursor, limit })}`),
  banUser: (id: number) => http.request<void>(`/admin/users/${id}/ban`, { method: 'POST' }),
  unbanUser: (id: number) => http.request<void>(`/admin/users/${id}/ban`, { method: 'DELETE' }),
}

export type { Comment }
