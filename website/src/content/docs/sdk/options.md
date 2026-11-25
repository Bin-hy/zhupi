---
title: 初始化选项
description: ZhupiSDK.init 的全部参数、默认值、CSS 变量覆盖与开发登录。
---

`init(options)` 返回一个批注实例。`siteKey` 必填，其余都有默认值。

## 参数总表

| 选项 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `siteKey` | `string` | 无（必填） | 站点公开标识 |
| `apiBase` | `string` | 从脚本地址 / 当前页面推导 | 批注服务地址，末尾斜杠会被去掉 |
| `container` | `HTMLElement \| string` | `document.body` | SDK 挂载位置 |
| `selectRoot` | `HTMLElement \| string` | `document.body` | 允许划词的范围 |
| `pageKey` | `string` | `location.pathname` | 页面归属键；SPA 用 `setPage` 更新 |
| `pageTitle` | `string` | `document.title` | 页面标题，用于展示与审核 |
| `locale` | `'zh-CN' \| 'en'` | `'zh-CN'` | 界面语言 |
| `theme` | `'auto' \| 'light' \| 'dark'` | `'auto'` | 主题；会记忆到本地 |
| `cssVariables` | `Record<string, string>` | 无 | 覆盖 Shadow DOM 内的 CSS 变量 |
| `devLogin` | `boolean` | `false` | 仅本地开发；需服务端 `DEV_LOGIN=1` |
| `on` | `Record<EventName, handler>` | 无 | 初始化时注册事件 |

## apiBase 的推导

按顺序取：

1. `options.apiBase`；
2. `<script>` 的 `src` 所在 origin（IIFE 引入时）；
3. ESM 模块 URL 的 origin（打包引入时）；
4. 当前页面的 `location.origin`。

自托管时建议显式传入，避免 CDN 与 API 不同源时推导错误：

```js
init({
  siteKey: 'YOUR_SITE_KEY',
  apiBase: 'https://annot.example.com',
})
```

## selectRoot

只允许在正文区域划词，避免导航、侧边栏被选中：

```js
init({
  siteKey: 'YOUR_SITE_KEY',
  selectRoot: '.article-body', // 或 HTMLElement
})
```

- 元素不存在时回退到 `document.body`。
- 如果正文在 iframe 中，需要在 iframe 内单独初始化 SDK（不跨 iframe 共享选区）。

## container

SDK 的抽屉、按钮与账号中心默认挂到 `document.body`。如果你有更合适的层级（例如应用根节点），可以指定：

```js
init({
  siteKey: 'YOUR_SITE_KEY',
  container: document.getElementById('app')!,
})
```

## pageKey 与 pageTitle

- `pageKey` 默认 `location.pathname`；即使带查询参数，默认也只取路径。
- 需要区分查询参数时显式传入，例如 `location.pathname + location.search`。
- `pageTitle` 只用于展示，不参与数据归属。

SPA 路由切换后调用：

```js
await instance.setPage({ pageKey: location.pathname, pageTitle: document.title })
```

## 主题与语言

```js
init({
  siteKey: 'YOUR_SITE_KEY',
  theme: 'dark',   // auto | light | dark
  locale: 'en',    // zh-CN | en
})
```

- 主题偏好按「API 地址 + siteKey」写入 `localStorage`。
- 如果初始化时**传了** `theme`，它会覆盖本地记忆；不传时才使用用户上次的选择，否则为 `auto`。
- 站点配置中的 `themeColor` 与 `defaultLocale` 会作为站点级默认值；`init` 的 `locale` 优先于站点默认值。
- 用户也可以在账号中心里修改主题与语言。

## CSS 变量

SDK 的样式封装在 Shadow DOM 内。需要微调时通过 `cssVariables` 覆盖：

```js
init({
  siteKey: 'YOUR_SITE_KEY',
  cssVariables: {
    '--annot-accent': '#c92a2a',
    '--annot-bg': '#ffffff',
    '--annotation-fab-right': '24px',
  },
})
```

可用变量：

| 变量 | 作用 |
| --- | --- |
| `--annot-accent` | 强调色；站点 `themeColor` 会作为默认值 |
| `--annot-bg` | 背景色 |
| `--annot-fg` | 前景文字色 |
| `--annot-muted` | 次要文字色 |
| `--annot-border` | 边框色 |
| `--annot-soft` | 浅色区块背景 |
| `--annot-danger` | 危险操作色 |
| `--annotation-fab-right` | 悬浮按钮距右侧距离 |
| `--annotation-fab-bottom` | 悬浮按钮距底部距离 |
| `--annotation-drawer-top` | 抽屉顶部偏移 |

深色模式会覆盖部分颜色变量；需要固定配色时同时设置亮/暗两套值，或直接指定 `theme`。

## devLogin

仅用于本地开发：

```js
init({ siteKey: 'YOUR_SITE_KEY', devLogin: true })
```

服务端必须同时设置 `DEV_LOGIN=1`，否则开发登录接口不可用。**不要在生产环境开启。**

## 事件注册

```js
init({
  siteKey: 'YOUR_SITE_KEY',
  on: {
    ready: ({ siteKey }) => console.log('ready', siteKey),
    statechange: ({ loggedIn, annotationCount }) => {
      // 同步到宿主导航
    },
  },
})
```

完整事件列表见 [方法与事件](/sdk/methods-events/)。
