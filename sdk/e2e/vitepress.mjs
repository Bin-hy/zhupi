import { chromium } from 'playwright-core'

function required(name) {
  const value = process.env[name]
  if (!value) throw new Error(`missing env ${name}`)
  return value
}

const CHROME = process.env.CHROME_PATH || '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'
const API = process.env.ANNOTATION_API || 'http://127.0.0.1:18090'
const SITE_KEY = required('SITE_KEY')
const PAGE = required('PAGE')

const loginResp = await fetch(`${API}/api/v1/auth/dev/login`, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ name: 'e2e-vp-user' }),
})
const login = await loginResp.json()
if (!login.token) throw new Error('dev login failed')

const browser = await chromium.launch({ executablePath: CHROME, headless: true })
try {
  const page = await browser.newPage({ viewport: { width: 1280, height: 900 } })
  page.on('console', (msg) => {
    if (msg.type() === 'error') console.log('[console.error]', msg.text())
  })
  page.on('pageerror', (err) => console.log('[pageerror]', err.message))
  await page.addInitScript(
    ({ key, token }) => localStorage.setItem(key, token),
    { key: `annotation-sdk:${API}:${SITE_KEY}`, token: login.token },
  )

  await page.goto(PAGE, { waitUntil: 'domcontentloaded', timeout: 120000 })
  await page.waitForSelector('.vp-doc', { timeout: 120000 })
  await page.waitForSelector('.fab', { timeout: 120000 })
  console.log('1) VitePress 页面已加载 SDK')

  await page.click('.fab')
  await page.click('[data-action="account"]')
  await page.waitForFunction(
    () => document.querySelector('.annotation-sdk-host')?.shadowRoot?.textContent?.includes('e2e-vp-user'),
    null,
    { timeout: 30000 },
  )
  console.log('2) 登录态在 SDK 中生效')
  await page.click('[data-action="close-account"]')
  await page.click('[data-action="close-drawer"]')

  const selected = await page.evaluate(() => {
    const paragraphs = document.querySelectorAll('.vp-doc p')
    for (const p of paragraphs) {
      const walker = document.createTreeWalker(p, NodeFilter.SHOW_TEXT)
      let node
      while ((node = walker.nextNode())) {
        if (node.parentElement?.closest('pre,code,svg,.katex')) continue
        const text = node.textContent || ''
        if (text.trim().length >= 12) {
          const range = document.createRange()
          range.setStart(node, 0)
          range.setEnd(node, Math.min(20, text.length))
          const sel = window.getSelection()
          sel.removeAllRanges()
          sel.addRange(range)
          document.dispatchEvent(new MouseEvent('mouseup', { bubbles: true }))
          return text.slice(0, 20)
        }
      }
    }
    return null
  })
  if (!selected) throw new Error('未找到可选中正文')
  console.log('3) 已选中:', JSON.stringify(selected))

  await page.waitForSelector('.toolbar', { state: 'visible', timeout: 15000 })
  await page.click('.toolbar button')
  await page.waitForSelector('.composer', { state: 'visible' })
  await page.fill('.composer textarea', '这是 VitePress 接入测试批注')
  await page.click('.composer .publish')
  await page.waitForSelector('mark.annot-mark', { timeout: 30000 })
  console.log('4) 批注创建成功')

  await page.reload({ waitUntil: 'domcontentloaded' })
  await page.waitForSelector('.vp-doc', { timeout: 120000 })
  await page.waitForSelector('mark.annot-mark', { timeout: 60000 })
  console.log('5) 刷新后高亮恢复')

  const pageKey = decodeURIComponent(new URL(PAGE).pathname)
  const list = await (
    await fetch(`${API}/api/v1/annotations?page=${encodeURIComponent(pageKey)}`, {
      headers: { 'X-Site-Key': SITE_KEY, Origin: 'http://127.0.0.1:15174' },
    })
  ).json()
  if (!list.annotations?.some((a) => a.quoteExact.includes('Interface') || a.comments?.some((c) => c.content.includes('VitePress')))) {
    throw new Error(`API 未返回 VitePress 批注: ${JSON.stringify(list).slice(0, 500)}`)
  }
  console.log('6) API 校验通过，批注数:', list.annotations.length)
  console.log('VITEPRESS E2E PASS')
} finally {
  await browser.close()
}
