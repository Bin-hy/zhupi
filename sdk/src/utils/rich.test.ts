import { describe, expect, it } from 'vitest'
import { renderRichText } from './rich'

describe('renderRichText', () => {
  it('escapes raw html', () => {
    expect(renderRichText('<img src=x onerror=alert(1)>')).toBe(
      '&lt;img src=x onerror=alert(1)&gt;',
    )
  })

  it('renders inline code, bold, italic and strikethrough', () => {
    expect(renderRichText('**重点** 与 `code` 以及 *斜体* ~~删除~~')).toBe(
      '<strong>重点</strong> 与 <code>code</code> 以及 <em>斜体</em> <del>删除</del>',
    )
  })

  it('renders markdown and bare links safely', () => {
    const html = renderRichText('见 [文档](https://example.com/a?b=1) 或 https://example.com/x。')
    expect(html).toContain('<a href="https://example.com/a?b=1" target="_blank" rel="noopener noreferrer">文档</a>')
    expect(html).toContain('<a href="https://example.com/x" target="_blank" rel="noopener noreferrer">https://example.com/x</a>')
    expect(html).toContain('。')
  })

  it('does not link non-http schemes', () => {
    expect(renderRichText('[x](javascript:alert(1))')).not.toContain('<a')
  })

  it('keeps query ampersands intact in links', () => {
    const html = renderRichText('https://example.com/a?b=1&c=2')
    expect(html).toContain('href="https://example.com/a?b=1&amp;c=2"')
  })

  it('keeps link labels escaped', () => {
    expect(renderRichText('[<b>x</b>](https://example.com)')).toContain('&lt;b&gt;x&lt;/b&gt;')
  })

  it('converts newlines to br', () => {
    expect(renderRichText('a\nb')).toBe('a<br>b')
  })

  it('does not format inside code spans', () => {
    expect(renderRichText('`**not bold**`')).toBe('<code>**not bold**</code>')
  })
})
