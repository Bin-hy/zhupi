---
title: 方法与事件
description: 实例方法、事件载荷、深链接、登录跳转与 orphan 批注重定位。
---

## 实例方法

| 方法 | 返回 | 说明 |
| --- | --- | --- |
| `setPage({ pageKey?, pageTitle? })` | `Promise<void>` | SPA 路由切换后更新页面上下文 |
| `refresh()` | `Promise<void>` | 重新拉取当前页面批注 |
| `login(providerId?)` | `void` | 登录；`providerId` 用于选择站点级 OIDC |
| `logout()` | `void` | 退出当前会话 |
| `getCurrentUser()` | `User \| null` | 当前用户 |
| `getAnnotationCount()` | `number` | 当前页面批注总数 |
| `setTheme(mode)` | `void` | `auto` / `light` / `dark` |
| `openDrawer()` | `void` | 打开批注抽屉 |
| `openAccount()` | `void` | 打开账号中心 |
| `on(event, handler)` | `() => void` | 订阅事件，返回取消订阅函数 |
| `destroy()` | `void` | 卸载 UI 与监听器 |

### setPage

切换页面时必须调用，否则新页面的批注会继续挂在旧 `pageKey` 下：

```js
await instance.setPage({
  pageKey: location.pathname,
  pageTitle: document.title,
})
```

它会重新拉取列表，并清理上一页的搜索、选中状态与未读记录；草稿按新页面切换存储键，旧页草稿仍保留。

### login

登录使用整页跳转（OAuth 流程），返回后会回到当前页面。不要把它当成弹窗：

```js
instance.login()          // GitHub（若已配置）
instance.login(providerId) // 指定站点 OIDC 提供商
```

不传 `providerId` 时按站点配置选择默认提供商。

### openDrawer 与 getAnnotationCount

适合在宿主导航里放一个入口：

```js
const count = instance.getAnnotationCount()
document.querySelector('#annotation-count')!.textContent = String(count)
button.addEventListener('click', () => instance.openDrawer())
```

## 事件

| 事件 | 载荷 | 触发时机 |
| --- | --- | --- |
| `ready` | `{ siteKey }` | 初始化完成 |
| `login` | `User` | 登录成功 |
| `logout` | `undefined` | 退出登录 |
| `created` | `Annotation` | 新建批注 |
| `updated` | `Annotation` | 批注状态或锚点更新 |
| `deleted` | `{ id }` | 批注删除 |
| `error` | `Error` | SDK 内部错误 |
| `pagechange` | `{ pageKey }` | 页面上下文变化 |
| `statechange` | `{ user, annotationCount, loggedIn }` | 登录态或数量变化 |

### 同步宿主导航

```js
instance.on('statechange', ({ loggedIn, user, annotationCount }) => {
  nav.setLoggedIn(loggedIn)
  nav.setAnnotationCount(annotationCount)
})
```

### 错误处理

```js
instance.on('error', (err) => {
  console.error('[zhupi]', err)
})
```

网络错误、权限错误与配额错误都会通过该事件暴露；同时请求失败会在 UI 内给出提示。

## 深链接

每条批注都有稳定的深链接：

```text
https://blog.example.com/post#annot-123
```

打开后 SDK 会自动定位并高亮对应批注。若页面内容变化导致无法自动定位，会进入 orphan 状态，提供「重新定位」入口，由用户手动选中新的文字。

## 销毁

组件卸载（例如微前端切换）时调用：

```js
instance.destroy()
```

会移除 Shadow DOM、事件监听、定时器与主题监听。销毁后该实例不可再用，需要重新 `init`。

## 取消订阅

```js
const off = instance.on('created', handler)
off() // 取消
```

`destroy()` 会同时取消所有订阅。
