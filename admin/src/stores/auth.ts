import { reactive } from 'vue'
import { api } from '../api'
import { http } from '../api/client'
import type { Me, Site } from '../api/types'

export const auth = reactive({
  me: null as Me | null,
  sites: [] as Site[],
  loaded: false,
})

export function hasToken(): boolean {
  return !!http.getToken()
}

export async function loadSession(): Promise<void> {
  const me = await api.me()
  const sites = await api.mySites()
  auth.me = me
  auth.sites = sites.sites
  auth.loaded = true
}

export function resetSession(): void {
  http.setToken(null)
  auth.me = null
  auth.sites = []
  auth.loaded = false
}

export function isSuperadmin(): boolean {
  return !!auth.me?.isSuperadmin
}

export function canAccessSite(id: number): boolean {
  return isSuperadmin() || auth.sites.some((site) => site.id === id)
}

function redirectStart(path: string): void {
  const returnTo = window.location.href.split('#')[0]
  window.location.href = `${path}?redirect=${encodeURIComponent(returnTo)}`
}

export function loginGitHub(): void {
  redirectStart('/api/v1/admin/auth/github/start')
}

export function loginOIDC(providerId: number): void {
  redirectStart(`/api/v1/admin/auth/oidc/${providerId}/start`)
}

export async function logout(): Promise<void> {
  try {
    await api.logout()
  } catch {
    /* 即使服务端失败也清理本地 */
  }
  resetSession()
}
