import { describe, expect, it } from 'vitest'
import { describeSelection, highlightAnnotations } from './anchor'
import type { Annotation } from './events'

describe('anchor', () => {
  it('describes a selection and restores highlight', async () => {
    document.body.innerHTML = '<main><p>这是第一段可划线的正文，用于锚点测试。</p></main>'
    const root = document.querySelector('main') as HTMLElement
    const paragraph = root.querySelector('p') as HTMLElement
    const textNode = paragraph.firstChild as Text

    const range = document.createRange()
    range.setStart(textNode, 0)
    range.setEnd(textNode, 10)

    const anchor = await describeSelection(range, root)
    expect(anchor.quote.exact).toBe(textNode.data.slice(0, 10))
    expect(anchor.section).toBe('')

    const annotation: Annotation = {
      id: 1,
      pageKey: '/doc',
      pageTitle: '',
      anchor,
      quoteExact: anchor.quote.exact,
      section: '',
      status: 'open',
      moderation: 'approved',
      visibility: 'visible',
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
      author: { id: 1, name: 'alice', avatarUrl: '' },
      comments: [],
    }

    const orphans: number[] = []
    const cleanup = await highlightAnnotations(root, [annotation], (id) => orphans.push(id))
    expect(orphans).toEqual([])
    expect(root.querySelector('mark.annot-mark')?.textContent).toBe(anchor.quote.exact)

    cleanup()
    expect(root.querySelector('mark.annot-mark')).toBeNull()
  })
})
