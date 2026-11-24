# Annotation Service 多站点划线批注产品 · Tasks

> 状态：草案 v0.1（等待评审）
> 输入：`.specs/multi-site-annotation/spec.md`、`.specs/multi-site-annotation/plan.md`
> 说明：任务按「可独立验证的最小工作单元」拆分；同一阶段内无依赖的任务可并行。

## 文件清单

### 新建

| 文件 | 职责 |
|---|---|
| `cmd/annotation-service/main.go` | 子命令入口：`serve` / `admin` |
| `internal/config/config.go` | 环境变量读取与校验 |
| `internal/domain/entities.go` | 站点、用户、身份、批注等实体 |
| `internal/domain/enums.go` | 状态枚举与校验 |
| `internal/domain/errors.go` | 领域错误与错误码 |
| `internal/crypto/secretbox.go` | AES-256-GCM 加解密 |
| `internal/store/store.go` | DB 打开、事务、迁移执行 |
| `internal/store/migrations/0001_init.sql` | 初始表结构 |
| `internal/store/site_repo.go` | 站点与来源白名单 |
| `internal/store/user_repo.go` | 全局用户 |
| `internal/store/identity_repo.go` | 登录身份 |
| `internal/store/session_repo.go` | 会话 |
| `internal/store/authflow_repo.go` | 一次性登录流程 |
| `internal/store/annotation_repo.go` | 批注 |
| `internal/store/comment_repo.go` | 评论 |
| `internal/store/provider_repo.go` | 站点 OIDC 提供商 |
| `internal/store/admin_repo.go` | 站点管理员与封禁 |
| `internal/auth/github.go` | 全局 GitHub OAuth |
| `internal/auth/oidc.go` | 站点 OIDC + PKCE + Discovery |
| `internal/auth/session.go` | 会话 token 生成与哈希 |
| `internal/service/site.go` | 站点业务 |
| `internal/service/auth.go` | 登录、绑定、会话业务 |
| `internal/service/annotation.go` | 批注与评论业务 |
| `internal/service/moderation.go` | 审核、封禁、角色 |
| `internal/service/admin.go` | 站点/用户管理 |
| `internal/httpapi/router.go` | 路由注册 |
| `internal/httpapi/middleware.go` | 请求上下文、站点解析、鉴权 |
| `internal/httpapi/cors.go` | 动态 CORS |
| `internal/httpapi/errors.go` | 统一错误响应 |
| `internal/httpapi/site_handlers.go` | site-config |
| `internal/httpapi/auth_handlers.go` | 登录/绑定/会话 |
| `internal/httpapi/annotation_handlers.go` | 批注/评论 |
| `internal/httpapi/admin_handlers.go` | 管理 API |
| `internal/httpapi/sdk_handlers.go` | SDK 静态资源 |
| `internal/ratelimit/limiter.go` | 令牌桶限流 |
| `internal/sdkassets/embed.go` | `go:embed` SDK 产物 |
| `internal/sdkassets/dist/.gitkeep` | 占位，保证无 SDK 构建时也能编译 |
| `sdk/package.json` | SDK 依赖与脚本 |
| `sdk/tsconfig.json` | TS 配置 |
| `sdk/vite.config.ts` | ESM + IIFE 构建 |
| `sdk/vitest.config.ts` | 单测配置 |
| `sdk/src/index.ts` | `init()` 与实例 API |
| `sdk/src/client.ts` | HTTP 客户端、token 存储 |
| `sdk/src/anchor.ts` | 选区锚定与高亮 |
| `sdk/src/state.ts` | 实例状态 |
| `sdk/src/events.ts` | 事件系统 |
| `sdk/src/i18n.ts` | 中英文案 |
| `sdk/src/ui/host.ts` | Shadow DOM 宿主 |
| `sdk/src/ui/toolbar.ts` | 选区浮动工具条 |
| `sdk/src/ui/composer.ts` | 批注编辑框 |
| `sdk/src/ui/drawer.ts` | 批注侧栏 |
| `sdk/src/ui/login.ts` | 登录面板 |
| `sdk/src/ui/styles.css` | SDK 样式 |
| `sdk/src/*.test.ts` | SDK 单测 |
| `examples/plain-html/index.html` | 无框架接入示例 |
| `sdk/e2e/annotation.spec.ts` | Playwright 端到端 |
| `Makefile` | 构建/测试入口 |

