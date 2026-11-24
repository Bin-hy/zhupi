import { describe, expect, it } from 'vitest'
import { ConfirmDialog } from './confirm'

describe('ConfirmDialog', () => {
  it('resolves true on confirm and false on cancel', async () => {
    const root = document.createElement('div').attachShadow({ mode: 'open' })
    const dialog = new ConfirmDialog(root)

    const confirmed = dialog.open({ message: 'sure?', confirmText: 'yes', cancelText: 'no' })
    ;(root.querySelector('.confirm') as HTMLButtonElement).click()
    await expect(confirmed).resolves.toBe(true)

    const cancelled = dialog.open({ message: 'sure?', confirmText: 'yes', cancelText: 'no' })
    ;(root.querySelector('.cancel') as HTMLButtonElement).click()
    await expect(cancelled).resolves.toBe(false)

    dialog.destroy()
  })
})
