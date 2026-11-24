import styles from './styles.css?inline'
import type { ThemeMode } from '../state'

export class Host {
  readonly host: HTMLElement
  readonly root: ShadowRoot

  constructor(container: HTMLElement, variables?: Record<string, string>) {
    this.host = document.createElement('div')
    this.host.className = 'zhupi-host'
    this.host.style.all = 'initial'
    this.root = this.host.attachShadow({ mode: 'open' })
    const style = document.createElement('style')
    style.textContent = styles
    this.root.appendChild(style)
    container.appendChild(this.host)
    this.setVariables(variables)
  }

  setVariables(variables?: Record<string, string>): void {
    if (!variables) return
    for (const [key, value] of Object.entries(variables)) {
      this.host.style.setProperty(key, value)
    }
  }

  setAccent(color: string): void {
    if (/^#([0-9a-f]{3}|[0-9a-f]{6})$/i.test(color)) {
      this.host.style.setProperty('--annot-accent', color)
    }
  }

  setTheme(mode: ThemeMode, resolved: 'light' | 'dark'): void {
    if (mode === 'auto') delete this.host.dataset.theme
    else this.host.dataset.theme = resolved
  }

  contains(node: Node | null): boolean {
    if (!node) return false
    return this.host === node || this.host.contains(node) || (node.getRootNode && node.getRootNode() === this.root)
  }

  destroy(): void {
    this.host.remove()
  }
}
