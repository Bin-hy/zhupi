# VitePress 站内接入打磨 · Tasks

> 状态：草案 v0.1（等待评审）
> 输入：`.specs/vitepress-integration/spec.md`、`plan.md`
> 说明：T01–T04 后端 total；T05–T13 SDK；T14–T18 博客接入；T19–T20 回归与文档。

## 文件清单

### 后端

| 操作 | 文件 | 职责 |
|---|---|---|
| 修改 | `internal/store/interfaces.go` | 新增 `CountPageAnnotations` |
| 修改 | `internal/store/annotation_repo.go` | 页面总数 SQL |
| 修改 | `internal/store/annotation_repo_test.go` | 计数测试 |
| 修改 | `internal/service/annotation.go` | `PageResult.Total` |
| 修改 | `internal/service/annotation_test.go` | filter 与 total |
| 修改 | `internal/httpapi/server_test.go` | total 响应 |

### SDK

| 操作 | 文件 | 职责 |
|---|---|---|
| 修改 | `sdk/src/state.ts` | `total` 字段 |
| 修改 | `sdk/src/client.ts` | 列表响应 total |
| 修改 | `sdk/src/events.ts` | `statechange` 事件 |
| 修改 | `sdk/src/index.ts` | 公共 API、commitState、selectRoot 重置 |
| 修改 | `sdk/src/ui/host.ts` | `cssVariables` |
| 修改 | `sdk/src/ui/styles.css` | 抽屉/FAB CSS 变量 |
| 修改 | `sdk/src/**/*.test.ts` | 新 API/状态测试 |

### 博客（go-campus）

| 操作 | 文件 | 职责 |
|---|---|---|
| 新建 | `docs/.vitepress/theme/annotation-bridge.ts` | 桥接状态与动作 |
| 新建 | `docs/.vitepress/theme/AnnotationNav.vue` | 顶部「批注 N」+ 登录/头像 |
| 修改 | `docs/.vitepress/theme/index.ts` | 单实例、暗色、导航 slot、错误降级 |
| 修改 | `docs/.vitepress/theme/custom.css` | 必要的全局微调（如有） |
| 修改 | `README.md` | 说明顶部入口与体验 |

### 测试

| 操作 | 文件 | 职责 |
|---|---|---|
| 修改 | `sdk/e2e/vitepress.mjs` | 导航、计数、暗色、SPA、首页、移动端 |

---

## 阶段 0：后端 total

### T01 仓储页面计数

**文件：** `internal/store/interfaces.go`、`annotation_repo.go`、`annotation_repo_test.go`
**依赖：** 无
**步骤：**
1. 新增 `CountPageAnnotations(ctx, siteID, pageKey, viewerID, includeUnapproved) (int,error)`。
2. SQL 与 `filter=all` 可见性一致，忽略状态与分页。
3. 空 pageKey 返回 0；跨站点不计数。

**验证：** 单测：公开/待审/作者/管理员四种计数正确。

### T02 服务层返回 total

**文件：** `internal/service/annotation.go`、`annotation_test.go`
**依赖：** T01
**步骤：**
1. `PageResult` 增加 `Total int json:"total"`。
2. `ListPage` 在列表查询后调用计数。
3. filter=mine/resolved/pending 时 total 仍为页面全部可见数。

**验证：** service 单测：切换 filter 与 cursor 时 total 不变。

### T03 HTTP 响应测试

**文件：** `internal/httpapi/server_test.go`
**依赖：** T02
**步骤：**
1. 断言列表响应包含 `total`。
2. 覆盖匿名、作者、管理员三种 total。

**验证：** `go test ./internal/...` 通过。

### T04 后端回归

**文件：** 后端全部
**依赖：** T03
**步骤：**
1. `gofmt`、`go vet`、`go test ./...`。
2. 确认现有列表/筛选测试不受影响。

**验证：** 命令全部通过。

---

## 阶段 1：SDK 增补

### T05 状态 total

**文件：** `sdk/src/state.ts`、`state.test.ts`
**依赖：** 无
**步骤：**
1. `SDKState` 增加 `total: number`，初始 0。
2. `mergeState` 自然支持。

**验证：** 初始状态 total=0。

### T06 客户端 total

**文件：** `sdk/src/client.ts`、`client.test.ts`
**依赖：** T05
**步骤：**
1. `listAnnotations` 返回类型增加 `total`。
2. 单元测试 mock 响应带 total。

**验证：** 类型检查 + 单测。

### T07 statechange 事件

**文件：** `sdk/src/events.ts`、`events.test.ts`（可新建）
**依赖：** T05
**步骤：**
1. `EventMap` 增加 `statechange: { user, annotationCount, loggedIn }`。
2. 保持现有事件不变。

**验证：** 类型检查 + 订阅/取消测试。

### T08 commitState 与事件发射

**文件：** `sdk/src/index.ts`
**依赖：** T07
**步骤：**
1. 新增 `commitState()`：更新 UI state 并 emit statechange。
2. 将现有 `this.ui.setState(this.state)` 调用替换为 `commitState()`。
3. `loadList` 使用响应 total（reset 或 append）。

**验证：** 单测/手动：登录、加载、创建、删除后事件载荷正确。

### T09 selectRoot 重新解析

**文件：** `sdk/src/index.ts`
**依赖：** 无
**步骤：**
1. 保存 `selectRoot` 原始选项。
2. 新增 `resolveSelectRoot()`：字符串每次查询；元素失效回退 body。
3. `setPage` 与 `loadList(reset)` 在生成高亮前重新解析。
4. `rehighlight` 使用最新 selectRoot。

