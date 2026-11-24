import { AnnotationClient, ApiError } from './client'
import { Emitter, type Annotation, type EventMap, type Identity, type SiteConfig, type User } from './events'
import { describeSelection, highlightAnnotations, selectionAllowed, sortByDocumentOrder } from './anchor'
import { AnnotationUI } from './ui/ui'
import type { ConfirmOptions } from './ui/confirm'
import {
  appendAnnotations,
  initialState,
  mergeState,
  removeAnnotation,
  upsertAnnotation,
  type Filter,
  type SDKState,
  type ThemeMode,
} from './state'
import { messages, normalizeLocale } from './i18n'
import { buildDeepLink, parseDeepLink } from './deeplink'
import {
  loadThemePreference,
  resolveTheme,
  saveThemePreference,
  systemTheme,
  watchSystemTheme,
} from './theme'

export type InitOptions = {
  siteKey: string
  pageKey?: string
  pageTitle?: string
  apiBase?: string
  container?: HTMLElement | string
  selectRoot?: HTMLElement | string
  locale?: 'zh-CN' | 'en'
  theme?: ThemeMode
  devLogin?: boolean
  cssVariables?: Record<string, string>
  on?: Partial<{ [K in keyof EventMap]: (payload: EventMap[K]) => void }>
}

const PAGE_SIZE = 50

export class AnnotationInstance {
  private readonly options: InitOptions
  private readonly client: AnnotationClient
  private readonly emitter = new Emitter()
  private readonly ui: AnnotationUI
  private selectRoot: HTMLElement
  private readonly selectRootOption: HTMLElement | string | undefined
  private readonly apiBase: string

  private state: SDKState
  private pageKey: string
  private pageTitle: string

  private savedRange: Range | null = null
  private selectedQuote = ''
  private cleanupHighlights: (() => void) | null = null
  private selectionTimer: number | undefined
  private stopThemeWatch: (() => void) | null = null
  private relocateTargetId: number | null = null
  private relocating = false
  private pendingDeepLinkId: number | null = null
  private loadSeq = 0
  private sessionsSeq = 0
  private destroyed = false

  constructor(options: InitOptions) {
    if (!options || !options.siteKey) throw new Error('zhupi: siteKey is required')
    this.options = options
    this.pageKey = normalizePageKey(options.pageKey || location.pathname)
    this.pageTitle = options.pageTitle || document.title
    this.apiBase = (options.apiBase || deriveApiBase()).replace(/\/+$/, '')
    this.client = new AnnotationClient(this.apiBase, options.siteKey)
    this.selectRootOption = options.selectRoot
    this.selectRoot = this.resolveSelectRoot()
    const container = resolveElement(options.container, document.body) as HTMLElement

    const storedTheme = loadThemePreference(this.apiBase, options.siteKey)
    const theme = options.theme || storedTheme || 'auto'
    this.state = mergeState(initialState(), {
      theme,
      locale: options.locale || 'zh-CN',
    })

    this.ui = new AnnotationUI(
      container,
      {
        onAnnotate: () => this.onAnnotate(),
        onCopyQuote: () => void this.copyQuote(),
        onLogin: (id) => this.login(id),
        onDevLogin: () => void this.devLogin(),
        onLogout: () => void this.logout(),
        onSubmit: (text) => void this.submitAnnotation(text),
        onCancelComposer: () => this.cancelComposer(),
        onReply: (id, text) => void this.reply(id, text),
        onResolve: (a) => void this.resolve(a),
        onDelete: (a) => void this.remove(a),
        onDeleteComment: (id) => void this.removeComment(id),
        onFocus: (id) => this.focus(id),
        onCopyLink: (id) => void this.copyLink(id),
        onRelocate: (id) => this.startRelocate(id),
        onCancelRelocate: () => this.cancelRelocate(),
        onFilterChange: (filter) => void this.changeFilter(filter),
        onLoadMore: () => void this.loadMore(),
        onRetry: () => void this.loadList(true),
        onToggleDrawer: (open) => this.toggleDrawer(open),
        onToggleAccount: (open) => void this.toggleAccount(open),
        onBind: (id) => void this.bind(id),
        onUnbind: (id) => void this.unbind(id),
        onRevokeSession: (id) => void this.revokeSession(id),
        onRevokeAllSessions: () => void this.revokeAllSessions(),
        onThemeChange: (mode) => this.changeTheme(mode),
        onLocaleChange: (locale) => this.changeLocale(locale),
      },
      { locale: this.state.locale, theme, devLogin: !!options.devLogin, cssVariables: options.cssVariables },
    )
    this.ui.setTheme(theme, resolveTheme(theme))
    this.stopThemeWatch = watchSystemTheme(() => {
      if (this.state.theme === 'auto') this.ui.setTheme('auto', systemTheme())
    })

    for (const [event, handler] of Object.entries(options.on || {})) {
      if (handler) this.emitter.on(event as keyof EventMap, handler as any)
    }

    document.addEventListener('mouseup', this.onPointerUp)
    document.addEventListener('touchend', this.onPointerUp)
    document.addEventListener('selectionchange', this.onSelectionChange)
    document.addEventListener('click', this.onDocumentClick)
    document.addEventListener('keydown', this.onKeydown)
  }

