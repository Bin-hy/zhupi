const FOCUSABLE =
  'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'

export class FocusTrap {
  private previous: HTMLElement | null = null
  private active = false

  constructor(
    private readonly container: HTMLElement,
    private readonly onEscape?: () => void,
  ) {}

  activate(): void {
    if (this.active) return
    this.active = true
    this.previous = this.currentActive()
    document.addEventListener('keydown', this.onKeydown, true)
    const target = this.focusable()[0]
    target?.focus()
  }

  release(): void {
    if (!this.active) return
    this.active = false
    document.removeEventListener('keydown', this.onKeydown, true)
    try {
      this.previous?.focus()
    } catch {
      /* ignore */
    }
    this.previous = null
  }

  private currentActive(): HTMLElement | null {
    const root = this.container.getRootNode()
    if (typeof ShadowRoot !== 'undefined' && root instanceof ShadowRoot) {
      return root.activeElement as HTMLElement | null
    }
    return document.activeElement as HTMLElement | null
  }

  private focusable(): HTMLElement[] {
    return Array.from(this.container.querySelectorAll<HTMLElement>(FOCUSABLE)).filter((el) => {
      if (el.hasAttribute('hidden') || el.closest('[hidden]')) return false
      return !el.hasAttribute('disabled')
    })
  }

  private onKeydown = (event: KeyboardEvent): void => {
    if (!this.active) return
    if (event.key === 'Escape') {
      this.onEscape?.()
      event.preventDefault()
      return
    }
    if (event.key !== 'Tab') return

    const items = this.focusable()
    if (items.length === 0) {
      event.preventDefault()
      return
    }
    const first = items[0]
    const last = items[items.length - 1]
    const root = this.container.getRootNode()
    const active = (
      typeof ShadowRoot !== 'undefined' && root instanceof ShadowRoot ? root.activeElement : document.activeElement
    ) as HTMLElement | null
    if (event.shiftKey && (active === first || !this.container.contains(active))) {
      last.focus()
      event.preventDefault()
    } else if (!event.shiftKey && active === last) {
      first.focus()
      event.preventDefault()
    }
  }
}
