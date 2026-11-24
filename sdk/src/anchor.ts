import type { Anchor, Annotation } from './events'

const EXCLUDE_SELECTOR = 'pre, svg, .mermaid, .katex, .annotation-sdk-host, script, style'

function closestExcluded(node: Node | null): Element | null {
  if (!node) return null
  const el = node.nodeType === Node.ELEMENT_NODE ? (node as Element) : node.parentElement
  return el ? el.closest(EXCLUDE_SELECTOR) : null
}

export function selectionAllowed(range: Range, scope: HTMLElement): boolean {
  if (!scope.contains(range.commonAncestorContainer)) return false
  if (closestExcluded(range.startContainer) || closestExcluded(range.endContainer)) return false
  try {
    const holder = document.createElement('div')
    holder.appendChild(range.cloneContents())
    if (holder.querySelector(EXCLUDE_SELECTOR)) return false
  } catch {
    return false
  }
  return true
}

function collectTextNodes(scope: HTMLElement): Text[] {
  const nodes: Text[] = []
  const walker = document.createTreeWalker(scope, NodeFilter.SHOW_TEXT)
  let current = walker.nextNode()
  while (current) {
    const parent = (current as Text).parentElement
    if (!parent || !parent.closest('script, style')) nodes.push(current as Text)
    current = walker.nextNode()
  }
  return nodes
}

function positionOfRange(range: Range, scope: HTMLElement): { start: number; end: number } | undefined {
  if (range.startContainer.nodeType !== Node.TEXT_NODE || range.endContainer.nodeType !== Node.TEXT_NODE) {
    return undefined
  }
  let start = -1
  let end = -1
  let offset = 0
  for (const node of collectTextNodes(scope)) {
    if (node === range.startContainer) start = offset + range.startOffset
    if (node === range.endContainer) end = offset + range.endOffset
    offset += node.data.length
    if (start >= 0 && end >= 0) break
  }
  if (start < 0 || end < 0) return undefined
  return { start, end }
}

function findSection(scope: HTMLElement, range: Range): string {
  const headings = Array.from(scope.querySelectorAll<HTMLElement>('h1, h2, h3, h4'))
  let label = ''
  for (const heading of headings) {
    const relation = heading.compareDocumentPosition(range.startContainer)
    if (relation & Node.DOCUMENT_POSITION_FOLLOWING) label = (heading.textContent || '').trim()
    else break
  }
  return label.slice(0, 200)
}

export async function describeSelection(range: Range, scope: HTMLElement): Promise<Anchor> {
  const dq: any = await import('dom-anchor-text-quote')
  const quote = dq.fromRange(scope, range)
  return {
    quote: {
      type: 'TextQuoteSelector',
      exact: quote.exact,
      prefix: quote.prefix || '',
      suffix: quote.suffix || '',
    },
    position: positionOfRange(range, scope),
    section: findSection(scope, range),
  }
}

export async function highlightAnnotations(
  scope: HTMLElement,
  list: Annotation[],
  onOrphan: (id: number) => void,
): Promise<() => void> {
  const cleanups: Array<() => void> = []
  if (list.length === 0) return () => {}

  const [{ highlightText }, dq]: any[] = await Promise.all([
    import('@apache-annotator/dom'),
    import('dom-anchor-text-quote'),
  ])

  for (const annotation of list) {
    const quote = annotation.anchor?.quote
    if (!quote?.exact) {
      onOrphan(annotation.id)
      continue
    }
    const selector: Record<string, string> = { exact: quote.exact }
    if (quote.prefix) selector.prefix = quote.prefix
    if (quote.suffix) selector.suffix = quote.suffix

    let range: Range | null = null
    try {
      range = dq.toRange(scope, selector)
    } catch (err) {
      console.warn('[annotation-sdk] anchor match failed', annotation.id, err)
    }
    if (!range) {
      onOrphan(annotation.id)
      continue
    }
    try {
      cleanups.push(
        highlightText(range, 'mark', {
          class: 'annot-mark',
          'data-annotation-id': String(annotation.id),
        }),
      )
    } catch (err) {
      console.warn('[annotation-sdk] highlight failed', annotation.id, err)
      onOrphan(annotation.id)
    }
  }

  return () => {
    for (const cleanup of cleanups) {
      try {
        cleanup()
      } catch {
        /* ignore */
      }
    }
    try {
      scope.normalize()
    } catch {
      /* ignore */
    }
  }
}

export function sortByDocumentOrder(scope: HTMLElement, list: Annotation[]): Annotation[] {
  const order = new Map<number, number>()
  scope.querySelectorAll<HTMLElement>('mark.annot-mark').forEach((mark, index) => {
    const id = Number(mark.dataset.annotationId)
    if (id && !order.has(id)) order.set(id, index)
  })
  return [...list].sort((a, b) => {
    const ai = order.get(a.id) ?? Number.MAX_SAFE_INTEGER
    const bi = order.get(b.id) ?? Number.MAX_SAFE_INTEGER
    return ai - bi
  })
}