### 重写 / 修改

| 文件 | 改动 |
|---|---|
| `go.mod` / `go.sum` | 增加 go-oidc、oauth2、sqlite 依赖 |
| `Dockerfile` | 增加 Node 构建 SDK 阶段 |
| `docker-compose.yml` | 环境变量与新镜像 |
| `.env.example` | 新配置项 |
| `README.md` | 产品化文档与 SDK 接入说明 |
| `deploy/*` | 保持反代示例，补充 SDK 静态路径说明 |

### 删除

| 文件 | 原因 |
|---|---|
| `cmd/server/main.go` | 被 `cmd/annotation-service` 取代 |
| `internal/auth/auth.go` / `auth_test.go` | 单站点认证重写 |
| `internal/config/config.go` | 配置项重写 |
| `internal/store/store.go` / `schema.sql` | 单站点模型重写 |
| `internal/httpapi/annotations.go` / `middleware.go` / `router.go` / `server_test.go` | 单站点 API 重写 |

> 旧实现已存在于 `go-campus/annotation-service`，本仓库删除后仍可回溯。

---

## 阶段 0：清理与骨架

### T00 移除旧单站点实现

**文件：** `cmd/`、`internal/`
**依赖：** 无
**步骤：**
1. 删除 `cmd/server/`。
2. 删除 `internal/auth/`、`internal/config/`、`internal/store/`、`internal/httpapi/`。
3. 保留 `Dockerfile`、`docker-compose.yml`、`deploy/`、`.env.example`、`README.md` 待后续重写。

**验证：** `find . -path ./.git -prune -o -name '*.go' -print` 无旧文件；`go.mod` 仍在。

### T01 依赖与模块骨架

**文件：** `go.mod`、`go.sum`
**依赖：** T00
**步骤：**
1. 保持模块名 `github.com/Bin-hy/go-campus/annotation-service`（发布前可统一改名）。
2. `go get github.com/coreos/go-oidc/v3 golang.org/x/oauth2 modernc.org/sqlite`。
3. 创建空目录 `cmd/annotation-service`、`internal/{domain,config,crypto,store,auth,service,httpapi,ratelimit,sdkassets}`。

**验证：** `go list -m all | grep -E 'go-oidc|oauth2|sqlite'` 输出三个依赖。

### T02 领域实体与错误

**文件：** `internal/domain/entities.go`、`enums.go`、`errors.go`
**依赖：** T01
**步骤：**
1. 定义 Site、User、Identity、Session、Anchor、Annotation、Comment、OIDCProvider、SiteAdmin、SiteBan、AuthFlow、Principal。
2. 定义枚举常量与 `Valid()` 方法；身份 `SiteID` 用 0 表示全局。
3. 定义 `ErrNotFound`、`ErrForbidden`、`ErrConflict`、`ErrInvalid` 及错误码映射。

**验证：** `go build ./internal/domain`。

### T03 配置加载

**文件：** `internal/config/config.go`、`config_test.go`
**依赖：** T01
**步骤：**
1. 读取 `ADDR`、`DB_PATH`、`PUBLIC_BASE_URL`、`SECRET_KEY`、`SESSION_TTL_HOURS`、`GITHUB_CLIENT_ID/SECRET`、`SUPERADMIN_GITHUB_LOGINS`、`TRUST_PROXY`、`ADMIN_ORIGINS`、`DEV_LOGIN`。
2. `SECRET_KEY` 必须为 32 字节（hex/base64 解码后）；缺失则启动失败。
3. 校验 `PUBLIC_BASE_URL` 为绝对 URL；提供默认值（本地 `http://127.0.0.1:8080`）。

**验证：** `go test ./internal/config`；缺少 SECRET_KEY 时返回错误。

### T04 密钥加密

**文件：** `internal/crypto/secretbox.go`、`secretbox_test.go`
**依赖：** T01
**步骤：**
1. 实现 `Seal(plain []byte) ([]byte, error)` 与 `Open(cipher []byte) ([]byte, error)`，AES-256-GCM，随机 nonce 前置。
2. 密钥长度校验。

**验证：** 单测覆盖往返、篡改失败、错误密钥失败。

