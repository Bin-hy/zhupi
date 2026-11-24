import { describe, expect, it, vi } from 'vitest'
import { FocusTrap } from './focus-trap'

describe('FocusTrap', () => {
  it('wraps tab focus and handles escape', () => {
    document.body.innerHTML = '<div id="c"><button id="a">a</button><button id="b">b</button></div>'
    const container = document.getElementById('c') as HTMLElement
    const a = document.getElementById('a') as HTMLButtonElement
    const b = document.getElementById('b') as HTMLButtonElement
    const onEscape = vi.fn()
    const trap = new FocusTrap(container, onEscape)

    trap.activate()
    expect(document.activeElement).toBe(a)

    b.focus()
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', bubbles: true }))
    expect(document.activeElement).toBe(a)

    a.focus()
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', shiftKey: true, bubbles: true }))
    expect(document.activeElement).toBe(b)

    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    expect(onEscape).toHaveBeenCalled()
    trap.release()
  })
})
