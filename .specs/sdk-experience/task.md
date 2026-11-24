# Annotation SDK 用户端体验完善 · Tasks

> 状态：草案 v0.1（等待评审）
> 输入：`.specs/sdk-experience/spec.md`、`.specs/sdk-experience/plan.md`
> 说明：后端改动集中在 T01–T06；T07 起为前端。每个任务都可单独验证。

## 文件清单

### 后端（修改）

| 文件 | 改动 |
|---|---|
| `internal/store/interfaces.go` | `AnnotationQuery`、`ListAnnotations`、`UpdateAnnotationAnchor` |
| `internal/store/annotation_repo.go` | 筛选 SQL、锚点更新 |
| `internal/store/annotation_repo_test.go` | 新增筛选/更新测试 |
| `internal/service/annotation.go` | `ListInput`/`UpdateAnnotationInput`、权限与校验 |
| `internal/service/annotation_test.go` | 新增 service 测试 |
| `internal/httpapi/annotation_handlers.go` | `filter` 参数、PATCH 扩展 |
| `internal/httpapi/server_test.go` | 新增 HTTP 集成测试 |
| `internal/httpapi/site_handlers.go` | provider `issuer` |
| `internal/httpapi/site_handlers_test.go` | 新增（或并入 server_test） |

### SDK（新建）

| 文件 | 职责 |
|---|---|
| `sdk/src/state.ts` | `SDKState`、Filter、状态辅助 |
| `sdk/src/theme.ts` | 主题偏好读取/持久化/系统监听 |
| `sdk/src/deeplink.ts` | `#annot-<id>` 解析/生成 |
| `sdk/src/utils/format.ts` | 相对时间、绝对时间、首字母 |
| `sdk/src/ui/host.ts` | Shadow DOM 宿主、CSS 变量、mark 样式注入 |
| `sdk/src/ui/focus-trap.ts` | 焦点陷阱 |
| `sdk/src/ui/toast.ts` | Toast + aria-live |
| `sdk/src/ui/confirm.ts` | Promise 确认框 |
| `sdk/src/ui/toolbar.ts` | 选区工具条 |
| `sdk/src/ui/composer.ts` | 批注编辑器 |
| `sdk/src/ui/drawer.ts` | 抽屉、筛选、列表、加载更多 |
| `sdk/src/ui/account.ts` | 身份、会话、主题、语言、退出 |
| `sdk/src/ui/ui.ts` | UI 门面（替换现有单文件实现） |
| `sdk/src/ui/styles.css` | 组件样式、响应式、对比度 |
| `sdk/src/*.test.ts` | 单元测试 |
| `sdk/e2e/sdk.spec.ts` | 主流程 |
| `sdk/e2e/account.spec.ts` | 身份/会话 |
| `sdk/e2e/mobile.spec.ts` | 移动端/可访问性 |
| `sdk/playwright.config.ts` | Playwright 配置 |

### SDK（修改）

| 文件 | 改动 |
|---|---|
| `sdk/src/index.ts` | 重写编排：筛选、分页、定位模式、深链接、账号 |
| `sdk/src/client.ts` | filter 参数、sessions、revoke、updateAnchor |
| `sdk/src/i18n.ts` | 新增文案、相对时间支持 |
| `sdk/src/events.ts` | 新增 updated/deleted/pagechange 语义 |
| `sdk/src/anchor.ts` | 保持，补充重定位辅助 |
| `examples/plain-html/index.html` | 演示筛选/账号/移动端 |
| `sdk/package.json` | 增加 `test:e2e` 脚本（如需要） |

---

## 阶段 0：后端最小补充

### T01 仓储筛选查询

**文件：** `internal/store/interfaces.go`、`annotation_repo.go`
**依赖：** 无
**步骤：**
1. 定义 `AnnotationQuery{SiteID,PageKey,ViewerID,Filter,IncludeUnapproved,Cursor,Limit}`。
2. 用条件拼装实现 `ListAnnotations(q)`：
   - all：公开可见 + 作者本人；
   - mine：仅 `created_by=viewer`，排除 deleted；
   - resolved：公开/本人 + `status=resolved`；
   - pending：仅 `created_by=viewer AND moderation=pending`；
   - IncludeUnapproved 可看全部非删除内容。
