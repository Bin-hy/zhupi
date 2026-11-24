# Annotation Service 管理后台 Web UI · Tasks

> 状态：草案 v0.1（等待评审）
> 输入：`.specs/admin-console/spec.md`、`plan.md`
> 说明：T01–T10 为后端补充；T11 起为 Vue 管理后台；T26 起为构建、E2E 与验收。

## 文件清单

### 后端（新建 / 修改）

| 操作 | 文件 | 职责 |
|---|---|---|
| 修改 | `internal/store/interfaces.go` | 新增 provider/admin/stats 仓储方法 |
| 修改 | `internal/store/provider_repo.go` | 全局 provider 查询 |
| 修改 | `internal/store/admin_repo.go` | 可管理站点、站点封禁列表 |
| 新建 | `internal/store/stats_repo.go` | 概览统计与最近待审 |
| 新建 | `internal/store/stats_repo_test.go` | 统计测试 |
| 修改 | `internal/service/auth.go` | 管理端登录、provider 列表、admin 回跳校验 |
| 修改 | `internal/service/admin.go` | AccessibleSites、Overview |
| 新建 | `internal/service/admin_test.go` | 角色范围与概览测试 |
| 新建 | `internal/httpapi/admin_auth_handlers.go` | `/admin/auth/*` |
| 新建 | `internal/httpapi/admin_overview_handlers.go` | `/me/sites`、`/admin/overview` |
| 修改 | `internal/httpapi/admin_handlers.go` | 站点封禁列表 |
| 修改 | `internal/httpapi/router.go` | 新路由与 `/admin` 静态托管 |
| 新建 | `internal/adminassets/embed.go` | `go:embed` 后台产物 |
| 新建 | `internal/httpapi/admin_static.go` | SPA fallback |
| 修改 | `internal/httpapi/server_test.go` | 管理端 API 集成测试 |

### 前端（新建）

| 文件 | 职责 |
|---|---|
| `admin/package.json` | Vue 3 / Element Plus / Router / Vite |
| `admin/vite.config.ts` | base `/admin/`、构建输出 |
| `admin/tsconfig.json` | TS strict |
| `admin/index.html` | 入口 |
| `admin/src/main.ts` | 挂载 Vue、Element Plus、Router |
| `admin/src/App.vue` | 根组件 |
| `admin/src/router.ts` | 路由与守卫 |
| `admin/src/api/types.ts` | 与后端对齐的类型 |
| `admin/src/api/client.ts` | 请求封装、token、401 |
| `admin/src/api/index.ts` | 业务 API 方法 |
| `admin/src/stores/auth.ts` | 登录态、用户、可管理站点 |
| `admin/src/layout/AdminLayout.vue` | 侧边栏 + 顶栏 |
| `admin/src/views/LoginView.vue` | 登录页 |
| `admin/src/views/OverviewView.vue` | 概览 |
| `admin/src/views/SitesView.vue` | 站点列表 |
| `admin/src/views/SiteDetailView.vue` | 设置/管理员/提供商/封禁 |
| `admin/src/views/ModerationView.vue` | 内容审核 |
| `admin/src/views/UsersView.vue` | 用户管理 |
| `admin/src/views/NotFoundView.vue` | 404 |
| `admin/src/components/StatCard.vue` | 统计卡片 |
| `admin/src/components/StatusTag.vue` | 状态标签 |
| `admin/src/components/SiteSelector.vue` | 站点选择器 |
| `admin/src/components/LoadMore.vue` | 加载更多 |
| `admin/src/components/EmptyState.vue` | 空态 |
| `admin/src/components/ErrorState.vue` | 错误态 |
| `admin/src/components/SiteFormDialog.vue` | 站点新建/编辑 |
| `admin/src/components/ProviderFormDialog.vue` | OIDC 提供商表单 |
| `admin/src/components/BanDialog.vue` | 站点封禁 |
| `admin/src/utils/format.ts` | 时间/文本格式化 |
| `admin/src/styles/index.css` | 全局样式与暗色 |
| `admin/src/**/*.test.ts` | 单测 |
| `admin/e2e/console.mjs` | Playwright 主流程 |

### 构建与文档

| 操作 | 文件 | 改动 |
|---|---|---|
| 修改 | `Makefile` | 新增 `admin` 目标，`build` 串联 sdk + admin + go |
| 修改 | `Dockerfile` | Node 阶段构建 sdk 与 admin |
| 修改 | `.gitignore` / `.dockerignore` | 忽略 admin 依赖/产物 |
| 修改 | `README.md` | 管理后台入口、开发与部署说明 |

---

## 阶段 0：后端补充

### T01 全局 Provider 查询

