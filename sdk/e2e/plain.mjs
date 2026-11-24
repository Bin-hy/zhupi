import { chromium } from 'playwright-core'

function required(name) {
  const value = process.env[name]
  if (!value) throw new Error(`missing env ${name}`)
  return value
}

const CHROME = process.env.CHROME_PATH || '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'
const API = process.env.ANNOTATION_API || 'http://127.0.0.1:18090'
const SITE_KEY = required('SITE_KEY')
const PAGE = process.env.PAGE || `http://127.0.0.1:15174/?siteKey=${SITE_KEY}&api=${API}`

const browser = await chromium.launch({ executablePath: CHROME, headless: true })
try {
  const page = await browser.newPage({ viewport: { width: 1280, height: 900 } })
  page.on('console', (msg) => {
    if (msg.type() === 'error') console.log('[console.error]', msg.text())
  })
  page.on('pageerror', (err) => console.log('[pageerror]', err.message))
  page.on('dialog', (dialog) => dialog.accept('sdk-user'))

  await page.goto(PAGE, { waitUntil: 'domcontentloaded' })
  await page.waitForSelector('.fab', { timeout: 30000 })
  console.log('1) SDK 已加载并渲染')

  await page.click('.fab')
  await page.waitForSelector('.drawer', { state: 'visible' })
  await page.getByRole('button', { name: '本地调试登录' }).click()
  await page.waitForFunction(() => document.querySelector('.annotation-sdk-host')?.shadowRoot?.textContent?.includes('sdk-user'), null, { timeout: 30000 })
  console.log('2) 调试登录成功')
  await page.click('[data-action="close-account"]')
  await page.click('[data-action="close-drawer"]')
  await page.waitForSelector('.drawer', { state: 'hidden' })

  const selected = await page.evaluate(() => {
    const p = document.querySelector('main p')
    if (!p) return null
    const walker = document.createTreeWalker(p, NodeFilter.SHOW_TEXT)
    let node
    while ((node = walker.nextNode())) {
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
    return null
  })
  if (!selected) throw new Error('未找到可选中文本')
  console.log('3) 已选中:', JSON.stringify(selected))

  await page.waitForSelector('.toolbar', { state: 'visible', timeout: 15000 })
  await page.click('.toolbar button')
  await page.waitForSelector('.composer', { state: 'visible' })
  await page.fill('.composer textarea', '这是 SDK 端到端测试写入的批注')
  await page.click('.composer .publish')

  await page.waitForSelector('mark.annot-mark', { timeout: 30000 })
  await page.waitForSelector('.card', { timeout: 30000 })
  const markText = await page.locator('mark.annot-mark').first().innerText()
  console.log('4) 批注创建成功，高亮:', JSON.stringify(markText))

  await page.reload({ waitUntil: 'domcontentloaded' })
  await page.waitForSelector('mark.annot-mark', { timeout: 30000 })
  const afterReload = await page.locator('mark.annot-mark').first().innerText()
  console.log('5) 刷新后高亮恢复:', JSON.stringify(afterReload))

  const list = await (
    await fetch(`${API}/api/v1/annotations?page=${encodeURIComponent('/')}`, {
      headers: { 'X-Site-Key': SITE_KEY, Origin: 'http://127.0.0.1:15174' },
    })
  ).json()
  if (!list.annotations || list.annotations.length < 1) {
    throw new Error(`API 未返回批注: ${JSON.stringify(list)}`)
  }
  console.log('6) API 校验通过，批注数:', list.annotations.length)
  console.log('SDK E2E PASS')
} finally {
  await browser.close()
}