3. 保留游标分页与评论加载（复用 `loadComments`）。

**验证：** 新增仓储测试，覆盖四种 filter 的返回集合与分页。

### T02 仓储锚点更新

**文件：** `internal/store/annotation_repo.go`、`annotation_repo_test.go`
**依赖：** T01
**步骤：**
1. 实现 `UpdateAnnotationAnchor(ctx, siteID, id, anchor)`：序列化 anchor，更新 `anchor_json`、`quote_exact`、`section`、`updated_at`。
2. 站点不匹配返回 `ErrNotFound`。

**验证：** 单测：更新后可读回新 anchor；跨站点更新失败。

### T03 服务层筛选与更新

**文件：** `internal/service/annotation.go`、`annotation_test.go`
**依赖：** T01、T02
**步骤：**
1. `ListPage` 接受 `ListInput{PageKey,Filter,Cursor,Limit}`；校验 filter 合法；mine/pending 要求登录。
2. 新增 `Update(ctx,p,siteID,id,UpdateAnnotationInput)`：
   - status 与 anchor 至少一个；
   - status 权限：作者/站点管理员/超管；
   - anchor 权限同上，复用 `validateAnchor`；
   - 返回更新后的批注。
3. 错误码：`invalid_filter`、`login_required`、`anchor_required`。

**验证：** service 单测：filter 行为、越权更新、空输入、锚点校验。

### T04 HTTP 参数与 PATCH 扩展

**文件：** `internal/httpapi/annotation_handlers.go`、`server_test.go`
**依赖：** T03
**步骤：**
1. `GET /annotations` 解析 `filter`（默认 all）。
2. `PATCH /annotations/{id}` 解码 `{status?,anchor?}`，调用 `Update`，返回更新后的批注。
3. 保持已有 status 行为兼容；非法 JSON 返回 400。

**验证：** httptest：四种 filter、锚点更新、越权 403、跨站点 404。

### T05 site-config 返回 issuer

**文件：** `internal/httpapi/site_handlers.go`、`server_test.go`
**依赖：** 无
**步骤：**
1. `authProviderView` 增加 `issuer,omitempty`。
2. OIDC provider 填入 `issuer`；GitHub 留空。

**验证：** httptest 断言 OIDC 条目含 issuer，且不包含 secret。

### T06 后端回归测试

**文件：** `internal/.../*_test.go`
**依赖：** T01–T05
**步骤：**
1. 运行 `gofmt`、`go vet`、`go test ./...`。
2. 确认原有站点隔离、审核、角色测试全部通过。

**验证：** 全部命令通过并记录输出。

---

## 阶段 1：SDK 基础层

### T07 状态模型

**文件：** `sdk/src/state.ts`、`state.test.ts`
**依赖：** 无
**步骤：**
1. 定义 `SDKState`、`Filter`、`ThemeMode`、初始状态。
2. 提供纯函数：`setFilter`、`appendAnnotations`（按 id 去重）、`resetList`、`setUser`。

**验证：** Vitest：筛选切换、追加去重、重置。

### T08 格式化工具

**文件：** `sdk/src/utils/format.ts`、`format.test.ts`
**依赖：** 无
**步骤：**
1. `relativeTime(iso, locale)` 使用 `Intl.RelativeTimeFormat`。
2. `absoluteTime(iso, locale)` 使用 `Intl.DateTimeFormat`。
3. `initials(name)` 取首字母/首字。

**验证：** 固定时间与 locale 的单测。

### T09 主题管理

**文件：** `sdk/src/theme.ts`、`theme.test.ts`
**依赖：** 无
**步骤：**
1. 读取/写入按 `apiBase+siteKey` 的本地偏好。
2. `resolveTheme(option, preference)` + 监听 `prefers-color-scheme`。
3. 暴露 `setTheme` 与取消监听。

**验证：** jsdom 单测：优先顺序、持久化、系统主题变化。

### T10 深链接

**文件：** `sdk/src/deeplink.ts`、`deeplink.test.ts`
**依赖：** 无
**步骤：**
1. `parseDeepLink(hash) -> number | null` 识别 `#annot-<id>`。
2. `buildLink(href, id)` 去掉 token、替换 hash。
3. 与其他 hash 参数共存时不破坏。