**文件：** `internal/store/interfaces.go`、`provider_repo.go`
**依赖：** 无
**步骤：**
1. 新增 `ProviderByIDAny(ctx,id)`。
2. 新增 `ListEnabledWithSite(ctx)`，返回 provider + siteName，供登录页展示。
3. 保持现有按站点方法不变。

**验证：** 单测：跨站点按 ID 查询；只返回启用 provider 且带站点名。

### T02 可管理站点与封禁列表

**文件：** `internal/store/interfaces.go`、`admin_repo.go`
**依赖：** 无
**步骤：**
1. 新增 `ListAdminSiteIDs(ctx,userID)`。
2. 新增 `ListSitesForAdmin(ctx,userID)`（join site_admins + sites）。
3. 新增 `ListSiteBans(ctx,siteID)`，join users 返回名称/头像。
4. 定义 `SiteBanView`。

**验证：** 单测：多站点分配、解绑后不返回、封禁列表字段完整。

### T03 概览统计仓储

**文件：** `internal/store/stats_repo.go`、`stats_repo_test.go`
**依赖：** T02
**步骤：**
1. 实现 CountSites、CountUsers、CountAnnotations、CountComments、CountDistinctAuthors。
2. 支持 `siteIDs` 过滤；空集合返回 0。
3. 实现 RecentPendingAnnotations / RecentPendingComments（各 limit 10）。
4. 动态 IN 占位符，避免 SQL 注入。

**验证：** 单测：跨站点统计、空站点集合、只统计 pending。

### T04 管理端登录服务

**文件：** `internal/service/auth.go`、`auth_test.go`
**依赖：** T01
**步骤：**
1. `StartAdminGitHub(returnTo)`：生成 login flow，siteID=0。
2. `StartAdminOIDC(providerID, returnTo)`：ProviderByIDAny；启用校验；siteID=provider.SiteID。
3. `safeAdminReturnTo`：只允许 PublicBaseURL 同源且路径以 `/admin` 开头。
4. `AdminProviders`：GitHub 配置状态 + 启用 OIDC 列表。

**验证：** 单测：合法/非法回跳、GitHub 未配置错误、OIDC provider 未启用拒绝。

### T05 可管理站点与概览服务

**文件：** `internal/service/admin.go`、`admin_test.go`
**依赖：** T02、T03
**步骤：**
1. `AccessibleSites(principal)`：超管全部，否则 ListSitesForAdmin。
2. `Overview(principal)`：计算站点集合与计数；普通用户全 0。
3. 站点管理员 users 口径为去重作者数；超管为全局用户数。
4. 最近待审各 10 条。

**验证：** 单测：超管/站点管理员/普通用户三种结果。

### T06 管理端登录 HTTP

**文件：** `internal/httpapi/admin_auth_handlers.go`、`router.go`
**依赖：** T04
**步骤：**
1. `GET /api/v1/admin/auth/providers`（公开）。
2. `GET /api/v1/admin/auth/github/start`。
3. `GET /api/v1/admin/auth/oidc/{providerID}/start`。
4. 302 到 IdP；错误使用统一错误格式。

**验证：** httptest：providers 返回结构；未配置 GitHub 返回 503；非法 redirect 回落到 `/admin/`。

### T07 概览与封禁 HTTP

**文件：** `internal/httpapi/admin_overview_handlers.go`、`admin_handlers.go`、`router.go`
**依赖：** T05、T02
**步骤：**
1. `GET /api/v1/me/sites`（Bearer）。
2. `GET /api/v1/admin/overview`（Bearer）。
3. `GET /api/v1/admin/sites/{siteID}/bans`（站点管理员/超管）。
4. 统一错误与权限校验。

**验证：** httptest：角色范围、跨站点 403、空站点返回 0。

### T08 后台静态资源托管

**文件：** `internal/adminassets/embed.go`、`internal/httpapi/admin_static.go`
**依赖：** 无
**步骤：**
1. `//go:embed all:dist` + `.gitkeep` 占位。
2. `GET /admin` 重定向到 `/admin/`。
3. `GET /admin/{path...}`：文件存在则按扩展名返回；否则返回 index.html。
4. index 不缓存；hash 资源长缓存。

**验证：** 启动服务后 `/admin/` 返回 HTML；`/admin/unknown/route` 返回 index；`/admin/assets/x.js` 正常。

### T09 后端集成测试

**文件：** `internal/httpapi/server_test.go`、`internal/service/admin_test.go`
**依赖：** T01–T08
**步骤：**
1. 覆盖管理端 provider 列表、登录回跳、`/me/sites`、overview、封禁列表。
2. 覆盖站点管理员跨站 403、普通用户空概览。
3. 覆盖统计口径。

