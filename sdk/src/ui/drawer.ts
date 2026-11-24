import type { Annotation } from '../events'
import type { Messages } from '../i18n'
import type { Filter, SDKState } from '../state'
import { relativeTime } from '../utils/format'
import { escapeAttr, escapeHtml } from '../utils/html'
import { avatar, renderAccount } from './account'

const FILTERS: Filter[] = ['all', 'mine', 'resolved', 'pending']

export class Drawer {
  readonly el: HTMLElement
  private readonly body: HTMLElement
  private readonly countEl: HTMLElement
  private readonly titleEl: HTMLElement
  private readonly accountBtn: HTMLButtonElement
  private state: SDKState | null = null

  constructor(root: ShadowRoot) {
    this.el = document.createElement('aside')
    this.el.className = 'drawer'
    this.el.hidden = true
    this.el.setAttribute('role', 'dialog')
    this.el.setAttribute('aria-label', 'Annotations')
    this.el.innerHTML = `
      <header>
        <strong class="drawer-title"></strong>
        <span class="pill drawer-count">0</span>
        <button class="btn link account-btn" data-action="account" type="button"></button>
        <button class="icon-btn" data-action="close-drawer" type="button" aria-label="close">×</button>
      </header>
      <div class="drawer-body"></div>
    `
    root.appendChild(this.el)
    this.body = this.el.querySelector('.drawer-body')!
    this.countEl = this.el.querySelector('.drawer-count')!
    this.titleEl = this.el.querySelector('.drawer-title')!
    this.accountBtn = this.el.querySelector('.account-btn')!
  }

  render(state: SDKState, msg: Messages, devLogin: boolean): void {
    this.state = state
    this.titleEl.textContent = state.accountOpen ? msg.account : msg.panelTitle
    this.countEl.textContent = String(state.annotations.length)
    this.countEl.hidden = state.accountOpen
    this.accountBtn.textContent = state.accountOpen ? msg.close : msg.account
    this.accountBtn.dataset.action = state.accountOpen ? 'close-account' : 'account'
    this.body.innerHTML = state.accountOpen ? renderAccount(state, msg, devLogin) : this.renderList(state, msg)
  }

  setVisible(visible: boolean): void {
    this.el.hidden = !visible
  }

  isOpen(): boolean {
    return !this.el.hidden
  }

  focusCard(id: number): void {
    const card = this.el.querySelector<HTMLElement>(`.card[data-card="${id}"]`)
    if (!card) return
    card.scrollIntoView({ behavior: 'smooth', block: 'center' })
    card.classList.add('flash')
    window.setTimeout(() => card.classList.remove('flash'), 1600)
  }

  query<T extends Element = HTMLElement>(selector: string): T | null {
    return this.el.querySelector(selector) as T | null
  }

  destroy(): void {
    this.el.remove()
  }

  private renderList(state: SDKState, msg: Messages): string {
    const tabs = `<div class="tabs" role="tablist">${FILTERS.map((filter) => {
      const label =
        filter === 'all'
          ? msg.filterAll
          : filter === 'mine'
            ? msg.filterMine
            : filter === 'resolved'
              ? msg.filterResolved
              : msg.filterPending
      return `<button class="tab${state.filter === filter ? ' active' : ''}" role="tab" aria-selected="${state.filter === filter}" data-action="filter" data-filter="${filter}" type="button">${label}</button>`
    }).join('')}</div>`

    if (state.loading && state.annotations.length === 0) {
      return tabs + `<div class="card skeleton"></div><div class="card skeleton"></div><div class="card skeleton"></div>`
    }
    if (state.error) {
      return tabs + `<div class="error-box"><p class="error">${escapeHtml(state.error)}</p><button class="btn" data-action="retry" type="button">${msg.retry}</button></div>`
    }
    if (state.annotations.length === 0) {
      return tabs + `<p class="empty">${msg.empty}</p>`
    }

    const cards = state.annotations.map((a) => this.renderAnnotation(a, state, msg)).join('')
    const more = state.nextCursor
      ? `<button class="btn block" data-action="load-more" type="button" ${state.loadingMore ? 'disabled' : ''}>${state.loadingMore ? msg.loadingMore : msg.loadMore}</button>`
      : ''
    return tabs + cards + more
  }

  private renderAnnotation(a: Annotation, state: SDKState, msg: Messages): string {
    const user = state.user
    const canManage = !!user && (user.isSuperadmin || user.id === a.author.id)
    const badges = [
      a.moderation === 'pending' ? `<span class="badge warn">${msg.pending}</span>` : '',
      a.moderation === 'rejected' ? `<span class="badge danger">${msg.rejected}</span>` : '',
      a.status === 'resolved' ? `<span class="badge">${msg.resolved}</span>` : '',
      state.orphans.has(a.id) ? `<span class="badge warn">${msg.orphan}</span>` : '',
    ].join('')

    const comments = a.comments
      .map((c) => {
        const canDelete = !!user && (user.isSuperadmin || user.id === c.author.id)
        return `<div class="comment">
          <div class="comment-meta">
            <strong>${escapeHtml(c.author.name)}</strong>
            <span>${relativeTime(c.createdAt, state.locale)}</span>
            ${canDelete ? `<button class="btn link" data-action="delete-comment" data-id="${c.id}" type="button">${msg.delete}</button>` : ''}
          </div>
          <p>${escapeHtml(c.content)}</p>
        </div>`
      })
      .join('')

    const hint =
      a.moderation === 'pending' ? `<p class="notice">${msg.pendingHint}</p>` : a.moderation === 'rejected' ? `<p class="notice warn">${msg.rejectedHint}</p>` : ''

    return `<article class="card${a.id === state.activeId ? ' active' : ''}${a.status === 'resolved' ? ' resolved' : ''}" data-card="${a.id}">
      <div class="meta">
        ${avatar(a.author.name, a.author.avatarUrl)}
        <strong>${escapeHtml(a.author.name)}</strong>
        <span>${relativeTime(a.createdAt, state.locale)}</span>
        ${badges}
      </div>
      <blockquote class="quote" data-action="focus" data-id="${a.id}">${escapeHtml(a.quoteExact)}</blockquote>
      ${hint}
      ${comments}
      ${
        user
          ? `<div class="reply">
              <input data-reply="${a.id}" maxlength="4000" placeholder="${msg.replyPlaceholder}" aria-label="${msg.reply}">
              <button class="btn" data-action="reply" data-id="${a.id}" type="button">${msg.reply}</button>
            </div>`
          : ''
      }
      <div class="actions">
        ${
          user
            ? `<button class="btn" data-action="resolve" data-id="${a.id}" type="button">${a.status === 'resolved' ? msg.reopen : msg.resolve}</button>`
            : ''
        }
        ${state.orphans.has(a.id) && canManage ? `<button class="btn" data-action="relocate" data-id="${a.id}" type="button">${msg.relocate}</button>` : ''}
        <button class="btn" data-action="copy-link" data-id="${a.id}" type="button">${msg.copyLink}</button>
        ${canManage ? `<button class="btn danger" data-action="delete" data-id="${a.id}" type="button">${msg.delete}</button>` : ''}
      </div>
    </article>`
  }
}

export { avatar, escapeAttr }
