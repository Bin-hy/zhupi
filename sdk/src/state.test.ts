import { describe, expect, it } from 'vitest'
import { appendAnnotations, initialState, mergeState, removeAnnotation, upsertAnnotation } from './state'
import type { Annotation } from './events'

function annotation(id: number): Annotation {
  return {
    id,
    pageKey: '/doc',
    pageTitle: '',
    anchor: { quote: { type: 'TextQuoteSelector', exact: `text-${id}`, prefix: '', suffix: '' } },
    quoteExact: `text-${id}`,
    section: '',
    status: 'open',
    moderation: 'approved',
    visibility: 'visible',
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    author: { id: 1, name: 'alice', avatarUrl: '' },
    comments: [],
  }
}

describe('state', () => {
  it('initial state', () => {
    const s = initialState()
    expect(s.filter).toBe('all')
    expect(s.theme).toBe('auto')
    expect(s.annotations).toEqual([])
    expect(s.total).toBe(0)
  })

  it('append dedupes by id', () => {
    const out = appendAnnotations([annotation(1), annotation(2)], [annotation(2), annotation(3)])
    expect(out.map((a) => a.id)).toEqual([1, 2, 3])
  })

  it('upsert updates and appends', () => {
    const updated = { ...annotation(1), status: 'resolved' as const }
    expect(upsertAnnotation([annotation(1)], updated)[0].status).toBe('resolved')
    expect(upsertAnnotation([annotation(1)], annotation(2))).toHaveLength(2)
  })

  it('remove and merge', () => {
    expect(removeAnnotation([annotation(1), annotation(2)], 1).map((a) => a.id)).toEqual([2])
    const next = mergeState(initialState(), { filter: 'mine' })
    expect(next.filter).toBe('mine')
  })
})