### T05 会话 token 工具

**文件：** `internal/auth/session.go`、`session_test.go`
**依赖：** T01
**步骤：**
1. `NewToken() (raw, hash string, err error)`：32 字节随机 + SHA-256。
2. `HashToken(raw)` 与常量时间比较工具。

**验证：** 单测验证长度、哈希一致性、随机性。

### T06 存储与迁移

**文件：** `internal/store/store.go`、`migrations/0001_init.sql`、`store_test.go`
**依赖：** T02
**步骤：**
1. `Open(path)` 使用 modernc SQLite，开启 WAL、foreign_keys、busy_timeout。
2. 实现 `Migrate()`：按文件名顺序执行嵌入 SQL，记录 `schema_migrations`，可重复执行。
3. 0001 建表：sites、site_origins、users、identities、sessions、site_admins、site_bans、oidc_providers、auth_flows、annotations、comments。
4. 添加必要索引与唯一约束（含 identities 唯一键、site_origins 唯一键）。
5. 检测旧单站点库：存在 `annotations` 表但没有 `schema_migrations` 时返回明确错误，提示使用新数据库文件。

**验证：** 单测在临时库连续执行两次 `Migrate()` 均成功，表数量符合预期。

### T07 限流与请求基础设施

**文件：** `internal/ratelimit/limiter.go`、`limiter_test.go`
**依赖：** T01
**步骤：**
1. 实现按 key 的令牌桶 `Allow(key string) bool`。
2. 提供 `New(rate, burst)` 与容量上限保护。

**验证：** 单测验证突发额度与恢复速率。

### T08 服务入口与健康检查

**文件：** `cmd/annotation-service/main.go`
**依赖：** T03、T06、T07
**步骤：**
1. 子命令解析：默认 `serve`，另有 `admin`（占位）。
2. 初始化配置、DB、迁移、HTTP Server；优雅退出。
3. 注册 `GET /healthz` 与 `GET /readyz`（DB Ping）。
4. 结构化日志（`log/slog`）与 request id 占位。

**验证：** `DEV_LOGIN=1 SECRET_KEY=... go run ./cmd/annotation-service serve` 后 `curl /healthz`、`/readyz` 正常。

---

## 阶段 1：站点与隔离

### T09 站点仓储

**文件：** `internal/store/site_repo.go`、`site_repo_test.go`
**依赖：** T06
**步骤：**
1. 实现 Create/ByID/ByKey/Update/RotateKey/ReplaceOrigins/List。
2. 方法内统一处理 `site_origins` 的读写。
3. 唯一冲突映射 `ErrConflict`。

**验证：** 单测覆盖创建、按 key 查询、轮换 key、来源替换。

### T10 站点服务

**文件：** `internal/service/site.go`
**依赖：** T09、T02
**步骤：**
1. `Resolve(siteKey, origin)`：校验站点存在、启用、Origin 精确匹配。
2. `Create/Update/RotateKey`：生成 32 字节 URL-safe siteKey；名称唯一。
3. 定义 `CreateSiteInput/UpdateSiteInput`。

**验证：** 单测：错误 Origin → 拒绝；停用站点 → 拒绝；重复名称 → 冲突。

### T11 站点解析中间件与动态 CORS

**文件：** `internal/httpapi/middleware.go`、`cors.go`、`errors.go`
**依赖：** T10
**步骤：**
1. `siteResolve`：从 `X-Site-Key` 解析站点，校验 Origin，注入 `SiteContext`。
2. 统一错误响应 `{error:{code,message,requestId}}`。
3. 动态 CORS：按站点 Origin 列表放行；允许 `Authorization, Content-Type, X-Site-Key`；处理预检；管理 API 仅在 `ADMIN_ORIGINS` 配置时放行。
4. 接入令牌桶限流：写操作按用户/IP，读操作按 IP。
5. 可选配置 `ALLOW_MISSING_ORIGIN`（仅测试）用于无 Origin 请求。

**验证：** httptest：白名单 Origin 返回 CORS 头；非白名单无 CORS 头；预检 204。

### T12 site-config 接口

