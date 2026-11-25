# zhupi-sdk

可嵌入的划线批注 SDK。选中网页上的文字 → 划线 → 批注与回复；UI 运行在 Shadow DOM 中，
不污染宿主样式。它是 [zhupi](https://github.com/Bin-hy/zhupi)（朱批）自托管服务的浏览器端，
需要配合一个已创建的站点使用。

## 安装

```bash
npm install zhupi-sdk
```

也可以直接用 `<script>` 引入：

```html
<script src="https://unpkg.com/zhupi-sdk/dist/zhupi.js"></script>
<script>
  ZhupiSDK.init({ siteKey: 'YOUR_SITE_KEY' })
</script>
```

## 快速开始

```ts
import { init } from 'zhupi-sdk'

const instance = init({
  siteKey: 'YOUR_SITE_KEY',
  apiBase: 'https://zhupi.example.com', // 可选；默认从脚本地址/当前页面推导
  selectRoot: '.article-body',          // 可选；允许划词的范围，默认 document.body
  pageKey: location.pathname,
  pageTitle: document.title,
  theme: 'auto',
})
```

`siteKey` 由 zhupi 服务端创建站点时签发：

```bash
zhupi admin site create --name blog --origin https://your.site
```

## 评论面板

- 搜索：服务端按引用、正文、页面、作者与评论内容匹配，输入后自动刷新；
- 排序：文档顺序 / 最新发布，偏好按站点保存在 `localStorage`；
- 未读：抽屉关闭时悬浮按钮显示红点，打开后按页面记录已读时间；
- 草稿：未发布的批注按页面自动保存，重新打开编辑器可恢复；
- 回复：支持 `@作者`、`⌘/Ctrl + Enter` 发布，评论支持轻量 Markdown；
- 快捷键：`/` 聚焦搜索，`j` / `k` 切换到下一条 / 上一条批注。

## 配置

| 选项 | 类型 | 说明 |
|------|------|------|
| `siteKey` | `string` | **必填**，站点公开 key |
| `apiBase` | `string` | 批注服务地址；默认从脚本地址或当前页面推导 |
| `selectRoot` | `HTMLElement \| string` | 允许划词的范围，默认 `document.body` |
| `container` | `HTMLElement \| string` | SDK 挂载位置，默认 `document.body` |
| `pageKey` | `string` | 页面标识，默认当前路径 |
| `pageTitle` | `string` | 页面标题 |
| `locale` | `'zh-CN' \| 'en'` | 界面语言，默认 `zh-CN` |
| `theme` | `'auto' \| 'light' \| 'dark'` | 主题，默认 `auto` |
| `cssVariables` | `Record<string, string>` | 覆盖 UI CSS 变量 |
| `devLogin` | `boolean` | 仅本地开发使用 |
| `on` | `Partial<Record<EventName, handler>>` | 初始化时注册事件 |

## 实例方法

| 方法 | 说明 |
|------|------|
| `setPage({ pageKey?, pageTitle? })` | SPA 路由切换后更新页面上下文 |
| `refresh()` | 重新拉取当前页面的批注 |
| `login(providerId?)` | 登录；`providerId` 用于选择站点级 OIDC 提供商 |
| `logout()` | 退出当前会话 |
| `getCurrentUser()` | 当前用户，未登录返回 `null` |
| `getAnnotationCount()` | 当前页面批注总数 |
| `setTheme(mode)` | 切换 `auto` / `light` / `dark` |
| `openDrawer()` | 打开批注抽屉 |
| `openAccount()` | 打开账号中心 |
| `on(event, handler)` | 订阅事件，返回取消订阅函数 |
| `destroy()` | 卸载 UI 与监听器 |

## 事件

| 事件 | 载荷 | 说明 |
|------|------|------|
| `ready` | `{ siteKey }` | SDK 初始化完成 |
| `login` | `User` | 登录成功 |
| `logout` | `undefined` | 退出登录 |
| `created` | `Annotation` | 新建批注 |
| `updated` | `Annotation` | 批注更新 |
| `deleted` | `{ id }` | 批注删除 |
| `error` | `Error` | SDK 内部错误 |
| `pagechange` | `{ pageKey }` | 页面上下文变化 |
| `statechange` | `{ user, annotationCount, loggedIn }` | 登录态或数量变化，适合同步宿主导航 |

## 类型

包内导出完整的公开类型，包括 `InitOptions`、`AnnotationInstance`、`Annotation`、
`AnnotationComment`、`User`、`Author`、`EventMap`、`Anchor`、`QuoteSelector`、
`SiteConfig`、`Identity`、`SessionView`、`CreateAnnotationInput` 与 `ThemeMode`。

## 主题与国际化

- 主题：`auto` 跟随系统，也可以显式指定 `light` / `dark`；切换后偏好写入本地存储。
- 语言：内置 `zh-CN` 与 `en`。
- 样式：全部封装在 Shadow DOM 内；需要微调时通过 `cssVariables` 覆盖。

## 浏览器支持

面向现代浏览器（ES2020+，支持 Shadow DOM 与 `ResizeObserver`）。构建产出：

- `dist/zhupi.esm.js`：ESM 入口，适合 bundler；
- `dist/zhupi.js`：IIFE，适合 `<script>` 与 CDN；
- `dist/index.d.ts`：TypeScript 类型声明。

## 与 zhupi 服务端的关系

SDK 只包含前端交互与 API 客户端。批注数据、账号、审核、站点配置都由 zhupi 服务端提供；
本包不内置服务端，也不发起任何未在配置中声明的网络请求。

服务端也可以自行托管同一份产物：`GET /sdk/v1/zhupi.js` 与 `GET /sdk/v1/zhupi.esm.js`。

## 许可证

[MIT](./LICENSE)