**验证：** `go test ./...` 全绿。

### T10 后端回归

**文件：** 后端全部
**依赖：** T09
**步骤：**
1. `gofmt`、`go vet`、`go test ./...`。
2. 确认原有多站点/SDK 接口不受影响。

**验证：** 命令全部通过。

---

## 阶段 1：前端骨架与基础设施

### T11 Vue 工程脚手架

**文件：** `admin/package.json`、`vite.config.ts`、`tsconfig.json`、`index.html`、`src/main.ts`、`App.vue`
**依赖：** 无
**步骤：**
1. 初始化 Vue 3 + Vite + TS。
2. 依赖：vue、vue-router、element-plus、@element-plus/icons-vue。
3. devDeps：vite、@vitejs/plugin-vue、typescript、vue-tsc、vitest、jsdom、playwright-core。
4. Vite `base: '/admin/'`，开发代理 `/api` 到 `http://127.0.0.1:8080`。
5. main.ts 挂载 Element Plus 与 Router。

**验证：** `npm install && npm run build` 产出 `dist/index.html` 与 assets。

### T12 API 类型与客户端

**文件：** `src/api/types.ts`、`client.ts`、`index.ts`、`client.test.ts`
**依赖：** T11
**步骤：**
1. 定义 Site/User/Provider/Ban/Overview/Queue 等类型。
2. `AdminApi` 封装 GET/POST/PATCH/DELETE，自动 Bearer。
3. 401 时清理 token 并触发 `auth:expired` 事件。
4. 错误归一为 `{status,code,message}`。
5. 覆盖所有业务方法。

**验证：** Vitest：header、401、错误映射、关键 URL。

### T13 认证 Store 与路由守卫

**文件：** `src/stores/auth.ts`、`src/router.ts`、`auth.test.ts`
**依赖：** T12
**步骤：**
1. token key `annotation-admin:token`；消费 `#annot_token` 并清理 URL。
2. `loadSession()`：/me + /me/sites。
3. `login(providerId)` / `logout()`。
4. 路由：/login、/、/sites、/sites/:id、/moderation、/users。
5. 守卫：未登录→/login；超管路由；站点访问校验；登录后回跳。

**验证：** 单测：token 消费、角色判断、站点访问；手动：未登录跳转。

### T14 后台布局

**文件：** `src/layout/AdminLayout.vue`、`src/styles/index.css`
**依赖：** T13
**步骤：**
1. 侧边栏菜单（按角色显示用户管理）。
2. 顶栏用户信息、站点数、退出。
3. 内容区 router-view；窄屏折叠菜单。
4. Element Plus 暗色跟随系统。

**验证：** 浏览器手动检查布局与菜单权限。

### T15 登录页

**文件：** `src/views/LoginView.vue`
**依赖：** T12、T13
**步骤：**
1. 拉取 `/admin/auth/providers`。
2. GitHub 按钮与 OIDC 列表。
3. 点击跳转管理端登录入口。
4. 无登录方式时显示配置提示；loading/error。

**验证：** 浏览器手动：未配置 GitHub 时提示；配置后跳转。

---

## 阶段 2：业务页面

### T16 概览页

**文件：** `src/views/OverviewView.vue`、`components/StatCard.vue`、`StatusTag.vue`、`EmptyState.vue`、`ErrorState.vue`
**依赖：** T14、T12
**步骤：**
1. 6 个统计卡片。
2. 最近待审批注/评论表格。
3. 空态与重试。
4. 点击「去审核」跳转 `/moderation?site=`。

**验证：** 不同角色数据正确；无数据/失败态。

### T17 站点列表与表单

**文件：** `src/views/SitesView.vue`、`components/SiteFormDialog.vue`
**依赖：** T14、T12
**步骤：**
1. 表格：名称、展示名、siteKey、状态、审核模式、来源、时间。
2. 复制/轮换 siteKey（确认）。
3. 新建/编辑对话框（超管）。
4. 来源动态增删、审核模式、主题色、语言、状态。
5. 站点管理员只读。

**验证：** 创建/编辑/轮换生效；旧 key 失效。

### T18 站点详情与设置

**文件：** `src/views/SiteDetailView.vue`
**依赖：** T17
**步骤：**
1. 读取 `:id`，加载站点。
2. Tab：设置 / 管理员 / OIDC / 封禁。
3. 设置 Tab 复用表单（超管）。
4. 非法/无权站点显示错误。

**验证：** 深链 `/admin/sites/1?tab=providers` 刷新正常。

### T19 管理员 Tab