**文件：** `internal/httpapi/site_handlers.go`、`router.go`
**依赖：** T11
**步骤：**
1. `GET /api/v1/site-config`：返回站点 displayName、themeColor、locale、moderationMode、启用的登录方式（GitHub + 各 OIDC 的 id/name）。
2. 不返回任何密钥。

**验证：** httptest 返回预期 JSON；停用站点返回 403。

### T13 站点隔离测试

**文件：** `internal/httpapi/site_isolation_test.go`
**依赖：** T11、T12
**步骤：**
1. 创建站点 A、B，各自来源。
2. 用 A 的 siteKey + B 的 Origin，断言拒绝。
3. 用 A 的 siteKey 读取数据，断言看不到 B 的数据（在批注接口完成后补全）。

**验证：** `go test ./internal/httpapi -run TestSiteIsolation` 通过。

---

## 阶段 2：账号与认证

### T14 用户仓储

**文件：** `internal/store/user_repo.go`、`user_repo_test.go`
**依赖：** T06
**步骤：**
1. 实现 Create/ByID/UpdateStatus/SetSuperadmin/List。
2. 列表使用游标分页。

**验证：** 单测覆盖创建、封禁、分页。

### T15 身份仓储

**文件：** `internal/store/identity_repo.go`、`identity_repo_test.go`
**依赖：** T06
**步骤：**
1. 实现 Create/ByID/ByProviderSubject/ListByUser/Delete/TouchLogin。
2. 全局身份 siteID=0；唯一键 `(provider, site_id, issuer, subject)`。

**验证：** 单测：重复创建冲突；GitHub 身份 siteID=0 唯一；按用户查询。

### T16 会话仓储

**文件：** `internal/store/session_repo.go`、`session_repo_test.go`
**依赖：** T06
**步骤：**
1. 实现 Create/ByTokenHash/ListByUser/Revoke/RevokeAllForUser/Touch/PurgeExpired。
2. 只存 token 哈希；撤销用 `revoked_at`。

**验证：** 单测：按哈希查询、撤销后不可用、批量撤销。

### T17 登录流程仓储

**文件：** `internal/store/authflow_repo.go`、`authflow_repo_test.go`
**依赖：** T06
**步骤：**
1. 实现 Create/Consume/PurgeExpired；Consume 原子删除并返回记录。
2. 记录包含 kind、mode、siteID、providerID、userID、returnTo、codeVerifier、nonce。

**验证：** 单测：同一 state 只能消费一次；过期记录被清理。

### T18 GitHub OAuth

**文件：** `internal/auth/github.go`、`github_test.go`
**依赖：** T17
**步骤：**
1. 实现 `Start(site, returnTo)`：生成 state，写入 auth_flows，返回授权 URL。
2. 实现 `Finish(code, state)`：消费流程、换 token、请求 `/user`，返回 ProviderUser。
3. OAuth 端点与 HTTP 客户端可注入，便于测试。

**验证：** 使用 httptest 假 GitHub 服务，完成 start → callback → 用户信息解析。

### T19 OIDC 提供商仓储与密钥

**文件：** `internal/store/provider_repo.go`、`provider_repo_test.go`
**依赖：** T06、T04
**步骤：**
1. 实现 Create/ByID/ListBySite/Update/Delete。
2. clientSecret 加密后入库；读取时返回密文，服务层解密使用。
3. 更新时若 clientSecret 为空表示「保持原值」。
4. 唯一约束：同一站点 issuer+clientID 不重复。

**验证：** 单测：密文可解密回原文；列表不返回明文（由服务层保证）。

### T20 OIDC 登录流程

**文件：** `internal/auth/oidc.go`、`oidc_test.go`
**依赖：** T17、T19
**步骤：**
1. 保存提供商时做 Discovery 校验（`issuer/.well-known/openid-configuration`）。
2. `Start(provider, site, returnTo)`：生成 state+nonce+PKCE(S256)，写 auth_flows，返回授权 URL。
3. `Finish(provider, code, state)`：消费流程、换 token、用 go-oidc 校验 ID Token、提取 sub/email/email_verified/name/picture。
4. 提供商配置缓存，更新时失效。

**验证：** 搭建 httptest 假 IdP（discovery + jwks + token），完成完整 flow 并校验 claim。

### T21 认证服务

