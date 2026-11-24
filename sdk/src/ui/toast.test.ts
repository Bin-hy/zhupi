import { afterEach, describe, expect, it, vi } from 'vitest'
import { Toast } from './toast'

afterEach(() => vi.useRealTimers())

describe('Toast', () => {
  it('shows message and announces', () => {
    vi.useFakeTimers()
    const root = document.createElement('div').attachShadow({ mode: 'open' })
    const toast = new Toast(root)
    toast.show('hello', 'error')
    const el = root.querySelector('.toast') as HTMLElement
    expect(el.hidden).toBe(false)
    expect(el.textContent).toBe('hello')
    expect(el.dataset.kind).toBe('error')
    vi.advanceTimersByTime(20)
    expect((root.querySelector('.sr-only') as HTMLElement).textContent).toBe('hello')
    vi.advanceTimersByTime(3000)
    expect(el.hidden).toBe(true)
    toast.destroy()
  })
})
