import type { Messages } from '../i18n'

export class Composer {
  readonly el: HTMLDivElement
  private readonly quoteEl: HTMLElement
  private readonly textarea: HTMLTextAreaElement
  private readonly errorEl: HTMLParagraphElement
  private readonly publishBtn: HTMLButtonElement
  private readonly cancelBtn: HTMLButtonElement
  private readonly titleEl: HTMLElement
  private readonly countEl: HTMLElement
  private readonly hintEl: HTMLElement
  private publishing = false
  private msg: Messages | null = null
  private draftKey = ''

  constructor(root: ShadowRoot, onSubmit: () => void) {
    this.el = document.createElement('div')
    this.el.className = 'composer'
    this.el.hidden = true
    this.el.setAttribute('role', 'dialog')
    this.el.setAttribute('aria-modal', 'true')
    this.el.innerHTML = `
      <div class="row">
        <strong class="composer-title"></strong>
        <button class="icon-btn" data-action="cancel" type="button" aria-label="close">×</button>
      </div>
      <blockquote class="quote"></blockquote>
      <textarea rows="4" maxlength="4000"></textarea>
      <div class="row count-row"><span class="composer-hint"></span><span class="count"></span></div>
      <div class="row end">
        <button class="btn cancel" data-action="cancel" type="button"></button>
        <button class="btn primary publish" data-action="publish" type="button"></button>
      </div>
      <p class="error" hidden></p>
    `
    root.appendChild(this.el)
    this.quoteEl = this.el.querySelector('.quote')!
    this.textarea = this.el.querySelector('textarea')!
    this.errorEl = this.el.querySelector('.error')!
    this.publishBtn = this.el.querySelector('.publish')!
    this.cancelBtn = this.el.querySelector('.cancel')!
    this.titleEl = this.el.querySelector('.composer-title')!
    this.countEl = this.el.querySelector('.count')!
    this.hintEl = this.el.querySelector('.composer-hint')!

    this.textarea.addEventListener('input', () => {
      this.updateCount()
      this.saveDraft()
    })
    this.textarea.addEventListener('keydown', (event) => {
      if (event.key === 'Enter' && (event.metaKey || event.ctrlKey)) {
        event.preventDefault()
        if (!this.publishing) onSubmit()
      }
    })
  }

  setMessages(msg: Messages): void {
    this.msg = msg
    this.titleEl.textContent = msg.composerTitle
    this.textarea.placeholder = msg.composerPlaceholder
    this.cancelBtn.textContent = msg.cancel
    this.publishBtn.textContent = this.publishing ? msg.publishing : msg.publish
    this.hintEl.textContent = msg.composerHint
  }

  setDraftKey(key: string): void {
    this.draftKey = key
  }

  open(quote: string, msg: Messages): void {
    this.setMessages(msg)
    this.quoteEl.textContent = quote
    this.textarea.value = this.loadDraft()
    this.updateCount()
    this.setError('')
    this.setPending(false)
    this.el.hidden = false
    this.textarea.focus()
  }

  close(): void {
    this.el.hidden = true
    this.setPending(false)
  }

  clearDraft(): void {
    if (!this.draftKey) return
    try {
      localStorage.removeItem(this.draftKey)
    } catch {
      /* ignore */
    }
  }

  value(): string {
    return this.textarea.value.trim()
  }

  setPending(pending: boolean): void {
    this.publishing = pending
    this.publishBtn.disabled = pending
    this.textarea.disabled = pending
    this.publishBtn.textContent = pending
      ? this.msg?.publishing ?? '…'
      : this.msg?.publish ?? '✓'
  }

  setError(message: string): void {
    this.errorEl.textContent = message
    this.errorEl.hidden = !message
  }

  isOpen(): boolean {
    return !this.el.hidden
  }

  focus(): void {
    this.textarea.focus()
  }

  private updateCount(): void {
    this.countEl.textContent = `${this.textarea.value.length}/4000`
    this.publishBtn.disabled = this.publishing || this.textarea.value.trim().length === 0
  }

  private loadDraft(): string {
    if (!this.draftKey) return ''
    try {
      return localStorage.getItem(this.draftKey) || ''
    } catch {
      return ''
    }
  }

  private saveDraft(): void {
    if (!this.draftKey) return
    try {
      const value = this.textarea.value
      if (value) localStorage.setItem(this.draftKey, value)
      else localStorage.removeItem(this.draftKey)
    } catch {
      /* ignore */
    }
  }
}