**文件：** `internal/service/auth.go`、`auth_test.go`
**依赖：** T14、T15、T16、T18、T20
**步骤：**
1. 登录：按身份查找/创建用户与身份；首次自动建号。
2. `returnTo` 必须属于该站点的 Origin 白名单，否则回落到站点首页。
3. 会话：创建 token、返回明文与过期时间；`Authenticate` 校验哈希、过期、撤销、全局封禁。
4. 绑定：已登录用户发起 bind flow，回调后绑定；目标身份已属于他人 → 冲突；不允许解绑最后一个身份。
5. 超管：GitHub handle 在 `SUPERADMIN_GITHUB_LOGINS` 中时自动提升。

**验证：** 单测：首次登录建号、二次登录复用、绑定冲突、解绑最后身份失败、封禁后 Authenticate 失败。

### T22 认证中间件与 Principal

**文件：** `internal/httpapi/middleware.go`
**依赖：** T21
**步骤：**
1. 解析 Bearer，调用 `Authenticate`，注入 `Principal`。
2. 提供 `requireUser`、`requireSiteRole`、`requireSuperadmin`。
3. 站点级请求检查站点封禁。

**验证：** httptest：无 token 401；封禁用户 403；普通用户访问管理接口 403。

### T23 认证与账号接口

**文件：** `internal/httpapi/auth_handlers.go`、`router.go`
**依赖：** T21、T22
**步骤：**
1. 登录：`GET /auth/github/start`、`GET /auth/oidc/{id}/start`；回调端点。
2. 账号：`GET /me`、`GET /me/identities`、`DELETE /me/identities/{id}`、`GET/DELETE /me/sessions`。
3. 绑定：`POST /me/identities/github/start`、`POST /me/identities/oidc/{id}/start`，返回 `{redirectUrl}`。
4. 回调成功：302 到 returnTo，fragment 带 `annot_token` 或 `annot_bind=ok`。

**验证：** httptest 覆盖登录回调、me、解绑、撤销会话。

### T24 开发登录与超管 CLI

**文件：** `internal/httpapi/auth_handlers.go`、`cmd/annotation-service/main.go`
**依赖：** T21、T23
**步骤：**
1. `DEV_LOGIN=1` 时提供 `POST /api/v1/auth/dev/login?name=`，直接创建用户与会话。
2. CLI：`admin promote --github-login`、`admin users list`。
3. CLI 直接使用 store/service，不经过 HTTP。

**验证：** dev 登录返回 token；CLI promote 后 `is_superadmin` 为 true。

---

## 阶段 3：批注与评论

### T25 批注仓储

**文件：** `internal/store/annotation_repo.go`、`annotation_repo_test.go`
**依赖：** T06
**步骤：**
1. 实现 Create/ByID/ListPage/UpdateStatus/SetVisibility/SetModeration/SoftDelete。
2. 列表查询同时支持公开可见与作者本人可见。
3. 所有方法带 siteID；跨站点返回 `ErrNotFound`。

**验证：** 单测：分页、站点隔离、作者可见待审内容。

### T26 评论仓储

**文件：** `internal/store/comment_repo.go`、`comment_repo_test.go`
**依赖：** T06
**步骤：**
1. 实现 Create/ByID/ListByAnnotation/SetModeration/SoftDelete。
2. 关联查询作者信息；所有方法带 siteID。

**验证：** 单测：创建、按批注查询、隐藏后公开查询不可见。

### T27 批注与评论服务

**文件：** `internal/service/annotation.go`、`annotation_test.go`
**依赖：** T25、T26、T10
**步骤：**
1. `ListPage`：按站点、pageKey、可见性查询并组装评论。
2. `Create`：校验锚点与内容；按站点审核模式设置 moderation；事务内写批注 + 首条评论。
3. `Resolve/Delete`：作者或站点管理员权限。
4. `AddComment/DeleteComment`：审核模式与权限。

**验证：** 单测：pre 模式写入 pending；作者可见、他人不可见；越权删除失败。

### T28 批注与评论接口

**文件：** `internal/httpapi/annotation_handlers.go`、`router.go`
**依赖：** T27、T22
**步骤：**
1. 注册公共接口：GET/POST/PATCH/DELETE annotations、POST/DELETE comments。
2. 请求体限长；字段校验与错误码映射。
3. `X-Site-Key` 必填；写接口要求登录。

