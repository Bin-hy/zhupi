---
title: 接入指南
description: 通过 script 标签或 ESM 把朱批 SDK 接入任意网站，并在 SPA 中正确切换页面。
---

接入分三步：创建站点拿到 `siteKey`、在页面加载 SDK、初始化实例。

## 1. 创建站点并拿到 siteKey

```bash
./bin/zhupi admin site create \
  --name blog \
  --origin https://blog.example.com
```

输出中的 `key` 就是 `siteKey`。`--origin` 必须包含实际访问页面时的完整来源（`scheme://host[:port]`），可重复传入多个。

## 2. 引入 SDK

### 从自己的服务加载（推荐）

服务端已经内嵌 SDK，不需要额外 CDN：

```html
<script src="https://annot.example.com/sdk/v1/zhupi.js"></script>
<script>
  ZhupiSDK.init({
    siteKey: 'YOUR_SITE_KEY',
    selectRoot: 'article',
  })
</script>
```

不传 `apiBase` 时，SDK 会从脚本地址推导服务地址（即 `https://annot.example.com`）。

### 从 npm / ESM 加载

```bash
npm install zhupi-sdk
```

```ts
import { init } from 'zhupi-sdk'

const instance = init({
  siteKey: 'YOUR_SITE_KEY',
  apiBase: 'https://annot.example.com',
  selectRoot: '.article-body',
})
```

SDK 构建产物：

| 文件 | 用途 |
| --- | --- |
| `dist/zhupi.esm.js` | ESM，适合 bundler |
| `dist/zhupi.js` | IIFE，适合 `<script>` / CDN |
| `dist/index.d.ts` | TypeScript 类型 |

也可以通过 unpkg 使用 npm 包：

```html
<script src="https://unpkg.com/zhupi-sdk/dist/zhupi.js"></script>
```

### 完整示例

```html
<!doctype html>
<html lang="zh-CN">
  <body>
    <article>
      <h1>一篇可以被批注的文章</h1>
      <p>选中这段文字，就会出现「批注」按钮。</p>
    </article>

    <script src="https://annot.example.com/sdk/v1/zhupi.js"></script>
    <script>
      const instance = ZhupiSDK.init({
        siteKey: 'YOUR_SITE_KEY',
        selectRoot: 'article',
        pageKey: location.pathname,
        pageTitle: document.title,
        theme: 'auto',
        locale: 'zh-CN',
        on: {
          ready: () => console.log('annotation ready'),
          created: (annotation) => console.log('created', annotation.id),
          error: (err) => console.error(err),
        },
      })
    </script>
  </body>
</html>
```

## 3. SPA / 路由切换

SPA 切换路由后，必须告诉 SDK 当前页面变了，否则批注仍归属旧页面：

```js
await instance.setPage({
  pageKey: location.pathname,
  pageTitle: document.title,
})
```

`setPage` 会重新拉取该页批注，清理上一页的搜索与选中状态，并切换到新页面的草稿。

### VitePress 示例

在自定义主题里初始化，并在路由变化后调用 `setPage`：

```ts
// .vitepress/theme/index.ts
import DefaultTheme from 'vitepress/theme'

let sdk: any

export default {
  extends: DefaultTheme,
  setup() {
    // 浏览器环境
    const script = document.createElement('script')
    script.src = 'https://annot.example.com/sdk/v1/zhupi.js'
    script.onload = () => {
      sdk = (window as any).ZhupiSDK.init({
        siteKey: 'YOUR_SITE_KEY',
        selectRoot: '.vp-doc',
        theme: 'auto',
      })
    }
    document.head.appendChild(script)
  },
}
```

VitePress 的 `router.onAfterRouteChanged` 或主题的 `watch` 中调用 `setPage` 即可；VitePress 的正文容器是 `.vp-doc`。

## 不同站点的接入位置

朱批不绑定特定技术栈。只要站点能提供一块稳定的正文容器，就可以接入：

| 站点类型 | 接入要点 |
| --- | --- |
| Hexo / Hugo / Jekyll / Astro | 全站引入脚本，用 `selectRoot` 指定文章正文 |
| VitePress / Docusaurus / Starlight | 指定 `pageKey`；客户端路由切换时调用 `setPage()` |
| Vue / React / Next.js / Nuxt | 使用 ESM SDK，在组件或路由生命周期中初始化与销毁 |
| Wiki / 知识库 / 在线阅读器 | 保持 `pageKey` 稳定，利用文本引用与模糊重定位跟随内容更新 |
| 任意静态页面 | 一段 `<script>` 即可，不需要现有后端配合 |

接入时通常只需要确认四件事：

1. 在 zhupi 后台创建站点，并把站点域名加入 Origin 白名单；
2. 用 `selectRoot` 限定允许划词的正文区域；
3. 用稳定的 `pageKey` 标识文章或文档页面；
4. SPA 路由变化时调用 `await instance.setPage(...)`。

## 验证接入

1. 打开页面，选中正文文字，应出现浮动的「批注」按钮。
2. 发布后刷新页面，划线仍在。
3. 打开开发者工具，确认请求头包含 `X-Site-Key`，且没有 `origin_not_allowed`。
4. 修改正文一两个字，确认批注能重新定位；大改后应出现 orphan 手动重定位入口。

仓库自带示例：

```bash
python3 -m http.server 5173
# 打开 /examples/plain-html/?siteKey=<key>&api=http://127.0.0.1:8080
```

## 常见接入错误

| 错误 | 原因 | 处理 |
| --- | --- | --- |
| `site_key_required` | 没有传 `siteKey` | 检查 `init` 参数 |
| `origin_not_allowed` | 页面 Origin 不在站点白名单 | 补充 `scheme://host[:port]` |
| `site_not_found` | siteKey 错误或已轮换 | 用 `admin site list` 确认 |
| 登录后回到错误域名 | `PUBLIC_BASE_URL` 配置错误 | 改成外部 HTTPS 地址 |
| CORS 预检失败 | 站点未启用或来源不匹配 | 检查站点状态与 Origin |
| 划线出现但刷新后消失 | 页面用了 SPA 但没调 `setPage` | 在路由变化后更新页面 |

更细的选项见 [初始化选项](/sdk/options/)，接口见 [方法与事件](/sdk/methods-events/)。