**验证：** 单测：解析、生成、token 不泄漏。

### T11 客户端扩展

**文件：** `sdk/src/client.ts`、`client.test.ts`
**依赖：** T07
**步骤：**
1. `listAnnotations(pageKey, filter, cursor, limit)` 增加 filter 参数。
2. 新增 `sessions()`、`revokeSession(id)`、`revokeAllSessions()`。
3. 新增 `updateAnnotation(id, {status?, anchor?})`。
4. 类型补充 `issuer`、`currentSessionId`。

**验证：** Vitest mock fetch：URL、header、body、401 处理。

### T12 国际化扩展

**文件：** `sdk/src/i18n.ts`
**依赖：** 无
**步骤：**
1. 补齐筛选、账号、会话、确认框、重定位、复制链接、错误文案。
2. 中英文双语，保持键一致。

**验证：** 类型检查确保两套文案键一致；Vitest 抽查关键文案。

### T13 事件扩展

**文件：** `sdk/src/events.ts`
**依赖：** 无
**步骤：**
1. `EventMap` 增加 `updated`、`deleted`，保持 `created/error/pagechange/login/logout/ready`。
2. 事件类型与载荷保持向后兼容。

**验证：** 类型检查 + 单测订阅/取消。

### T14 焦点陷阱

**文件：** `sdk/src/ui/focus-trap.ts`、`focus-trap.test.ts`
**依赖：** 无
**步骤：**
1. `trapFocus(container, {onEscape})`：Tab 循环、Esc 回调。
2. `release()` 恢复触发元素焦点。

**验证：** jsdom 单测：Tab 循环、Esc、释放。

### T15 Toast 与播报

**文件：** `sdk/src/ui/toast.ts`、`toast.test.ts`
**依赖：** T14
**步骤：**
1. Toast 成功/错误样式与自动消失。
2. `aria-live="polite"` 区域，`announce(message)`。

**验证：** jsdom 单测：显示/隐藏、aria-live 文本更新。

---

## 阶段 2：UI 组件

### T16 Shadow DOM 宿主

**文件：** `sdk/src/ui/host.ts`、`host.test.ts`
**依赖：** T09
**步骤：**
1. 创建宿主与 Shadow Root，注入样式。
2. 设置/更新 CSS 变量与 `data-theme`。
3. 向宿主文档注入 `mark.annot-mark` 样式（可更新 accent）。
4. `destroy()` 移除宿主与全局 mark 样式。

**验证：** jsdom 单测：Shadow Root、主题属性、销毁。

### T17 选区工具条

**文件：** `sdk/src/ui/toolbar.ts`、`toolbar.test.ts`
**依赖：** T16
**步骤：**
1. 渲染「批注」「复制引用」。
2. 根据选区 rect 定位，避让视口边缘。
3. 键盘可达（按钮可 Tab/Enter）。

**验证：** jsdom 单测：显示/隐藏、按钮回调、位置钳制。

### T18 批注编辑器

**文件：** `sdk/src/ui/composer.ts`、`composer.test.ts`
**依赖：** T16
**步骤：**
1. 引用文本、textarea、字数计数、发布/取消、错误区。
2. Ctrl/Cmd+Enter 提交；pending 禁用；Esc 取消。
3. 关闭后释放焦点。

**验证：** jsdom 单测：提交内容、pending、错误、快捷键。

### T19 确认框

**文件：** `sdk/src/ui/confirm.ts`、`confirm.test.ts`
**依赖：** T14、T16
**步骤：**
1. `confirm(options): Promise<boolean>`。
2. 焦点陷阱、Esc 取消、危险按钮样式。
3. 关闭后恢复焦点。

**验证：** jsdom 单测：确认/取消/Esc、Promise 结果。

### T20 抽屉、筛选与列表

