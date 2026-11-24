import type { Messages } from '../i18n'

export class Toolbar {
  readonly el: HTMLDivElement

  constructor(root: ShadowRoot) {
    this.el = document.createElement('div')
    this.el.className = 'toolbar'
    this.el.hidden = true
    this.el.setAttribute('role', 'toolbar')
    this.el.innerHTML = `
      <button class="btn primary" data-action="annotate" type="button"></button>
      <button class="btn" data-action="copy-quote" type="button"></button>
    `
    root.appendChild(this.el)
  }

  show(rect: { left: number; top: number; width: number; height: number }, loggedIn: boolean, msg: Messages): void {
    const annotate = this.el.querySelector<HTMLButtonElement>('[data-action="annotate"]')!
    annotate.textContent = loggedIn ? msg.annotate : msg.loginToAnnotate
    this.el.querySelector<HTMLButtonElement>('[data-action="copy-quote"]')!.textContent = msg.copyQuote

    const x = Math.min(Math.max(rect.left + rect.width / 2, 90), window.innerWidth - 90)
    const y = Math.max(8, rect.top - 10)
    this.el.style.left = `${x}px`
    this.el.style.top = `${y}px`
    this.el.hidden = false
  }

  hide(): void {
    this.el.hidden = true
  }

  isVisible(): boolean {
    return !this.el.hidden
  }

  destroy(): void {
    this.el.remove()
  }
}
