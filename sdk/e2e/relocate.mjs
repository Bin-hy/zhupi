import { chromium } from 'playwright-core'

function required(name) {
  const value = process.env[name]
  if (!value) throw new Error(`missing env ${name}`)
  return value
}

const CHROME = process.env.CHROME_PATH || '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'
const API = process.env.ANNOTATION_API || 'http://127.0.0.1:18090'
const SITE_KEY = required('SITE_KEY')
const ORIGIN = process.env.SITE_ORIGIN || 'http://127.0.0.1:15174'
const PAGE = process.env.PAGE || `${ORIGIN}/?siteKey=${SITE_KEY}&api=${API}`
const TOKEN_KEY = `annotation-sdk:${API}:${SITE_KEY}`

const login=async n=>(await (await fetch(`${API}/api/v1/auth/dev/login`,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({name:n})})).json())
const user=await login('e2e-reloc')
const created=await (await fetch(`${API}/api/v1/annotations`,{method:'POST',headers:{'Content-Type':'application/json','X-Site-Key':SITE_KEY,Origin:'http://127.0.0.1:15174',Authorization:`Bearer ${user.token}`},body:JSON.stringify({pageKey:'/',pageTitle:'demo',anchor:{quote:{type:'TextQuoteSelector',exact:'这段文字不存在XYZ',prefix:'',suffix:''}},content:'orphan 测试批注'})})).json()
if(!created.id) throw new Error('create failed '+JSON.stringify(created))
const browser=await chromium.launch({executablePath:CHROME,headless:true})
try{
 const page=await browser.newPage({viewport:{width:1280,height:900}})
 page.on('pageerror',e=>console.log('[pageerror]',e.message))
 await page.addInitScript(({k,t})=>localStorage.setItem(k,t),{k:TOKEN_KEY,t:user.token})
 await page.goto(PAGE,{waitUntil:'domcontentloaded'})
 await page.waitForSelector('.fab',{timeout:30000})
 await page.click('.fab')
 await page.waitForSelector(`[data-card="${created.id}"] [data-action="relocate"]`,{timeout:30000})
 console.log('1) orphan 批注显示重定位入口')

 // 筛选
 await page.click('[data-action="filter"][data-filter="mine"]')
 await page.waitForTimeout(600)
 const mine=await page.locator('.card').count()
 await page.click('[data-action="filter"][data-filter="all"]')
 await page.waitForTimeout(600)
 const all=await page.locator('.card').count()
 console.log('2) 筛选：我的', mine, '全部', all)
 if(mine<1||all<mine) throw new Error('filter result invalid')

 await page.click(`[data-card="${created.id}"] [data-action="relocate"]`)
 await page.waitForSelector('.relocate-banner',{state:'visible'})
 const selected=await page.evaluate(()=>{
   const p=document.querySelector('main p')
   const walker=document.createTreeWalker(p,NodeFilter.SHOW_TEXT)
   let node
   while((node=walker.nextNode())){
     const t=node.textContent||''
     if(t.trim().length>=10){
       const start=t.search(/\S/)
       const r=document.createRange(); r.setStart(node,start); r.setEnd(node,start+10)
       const s=window.getSelection(); s.removeAllRanges(); s.addRange(r)
       document.dispatchEvent(new Event('selectionchange'))
       return t.slice(start,start+10)
     }
   }
   return null
 })
 if(!selected) throw new Error('no selectable text')
 console.log('3) 选择新位置:', JSON.stringify(selected))
 await page.waitForSelector('.relocate-banner',{state:'hidden',timeout:30000})
 await page.waitForSelector('mark.annot-mark',{timeout:30000})
 console.log('4) 重定位后高亮恢复')

 const list=await (await fetch(`${API}/api/v1/annotations?page=${encodeURIComponent('/')}`,{headers:{'X-Site-Key':SITE_KEY,Origin:'http://127.0.0.1:15174'}})).json()
 const target=list.annotations.find(a=>a.id===created.id)
 if(!target||target.quoteExact!==selected) throw new Error('anchor not updated: '+JSON.stringify(target))
 console.log('5) API 锚点已更新为:', JSON.stringify(target.quoteExact))
 console.log('RELOCATE E2E PASS')
} finally { await browser.close() }
