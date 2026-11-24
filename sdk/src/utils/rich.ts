import { escapeAttr, escapeHtml } from './html'

const TOKEN = '\u0000'

/**
 * 渲染批注/评论的轻量富文本：支持行内代码、粗体、斜体、删除线、链接与换行。
 * 先转义全部 HTML，再只放行白名单语法，因此不会执行用户输入的标签或脚本。
 */
export function renderRichText(raw: string): string {
  const tokens: string[] = []
  const stash = (html: string): string => `${TOKEN}${tokens.push(html) - 1}${TOKEN}`

  let text = escapeHtml(String(raw).replace(/\u0000/g, ''))

  text = text.replace(/`([^`\n]+)`/g, (_, code: string) => stash(`<code>${code}</code>`))

  text = text.replace(/\[([^\]\n]+)\]\((https?:\/\/[^\s)]+)\)/g, (_, label: string, url: string) =>
    stash(link(url, label)),
  )

  text = text.replace(/\bhttps?:\/\/[^\s<>"']+/g, (rawUrl: string) => {
    const trailing = rawUrl.match(/[.,;:!?)\]}，。；：！？、】》」』]+$/)?.[0] ?? ''
    const url = trailing ? rawUrl.slice(0, -trailing.length) : rawUrl
    return `${stash(link(url, url))}${trailing}`
  })

  text = text
    .replace(/\*\*([^*\n]+)\*\*/g, '<strong>$1</strong>')
    .replace(/~~([^~\n]+)~~/g, '<del>$1</del>')
    .replace(/(^|[^*])\*([^*\n]+)\*/g, '$1<em>$2</em>')
    .replace(/\n/g, '<br>')

  return text.replace(/\u0000(\d+)\u0000/g, (_, index: string) => tokens[Number(index)] ?? '')
}

function link(url: string, label: string): string {
  const decoded = decodeEntities(url)
  const safe = /^https?:\/\//i.test(decoded) ? decoded : '#'
  return `<a href="${escapeAttr(safe)}" target="_blank" rel="noopener noreferrer">${label}</a>`
}

/** 文本先整体转义过，构造 href 前需要还原实体，避免 & 被二次转义。 */
function decodeEntities(value: string): string {
  return value
    .replace(/&amp;/g, '&')
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'")
}