**验证：** httptest 覆盖创建、列表、回复、解决、删除、401/403/400。

### T29 批注隔离与可见性测试

**文件：** `internal/httpapi/annotation_isolation_test.go`
**依赖：** T28
**步骤：**
1. 两个站点各建批注，断言互不可见。
2. 同站点两个 pageKey 互不混。
3. 待审内容对未登录/他人不可见，对作者可见。

**验证：** `go test ./internal/httpapi -run TestAnnotationIsolation` 通过。

---

## 阶段 4：治理与管理

### T30 管理员与封禁仓储

**文件：** `internal/store/admin_repo.go`、`admin_repo_test.go`
**依赖：** T06
**步骤：**
1. 实现站点管理员增删查、是否管理员。
2. 实现站点封禁/解封/查询。

**验证：** 单测：重复添加幂等、封禁查询、跨站点隔离。

### T31 治理服务

**文件：** `internal/service/moderation.go`、`moderation_test.go`
**依赖：** T30、T25、T26
**步骤：**
1. 审核队列：待审批注与评论。
2. 审核动作：approve/reject/hide/restore。
3. 封禁：站点封禁检查；全局封禁撤销全部会话。
4. 权限：站点管理员仅本站；超管全部。

**验证：** 单测：越权审核失败；通过后公开可见；封禁后站点请求 403。

### T32 管理服务

**文件：** `internal/service/admin.go`、`admin_test.go`
**依赖：** T09、T14、T30
**步骤：**
1. 站点 CRUD、轮换 key。
2. 指定/撤销站点管理员。
3. 全局用户列表与封禁。
4. 操作写结构化日志。

**验证：** 单测：角色校验、站点管理员分配与撤销。

### T33 管理 API：站点与用户

**文件：** `internal/httpapi/admin_handlers.go`、`router.go`
**依赖：** T32、T22
**步骤：**
1. 注册 `/admin/sites*`、`/admin/users*`。
2. 全部要求超管；统一分页与错误。

**验证：** httptest：超管可 CRUD；站点管理员 403。

### T34 管理 API：审核与提供商

**文件：** `internal/httpapi/admin_handlers.go`
**依赖：** T31、T19、T22
**步骤：**
1. 审核队列、审核动作、站点封禁。
2. OIDC 提供商 CRUD；响应不含 secret。
3. 权限：站点管理员（本站）或超管。

**验证：** httptest：审核通过/拒绝、封禁、提供商创建与更新、密钥不回显。

### T35 管理 CLI

**文件：** `cmd/annotation-service/main.go`
**依赖：** T32、T33
**步骤：**
1. `admin site create/list/rotate-key`。
2. `admin site add-admin/remove-admin`。
3. `admin promote`、`admin users list`。
4. 输出人类可读表格或 JSON（`--json`）。

**验证：** 在临时库执行 CLI，查询 DB 断言结果。

---

## 阶段 5：SDK

### T36 SDK 工程脚手架

**文件：** `sdk/package.json`、`tsconfig.json`、`vite.config.ts`、`vitest.config.ts`
**依赖：** 无（可与后端并行）
**步骤：**
1. 初始化 TypeScript + Vite library mode，输出 ESM/IIFE/d.ts，全局名 `AnnotationSDK`。
2. 依赖：`dom-anchor-text-quote`、`@apache-annotator/dom`、`@apache-annotator/selector`。
3. devDeps：vite、typescript、vitest、jsdom、@playwright/test。
4. 脚本：`build`、`test`、`typecheck`。

**验证：** `cd sdk && npm install && npm run build` 产出 `dist/`。

### T37 SDK HTTP 客户端

**文件：** `sdk/src/client.ts`、`client.test.ts`
**依赖：** T36
**步骤：**
1. 自动附加 `X-Site-Key` 与 Bearer；统一错误类型。
2. token 存储按 `apiBase + siteKey` 隔离；处理登录回调 fragment。
3. `apiBase` 默认从 SDK 脚本自身 URL 推导（`document.currentScript` 或 import.meta.url）。
4. 封装 site-config、annotations、comments、me、identities、sessions API。

**验证：** Vitest mock fetch：header 正确、401 清理 token、错误映射。

### T38 SDK 锚定模块