  async start(): Promise<void> {
    try {
      const auth = this.client.consumeAuthFromHash()
      const config = await this.client.siteConfig()
      const locale = normalizeLocale(this.options.locale || config.locale)
      this.state = mergeState(this.state, { siteConfig: config, locale })
      this.ui.setLocale(locale)
      this.ui.setAccent(config.themeColor)
      injectMarkStyles(config.themeColor)

      this.pendingDeepLinkId = parseDeepLink(location.hash)
      await this.refreshUser()
      if (this.state.user) await this.refreshIdentities()
      if (auth.loggedIn) this.emitter.emit('login', this.state.user as User)
      if (auth.bound) {
        this.ui.toast(messages(locale).bindSuccess)
        if (this.state.user) await this.refreshIdentities()
      }
      await this.loadList(true)
      this.emitter.emit('ready', { siteKey: this.options.siteKey })
    } catch (err) {
      this.handleError(err)
    }
  }

  async refresh(): Promise<void> {
    await this.loadList(true)
  }

  async setPage(page: { pageKey?: string; pageTitle?: string }): Promise<void> {
    this.pageKey = normalizePageKey(page.pageKey ?? location.pathname)
    this.pageTitle = page.pageTitle ?? document.title
    this.savedRange = null
    this.selectedQuote = ''
    this.ui.closeComposer()
    this.ui.hideToolbar()
    this.cancelRelocate()
    this.selectRoot = this.resolveSelectRoot()
    this.emitter.emit('pagechange', { pageKey: this.pageKey })
    await this.loadList(true)
  }

  login(providerId?: number): void {
    const providers = this.state.siteConfig?.providers || []
    let id = providerId
    if (id === undefined) {
      const github = providers.find((p) => p.type === 'github')
      id = github ? github.id : providers[0]?.id ?? 0
    }
    location.href = this.client.loginUrl(id, location.href.split('#')[0])
  }

  async logout(): Promise<void> {
    const msg = messages(this.state.locale)
    const ok = await this.ui.confirm({
      title: msg.confirmTitle,
      message: msg.confirmLogout,
      confirmText: msg.logout,
      cancelText: msg.cancel,
    })
    if (!ok) return
    await this.client.logout()
    this.state = mergeState(this.state, {
      user: null,
      identities: [],
      sessions: [],
      currentSessionId: null,
      sessionsLoaded: false,
      accountOpen: false,
      siteBanned: false,
    })
    this.commitState()
    this.emitter.emit('logout', undefined)
    await this.loadList(true)
  }

  getCurrentUser(): User | null {
    return this.state.user
  }

  getAnnotationCount(): number {
    return this.state.total
  }

  setTheme(mode: ThemeMode): void {
    this.changeTheme(mode)
  }

  openDrawer(): void {
    this.ui.openDrawer()
  }

  openAccount(): void {
    this.ui.openAccount()
  }

  on<K extends keyof EventMap>(event: K, handler: (payload: EventMap[K]) => void): () => void {
    return this.emitter.on(event, handler)
  }

