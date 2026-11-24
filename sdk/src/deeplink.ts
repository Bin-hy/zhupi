const PREFIX = 'annot-'

/** 从 location.hash 中解析 #annot-<id>，忽略其他参数。 */
export function parseDeepLink(hash: string): number | null {
  if (!hash) return null
  const raw = hash.replace(/^#/, '')
  for (const part of raw.split('&')) {
    if (!part.startsWith(PREFIX)) continue
    const id = Number(part.slice(PREFIX.length))
    if (Number.isInteger(id) && id > 0) return id
  }
  return null
}

/** 生成带 #annot-<id> 的链接，移除登录 token，保留其他 hash 参数。 */
export function buildDeepLink(href: string, id: number): string {
  const url = new URL(href)
  const params = new URLSearchParams(url.hash.replace(/^#/, ''))
  params.delete('annot_token')
  params.delete('annot_bind')
  const kept: string[] = []
  params.forEach((value, key) => {
    if (!key.startsWith(PREFIX)) kept.push(`${key}=${value}`)
  })
  kept.push(`${PREFIX}${id}`)
  url.hash = kept.join('&')
  return url.toString()
}
