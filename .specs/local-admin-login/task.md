# 本地管理员账号密码登录 · Tasks

> 状态：已确认（v1.0）
> 输入：`.specs/local-admin-login/spec.md`、`plan.md`
> 说明：按顺序执行；每个任务完成后运行对应验证。

## 文件清单

### 后端

| 操作 | 文件 | 职责 |
|---|---|---|
| 新建 | `internal/store/migrations/0002_local_credentials.sql` | 本地凭据表 |
| 修改 | `internal/domain/entities.go` | `LocalCredential`、`Principal` 扩展 |
| 修改 | `internal/store/interfaces.go` | `LocalCredentialRepo`、会话吊销方法 |
| 新建 | `internal/store/local_credential_repo.go` | 本地凭据 CRUD |
| 修改 | `internal/store/session_repo.go` | `RevokeOtherSessionsForUser` |
| 新建 | `internal/auth/password.go` | bcrypt 哈希/校验/生成 |
| 新建 | `internal/auth/password_test.go` | 密码工具单测 |
| 修改 | `internal/config/config.go` | `LOCAL_ADMIN_*` 配置 |
| 修改 | `internal/service/auth.go` | bootstrap / 本地登录 / 改密 / 重置 |
| 新建 | `internal/service/auth_local_test.go` | 服务层测试 |
| 修改 | `internal/httpapi/admin_auth_handlers.go` | 本地登录接口 |
| 修改 | `internal/httpapi/auth_handlers.go` | 改密接口、`/me` 字段 |
| 修改 | `internal/httpapi/middleware.go` | `requirePasswordChanged` |
| 修改 | `internal/httpapi/router.go` | 新路由与中间件 |
| 修改 | `internal/httpapi/server_test.go` | 集成测试 |
| 修改 | `cmd/zhupi/main.go` | `admin password reset` |
| 修改 | `README.md` | 环境变量与首次登录说明 |

### 前端

| 操作 | 文件 | 职责 |
|---|---|---|
| 修改 | `admin/src/api/types.ts` | `Me`、`AdminAuthProviders` 扩展 |
| 修改 | `admin/src/api/index.ts` | `localLogin`、`changePassword` |
| 修改 | `admin/src/stores/auth.ts` | 本地登录与强制改密状态 |
| 修改 | `admin/src/views/LoginView.vue` | 账号密码表单 |
| 新建 | `admin/src/views/AccountView.vue` | 账号设置页 |
| 修改 | `admin/src/router.ts` | `/account` 与守卫 |
| 修改 | `admin/src/layout/AdminLayout.vue` | 账号设置入口 |

## 任务

### T01 数据库与领域模型

- 新增迁移 `0002_local_credentials.sql`。
- 新增 `domain.LocalCredential`，扩展 `domain.Principal`。
- 验证：`go test ./internal/store/...`。

### T02 存储层

- 实现 `LocalCredentialRepo` 与 `RevokeOtherSessionsForUser`。
- 验证：新增 store 测试覆盖创建、按用户名/用户 ID 查询、更新密码、大小写不敏感。

### T03 密码工具

- 实现 bcrypt 哈希、校验、策略、随机密码。
- 验证：`go test ./internal/auth/...`。

### T04 配置

- 增加 `LocalAdminUsername`、`LocalAdminPassword`、`LocalAdminResetPassword`。
- 验证：`go test ./internal/config/...`（如无测试则 `go build ./...`）。

### T05 服务层

- 实现 `BootstrapLocalAdmin`、`LocalLogin`、`ChangePassword`、`ResetLocalPassword`。
- `AuthenticateSession` 填充本地账号信息。
- 验证：`go test ./internal/service/...`。

### T06 HTTP 接口与中间件

- 新增本地登录、改密接口；扩展 `/me` 与 providers。
- 新增 `requirePasswordChanged` 并挂到管理路由。
- 验证：`go test ./internal/httpapi/...`。

### T07 CLI

- 新增 `admin password reset`。
- 验证：`go run ./cmd/zhupi admin password reset --help` 与临时数据库实测。

### T08 管理后台

- 登录页本地表单、账号设置页、路由守卫、布局入口。
- 验证：`cd admin && npm run typecheck && npm test && npm run build`。

### T09 文档

- README 增加环境变量、首次登录、重置密码说明。
- 验证：人工校对命令可复制执行。

### T10 全量验证

- `gofmt -l .`、`go vet ./...`、`go test ./...`。
- `cd admin && npm run typecheck && npm test && npm run build`。
- 本地启动服务，走通「默认登录 → 强制改密 → 正常访问」。

### T11 部署

- 构建镜像，`docker save | ssh` 传到服务器。
- 更新服务器 `.env`（保留已有 SECRET_KEY / GitHub 配置）。
- `docker compose up -d`，验证 `https://zhupi.gataccay.com` 登录流程。