**文件：** `sdk/src/anchor.ts`、`anchor.test.ts`
**依赖：** T36
**步骤：**
1. 选区校验（排除 pre/svg/mermaid/katex）。
2. 用 `dom-anchor-text-quote.fromRange` 生成锚点。
3. 用 `toRange` 模糊定位，用 `@apache-annotator/dom.highlightText` 渲染/清理 `<mark>`。
4. 记录正文顺序用于侧栏排序，匹配失败回调 orphan。

**验证：** jsdom 单测：锚点往返、改一字仍匹配、排除区不匹配。

### T39 SDK 状态与事件

**文件：** `sdk/src/state.ts`、`events.ts`、`i18n.ts`
**依赖：** T36
**步骤：**
1. 实例状态：站点配置、当前用户、当前页、批注列表、orphan 集合。
2. 事件：ready/login/logout/created/error；返回取消订阅函数。
3. 中英文案字典。

**验证：** 单测：事件订阅/取消、状态切换。

### T40 SDK UI 宿主与样式

**文件：** `sdk/src/ui/host.ts`、`styles.css`
**依赖：** T36
**步骤：**
1. 创建宿主元素与 Shadow Root；注入样式。
2. 亮/暗主题与站点主题色 CSS 变量。
3. 提供挂载/卸载、Toast、确认框基础组件。

**验证：** jsdom 单测：Shadow Root 存在、样式随 theme 变化、destroy 移除宿主。

### T41 SDK 选区工具条与编辑框

**文件：** `sdk/src/ui/toolbar.ts`、`composer.ts`
**依赖：** T38、T40、T39
**步骤：**
1. 监听 mouseup/touchend，显示「批注」按钮；未登录显示「登录后批注」。
2. 编辑框：引用文本、textarea、发布/取消、错误提示、长度限制。
3. 键盘 Escape 关闭；按钮阻止选区丢失。

**验证：** jsdom 单测：选中后工具条出现；发布调用 client 一次。

### T42 SDK 高亮、侧栏与评论

**文件：** `sdk/src/ui/drawer.ts`
**依赖：** T38、T40、T37
**步骤：**
1. 页面加载渲染高亮，点击高亮打开侧栏并定位。
2. 侧栏列表按正文顺序；作者、时间、状态、orphan 标记。
3. 回复、解决/重新打开、删除（作者或管理员）。
4. 登录状态变化时刷新。

**验证：** jsdom 单测：列表渲染、点击高亮、回复调用。

### T43 SDK 登录面板与实例 API

**文件：** `sdk/src/ui/login.ts`、`sdk/src/index.ts`
**依赖：** T37、T39、T40、T42
**步骤：**
1. 登录面板：GitHub + 站点 OIDC 列表；整页跳转。
2. `init()`：拉 site-config、消费 token、渲染 UI、加载当前页。
3. 实例 API：`setPage/refresh/login/logout/getCurrentUser/on/destroy`。
4. SPA 路由切换清理旧高亮。

**验证：** 单测：init 调用顺序、setPage 重新拉取、destroy 清理。

### T44 SDK 接入示例

**文件：** `examples/plain-html/index.html`
**依赖：** T43
**步骤：**
1. 通过 `<script src="/sdk/v1/annotation-sdk.js">` 引入。
2. 一段可划线的示例正文。
3. `AnnotationSDK.init({ siteKey })`。
4. E2E 时用静态服务器托管示例页（如 `python3 -m http.server`），并把该 Origin 加入站点白名单。

**验证：** 由服务托管后浏览器可完成登录与批注（E2E 覆盖）。

### T45 SDK 构建与内嵌

**文件：** `internal/sdkassets/embed.go`、`internal/sdkassets/dist/.gitkeep`、`internal/httpapi/sdk_handlers.go`、`Makefile`
**依赖：** T36、T43
**步骤：**
1. `//go:embed all:dist`；未构建时返回 404 与提示。
2. `GET /sdk/v1/annotation-sdk.js` 与 `/sdk/v1/annotation-sdk.esm.js`，设置缓存头。
3. `make sdk` 构建并复制到 `internal/sdkassets/dist/`。
4. `make build` = sdk + go build。

**验证：** `make build` 后启动服务，`curl /sdk/v1/annotation-sdk.js` 返回 200 且内容为 IIFE。

