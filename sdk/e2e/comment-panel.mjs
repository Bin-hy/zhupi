import { chromium } from 'playwright-core'

function required(name) {
  const value = process.env[name]
  if (!value) throw new Error(`missing env ${name}`)
  return value
}

const CHROME = process.env.CHROME_PATH || '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'
const API = (process.env.ANNOTATION_API || 'http://127.0.0.1:18090').replace(/\/+$/, '')
const SITE_KEY = required('SITE_KEY')
const ORIGIN = process.env.SITE_ORIGIN || 'http://127.0.0.1:15174'
const PAGE = process.env.PAGE || `${ORIGIN}/?siteKey=${SITE_KEY}&api=${API}`

async function devLogin(name) {
  const res = await fetch(`${API}/api/v1/auth/dev/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name }),
  })
  if (!res.ok) throw new Error(`dev login failed: ${res.status}`)
  return res.json()
}

async function createAnnotation(token, content, exact) {
  const res = await fetch(`${API}/api/v1/annotations`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-Site-Key': SITE_KEY,
      Origin: ORIGIN,
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({
      pageKey: '/',
      pageTitle: 'SDK E2E',
      anchor: { quote: { type: 'TextQuoteSelector', exact, prefix: '', suffix: '' } },
      content,
    }),
  })
  if (!res.ok) throw new Error(`create annotation failed: ${res.status} ${await res.text()}`)
  return res.json()
}

const stamp = Date.now()
const author = await devLogin(`e2e-panel-author-${stamp}`)
const firstContent = `第一条评论内容 ${stamp}`
const marker = `panel-${stamp}`
const secondContent = `第二条评论内容 ${marker}`
await createAnnotation(author.token, firstContent, '这是一段用于端到端测试的正文')
await new Promise((resolve) => setTimeout(resolve, 20))
await createAnnotation(author.token, secondContent, '第二段文字，用于验证')

const browser = await chromium.launch({ executablePath: CHROME, headless: true })
try {
  const page = await browser.newPage({ viewport: { width: 1280, height: 900 } })
  page.on('console', (msg) => {
    if (msg.type() === 'error') console.log('[console.error]', msg.text())
  })
  page.on('pageerror', (err) => console.log('[pageerror]', err.message))
  page.on('dialog', (dialog) => dialog.accept(`e2e-panel-user-${stamp}`))

  await page.goto(PAGE, { waitUntil: 'domcontentloaded' })
  await page.waitForSelector('.fab', { timeout: 30000 })
  await page.click('.fab')
  await page.waitForSelector('.drawer', { state: 'visible' })
  await page.getByRole('button', { name: '本地调试登录' }).click()
  await page.waitForFunction(
    (name) => document.querySelector('.zhupi-host')?.shadowRoot?.textContent?.includes(name),
    `e2e-panel-user-${stamp}`,
    { timeout: 30000 },
  )
  await page.click('[data-action="close-account"]')
  await page.waitForFunction(
    () => document.querySelectorAll('.zhupi-host')?.length === 1 &&
      document.querySelector('.zhupi-host').shadowRoot.querySelectorAll('.card').length >= 2,
    null,
    { timeout: 30000 },
  )
  console.log('1) 评论面板已加载两条批注')

  const search = page.locator('.zhupi-host').locator('.search-input')
  await search.fill(marker)
  await page.waitForFunction(
    () => document.querySelector('.zhupi-host').shadowRoot.querySelectorAll('.card').length === 1,
    null,
    { timeout: 30000 },
  )
  const searched = await page.locator('.zhupi-host').locator('.card').first().innerText()
  if (!searched.includes(secondContent)) throw new Error('search did not return the expected annotation')
  console.log('2) 服务端搜索命中评论正文')

  await search.fill('')
  await page.waitForFunction(
    () => document.querySelector('.zhupi-host').shadowRoot.querySelectorAll('.card').length >= 2,
    null,
    { timeout: 30000 },
  )
  await page.locator('.zhupi-host').locator('.sort-select').selectOption('newest')
  await page.waitForFunction(
    (content) => document.querySelector('.zhupi-host').shadowRoot.querySelector('.card')?.textContent?.includes(content),
    secondContent,
    { timeout: 30000 },
  )
  console.log('3) 最新发布排序生效')

  const firstCard = page.locator('.zhupi-host').locator('.card').first()
  await firstCard.locator('textarea[data-reply]').fill('端到端回复内容')
  await firstCard.locator('[data-action="reply"]').click()
  await page.waitForFunction(
    () => document.querySelector('.zhupi-host').shadowRoot.textContent.includes('端到端回复内容'),
    null,
    { timeout: 30000 },
  )
  console.log('4) 回复即时追加，无需刷新')

  await page.click('[data-action="close-drawer"]')
  await page.waitForSelector('.drawer', { state: 'hidden' })
  const selectParagraph = async () => {
    await page.evaluate(() => {
      const p = document.querySelectorAll('main p')[1]
      if (!p) throw new Error('paragraph not found')
      const walker = document.createTreeWalker(p, NodeFilter.SHOW_TEXT)
      let node
      while ((node = walker.nextNode())) {
        const text = node.textContent || ''
        if (text.trim().length < 12) continue
        const range = document.createRange()
        range.setStart(node, 0)
        range.setEnd(node, 12)
        const sel = window.getSelection()
        sel.removeAllRanges()
        sel.addRange(range)
        document.dispatchEvent(new MouseEvent('mouseup', { bubbles: true }))
        return
      }
      throw new Error('no selectable text')
    })
    await page.waitForSelector('.toolbar', { state: 'visible', timeout: 15000 })
  }
  await selectParagraph()
  await page.click('.toolbar button')
  await page.waitForSelector('.composer', { state: 'visible' })
  await page.fill('.composer textarea', '未发布的草稿内容')
  await page.click('.composer .cancel')
  await page.waitForSelector('.composer', { state: 'hidden' })
  await selectParagraph()
  await page.click('.toolbar button')
  await page.waitForSelector('.composer', { state: 'visible' })
  const draft = await page.locator('.composer textarea').inputValue()
  if (draft !== '未发布的草稿内容') throw new Error(`draft not restored: ${JSON.stringify(draft)}`)
  await page.click('.composer .cancel')
  console.log('5) 批注草稿可恢复')

  console.log('SDK COMMENT PANEL E2E PASS')
} finally {
  await browser.close()
}