**文件：** `src/views/SiteDetailView.vue`
**依赖：** T18
**步骤：**
1. 列表：用户 ID、授予时间。
2. 添加管理员（用户 ID/搜索）。
3. 移除确认。

**验证：** 添加/移除后列表更新；不存在用户报错。

### T20 OIDC 提供商 Tab

**文件：** `src/views/SiteDetailView.vue`、`components/ProviderFormDialog.vue`
**依赖：** T18
**步骤：**
1. 列表：名称、issuer、clientId、状态、更新时间。
2. 新增/编辑：secret 留空保持原值。
3. 删除/停用确认。
4. Discovery 失败展示后端错误。

**验证：** 新增后 site-config 可见；secret 不回显。

### T21 封禁 Tab

**文件：** `src/views/SiteDetailView.vue`、`components/BanDialog.vue`
**依赖：** T18
**步骤：**
1. 列表：用户、原因、时间。
2. 通过用户 ID 添加封禁。
3. 解除封禁确认。

**验证：** 封禁后该用户该站点写请求 403；其他站点不受影响。

### T22 内容审核页

**文件：** `src/views/ModerationView.vue`、`components/SiteSelector.vue`、`LoadMore.vue`
**依赖：** T14、T12
**步骤：**
1. 站点选择器（来自 /me/sites）。
2. 状态标签：待审/已通过/已拒绝。
3. 批注表与评论表。
4. 操作：通过/拒绝/隐藏/恢复，二次确认。
5. 游标加载更多。

**验证：** 审核后列表与公开可见性变化；跨站 403。

### T23 用户管理页

**文件：** `src/views/UsersView.vue`
**依赖：** T14、T12
**步骤：**
1. 表格：ID、名称、状态、超管、时间。
2. 客户端搜索。
3. 全局封禁/解封（确认提示会话失效）。
4. 仅超管可见。

**验证：** 封禁后用户会话失效；站点管理员访问被拒。

### T24 全站样式与共享组件收口

**文件：** `src/components/*`、`src/styles/index.css`
**依赖：** T16–T23
**步骤：**
1. 统一表格、按钮、间距、空态、错误态。
2. 状态标签颜色。
3. 暗色对比度检查。
4. 窄屏表格横向滚动。

**验证：** 桌面/窄屏视觉检查。

### T25 前端单元测试

**文件：** `src/**/*.test.ts`
**依赖：** T12、T13
**步骤：**
1. API client：header/401/错误。
2. auth store：token 消费、角色、站点访问。
3. format：时间/文本。

**验证：** `npm test` 全绿。

---

## 阶段 3：构建、托管与验收

### T26 构建与内嵌

**文件：** `Makefile`、`Dockerfile`、`.gitignore`、`.dockerignore`、`internal/adminassets/embed.go`
**依赖：** T11、T08
**步骤：**
1. `make admin`：npm ci + build + 复制到 `internal/adminassets/dist`。
2. `make build` 串联 sdk → admin → go。
3. Docker Node 阶段同时构建 sdk 与 admin。
4. 忽略 admin/node_modules、admin/dist、internal/adminassets/dist。

**验证：** `make build` 后 `/admin/` 可访问；Dockerfile 结构正确。

### T27 管理后台 E2E

**文件：** `admin/e2e/console.mjs`
**依赖：** T26、T10
**步骤：**
1. 用 CLI 创建站点 + 预置待审批注/用户。
2. 通过 `/api/v1/auth/dev/login` 获取 token，注入 `annotation-admin:token`。
3. 验证：概览、站点列表、站点详情、审核通过、用户封禁。
4. 验证登录页（无 token）渲染 provider 列表/空态。
5. 深链刷新不 404。

**验证：** `node e2e/console.mjs` 通过。

### T28 全量回归与文档

**文件：** 全部
**依赖：** T27
**步骤：**
1. `gofmt`、`go vet`、`go test ./...`。
2. `cd admin && npm run typecheck && npm test && npm run build`。
3. `make build`；`/admin/` 与 `/sdk/v1/annotation-sdk.js` 均可访问。
4. 博客 `npm run docs:build`。
5. README 增加管理后台入口、开发命令、环境变量与权限说明。

**验证：** 全部命令通过并记录证据。

---

## 执行顺序

```
后端：T01 → T02 → T03 → T04 → T05 → T06 → T07 → T08 → T09 → T10
前端：T11 → T12 → T13 → T14 → T15 → T16 → T17 → T18
                                              ├→ T19
                                              ├→ T20
                                              └→ T21
             T14 → T22（审核）
             T14 → T23（用户）
T16–T23 → T24 → T25
T10 + T11 + T08 → T26 → T27 → T28
```