**验证：** jsdom 单测：替换 `.vp-doc` 节点后 setPage 能在新节点高亮。

### T10 公共实例 API

**文件：** `sdk/src/index.ts`
**依赖：** T08
**步骤：**
1. 新增 `setTheme(mode)`。
2. 新增 `openDrawer()`、`openAccount()`。
3. 新增 `getAnnotationCount()`。
4. 更新导出的 `AnnotationInstance` 类型。

**验证：** 类型检查；单测/浏览器冒烟。

### T11 CSS 变量

**文件：** `sdk/src/ui/host.ts`、`sdk/src/ui/ui.ts`、`sdk/src/ui/styles.css`、`sdk/src/index.ts`
**依赖：** 无
**步骤：**
1. `InitOptions.cssVariables`，Host 写入宿主 style。
2. 抽屉 `top/height` 使用 `--annotation-drawer-top`。
3. FAB `bottom/right` 使用 `--annotation-fab-bottom/right`。
4. 默认值保持现有行为。

**验证：** 单测/浏览器：设置变量后位置变化。

### T12 SDK 单测补充

**文件：** `sdk/src/**/*.test.ts`
**依赖：** T05–T11
**步骤：**
1. total 透传。
2. statechange 载荷。
3. selectRoot 重置。
4. cssVariables。

**验证：** `npm test` 全绿。

### T13 SDK 构建

**文件：** `sdk/`
**依赖：** T12
**步骤：**
1. `npm run typecheck`。
2. `npm run build`。
3. `make sdk` 复制到 `internal/sdkassets/dist`。

**验证：** 构建成功，gzip 体积记录。

---

## 阶段 2：博客接入

### T14 桥接状态

**文件：** `docs/.vitepress/theme/annotation-bridge.ts`
**依赖：** 无
**步骤：**
1. reactive 状态：status / hasDoc / user / count / actions。
2. 提供 reset / setReady / setError 辅助函数。

**验证：** 类型检查；博客构建通过。

### T15 顶部导航组件

**文件：** `docs/.vitepress/theme/AnnotationNav.vue`
**依赖：** T14、T10
**步骤：**
1. 仅 hasDoc 时渲染。
2. error → 禁用「批注不可用」；loading → loading。
3. ready → 「批注 N」+ 登录/头像。
4. 点击调用 bridge.actions。
5. 使用 VitePress CSS 变量，移动端隐藏用户名。

**验证：** 手动/Playwright：桌面与窄屏显示正确。

### T16 主题生命周期接入

**文件：** `docs/.vitepress/theme/index.ts`
**依赖：** T14、T15、T13
**步骤：**
1. 维护单实例与脚本加载 Promise。
2. 路由 nextTick：无 `.vp-doc` → destroy + reset；有且已有实例 → setPage；有且无实例 → init。
3. init 参数：selectRoot、theme、cssVariables、事件。
4. watch(isDark) → instance.setTheme。
5. 错误 → bridge.status='error'；下次路由重试。
6. Layout 增加 `nav-bar-content-after` 渲染 AnnotationNav。
7. onUnmounted destroy。

**验证：** 手动/Playwright：文档页切换不重复实例、首页隐藏、暗色同步。

### T17 VitePress E2E 扩展

**文件：** `sdk/e2e/vitepress.mjs`
**依赖：** T16
**步骤：**
1. 等待 `.annotation-nav`，断言「批注 N」。
2. 点击导航按钮打开抽屉（不再依赖 FAB）。
3. 切换 VitePress 暗色，断言 SDK host `data-theme` 变化。
4. SPA 切换到另一文档页，断言只有一个 `.annotation-sdk-host` 且旧高亮清除。
5. 回到首页，断言导航入口隐藏；再回文档页恢复。
6. 390×844 视口检查导航与抽屉布局。

**验证：** 脚本通过。

### T18 博客构建与手动检查

**文件：** go-campus
**依赖：** T16
**步骤：**
1. `npm run docs:build`。
2. 检查 Artalk、Viewer、mermaid、搜索无明显回归。

**验证：** 构建通过；关键页面手动检查。

---

## 阶段 3：回归与文档

### T19 全量回归

**文件：** 全部
**依赖：** T04、T13、T17、T18
**步骤：**
1. `gofmt -l .`、`go vet ./...`、`go test ./...`。
2. `cd sdk && npm run typecheck && npm test && npm run build`。
3. `cd admin && npm run typecheck && npm test && npm run build`。
4. `make build`。
5. 博客 `npm run docs:build`。
6. VitePress E2E 与 SDK/管理后台既有 E2E 冒烟。

**验证：** 全部通过并记录证据。

### T20 文档与提交

**文件：** `README.md`（annotation-service）、`go-campus/README.md`
**依赖：** T19
**步骤：**
1. 说明顶部导航入口、暗色同步、单实例行为。
2. 记录 SDK 新公共 API。
3. 更新验收记录到 checklist。
4. 提交两个仓库的改动。

**验证：** `git status` 干净；提交信息符合规范。

---

## 执行顺序

```
后端：T01 → T02 → T03 → T04
SDK：T05 → T06 → T07 → T08 → T09 → T10 → T11 → T12 → T13
博客：T14 → T15 → T16 → T17 → T18
回归：T19 → T20
```
