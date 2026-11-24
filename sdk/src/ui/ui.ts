import type { Messages } from '../i18n'
import { messages } from '../i18n'
import type { Annotation } from '../events'
import type { Filter, SDKState, ThemeMode } from '../state'
import { Composer } from './composer'
import { ConfirmDialog, type ConfirmOptions } from './confirm'
import { Drawer } from './drawer'
import { FocusTrap } from './focus-trap'
import { Host } from './host'
import { Toast, type ToastKind } from './toast'
import { Toolbar } from './toolbar'

export type UIHandlers = {
  onAnnotate: () => void
  onCopyQuote: () => void
  onLogin: (providerId: number) => void
  onDevLogin: () => void
  onLogout: () => void
  onSubmit: (text: string) => void
  onCancelComposer: () => void
  onReply: (id: number, text: string) => void
  onResolve: (annotation: Annotation) => void
  onDelete: (annotation: Annotation) => void
  onDeleteComment: (id: number) => void
  onFocus: (id: number) => void
  onCopyLink: (id: number) => void
  onRelocate: (id: number) => void
  onCancelRelocate: () => void
  onFilterChange: (filter: Filter) => void
  onLoadMore: () => void
  onRetry: () => void
  onToggleDrawer: (open: boolean) => void
  onToggleAccount: (open: boolean) => void
  onBind: (providerId: number) => void
  onUnbind: (identityId: number) => void
  onRevokeSession: (sessionId: number) => void
  onRevokeAllSessions: () => void
  onThemeChange: (theme: ThemeMode) => void
  onLocaleChange: (locale: 'zh-CN' | 'en') => void
}

export type UIOptions = {
  locale: 'zh-CN' | 'en'
  theme: ThemeMode
  devLogin: boolean
}

export class AnnotationUI {
  readonly host: Host
  private readonly handlers: UIHandlers
  private readonly options: UIOptions
  private readonly toolbar: Toolbar
  private readonly composer: Composer
  private readonly drawer: Drawer
  private readonly confirmDialog: ConfirmDialog
  private readonly toastView: Toast
  private readonly fab: HTMLButtonElement
  private readonly fabCount: HTMLSpanElement
  private readonly fabLabel: HTMLSpanElement
  private readonly relocateBanner: HTMLDivElement
  private readonly relocateText: HTMLSpanElement
  private readonly drawerTrap: FocusTrap
  private readonly composerTrap: FocusTrap
  private state: SDKState | null = null
  private msg: Messages
  private drawerTrapActive = false

  constructor(container: HTMLElement, handlers: UIHandlers, options: UIOptions) {
    this.handlers = handlers
    this.options = options
    this.msg = messages(options.locale)
    this.host = new Host(container)

    this.fab = document.createElement('button')
    this.fab.className = 'fab'
    this.fab.type = 'button'
    this.fab.dataset.action = 'toggle-drawer'
    this.fab.innerHTML = `<span aria-hidden="true">✎</span><span class="fab-label"></span><span class="count" hidden></span>`
    this.host.root.appendChild(this.fab)
    this.fabLabel = this.fab.querySelector('.fab-label')!
    this.fabCount = this.fab.querySelector('.count')!

    this.relocateBanner = document.createElement('div')
    this.relocateBanner.className = 'relocate-banner'
    this.relocateBanner.hidden = true
    this.relocateBanner.innerHTML = `<span class="relocate-text"></span><button class="btn" data-action="relocate-cancel" type="button"></button>`
    this.host.root.appendChild(this.relocateBanner)
    this.relocateText = this.relocateBanner.querySelector('.relocate-text')!

    this.toolbar = new Toolbar(this.host.root)
    this.composer = new Composer(this.host.root, () => this.handlers.onSubmit(this.composer.value()))
    this.drawer = new Drawer(this.host.root)
    this.confirmDialog = new ConfirmDialog(this.host.root)
    this.toastView = new Toast(this.host.root)

    this.drawerTrap = new FocusTrap(this.drawer.el, () => this.handlers.onToggleDrawer(false))
    this.composerTrap = new FocusTrap(this.composer.el, () => this.handlers.onCancelComposer())

    this.host.root.addEventListener('click', (event) => this.onClick(event))
    this.host.root.addEventListener('keydown', (event) => {
      const key = (event as KeyboardEvent).key
      if (key === 'Escape' && this.toolbar.isVisible()) {
        this.toolbar.hide()
      }
    })

    this.updateLabels()
  }