  destroy(): void {
    if (this.destroyed) return
    this.destroyed = true
    document.removeEventListener('mouseup', this.onPointerUp)
    document.removeEventListener('touchend', this.onPointerUp)
    document.removeEventListener('selectionchange', this.onSelectionChange)
    document.removeEventListener('click', this.onDocumentClick)
    document.removeEventListener('keydown', this.onKeydown)
    if (this.selectionTimer) window.clearTimeout(this.selectionTimer)
    this.stopThemeWatch?.()
    if (this.cleanupHighlights) {
      this.cleanupHighlights()
      this.cleanupHighlights = null
    }
    this.ui.destroy()
  }

  // ---- data ----

  private async loadList(reset: boolean, limit = PAGE_SIZE): Promise<void> {
    this.selectRoot = this.resolveSelectRoot()
    const seq = ++this.loadSeq
    if (reset) {
      this.cleanup()
      this.state = mergeState(this.state, {
        annotations: [],
        total: 0,
        orphans: new Set(),
        activeId: null,
        nextCursor: 0,
        loading: true,
        loadingMore: false,
        error: '',
      })
    } else {
      if (!this.state.nextCursor || this.state.loadingMore) return
      this.state = mergeState(this.state, { loadingMore: true })
    }
    this.commitState()

    try {
      const result = await this.client.listAnnotations(
        this.pageKey,
        this.state.filter,
        reset ? 0 : this.state.nextCursor,
        limit,
      )
      if (seq !== this.loadSeq) return
      const annotations = reset
        ? result.annotations
        : appendAnnotations(this.state.annotations, result.annotations)
      this.state = mergeState(this.state, {
        annotations,
        total: result.total,
        nextCursor: result.nextCursor,
        loading: false,
        loadingMore: false,
        error: '',
      })
      await this.rehighlight()
      this.commitState()
      this.applyPendingDeepLink()
    } catch (err) {
      if (seq !== this.loadSeq) return
      const message = this.errorMessage(err)
      this.state = mergeState(this.state, { loading: false, loadingMore: false, error: message })
      this.commitState()
      this.handleError(err, false)
    }
  }

  private async rehighlight(): Promise<void> {
    if (this.cleanupHighlights) {
      this.cleanupHighlights()
      this.cleanupHighlights = null
    }
    const orphans = new Set<number>()
    this.cleanupHighlights = await highlightAnnotations(this.selectRoot, this.state.annotations, (id) =>
      orphans.add(id),
    )
    const annotations = sortByDocumentOrder(this.selectRoot, this.state.annotations)
    this.state = mergeState(this.state, { orphans, annotations })
  }

  private async changeFilter(filter: Filter): Promise<void> {
    if ((filter === 'mine' || filter === 'pending') && !this.state.user) {
      this.ui.openAccount()
      this.ui.toast(messages(this.state.locale).loginToAnnotate)
      return
    }
    this.state = mergeState(this.state, { filter })
    this.commitState()
    await this.loadList(true)
  }

  private async loadMore(): Promise<void> {
    await this.loadList(false)
  }

