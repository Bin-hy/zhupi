import type { Annotation } from '../events'
import type { Messages } from '../i18n'
import type { Filter, SDKState } from '../state'
import { searchAnnotations, sortForDisplay } from '../state'
import { absoluteTime, relativeTime } from '../utils/format'
import { escapeAttr, escapeHtml } from '../utils/html'
import { renderRichText } from '../utils/rich'
import { avatar, renderAccount } from './account'

const FILTERS: Filter[] = ['all', 'mine', 'resolved', 'pending']

export class Drawer {
  readonly el: HTMLElement
  private readonly body: HTMLElement
  private readonly countEl: HTMLElement
  private readonly titleEl: HTMLElement
  private readonly accountBtn: HTMLButtonElement
  private readonly tools: HTMLElement
  private readonly searchInput: HTMLInputElement
  private readonly sortSelect: HTMLSelectElement
  private readonly tabsEl: HTMLElement
  private readonly navEl: HTMLElement
  private readonly navPos: HTMLElement
  private readonly prevBtn: HTMLButtonElement
  private readonly nextBtn: HTMLButtonElement
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
        <div class="drawer-nav">
          <button class="icon-btn nav-btn" data-action="prev-annotation" type="button">‹</button>
          <span class="nav-pos"></span>
          <button class="icon-btn nav-btn" data-action="next-annotation" type="button">›</button>
        </div>
        <button class="btn link account-btn" data-action="account" type="button"></button>
        <button class="icon-btn close-btn" data-action="close-drawer" type="button" aria-label="close">×</button>
      </header>
      <div class="drawer-tools">
        <div class="search-row">
          <input class="search-input" data-role="search" type="search" autocomplete="off" />
          <select class="sort-select" data-role="sort">
            <option value="position"></option>
            <option value="newest"></option>
          </select>
        </div>
        <div class="tabs" role="tablist"></div>
      </div>
      <div class="drawer-body"></div>
    `
    root.appendChild(this.el)
    this.body = this.el.querySelector('.drawer-body')!
    this.countEl = this.el.querySelector('.drawer-count')!
    this.titleEl = this.el.querySelector('.drawer-title')!
    this.accountBtn = this.el.querySelector('.account-btn')!
    this.tools = this.el.querySelector('.drawer-tools')!
    this.searchInput = this.el.querySelector('.search-input')!
    this.sortSelect = this.el.querySelector('.sort-select')!
    this.tabsEl = this.el.querySelector('.tabs')!
    this.navEl = this.el.querySelector('.drawer-nav')!
    this.navPos = this.el.querySelector('.nav-pos')!
    this.prevBtn = this.el.querySelector('[data-action="prev-annotation"]')!
    this.nextBtn = this.el.querySelector('[data-action="next-annotation"]')!
  }

  render(state: SDKState, msg: Messages, devLogin: boolean): void {
    this.state = state
    this.titleEl.textContent = state.accountOpen ? msg.account : msg.panelTitle
    this.countEl.textContent = String(state.total)
    this.countEl.hidden = state.accountOpen
    this.accountBtn.textContent = state.accountOpen ? msg.close : msg.account
    this.accountBtn.dataset.action = state.accountOpen ? 'close-account' : 'account'
    this.tools.hidden = state.accountOpen
    this.navEl.hidden = state.accountOpen

    if (state.accountOpen) {
      this.body.innerHTML = renderAccount(state, msg, devLogin)
      return
    }

    this.searchInput.placeholder = msg.searchPlaceholder
    this.searchInput.setAttribute('aria-label', msg.searchPlaceholder)
    if (this.searchInput.value !== state.search) this.searchInput.value = state.search
    this.sortSelect.setAttribute('aria-label', msg.sortLabel)
    this.sortSelect.options[0].textContent = msg.sortPosition
    this.sortSelect.options[1].textContent = msg.sortNewest
    this.sortSelect.value = state.sort

    this.tabsEl.innerHTML = FILTERS.map((filter) => {
      const label =
        filter === 'all'
          ? msg.filterAll
          : filter === 'mine'
            ? msg.filterMine
            : filter === 'resolved'
              ? msg.filterResolved
              : msg.filterPending
      const count = state.counts[filter] ?? 0
      return `<button class="tab${state.filter === filter ? ' active' : ''}" role="tab" aria-selected="${state.filter === filter}" data-action="filter" data-filter="${filter}" type="button">${label}<span class="tab-count">${count}</span></button>`
    }).join('')

    const list = this.visible(state)
    const index = list.findIndex((a) => a.id === state.activeId)
    this.prevBtn.disabled = list.length === 0
    this.nextBtn.disabled = list.length === 0
    this.prevBtn.setAttribute('aria-label', msg.prevAnnotation)
    this.nextBtn.setAttribute('aria-label', msg.nextAnnotation)
    this.navPos.textContent = list.length ? `${index >= 0 ? index + 1 : 0}/${list.length}` : ''

    this.body.innerHTML = this.renderList(state, msg, list)
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

  focusSearch(): void {
    this.searchInput.focus()
    this.searchInput.select()
  }

  destroy(): void {
    this.el.remove()
  }

  private visible(state: SDKState): Annotation[] {
    return searchAnnotations(sortForDisplay(state.annotations, state.sort), state.search)
  }

  private renderList(state: SDKState, msg: Messages, list: Annotation[]): string {
    if (state.loading && state.annotations.length === 0) {
      return `<div class="card skeleton"></div><div class="card skeleton"></div><div class="card skeleton"></div>`
    }
    if (state.error) {
      return `<div class="error-box"><p class="error">${escapeHtml(state.error)}</p><button class="btn" data-action="retry" type="button">${msg.retry}</button></div>`
    }
    if (list.length === 0) return this.renderEmpty(state, msg)

    const cards = list.map((a) => this.renderAnnotation(a, state, msg)).join('')
    const more = state.nextCursor
      ? `<button class="btn block" data-action="load-more" type="button" ${state.loadingMore ? 'disabled' : ''}>${state.loadingMore ? msg.loadingMore : msg.loadMore}</button>`
      : ''
    return cards + more
  }

  private renderEmpty(state: SDKState, msg: Messages): string {
    if (state.search.trim()) {
      const more = state.nextCursor
        ? `<button class="btn" data-action="load-more" type="button" ${state.loadingMore ? 'disabled' : ''}>${state.loadingMore ? msg.loadingMore : msg.loadMore}</button>`
        : ''
      return `<div class="empty-state">
        <p>${msg.searchEmpty}</p>
        <p class="muted">${msg.searchEmptyHint}</p>
        <button class="btn" data-action="clear-search" type="button">${msg.clearSearch}</button>
        ${more}
      </div>`
    }
    const text =
      state.filter === 'mine'
        ? msg.emptyMine
        : state.filter === 'resolved'
          ? msg.emptyResolved
          : state.filter === 'pending'
            ? msg.emptyPending
            : msg.empty
    return `<div class="empty-state"><p>${text}</p></div>`
  }

  private renderAnnotation(a: Annotation, state: SDKState, msg: Messages): string {
    const user = state.user
    const canManage = !!user && (user.isSuperadmin || user.id === a.author.id)
    const badges = [
      state.unreadIds.has(a.id) ? `<span class="badge new">${msg.newBadge}</span>` : '',
      a.moderation === 'pending' ? `<span class="badge warn">${msg.pending}</span>` : '',
      a.moderation === 'rejected' ? `<span class="badge danger">${msg.rejected}</span>` : '',
      a.status === 'resolved' ? `<span class="badge">${msg.resolved}</span>` : '',
      state.orphans.has(a.id) ? `<span class="badge warn">${msg.orphan}</span>` : '',
    ].join('')

    const comments = a.comments
      .map((c) => {
        const canDelete = !!user && (user.isSuperadmin || user.id === c.author.id)
        const moderation =
          c.moderation === 'pending'
            ? `<span class="badge warn">${msg.pending}</span>`
            : c.moderation === 'rejected'
              ? `<span class="badge danger">${msg.rejected}</span>`
              : ''
        return `<div class="comment">
          <div class="comment-head">
            ${avatar(c.author.name, c.author.avatarUrl)}
            <div class="comment-meta">
              <strong>${escapeHtml(c.author.name)}</strong>
              <span title="${escapeAttr(absoluteTime(c.createdAt, state.locale))}">${relativeTime(c.createdAt, state.locale)}</span>
              ${moderation}
            </div>
            <div class="comment-tools">
              ${user ? `<button class="btn link" data-action="reply-to" data-id="${a.id}" data-name="${escapeAttr(c.author.name)}" type="button">${msg.reply}</button>` : ''}
              ${canDelete ? `<button class="btn link" data-action="delete-comment" data-id="${c.id}" type="button">${msg.delete}</button>` : ''}
            </div>
          </div>
          <div class="comment-body">${renderRichText(c.content)}</div>
        </div>`
      })
      .join('')

    const hint =
      a.moderation === 'pending'
        ? `<p class="notice">${msg.pendingHint}</p>`
        : a.moderation === 'rejected'
          ? `<p class="notice warn">${msg.rejectedHint}</p>`
          : ''

    return `<article class="card${a.id === state.activeId ? ' active' : ''}${a.status === 'resolved' ? ' resolved' : ''}" data-card="${a.id}">
      <div class="meta">
        ${avatar(a.author.name, a.author.avatarUrl)}
        <strong>${escapeHtml(a.author.name)}</strong>
        <span title="${escapeAttr(absoluteTime(a.createdAt, state.locale))}">${relativeTime(a.createdAt, state.locale)}</span>
        ${badges}
      </div>
      <blockquote class="quote" data-action="focus" data-id="${a.id}" title="${escapeAttr(msg.jumpToText)}">${escapeHtml(a.quoteExact)}</blockquote>
      ${hint}
      ${comments}
      ${
        user
          ? `<div class="reply">
              <textarea data-reply="${a.id}" rows="1" maxlength="4000" placeholder="${escapeAttr(msg.replyPlaceholder)}" aria-label="${escapeAttr(msg.reply)}"></textarea>
              <button class="btn primary" data-action="reply" data-id="${a.id}" type="button">${msg.reply}</button>
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
        <button class="btn" data-action="copy-quote-text" data-id="${a.id}" type="button">${msg.copyQuote}</button>
        <button class="btn" data-action="copy-link" data-id="${a.id}" type="button">${msg.copyLink}</button>
        ${canManage ? `<button class="btn danger" data-action="delete" data-id="${a.id}" type="button">${msg.delete}</button>` : ''}
      </div>
    </article>`
  }
}

export { avatar, escapeAttr }