  get shadowRoot(): ShadowRoot {
    return this.host.root
  }

  setState(state: SDKState): void {
    this.state = state
    this.fabCount.textContent = state.annotations.length ? String(state.annotations.length) : ''
    this.fabCount.hidden = state.annotations.length === 0
    this.drawer.setVisible(state.drawerOpen)
    if (state.drawerOpen) {
      if (!this.drawerTrapActive) {
        this.drawerTrap.activate()
        this.drawerTrapActive = true
      }
      const focus = this.captureFocus()
      this.drawer.render(state, this.msg, this.options.devLogin)
      this.restoreFocus(focus)
    } else {
      if (this.drawerTrapActive) {
        this.drawerTrap.release()
        this.drawerTrapActive = false
      }
      this.drawer.render(state, this.msg, this.options.devLogin)
    }
  }

  setLocale(locale: 'zh-CN' | 'en'): void {
    this.msg = messages(locale)
    this.composer.setMessages(this.msg)
    this.updateLabels()
    if (this.state) this.drawer.render(this.state, this.msg, this.options.devLogin)
  }

  setTheme(mode: ThemeMode, resolved: 'light' | 'dark'): void {
    this.host.setTheme(mode, resolved)
  }

  setAccent(color: string): void {
    this.host.setAccent(color)
  }

  showToolbar(rect: { left: number; top: number; width: number; height: number }, loggedIn: boolean): void {
    this.toolbar.show(rect, loggedIn, this.msg)
  }

  hideToolbar(): void {
    this.toolbar.hide()
  }

  openComposer(quote: string): void {
    this.suspendDrawerTrap()
    this.composer.open(quote, this.msg)
    this.composerTrap.activate()
    this.composer.focus()
  }

  closeComposer(): void {
    this.composer.close()
    this.composerTrap.release()
    this.resumeDrawerTrap()
  }

  setComposerPending(pending: boolean): void {
    this.composer.setPending(pending)
  }

  setComposerError(message: string): void {
    this.composer.setError(message)
  }

  isComposerOpen(): boolean {
    return this.composer.isOpen()
  }

  openDrawer(): void {
    this.handlers.onToggleDrawer(true)
  }

  closeDrawer(): void {
    this.handlers.onToggleDrawer(false)
  }

  isDrawerOpen(): boolean {
    return this.drawer.isOpen()
  }

  isModalOpen(): boolean {
    return this.composer.isOpen() || this.confirmDialog.isOpen()
  }

  openAccount(): void {
    this.handlers.onToggleDrawer(true)
    this.handlers.onToggleAccount(true)
  }

  closeAccount(): void {
    this.handlers.onToggleAccount(false)
  }

  setRelocateMode(active: boolean, quote?: string): void {
    this.relocateBanner.hidden = !active
    this.relocateText.textContent = active
      ? quote
        ? `${this.msg.relocateHint}\n“${quote}”`
        : this.msg.relocateHint
      : ''
    this.relocateBanner.querySelector('button')!.textContent = this.msg.cancel
  }

  confirm(options: ConfirmOptions): Promise<boolean> {
    this.suspendDrawerTrap()
    return this.confirmDialog.open(options).finally(() => this.resumeDrawerTrap())
  }

  toast(message: string, kind: ToastKind = 'success'): void {
    this.toastView.show(message, kind)
  }

  announce(message: string): void {
    this.toastView.announce(message)
  }

  focusCard(id: number): void {
    this.drawer.focusCard(id)
  }

  contains(node: Node | null): boolean {
    return this.host.contains(node)
  }

  destroy(): void {
    this.drawerTrap.release()
    this.composerTrap.release()
    this.confirmDialog.destroy()
    this.toastView.destroy()
    this.toolbar.destroy()
    this.drawer.destroy()
    this.composer.el.remove()
    this.host.destroy()
  }

  private updateLabels(): void {
    this.fabLabel.textContent = this.msg.annotate
    this.fab.setAttribute('aria-label', this.msg.panelTitle)
    this.relocateBanner.querySelector('button')!.textContent = this.msg.cancel
  }

  private suspendDrawerTrap(): void {
    if (this.drawerTrapActive) {
      this.drawerTrap.release()
      this.drawerTrapActive = false
    }
  }

