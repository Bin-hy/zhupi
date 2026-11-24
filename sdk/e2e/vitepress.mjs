import { chromium } from 'playwright-core'

function required(name) {
  const value = process.env[name]
  if (!value) throw new Error(`missing env ${name}`)
  return value
}

const CHROME =
  process.env.CHROME_PATH || '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'
const API = (process.env.ANNOTATION_API || 'http://127.0.0.1:18090').replace(/\/+$/, '')
const SITE_KEY = required('SITE_KEY')
const PAGE = required('PAGE')
const ORIGIN = new URL(PAGE).origin
const TOKEN_KEY = `annotation-sdk:${API}:${SITE_KEY}`

async function devLogin(name) {
  const res = await fetch(`${API}/api/v1/auth/dev/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name }),
  })
  return res.json()
}

async function pageTotal(pageKey) {
  const res = await fetch(`${API}/api/v1/annotations?page=${encodeURIComponent(pageKey)}&limit=1`, {
    headers: { 'X-Site-Key': SITE_KEY, Origin: ORIGIN },
  })
  return (await res.json()).total
}

async function navCount(page) {
  const el = page.locator('.annotation-nav-count')
  if ((await el.count()) === 0) return 0
  return Number((await el.innerText()).trim())
}

const login = await devLogin('e2e-vp-user')
if (!login.token) throw new Error('dev login failed')

const browser = await chromium.launch({ executablePath: CHROME, headless: true })
try {
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 } })
  await context.addInitScript(({ key, token }) => localStorage.setItem(key, token), {
    key: TOKEN_KEY,
    token: login.token,
  })
  const page = await context.newPage()
  page.on('console', (msg) => {
    if (msg.type() === 'error') console.log('[console.error]', msg.text())
  })
  page.on('pageerror', (err) => console.log('[pageerror]', err.message))

  await page.goto(PAGE, { waitUntil: 'domcontentloaded', timeout: 120000 })
  await page.waitForSelector('.vp-doc', { timeout: 120000 })
  await page.waitForSelector('.annotation-nav', { timeout: 120000 })
  await page.waitForSelector('.annotation-nav-user', { timeout: 60000 })
  console.log('1) 顶部导航入口已渲染')

  let pageKey = decodeURIComponent(new URL(PAGE).pathname)
  let total = await pageTotal(pageKey)
  let count = await navCount(page)
  if (count !== total) throw new Error(`nav count ${count} != total ${total}`)
  console.log('2) 导航计数与页面总数一致:', count)

  await page.click('.annotation-nav-user')
  await page.waitForFunction(
    () =>
      document
        .querySelector('.annotation-sdk-host')
        ?.shadowRoot?.textContent?.includes('e2e-vp-user'),
    null,
    { timeout: 30000 },
  )
  const drawerTop = await page.evaluate(() => {
    const el = document.querySelector('.annotation-sdk-host')?.shadowRoot?.querySelector('.drawer')
    return el ? Math.round(el.getBoundingClientRect().top) : 0
  })
  if (drawerTop < 40) throw new Error(`drawer should start below nav, got ${drawerTop}`)
  console.log('3) 账号面板与登录态生效，抽屉 top:', drawerTop)
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
        const start = text.search(/\S/)
        if (start >= 0 && text.trim().length >= 12) {
          const range = document.createRange()
          range.setStart(node, start)
          range.setEnd(node, start + 12)
          const sel = window.getSelection()
          sel.removeAllRanges()
          sel.addRange(range)
          document.dispatchEvent(new MouseEvent('mouseup', { bubbles: true }))
          return text.slice(start, start + 12)
        }
      }
    }
    return null
  })
  if (!selected) throw new Error('未找到可选中正文')
  await page.waitForSelector('.toolbar', { state: 'visible', timeout: 15000 })
  await page.click('.toolbar button[data-action="annotate"]')
  await page.waitForSelector('.composer', { state: 'visible' })
  await page.fill('.composer textarea', 'VitePress 顶部导航接入测试')
  await page.click('.composer .publish')
  await page.waitForSelector('mark.annot-mark', { timeout: 30000 })
  await page.waitForTimeout(600)
  const newCount = await navCount(page)
  if (newCount !== total + 1) throw new Error(`nav count after create ${newCount} != ${total + 1}`)
  console.log('4) 创建批注后导航计数 +1:', newCount)

  const beforeTheme = await page.evaluate(
    () => document.querySelector('.annotation-sdk-host')?.dataset.theme,
  )
  await page.click('.VPSwitchAppearance')
  await page.waitForTimeout(300)
  const afterTheme = await page.evaluate(
    () => document.querySelector('.annotation-sdk-host')?.dataset.theme,
  )
  if (!afterTheme || afterTheme === beforeTheme) {
    throw new Error(`dark sync failed: ${beforeTheme} -> ${afterTheme}`)
  }
  console.log('5) 暗色同步:', beforeTheme, '->', afterTheme)

  const currentPath = new URL(page.url()).pathname
  const links = await page.$$eval('.VPSidebarItem a[href]', (els) =>
    els.map((el) => el.getAttribute('href')).filter(Boolean),
  )
  const target = links.find((href) => decodeURIComponent(href) !== decodeURIComponent(currentPath))
  if (!target) throw new Error('no sibling doc link found')
  await page.click(`.VPSidebarItem a[href="${target}"]`)
  await page.waitForFunction(
    (oldPath) => decodeURIComponent(location.pathname) !== decodeURIComponent(oldPath),
    currentPath,
    { timeout: 60000 },
  )
  await page.waitForSelector('.vp-doc', { timeout: 60000 })
  await page.waitForTimeout(800)
  const hosts = await page.locator('.annotation-sdk-host').count()
  if (hosts !== 1) throw new Error(`expected 1 SDK host after setPage, got ${hosts}`)
  pageKey = decodeURIComponent(new URL(page.url()).pathname)
  total = await pageTotal(pageKey)
  count = await navCount(page)
  if (count !== total) throw new Error(`nav count after route change ${count} != ${total}`)
  console.log('6) 文档页切换（setPage）正常，实例数:', hosts)

  await page.click('.VPNavBarTitle a')
  await page.waitForSelector('.annotation-nav', { state: 'detached', timeout: 60000 })
  if ((await page.locator('.annotation-sdk-host').count()) !== 0) {
    throw new Error('SDK host not removed on home')
  }
  console.log('7) 首页隐藏入口并销毁实例')
  await page.goBack()
  await page.waitForSelector('.annotation-nav', { timeout: 60000 })
  await page.waitForSelector('.annotation-sdk-host', { state: 'attached', timeout: 60000 })
  await page.waitForSelector('.fab', { timeout: 60000 })
  console.log('8) 返回文档页恢复实例')

  // 移动端
  const mobileContext = await browser.newContext({ viewport: { width: 390, height: 844 } })
  await mobileContext.addInitScript(({ key, token }) => localStorage.setItem(key, token), {
    key: TOKEN_KEY,
    token: login.token,
  })
  const mobile = await mobileContext.newPage()
  await mobile.goto(PAGE, { waitUntil: 'domcontentloaded', timeout: 120000 })
  await mobile.waitForSelector('.annotation-nav', { timeout: 120000 })
  const mobileNameDisplay = await mobile.evaluate(() => {
    const el = document.querySelector('.annotation-nav-name')
    return el ? getComputedStyle(el).display : 'none'
  })
  await mobile.click('.annotation-nav-btn')
  await mobile.waitForSelector('.drawer', { state: 'visible' })
  const drawerWidth = await mobile.evaluate(() => {
    const el = document.querySelector('.annotation-sdk-host')?.shadowRoot?.querySelector('.drawer')
    return el ? Math.round(el.getBoundingClientRect().width) : 0
  })
  if (mobileNameDisplay !== 'none' || drawerWidth < 380) {
    throw new Error(`mobile layout failed: name=${mobileNameDisplay} drawer=${drawerWidth}`)
  }
  console.log('9) 移动端导航与抽屉正常')

  // 错误降级
  const errorContext = await browser.newContext({ viewport: { width: 1440, height: 900 } })
  const errorPage = await errorContext.newPage()
  await errorPage.route('**/sdk/v1/annotation-sdk.js', (route) => route.abort())
  await errorPage.goto(PAGE, { waitUntil: 'domcontentloaded', timeout: 120000 })
  await errorPage.waitForSelector('.annotation-nav', { timeout: 120000 })
  await errorPage.waitForSelector('text=批注不可用', { timeout: 60000 })
  console.log('10) SDK 加载失败时入口降级')

  console.log('VITEPRESS NAV E2E PASS')
} finally {
  await browser.close()
}
