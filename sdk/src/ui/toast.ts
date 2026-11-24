export type ToastKind = 'success' | 'error'

export class Toast {
  private readonly el: HTMLDivElement
  private readonly live: HTMLDivElement
  private timer: number | undefined

  constructor(root: ShadowRoot) {
    this.el = document.createElement('div')
    this.el.className = 'toast'
    this.el.hidden = true
    this.el.setAttribute('role', 'status')
    root.appendChild(this.el)

    this.live = document.createElement('div')
    this.live.className = 'sr-only'
    this.live.setAttribute('aria-live', 'polite')
    this.live.setAttribute('aria-atomic', 'true')
    root.appendChild(this.live)
  }

  show(message: string, kind: ToastKind = 'success'): void {
    this.el.textContent = message
    this.el.dataset.kind = kind
    this.el.hidden = false
    this.announce(message)
    if (this.timer) window.clearTimeout(this.timer)
    this.timer = window.setTimeout(() => {
      this.el.hidden = true
    }, 2800)
  }

  announce(message: string): void {
    this.live.textContent = ''
    window.setTimeout(() => {
      this.live.textContent = message
    }, 10)
  }

  destroy(): void {
    if (this.timer) window.clearTimeout(this.timer)
    this.el.remove()
    this.live.remove()
  }
}