  private resumeDrawerTrap(): void {
    if (this.state?.drawerOpen && !this.drawerTrapActive && !this.composer.isOpen() && !this.confirmDialog.isOpen()) {
      this.drawerTrap.activate()
      this.drawerTrapActive = true
    }
  }

  private captureFocus(): string | null {
    const active = this.host.root.activeElement as HTMLElement | null
    if (!active) return null
    const keys: Array<[string, string | undefined]> = [
      ['action', active.dataset.action],
      ['id', active.dataset.id],
      ['filter', active.dataset.filter],
      ['theme', active.dataset.theme],
      ['locale', active.dataset.locale],
      ['provider', active.dataset.provider],
      ['reply', active.dataset.reply],
    ]
    const parts = keys
      .filter(([, value]) => value)
      .map(([key, value]) => `[data-${key}="${cssEscape(value!)}"]`)
    return parts.length ? parts.join('') : null
  }

  private restoreFocus(selector: string | null): void {
    if (!selector) return
    const el = this.host.root.querySelector<HTMLElement>(selector)
    el?.focus()
  }

  private onClick(event: Event): void {
    const target = (event.target as Element | null)?.closest('[data-action]') as HTMLElement | null
    if (!target) return
    const action = target.dataset.action
    const id = Number(target.dataset.id || 0)
    const providerId = Number(target.dataset.provider || 0)
    switch (action) {
      case 'toggle-drawer':
        this.handlers.onToggleDrawer(!this.drawer.isOpen())
        break
      case 'close-drawer':
        this.handlers.onToggleDrawer(false)
        break
      case 'account':
        this.handlers.onToggleDrawer(true)
        this.handlers.onToggleAccount(true)
        break
      case 'close-account':
        this.handlers.onToggleAccount(false)
        break
      case 'annotate':
        this.handlers.onAnnotate()
        this.toolbar.hide()
        break
      case 'copy-quote':
        this.handlers.onCopyQuote()
        this.toolbar.hide()
        break
      case 'publish':
        this.handlers.onSubmit(this.composer.value())
        break
      case 'cancel':
        this.handlers.onCancelComposer()
        break
      case 'filter':
        this.handlers.onFilterChange(target.dataset.filter as Filter)
        break
      case 'load-more':
        this.handlers.onLoadMore()
        break
      case 'retry':
        this.handlers.onRetry()
        break
      case 'reply': {
        const input = this.drawer.query<HTMLInputElement>(`input[data-reply="${id}"]`)
        const text = input?.value.trim() || ''
        if (text) this.handlers.onReply(id, text)
        break
      }
      case 'resolve':
        this.withAnnotation(id, (a) => this.handlers.onResolve(a))
        break
      case 'delete':
        this.withAnnotation(id, (a) => this.handlers.onDelete(a))
        break
      case 'delete-comment':
        this.handlers.onDeleteComment(id)
        break
      case 'focus':
        this.handlers.onFocus(id)
        break
      case 'copy-link':
        this.handlers.onCopyLink(id)
        break
      case 'relocate':
        this.handlers.onRelocate(id)
        break
      case 'relocate-cancel':
        this.handlers.onCancelRelocate()
        break
      case 'login':
        this.handlers.onLogin(providerId)
        break
      case 'dev-login':
        this.handlers.onDevLogin()
        break
      case 'logout':
        this.handlers.onLogout()
        break
      case 'bind':
        this.handlers.onBind(providerId)
        break
      case 'unbind':
        this.handlers.onUnbind(id)
        break
      case 'revoke-session':
        this.handlers.onRevokeSession(id)
        break
      case 'revoke-all':
        this.handlers.onRevokeAllSessions()
        break
      case 'theme':
        this.handlers.onThemeChange(target.dataset.theme as ThemeMode)
        break
      case 'locale':
        this.handlers.onLocaleChange(target.dataset.locale as 'zh-CN' | 'en')
        break
    }
  }

  private withAnnotation(id: number, callback: (annotation: Annotation) => void): void {
    const annotation = this.state?.annotations.find((a) => a.id === id)
    if (annotation) callback(annotation)
  }
}

function cssEscape(value: string): string {
  if (typeof CSS !== 'undefined' && typeof CSS.escape === 'function') return CSS.escape(value)
  return value.replace(/["\\]/g, '\\$&')
}
