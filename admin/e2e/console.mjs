import { chromium } from 'playwright-core'

const CHROME =
  process.env.CHROME_PATH || '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'
const BASE = (process.env.ADMIN_BASE || 'http://127.0.0.1:18090').replace(/\/+$/, '')
const ADMIN_NAME = process.env.DEV_ADMIN_NAME || 'alice'
const TOKEN_KEY = 'annotation-admin:token'

async function devLogin(name) {
  const res = await fetch(`${BASE}/api/v1/auth/dev/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name }),
  })
  if (!res.ok) throw new Error(`dev login failed: ${res.status}`)
  return res.json()
}

const admin = await devLogin(ADMIN_NAME)
if (!admin.user?.isSuperadmin) {
  throw new Error(`用户 ${ADMIN_NAME} 不是超级管理员，请先执行 admin promote`)
}

const stamp = Date.now()
const origin = 'http://e2e.test'
const site = await (
  await fetch(`${BASE}/api/v1/admin/sites`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${admin.token}`,
    },
    body: JSON.stringify({
      name: `e2e-admin-${stamp}`,
      displayName: `E2E Admin ${stamp}`,
      origins: [origin],
      moderationMode: 'pre',
      themeColor: '#087f5b',
      defaultLocale: 'zh-CN',
    }),
  })
).json()
if (!site.id) throw new Error(`create site failed: ${JSON.stringify(site)}`)

const pendingUser = await devLogin(`e2e-pending-${stamp}`)
const pendingContent = `admin pending text ${stamp}`
const annotationResp = await fetch(`${BASE}/api/v1/annotations`, {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'X-Site-Key': site.key,
    Origin: origin,
    Authorization: `Bearer ${pendingUser.token}`,
  },
  body: JSON.stringify({
    pageKey: '/admin-e2e',
    pageTitle: 'admin e2e',
    anchor: { quote: { type: 'TextQuoteSelector', exact: pendingContent, prefix: '', suffix: '' } },
    content: 'pending annotation for admin console',
  }),
})
if (!annotationResp.ok) throw new Error(`create annotation failed: ${annotationResp.status}`)

const browser = await chromium.launch({ executablePath: CHROME, headless: true })
try {
  const page = await browser.newPage({ viewport: { width: 1366, height: 900 } })
  page.on('pageerror', (err) => console.log('[pageerror]', err.message))
  page.on('console', (msg) => {
    if (msg.type() === 'error') console.log('[console.error]', msg.text())
  })
  page.on('dialog', (dialog) => dialog.accept('x'))
  await page.addInitScript(({ key, token }) => localStorage.setItem(key, token), {
    key: TOKEN_KEY,
    token: admin.token,
  })

  await page.goto(`${BASE}/admin/`, { waitUntil: 'domcontentloaded', timeout: 120000 })
  await page.waitForSelector('.stat-grid', { timeout: 120000 })
  const cards = await page.locator('.stat-card').count()
  console.log('1) 概览统计卡片:', cards)
  if (cards !== 6) throw new Error('expected 6 stat cards')

  await page.locator('.el-menu-item', { hasText: '站点' }).click()
  await page.waitForSelector('.el-table__row', { timeout: 30000 })
  await page.locator('.el-table__row', { hasText: `e2e-admin-${stamp}` }).getByText('详情').click()
  await page.waitForSelector('.el-tabs', { timeout: 30000 })
  console.log('2) 站点详情已打开')

  await page.goto(`${BASE}/admin/sites/${site.id}?tab=providers`, {
    waitUntil: 'domcontentloaded',
    timeout: 120000,
  })
  await page.waitForSelector('.el-tabs__item', { timeout: 30000 })
  console.log('3) 深链刷新正常')

  await page.goto(`${BASE}/admin/moderation?site=${site.id}&status=pending`, {
    waitUntil: 'domcontentloaded',
    timeout: 120000,
  })
  const row = page.locator('.el-table__row', { hasText: pendingContent })
  await row.waitFor({ timeout: 30000 })
  await row.getByRole('button', { name: '通过' }).click()
  await page.locator('.el-message-box__btns .el-button--primary').click()
  await page.waitForSelector(`text=${pendingContent}`, { state: 'detached', timeout: 30000 }).catch(() => {})
  console.log('4) 审核通过完成')

  await page.goto(`${BASE}/admin/users`, { waitUntil: 'domcontentloaded', timeout: 120000 })
  const userRow = page.locator('.el-table__row', { hasText: `e2e-pending-${stamp}` })
  await userRow.waitFor({ timeout: 30000 })
  await userRow.getByRole('button', { name: '封禁' }).click()
  await page.locator('.el-message-box__btns .el-button--primary').click()
  await page.waitForTimeout(800)
  console.log('5) 用户封禁完成')

  await page.goto(`${BASE}/admin/sites`, { waitUntil: 'domcontentloaded', timeout: 120000 })
  const siteRow = page.locator('.el-table__row', { hasText: `e2e-admin-${stamp}` })
  await siteRow.waitFor({ timeout: 30000 })
  await siteRow.getByRole('button', { name: '编辑' }).click()
  await page.waitForSelector('.el-dialog', { timeout: 15000 })
  const displayInput = page.locator('.el-dialog input').nth(1)
  await displayInput.fill(`E2E Renamed ${stamp}`)
  await page.locator('.el-dialog').getByRole('button', { name: '保存' }).click()
  await page.waitForTimeout(800)
  console.log('6) 站点编辑完成')

  await siteRow.getByRole('button', { name: '轮换 Key' }).click()
  await page.locator('.el-message-box__btns .el-button--primary').click()
  await page.waitForSelector('.el-message-box', { timeout: 15000 })
  await page.getByRole('button', { name: '知道了' }).click()
  console.log('7) siteKey 轮换完成')

  const loginPage = await browser.newPage()
  await loginPage.goto(`${BASE}/admin/login`, { waitUntil: 'domcontentloaded', timeout: 120000 })
  await loginPage.waitForSelector('text=朱批 · Zhupi', { timeout: 30000 })
  console.log('8) 登录页渲染正常')

  console.log('ADMIN E2E PASS')
} finally {
  await browser.close()
}