  private async refreshUser(): Promise<void> {
    if (!this.client.getToken()) {
      this.state = mergeState(this.state, { user: null })
      return
    }
    try {
      const user = await this.client.me()
      this.state = mergeState(this.state, { user })
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        this.client.setToken(null)
        this.state = mergeState(this.state, { user: null })
      } else {
        throw err
      }
    }
  }

  private async refreshIdentities(): Promise<void> {
    if (!this.state.user) {
      this.state = mergeState(this.state, { identities: [] })
      return
    }
    try {
      const result = await this.client.identities()
      this.state = mergeState(this.state, { identities: result.identities })
    } catch {
      this.state = mergeState(this.state, { identities: [] })
    }
    this.commitState()
  }

  private async refreshSessions(): Promise<void> {
    if (!this.state.user) return
    const seq = ++this.sessionsSeq
    this.state = mergeState(this.state, { sessionsLoading: true })
    this.commitState()
    try {
      const result = await this.client.sessions()
      if (seq !== this.sessionsSeq) return
      const sessions = result.sessions.map((s) => ({ ...s, current: s.id === result.currentSessionId }))
      this.state = mergeState(this.state, {
        sessions,
        currentSessionId: result.currentSessionId,
        sessionsLoading: false,
        sessionsLoaded: true,
      })
    } catch (err) {
      if (seq !== this.sessionsSeq) return
      this.state = mergeState(this.state, { sessionsLoading: false, sessionsLoaded: true })
      this.handleError(err)
    }
    this.commitState()
  }

  // ---- selection ----

  private onPointerUp = (event: Event): void => {
    if (this.destroyed) return
    if (this.ui.contains(event.target as Node | null)) return
    window.setTimeout(() => this.captureSelection(), 10)
  }

  private onSelectionChange = (): void => {
    if (this.destroyed) return
    if (this.selectionTimer) window.clearTimeout(this.selectionTimer)
    this.selectionTimer = window.setTimeout(() => {
      if (this.relocateTargetId) {
        const range = this.currentRange()
        if (range) void this.applyRelocation(range)
        return
      }
      this.captureSelection()
    }, 350)
  }

  private currentRange(): Range | null {
    const selection = window.getSelection()
    if (!selection || selection.isCollapsed || selection.rangeCount === 0) return null
    const range = selection.getRangeAt(0)
    if (!selectionAllowed(range, this.selectRoot)) return null
    if (selection.toString().trim().length < 2) return null
    return range
  }

  private captureSelection(): void {
    if (this.destroyed || this.ui.isComposerOpen() || this.relocateTargetId) return
    const range = this.currentRange()
    if (!range) {
      this.ui.hideToolbar()
      return
    }
    this.savedRange = range.cloneRange()
    const quote = range.toString().trim()
    this.selectedQuote = quote.length > 80 ? `${quote.slice(0, 80)}…` : quote
    const rect = range.getBoundingClientRect()
    const box = rect.width || rect.height ? rect : range.getClientRects()[0]
    if (!box) return
    this.ui.showToolbar(box, !!this.state.user)
  }

  private onDocumentClick = (event: MouseEvent): void => {
    if (this.destroyed) return
    const target = event.target as Element | null
    if (this.ui.contains(target as Node | null)) return
    const mark = target?.closest?.('mark.annot-mark') as HTMLElement | null
    if (!mark) return
    const id = Number(mark.dataset.annotationId)
    if (id) this.focus(id)
  }

  private onKeydown = (event: KeyboardEvent): void => {
    if (event.key !== 'Escape') return
    if (this.ui.isModalOpen()) return
    if (this.relocateTargetId) {
      this.cancelRelocate()
      return
    }
    this.ui.hideToolbar()
  }

  private onAnnotate(): void {
    if (!this.state.user) {
      this.ui.openAccount()
      return
    }
    if (!this.savedRange) return
    this.ui.openComposer(this.selectedQuote)
  }

  private cancelComposer(): void {
    this.savedRange = null
    this.selectedQuote = ''
    this.ui.closeComposer()
    window.getSelection()?.removeAllRanges()
  }

  private async submitAnnotation(text: string): Promise<void> {
    if (!this.savedRange) return
    const msg = messages(this.state.locale)
    this.ui.setComposerPending(true)
    this.ui.setComposerError('')
    try {
      const anchor = await describeSelection(this.savedRange, this.selectRoot)
      const created = await this.client.createAnnotation({
        pageKey: this.pageKey,
        pageTitle: this.pageTitle,
        anchor,
        content: text,
      })
      this.cancelComposer()
      await this.loadList(true, Math.min(200, Math.max(PAGE_SIZE, this.state.annotations.length)))
      this.ui.openDrawer()
      this.ui.toast(msg.createSuccess)
      this.emitter.emit('created', created)
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        this.state = mergeState(this.state, { user: null })
        this.ui.setComposerError(msg.loginExpired)
      } else {
        this.ui.setComposerError(this.errorMessage(err))
      }
    } finally {
      this.ui.setComposerPending(false)
    }
  }

  // ---- item actions ----

  private async reply(id: number, text: string): Promise<void> {
    const msg = messages(this.state.locale)
    try {
      await this.client.addComment(id, text)
      await this.loadList(true, Math.min(200, Math.max(PAGE_SIZE, this.state.annotations.length)))
      this.ui.toast(msg.replySuccess)
    } catch (err) {
      this.handleError(err)
    }
  }

  private async resolve(annotation: Annotation): Promise<void> {
    const msg = messages(this.state.locale)
    try {
      const status = annotation.status === 'resolved' ? 'open' : 'resolved'
      const updated = await this.client.updateAnnotation(annotation.id, { status })
      this.state = mergeState(this.state, { annotations: upsertAnnotation(this.state.annotations, updated) })
      this.commitState()
      this.ui.toast(msg.resolveSuccess)
      this.emitter.emit('updated', updated)
    } catch (err) {
      this.handleError(err)
    }
  }

  private async remove(annotation: Annotation): Promise<void> {
    const msg = messages(this.state.locale)
    const ok = await this.ui.confirm({
      title: msg.confirmTitle,
      message: msg.confirmDeleteAnnotation,
      confirmText: msg.delete,
      cancelText: msg.cancel,
      danger: true,
    })
    if (!ok) return
    try {
      await this.client.deleteAnnotation(annotation.id)
      this.state = mergeState(this.state, {
        annotations: removeAnnotation(this.state.annotations, annotation.id),
        activeId: null,
      })
      await this.rehighlight()
      this.commitState()
      this.ui.toast(msg.deleteSuccess)
      this.emitter.emit('deleted', { id: annotation.id })
    } catch (err) {
      this.handleError(err)
    }
  }

  private async removeComment(id: number): Promise<void> {
    const msg = messages(this.state.locale)
    const ok = await this.ui.confirm({
      title: msg.confirmTitle,
      message: msg.confirmDeleteComment,
      confirmText: msg.delete,
      cancelText: msg.cancel,
      danger: true,
    })
    if (!ok) return
    try {
      await this.client.deleteComment(id)
      await this.loadList(true, Math.min(200, Math.max(PAGE_SIZE, this.state.annotations.length)))
      this.ui.toast(msg.deleteSuccess)
    } catch (err) {
      this.handleError(err)
    }
  }

  private focus(id: number): void {
    this.state = mergeState(this.state, { activeId: id, drawerOpen: true })
    this.commitState()
    this.ui.focusCard(id)
    const mark = this.selectRoot.querySelector<HTMLElement>(`mark.annot-mark[data-annotation-id="${id}"]`)
    if (!mark) return
    mark.scrollIntoView({ behavior: 'smooth', block: 'center' })
    mark.classList.add('is-active')
    window.setTimeout(() => mark.classList.remove('is-active'), 1600)
  }

  private async copyQuote(): Promise<void> {
    const msg = messages(this.state.locale)
    if (!this.selectedQuote) return
    const ok = await copyText(this.selectedQuote)
    this.ui.toast(ok ? msg.copied : msg.copyFailed, ok ? 'success' : 'error')
  }

  private async copyLink(id: number): Promise<void> {
    const msg = messages(this.state.locale)
    const link = buildDeepLink(location.href, id)
    const ok = await copyText(link)
    this.ui.toast(ok ? msg.copied : `${msg.copyFailed}\n${link}`, ok ? 'success' : 'error')
  }

  // ---- relocate ----

  private startRelocate(id: number): void {
    const annotation = this.state.annotations.find((a) => a.id === id)
    if (!annotation) return
    this.relocateTargetId = id
    this.ui.hideToolbar()
    this.ui.closeDrawer()
    this.ui.setRelocateMode(true, annotation.quoteExact)
    this.ui.announce(messages(this.state.locale).relocateHint)
  }

  private cancelRelocate(): void {
    if (!this.relocateTargetId) return
    this.relocateTargetId = null
    this.relocating = false
    this.ui.setRelocateMode(false)
  }

  private async applyRelocation(range: Range): Promise<void> {
    if (!this.relocateTargetId || this.relocating) return
    const id = this.relocateTargetId
    const msg = messages(this.state.locale)
    this.relocating = true
    try {
      const anchor = await describeSelection(range, this.selectRoot)
      const updated = await this.client.updateAnnotation(id, { anchor })
      this.relocateTargetId = null
      this.state = mergeState(this.state, {
        annotations: upsertAnnotation(this.state.annotations, updated),
      })
      this.ui.setRelocateMode(false)
      await this.loadList(true, Math.min(200, Math.max(PAGE_SIZE, this.state.annotations.length)))
      this.ui.toast(msg.relocateSuccess)
      this.emitter.emit('updated', updated)
    } catch (err) {
      this.ui.toast(this.errorMessage(err), 'error')
      this.emitter.emit('error', err instanceof Error ? err : new Error(msg.relocateFailed))
    } finally {
      this.relocating = false
    }
  }

  // ---- account ----

  private toggleDrawer(open: boolean): void {
    this.state = mergeState(this.state, {
      drawerOpen: open,
      accountOpen: open ? this.state.accountOpen || !this.state.user : false,
    })
    this.commitState()
  }

  private async toggleAccount(open: boolean): Promise<void> {
    this.state = mergeState(this.state, { drawerOpen: true, accountOpen: open })
    this.commitState()
    if (open && this.state.user && !this.state.sessionsLoaded) await this.refreshSessions()
  }

  private async bind(providerId: number): Promise<void> {
    try {
      const url = await this.client.bindStart(providerId, location.href.split('#')[0])
      location.href = url
    } catch (err) {
      this.handleError(err)
    }
  }

  private async unbind(identityId: number): Promise<void> {
    const msg = messages(this.state.locale)
    const ok = await this.ui.confirm({
      title: msg.confirmTitle,
      message: msg.confirmUnbind,
      confirmText: msg.unbind,
      cancelText: msg.cancel,
    })
    if (!ok) return
    try {
      await this.client.unbindIdentity(identityId)
      await this.refreshIdentities()
      this.ui.toast(msg.unbindSuccess)
    } catch (err) {
      this.handleError(err)
    }
  }

  private async revokeSession(sessionId: number): Promise<void> {
    const msg = messages(this.state.locale)
    const ok = await this.ui.confirm({
      title: msg.confirmTitle,
      message: msg.confirmRevokeSession,
      confirmText: msg.revoke,
      cancelText: msg.cancel,
    })
    if (!ok) return
    try {
      await this.client.revokeSession(sessionId)
      if (sessionId === this.state.currentSessionId) {
        this.client.setToken(null)
        this.state = mergeState(this.state, {
          user: null,
          identities: [],
          sessions: [],
          currentSessionId: null,
          sessionsLoaded: false,
          accountOpen: false,
        })
        this.commitState()
        this.emitter.emit('logout', undefined)
        await this.loadList(true)
      } else {
        await this.refreshSessions()
      }
      this.ui.toast(msg.sessionRevoked)
    } catch (err) {
      this.handleError(err)
    }
  }

  private async revokeAllSessions(): Promise<void> {
    const msg = messages(this.state.locale)
    const ok = await this.ui.confirm({
      title: msg.confirmTitle,
      message: msg.confirmRevokeAll,
      confirmText: msg.revokeAll,
      cancelText: msg.cancel,
      danger: true,
    })
    if (!ok) return
    try {
      await this.client.revokeAllSessions()
      this.client.setToken(null)
      this.state = mergeState(this.state, {
        user: null,
        identities: [],
        sessions: [],
        currentSessionId: null,
        sessionsLoaded: false,
        accountOpen: false,
      })
      this.commitState()
      this.emitter.emit('logout', undefined)
      await this.loadList(true)
      this.ui.toast(msg.sessionRevoked)
    } catch (err) {
      this.handleError(err)
    }
  }

  private async devLogin(): Promise<void> {
    const name = window.prompt('本地调试用户名', '本地用户')
    if (!name || !name.trim()) return
    try {
      await this.client.devLogin(name.trim())
      await this.refreshUser()
      await this.refreshIdentities()
      await this.loadList(true)
      if (this.state.user) this.emitter.emit('login', this.state.user)
    } catch (err) {
      this.handleError(err)
    }
  }

  private changeTheme(mode: ThemeMode): void {
    this.state = mergeState(this.state, { theme: mode })
    saveThemePreference(this.apiBase, this.options.siteKey, mode)
    this.ui.setTheme(mode, resolveTheme(mode))
    this.commitState()
  }

  private changeLocale(locale: 'zh-CN' | 'en'): void {
    this.state = mergeState(this.state, { locale })
    this.ui.setLocale(locale)
    this.commitState()
  }

  // ---- misc ----

  private applyPendingDeepLink(): void {
    if (!this.pendingDeepLinkId) return
    const id = this.pendingDeepLinkId
    this.pendingDeepLinkId = null
    if (this.state.annotations.some((a) => a.id === id)) {
      this.toggleDrawer(true)
      this.focus(id)
    }
  }

  private commitState(): void {
    this.ui.setState(this.state)
    this.emitter.emit('statechange', {
      user: this.state.user,
      annotationCount: this.state.total,
      loggedIn: !!this.state.user,
    })
  }

  private resolveSelectRoot(): HTMLElement {
    const option = this.selectRootOption
    if (typeof option === 'string') {
      return document.querySelector<HTMLElement>(option) || document.body
    }
    if (option instanceof HTMLElement) {
      return option.isConnected ? option : document.body
    }
    return document.body
  }

  private cleanup(): void {
    if (this.cleanupHighlights) {
      this.cleanupHighlights()
      this.cleanupHighlights = null
    }
  }

  private errorMessage(err: unknown): string {
    const msg = messages(this.state.locale)
    if (err instanceof ApiError) {
      if (err.status === 401) return msg.loginExpired
      if (err.status === 403) return err.code === 'site_banned' ? msg.readOnlyBanned : msg.forbidden
      if (err.status === 404) return msg.notFound
      if (err.status === 429) return msg.rateLimited
      return err.message || msg.errorGeneric
    }
    if (err instanceof TypeError) return msg.networkError
    if (err instanceof Error) return err.message || msg.errorGeneric
    return msg.errorGeneric
  }

  private handleError(err: unknown, toast = true): void {
    const message = this.errorMessage(err)
    if (err instanceof ApiError) {
      if (err.status === 401) {
        this.client.setToken(null)
        this.state = mergeState(this.state, { user: null, identities: [], sessions: [] })
      }
      if (err.code === 'site_banned') {
        this.state = mergeState(this.state, { siteBanned: true })
      }
    }
    if (toast) this.ui.toast(message, 'error')
    this.commitState()
    this.emitter.emit('error', err instanceof Error ? err : new Error(message))
  }
}