**文件：** `sdk/src/ui/drawer.ts`、`drawer.test.ts`
**依赖：** T07、T08、T16、T17、T18、T19
**步骤：**
1. 头部：标题、数量、筛选标签（全部/我的/已解决/待审）、账号入口、关闭。
2. 卡片：作者头像/首字母、相对时间、状态徽章、评论、操作按钮。
3. 加载骨架、空态、错误重试、加载更多。
4. 窄屏底部面板样式；当前批注卡片高亮。

**验证：** jsdom 单测：筛选回调、卡片渲染、加载更多、空态。

### T21 账号面板

**文件：** `sdk/src/ui/account.ts`、`account.test.ts`
**依赖：** T11、T14、T16、T19
**步骤：**
1. 账号信息、退出。
2. 身份列表 + 绑定按钮（已绑定隐藏）+ 解绑。
3. 会话列表（当前标记）+ 撤销单个/全部。
4. 主题切换、语言切换入口。
5. 封禁/只读状态提示。

**验证：** jsdom 单测：已绑定过滤、解绑回调、会话渲染、撤销回调。

### T22 UI 门面与样式

**文件：** `sdk/src/ui/ui.ts`、`styles.css`
**依赖：** T16–T21
**步骤：**
1. 组合组件，暴露 `setState/setLocale/showToolbar/openComposer/confirm/toast/announce/...`。
2. 样式覆盖桌面/移动、亮/暗、对比度、触控目标 ≥44px。
3. 不修改宿主全局样式。

**验证：** 类型检查 + 单测门面方法调用正确。

### T23 登录面板

**文件：** `sdk/src/ui/account.ts` 或独立 `ui/login.ts`
**依赖：** T21、T05
**步骤：**
1. 未登录时展示 provider 列表（GitHub/OIDC）。
2. 已绑定判断使用 issuer；devLogin 仅在开启时显示。
3. 选择后回调 `onLogin(providerId)`。

**验证：** jsdom 单测：provider 渲染、已绑定隐藏、回调参数。

---

## 阶段 3：编排与交互

### T24 实例初始化与列表加载

**文件：** `sdk/src/index.ts`
**依赖：** T07–T13、T22
**步骤：**
1. 初始化：apiBase、client、theme、host/UI、事件。
2. 消费 hash token；拉 site-config；应用 locale/theme；拉 me/identities。
3. `loadList(reset)` 支持 filter/cursor；追加去重；更新 orphan 与 activeId。

**验证：** 单测/mock：初始化调用顺序、首屏加载、筛选重置。

### T25 选区、工具条与创建

**文件：** `sdk/src/index.ts`
**依赖：** T24、T17、T18
**步骤：**
1. mouseup/touchend/selectionchange 防抖捕获选区。
2. 未登录 → 登录面板；已登录 → 编辑器。
3. 发布：describeSelection → create → 重新加载 → Toast → 事件。

**验证：** jsdom 单测：选区捕获、发布调用、错误保留内容。

### T26 回复、解决、删除与复制

**文件：** `sdk/src/index.ts`
**依赖：** T24、T19、T20
**步骤：**
1. 回复后刷新当前页（保持 filter/cursor 已加载数据）。
2. 解决/重新打开。
3. 删除走 `confirm()`；成功 Toast + `deleted` 事件。
4. 复制链接：`buildLink` + `navigator.clipboard`，失败降级提示。

**验证：** 单测：confirm 拒绝不调用 API；复制降级路径。

### T27 账号、绑定与会话

**文件：** `sdk/src/index.ts`
**依赖：** T21、T23
**步骤：**
1. 打开账号时懒加载 sessions。
2. 绑定/解绑后刷新 identities。
3. 撤销会话/退出所有；撤销当前会话后回到未登录。
4. 退出登录清理 token、identities、sessions。

**验证：** 单测：懒加载、撤销当前后的状态、绑定刷新。

### T28 orphan 重定位

**文件：** `sdk/src/index.ts`、`anchor.ts`
**依赖：** T24、T11
**步骤：**
1. 进入定位模式：收起抽屉、显示提示、记录目标批注 id。
2. 下一次选区生成 anchor → `PATCH /annotations/{id}`。
3. 成功：退出模式、重建高亮、Toast、`updated` 事件；失败保留原数据。
4. Esc/取消退出模式。

**验证：** 单测：定位模式状态机、成功/失败/取消。

### T29 深链接

