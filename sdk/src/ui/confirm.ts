import { FocusTrap } from './focus-trap'

export type ConfirmOptions = {
  title?: string
  message: string
  confirmText?: string
  cancelText?: string
  danger?: boolean
}

export class ConfirmDialog {
  private readonly el: HTMLDivElement
  private readonly titleEl: HTMLElement
  private readonly messageEl: HTMLElement
  private readonly confirmBtn: HTMLButtonElement
  private readonly cancelBtn: HTMLButtonElement
  private readonly trap: FocusTrap
  private resolver: ((value: boolean) => void) | null = null

  constructor(root: ShadowRoot) {
    this.el = document.createElement('div')
    this.el.className = 'confirm-overlay'
    this.el.hidden = true
    this.el.innerHTML = `
      <div class="confirm-dialog" role="alertdialog" aria-modal="true">
        <h3 class="confirm-title"></h3>
        <p class="confirm-message"></p>
        <div class="row end">
          <button class="btn cancel" type="button"></button>
          <button class="btn primary confirm" type="button"></button>
        </div>
      </div>
    `
    root.appendChild(this.el)
    this.titleEl = this.el.querySelector('.confirm-title')!
    this.messageEl = this.el.querySelector('.confirm-message')!
    this.confirmBtn = this.el.querySelector('.confirm')!
    this.cancelBtn = this.el.querySelector('.cancel')!
    this.trap = new FocusTrap(this.el.querySelector('.confirm-dialog') as HTMLElement, () => this.close(false))
    this.cancelBtn.addEventListener('click', () => this.close(false))
    this.confirmBtn.addEventListener('click', () => this.close(true))
  }

  open(options: ConfirmOptions): Promise<boolean> {
    this.titleEl.textContent = options.title || ''
    this.titleEl.hidden = !options.title
    this.messageEl.textContent = options.message
    this.confirmBtn.textContent = options.confirmText || 'OK'
    this.cancelBtn.textContent = options.cancelText || 'Cancel'
    this.confirmBtn.classList.toggle('danger', !!options.danger)
    this.el.hidden = false
    this.trap.activate()
    this.cancelBtn.focus()
    return new Promise<boolean>((resolve) => {
      this.resolver = resolve
    })
  }

  isOpen(): boolean {
    return !this.el.hidden
  }

  destroy(): void {
    this.trap.release()
    this.el.remove()
  }

  private close(result: boolean): void {
    if (this.el.hidden) return
    this.el.hidden = true
    this.trap.release()
    const resolve = this.resolver
    this.resolver = null
    resolve?.(result)
  }
}