export function init(options: InitOptions): AnnotationInstance {
  const instance = new AnnotationInstance(options)
  void instance.start()
  return instance
}

function deriveApiBase(): string {
  try {
    const script = document.currentScript as HTMLScriptElement | null
    if (script?.src) return new URL(script.src).origin
    const meta = import.meta as any
    if (meta?.url) return new URL(meta.url).origin
  } catch {
    /* ignore */
  }
  return location.origin
}

function resolveElement(value: HTMLElement | string | undefined, fallback: HTMLElement): HTMLElement | null {
  if (!value) return fallback
  if (typeof value === 'string') return document.querySelector<HTMLElement>(value) || fallback
  return value
}

function normalizePageKey(raw: string): string {
  try {
    return decodeURI(raw)
  } catch {
    return raw
  }
}

async function copyText(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text)
    return true
  } catch {
    try {
      const area = document.createElement('textarea')
      area.value = text
      area.style.position = 'fixed'
      area.style.opacity = '0'
      document.body.appendChild(area)
      area.select()
      const ok = document.execCommand('copy')
      area.remove()
      return ok
    } catch {
      return false
    }
  }
}

function injectMarkStyles(accent: string): void {
  const id = 'zhupi-mark-styles'
  let style = document.getElementById(id) as HTMLStyleElement | null
  if (!style) {
    style = document.createElement('style')
    style.id = id
    document.head.appendChild(style)
  }
  const safeAccent = /^#([0-9a-f]{3}|[0-9a-f]{6})$/i.test(accent) ? accent : '#087f5b'
  style.textContent = `
    mark.annot-mark {
      background: color-mix(in srgb, ${safeAccent} 22%, transparent);
      color: inherit;
      border-radius: 3px;
      padding: 0 1px;
      cursor: pointer;
      box-shadow: inset 0 -1px 0 ${safeAccent};
    }
    mark.annot-mark:hover { background: color-mix(in srgb, ${safeAccent} 36%, transparent); }
    mark.annot-mark.is-active { background: color-mix(in srgb, ${safeAccent} 55%, transparent); }
    @media (prefers-color-scheme: dark) {
      mark.annot-mark { background: color-mix(in srgb, ${safeAccent} 30%, transparent); }
    }
  `
}

export default { init }