**文件：** `sdk/src/index.ts`
**依赖：** T10、T24
**步骤：**
1. 初始化时解析 `#annot-<id>` 并在列表加载后定位。
2. 登录回跳 token 消费后保留深链接。
3. 复制链接不包含 token。

**验证：** 单测：有/无深链接、token+深链接共存。

### T30 生命周期、主题与事件

**文件：** `sdk/src/index.ts`
**依赖：** T24
**步骤：**
1. `setPage` 清理高亮、重置 cursor/filter 与列表。
2. `destroy` 移除监听、UI、mark 样式。
3. 主题/语言切换实时更新 UI。
4. 事件载荷与取消订阅。

**验证：** 单测：setPage、destroy、重复 init 不叠加。

### T31 示例页更新

**文件：** `examples/plain-html/index.html`
**依赖：** T24–T30
**步骤：**
1. 演示筛选、账号面板、主题切换、复制链接。
2. 保留 `?siteKey=&api=` 参数与 devLogin。

**验证：** 浏览器手动检查（E2E 覆盖）。

---

## 阶段 4：测试与验收

### T32 SDK 单元测试汇总

**文件：** `sdk/src/**/*.test.ts`
**依赖：** T07–T30
**步骤：**
1. 覆盖 state、format、theme、deeplink、client、focus-trap、toast、confirm、drawer、account。
2. 确保无网络依赖。

**验证：** `npm test` 全绿。

### T33 Playwright 主流程

**文件：** `sdk/e2e/sdk.spec.ts`、`sdk/playwright.config.ts`、`sdk/package.json`
**依赖：** T31、T06
**步骤：**
1. 在 `sdk/package.json` 增加 `@playwright/test` 与 `test:e2e` 脚本；创建 Playwright 配置。
2. 测试前置：启动服务、CLI 创建站点、启动 plain-html 静态服务器。
3. 创建站点、登录、划线、发布、筛选切换、加载更多。
4. 刷新恢复；复制链接后新开页面定位。
5. 解决/重新打开/回复/删除。

**验证：** `npx playwright test sdk.spec.ts` 通过。

### T34 Playwright 账号与会话

**文件：** `sdk/e2e/account.spec.ts`
**依赖：** T27、T33
**步骤：**
1. 多次 dev 登录制造多个会话。
2. 打开账号面板：会话列表、当前标记、撤销单个、退出所有。
3. 身份列表渲染；已绑定 provider 不显示绑定按钮（OIDC 可用 API 预置）。
4. 退出后 UI 回到未登录。

**验证：** Playwright 通过。

### T35 Playwright 移动端与可访问性

**文件：** `sdk/e2e/mobile.spec.ts`
**依赖：** T33
**步骤：**
1. 390×844 视口：抽屉全宽、触控目标 ≥44px。
2. 键盘：Tab 到工具条/抽屉、Esc 关闭、焦点恢复。
3. `aria-live` 在创建/错误时更新。

**验证：** Playwright 通过。

### T36 构建与体积

**文件：** `sdk/`、`internal/sdkassets/`
**依赖：** T32–T35
**步骤：**
1. `npm run typecheck && npm test && npm run build`。
2. 检查 IIFE gzip ≤ 100KB。
3. `make sdk` 复制产物，`go build ./...`。

**验证：** 命令通过，记录 gzip 体积。

### T37 全量回归

**文件：** 全部
**依赖：** T36
**步骤：**
1. `gofmt -l .`、`go vet ./...`、`go test ./...`。
2. SDK typecheck/单测/build。
3. `make build`；plain-html 与 VitePress E2E。
4. 博客 `npm run docs:build`。

**验证：** 全部通过并记录证据。

---

## 执行顺序

```
后端：T01 → T02 → T03 → T04 → T05 → T06
                 ↘
前端基础：T07 → T08 → T09 → T10 → T11 → T12 → T13 → T14 → T15
                                      ↘
UI：T16 → T17 → T18 → T19 → T20 → T21 → T22 → T23
                                      ↘
编排：T24 → T25 → T26 → T27 → T28 → T29 → T30 → T31
                                      ↘
测试：T32 → T33 → T34 → T35 → T36 → T37
```
