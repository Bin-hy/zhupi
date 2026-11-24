import type { AuthProvider, Identity } from '../events'
import type { Messages } from '../i18n'
import type { SDKState, ThemeMode } from '../state'
import { initials, relativeTime } from '../utils/format'
import { escapeAttr, escapeHtml } from '../utils/html'

export function renderAccount(state: SDKState, msg: Messages, devLogin: boolean): string {
  if (!state.user) return renderLogin(state, msg, devLogin)
  const user = state.user
  const theme = (mode: ThemeMode, label: string) =>
    `<button class="seg${state.theme === mode ? ' active' : ''}" data-action="theme" data-theme="${mode}" type="button">${label}</button>`
  const locale = (value: 'zh-CN' | 'en', label: string) =>
    `<button class="seg${state.locale === value ? ' active' : ''}" data-action="locale" data-locale="${value}" type="button">${label}</button>`

  return `
    <section class="account">
      <div class="profile">
        ${avatar(user.displayName, user.avatarUrl)}
        <div class="profile-main">
          <strong>${escapeHtml(user.displayName || String(user.id))}</strong>
          <span class="muted">#${user.id}</span>
        </div>
        <button class="btn" data-action="logout" type="button">${msg.logout}</button>
      </div>
      ${state.siteBanned ? `<p class="notice warn">${msg.readOnlyBanned}</p>` : ''}

      <h4>${msg.theme}</h4>
      <div class="segmented">
        ${theme('auto', msg.themeAuto)}${theme('light', msg.themeLight)}${theme('dark', msg.themeDark)}
      </div>

      <h4>${msg.language}</h4>
      <div class="segmented">${locale('zh-CN', '中文')}${locale('en', 'English')}</div>

      <h4>${msg.identities}</h4>
      <div class="list">
        ${state.identities.length ? state.identities.map((i) => renderIdentity(i, msg)).join('') : `<p class="muted">${msg.identitiesEmpty}</p>`}
      </div>
      <div class="actions">
        ${(state.siteConfig?.providers || [])
          .filter((p) => !isBound(p, state.identities))
          .map(
            (p) =>
              `<button class="btn" data-action="bind" data-provider="${p.id}" type="button">${msg.bind} ${escapeHtml(
                p.type === 'github' ? 'GitHub' : p.name,
              )}</button>`,
          )
          .join('')}
      </div>

      <h4>${msg.sessions}</h4>
      ${
        state.sessionsLoading
          ? `<p class="muted">${msg.loading}</p>`
          : state.sessions.length
            ? `<div class="list">${state.sessions.map((s) => renderSession(s, msg, state.locale)).join('')}</div>`
            : `<p class="muted">${msg.sessionsEmpty}</p>`
      }
      <div class="actions">
        <button class="btn danger" data-action="revoke-all" type="button">${msg.revokeAll}</button>
      </div>
    </section>
  `
}

function renderLogin(state: SDKState, msg: Messages, devLogin: boolean): string {
  const providers = state.siteConfig?.providers || []
  const buttons = providers.length
    ? providers.map(
        (p) =>
          `<button class="btn primary block" data-action="login" data-provider="${p.id}" type="button">${escapeHtml(
            p.type === 'github' ? 'GitHub' : p.name,
          )}</button>`,
      )
    : [`<button class="btn primary block" data-action="login" data-provider="0" type="button">${msg.login}</button>`]
  return `
    <section class="account login-panel">
      <h3>${msg.loginTitle}</h3>
      <p class="muted">${msg.loginSubtitle}</p>
      <div class="actions column">${buttons.join('')}</div>
      ${devLogin ? `<button class="btn link" data-action="dev-login" type="button">${msg.localLogin}</button>` : ''}
    </section>
  `
}

function renderIdentity(identity: Identity, msg: Messages): string {
  const label = identity.provider === 'github' ? 'GitHub' : identity.issuer || identity.provider
  const detail = identity.handle || identity.email || ''
  return `<div class="list-item">
    <div><strong>${escapeHtml(label)}</strong><span class="muted"> ${escapeHtml(detail)}</span></div>
    <button class="btn link" data-action="unbind" data-id="${identity.id}" type="button">${msg.unbind}</button>
  </div>`
}

function renderSession(session: SDKState['sessions'][number], msg: Messages, locale: 'zh-CN' | 'en'): string {
  return `<div class="list-item">
    <div>
      <strong>${escapeHtml(shortUA(session.userAgent))}</strong>
      ${session.current ? `<span class="badge">${msg.currentSession}</span>` : ''}
      <div class="muted">${escapeHtml(session.ip || '-')} · ${relativeTime(session.lastSeenAt, locale)}</div>
    </div>
    <button class="btn link" data-action="revoke-session" data-id="${session.id}" type="button">${msg.revoke}</button>
  </div>`
}

function isBound(provider: AuthProvider, identities: Identity[]): boolean {
  if (provider.type === 'github') return identities.some((i) => i.provider === 'github')
  if (!provider.issuer) return false
  return identities.some((i) => i.provider === 'oidc' && i.issuer === provider.issuer)
}

function avatar(name: string, url: string): string {
  if (url) return `<img class="avatar" src="${escapeAttr(url)}" alt="">`
  return `<span class="avatar fallback">${escapeHtml(initials(name))}</span>`
}

function shortUA(ua: string): string {
  if (!ua) return 'Unknown'
  const browser = /Edg\//.test(ua) ? 'Edge' : /Chrome\//.test(ua) ? 'Chrome' : /Firefox\//.test(ua) ? 'Firefox' : /Safari\//.test(ua) ? 'Safari' : 'Browser'
  const os = /Mac OS X/.test(ua) ? 'macOS' : /Windows/.test(ua) ? 'Windows' : /Android/.test(ua) ? 'Android' : /iPhone|iPad/.test(ua) ? 'iOS' : 'Unknown OS'
  return `${browser} · ${os}`
}

export { avatar, isBound, shortUA }
