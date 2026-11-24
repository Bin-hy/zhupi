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
const TOKEN_KEY = `annotation-sdk:${API}:${SITE_KEY}`

async function devLogin(name) {
  const res = await fetch(`${API}/api/v1/auth/dev/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name }),
  })
  return res.json()
}

const first = await devLogin('e2e-extra4')
await devLogin('e2e-extra4')
await devLogin('e2e-extra4')

const browser = await chromium.launch({ executablePath: CHROME, headless: true })
try {
  const page = await browser.newPage({ viewport: { width: 1280, height: 900 } })
  page.on('pageerror', (err) => console.log('[pageerror]', err.message))
  page.on('dialog', (dialog) => dialog.accept('e2e-extra4'))
  await page.addInitScript(({ key, token }) => localStorage.setItem(key, token), {
    key: TOKEN_KEY,
    token: first.token,
  })
  await page.goto(PAGE, { waitUntil: 'domcontentloaded', timeout: 120000 })
  await page.waitForSelector('.fab', { timeout: 120000 })
  await page.click('.fab')
  await page.click('[data-action="account"]')
  await page.waitForFunction(
    () => document.querySelector('.annotation-sdk-host')?.shadowRoot?.textContent?.includes('e2e-extra4'),
    null,
    { timeout: 30000 },
  )

  await page.waitForSelector('[data-action="revoke-session"]', { timeout: 30000 })
  const before = await page.locator('[data-action="revoke-session"]').count()
  console.log('1) 会话数量:', before)
  if (before < 2) throw new Error('expected multiple sessions')

  await page.locator('[data-action="revoke-session"]').first().click()
  await page.click('.confirm')
  let after = before
  for (let i = 0; i < 60; i++) {
    await page.waitForTimeout(250)
    after = await page.locator('[data-action="revoke-session"]').count()
    if (after === before - 1) break
  }
  console.log('2) 撤销单个会话后:', after)
  if (after !== before - 1) throw new Error(`session count did not decrease: ${before} -> ${after}`)

  await page.click('[data-action="revoke-all"]')
  await page.click('.confirm')
  await page.waitForFunction(
    () => !localStorage.getItem('annotation-sdk:' + 'http://127.0.0.1:18090' + ':' + 'O8sDo-s-GpEZK02vC8tUTYvbdCUB02LZ'),
    null,
    { timeout: 30000 },
  )
  console.log('3) 退出所有设备后本地 token 已清除')

  // 移动端
  const mobile = await browser.newPage({ viewport: { width: 390, height: 844 } })
  await mobile.addInitScript(({ key, token }) => localStorage.setItem(key, token), {
    key: TOKEN_KEY,
    token: (await devLogin('e2e-mobile')).token,
  })
  await mobile.goto(PAGE, { waitUntil: 'domcontentloaded', timeout: 120000 })
  await mobile.waitForSelector('.fab', { timeout: 120000 })
  await mobile.click('.fab')
  const metrics = await mobile.evaluate(() => {
    const root = document.querySelector('.annotation-sdk-host')?.shadowRoot
    const drawer = root?.querySelector('.drawer')
    const tab = root?.querySelector('.tab')
    const fab = root?.querySelector('.fab')
    return {
      drawerWidth: drawer ? Math.round(drawer.getBoundingClientRect().width) : 0,
      tabHeight: tab ? Math.round(tab.getBoundingClientRect().height) : 0,
      fabHeight: fab ? Math.round(fab.getBoundingClientRect().height) : 0,
    }
  })
  console.log('4) 移动端尺寸:', JSON.stringify(metrics))
  if (metrics.drawerWidth < 380) throw new Error('drawer should be full width on mobile')
  if (metrics.tabHeight < 44) throw new Error('tab touch target too small')

  // 键盘焦点陷阱
  for (let i = 0; i < 6; i++) await mobile.keyboard.press('Tab')
  const trapped = await mobile.evaluate(() => {
    const host = document.querySelector('.annotation-sdk-host')
    const root = host?.shadowRoot
    const active = root?.activeElement
    const drawer = root?.querySelector('.drawer')
    return !!active && !!drawer && drawer.contains(active)
  })
  console.log('5) 键盘焦点保持在抽屉内:', trapped)
  if (!trapped) throw new Error('focus escaped the drawer')

  await mobile.keyboard.press('Escape')
  const closed = await mobile.evaluate(() => {
    const root = document.querySelector('.annotation-sdk-host')?.shadowRoot
    return root?.querySelector('.drawer')?.hasAttribute('hidden')
  })
  console.log('6) Escape 关闭抽屉:', closed)
  if (!closed) throw new Error('Escape did not close drawer')

  console.log('EXTRA E2E PASS')
} finally {
  await browser.close()
}