### T46 SDK 单测与 E2E

**文件：** `sdk/src/*.test.ts`、`sdk/e2e/annotation.spec.ts`
**依赖：** T43、T45、T35
**步骤：**
1. 汇总 Vitest 单测。
2. Playwright：CLI 创建站点 A/B → 启动服务与示例页静态服务器 → dev 登录 → 选中文字 → 发布 → 刷新恢复。
3. 跨站隔离：站点 A 的批注不出现在站点 B。
4. 审核：pre 模式下他人不可见。
5. 可访问性冒烟：工具条与侧栏可键盘关闭。

**验证：** `npm test` 与 `npx playwright test` 全绿。

---

## 阶段 6：部署、文档与最终验收

### T47 部署文件更新

**文件：** `Dockerfile`、`docker-compose.yml`、`.env.example`、`deploy/*`
**依赖：** T45
**步骤：**
1. Dockerfile 多阶段：Node 构建 SDK → Go embed 编译 → Alpine 运行。
2. compose 增加 `SECRET_KEY`、`PUBLIC_BASE_URL`、`SUPERADMIN_GITHUB_LOGINS`。
3. `.env.example` 覆盖全部新配置，标注 OIDC 回调地址格式。
4. 反代示例补充 SDK 静态路径与 CORS 说明。

**验证：** `docker build` 成功；容器启动后 `/healthz` 与 `/sdk/v1/annotation-sdk.js` 正常。

### T48 README 产品化

**文件：** `README.md`
**依赖：** T47
**步骤：**
1. 产品定位、架构图、角色说明。
2. 快速开始：Docker、CLI 创建站点、SDK 接入。
3. GitHub/OIDC 配置、回调地址、超管引导。
4. 运维：备份、迁移、限流、升级。
5. 明确 V1 边界（无管理后台、无密码登录等）。

**验证：** 按 README 从空库走通一次本地部署。

### T49 全量验证

**文件：** 全部
**依赖：** T46、T47、T48
**步骤：**
1. `gofmt -l .`、`go vet ./...`、`go test ./...`。
2. `cd sdk && npm run typecheck && npm test && npm run build`。
3. `make build`、Docker 构建、Playwright E2E。
4. 检查无旧单站点接口残留。

**验证：** 所有命令通过，记录输出。

---

## 阶段 7：外部接入（go-campus）

### T50 VitePress 改用 SDK

**文件：** `../go-campus/docs/.vitepress/theme/index.ts`、删除 `../go-campus/docs/.vitepress/theme/annotations/`、`../go-campus/docs/.env.example`
**依赖：** T45
**步骤：**
1. 删除手写批注层与样式，保留 Artalk。
2. 注入 `<script src="<API>/sdk/v1/annotation-sdk.js">`。
3. 在主题挂载后 `AnnotationSDK.init({ siteKey: import.meta.env.VITE_ANNOTATION_SITE_KEY, pageKey: route.path })`，并监听路由调用 `setPage`。
4. 更新 `.env.example`：`VITE_ANNOTATION_API`、`VITE_ANNOTATION_SITE_KEY`。
5. 通过 CLI 创建站点并为博客域名配置 Origin。

**验证：** `npm run docs:build` 通过；浏览器登录、划线、刷新恢复；原有 Artalk/图片放大/mermaid 不受影响。

---

## 执行顺序

```
T00 → T01 → T02 → T03
                 ├→ T04 → T19
                 ├→ T05
                 └→ T06 → T09 → T10 → T11 → T12 → T13
                          ├→ T14 ┐
                          ├→ T15 ├→ T21 → T22 → T23 → T24
                          ├→ T16 ┘        └→ T28 → T29
                          ├→ T17 → T18 ┐
                          ├→ T19 → T20 ┘
                          ├→ T25 ┐
                          ├→ T26 ┴→ T27 → T28
                          └→ T30 → T31 → T34
                                    T32 → T33 → T35

SDK（可与后端阶段 1–4 并行）：
T36 → T37 → T38 → T39 → T40 → T41 → T42 → T43 → T44
                                        ├→ T45 → T46
                                        └→ T47 → T48 → T49 → T50

T07 → T08（随 T06 完成即可）
```
