import { describe, expect, it } from 'vitest'
import {
  appendAnnotations,
  appendComment,
  initialState,
  lastActivityAt,
  mergeState,
  removeAnnotation,
  searchAnnotations,
  sortForDisplay,
  upsertAnnotation,
} from './state'
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

  it('appends comments without duplicating', () => {
    const comment = {
      id: 9,
      annotationId: 1,
      content: 'hello',
      moderation: 'approved',
      visibility: 'visible',
      createdAt: '2026-01-02T00:00:00Z',
      author: { id: 2, name: 'bob', avatarUrl: '' },
    }
    const once = appendComment([annotation(1)], 1, comment)
    expect(once[0].comments).toHaveLength(1)
    expect(appendComment(once, 1, comment)[0].comments).toHaveLength(1)
  })

  it('sorts newest first without mutating input', () => {
    const older = { ...annotation(1), createdAt: '2026-01-01T00:00:00Z' }
    const newer = { ...annotation(2), createdAt: '2026-01-03T00:00:00Z' }
    const list = [older, newer]
    expect(sortForDisplay(list, 'newest').map((a) => a.id)).toEqual([2, 1])
    expect(sortForDisplay(list, 'position').map((a) => a.id)).toEqual([1, 2])
    expect(list.map((a) => a.id)).toEqual([1, 2])
  })

  it('searches quote, comment and author with all terms', () => {
    const item = {
      ...annotation(1),
      quoteExact: '部署 Cloudflare Pages',
      comments: [
        {
          id: 1,
          annotationId: 1,
          content: '记得配置环境变量',
          moderation: 'approved' as const,
          visibility: 'visible',
          createdAt: '2026-01-01T00:00:00Z',
          author: { id: 2, name: 'bob', avatarUrl: '' },
        },
      ],
    }
    expect(searchAnnotations([item], 'cloudflare 环境变量')).toHaveLength(1)
    expect(searchAnnotations([item], 'bob 部署')).toHaveLength(1)
    expect(searchAnnotations([item], '不存在')).toHaveLength(0)
    expect(searchAnnotations([item], '  ')).toHaveLength(1)
  })

  it('uses latest comment time as activity', () => {
    const item = {
      ...annotation(1),
      createdAt: '2026-01-01T00:00:00Z',
      updatedAt: '2026-01-01T00:00:00Z',
      comments: [
        {
          id: 1,
          annotationId: 1,
          content: 'x',
          moderation: 'approved' as const,
          visibility: 'visible',
          createdAt: '2026-01-05T00:00:00Z',
          author: { id: 2, name: 'bob', avatarUrl: '' },
        },
      ],
    }
    expect(lastActivityAt(item)).toBe(Date.parse('2026-01-05T00:00:00Z'))
  })
})
