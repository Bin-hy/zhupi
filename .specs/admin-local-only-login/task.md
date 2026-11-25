# 管理后台仅本地账号密码登录 · Tasks

> 状态：已确认（v1.0）
> 输入：`.specs/admin-local-only-login/spec.md`、`.specs/admin-local-only-login/plan.md`

## 文件清单

| 操作 | 文件 | 职责 |
| --- | --- | --- |
| 新建 | `internal/store/migrations/0003_session_auth_method.sql` | 会话来源列 |
| 修改 | `internal/domain/enums.go` | AuthMethod 常量 |
| 修改 | `internal/domain/entities.go` | Session.AuthMethod、LocalAdminAccount |
| 修改 | `internal/store/session_repo.go` | 读写会话来源 |
| 修改 | `internal/store/local_credential_repo.go` | 列表、计数、事务建号、冲突映射 |
| 修改 | `internal/store/interfaces.go` | LocalCredentialRepo 扩展 |
| 新建 | `internal/auth/username.go` | 用户名校验 |
| 新建 | `internal/auth/username_test.go` | 用户名校验测试 |
| 修改 | `internal/service/auth.go` | 会话来源、后台绑定、解绑规则、移除自动超管 |
| 新建 | `internal/service/bind_error.go` | 绑定失败携带流程 |
| 修改 | `internal/service/admin.go` | 账号生命周期、超管守卫 |
| 新建 | `internal/service/auth_bind_test.go` | 绑定冲突与解绑规则测试 |
| 修改 | `internal/httpapi/middleware.go` | requireLocalSession |
| 修改 | `internal/httpapi/router.go` | 路由增删与统一管理链 |
| 修改 | `internal/httpapi/admin_auth_handlers.go` | 仅本地登录、后台绑定 |
| 新建 | `internal/httpapi/admin_account_handlers.go` | 管理员管理接口 |
| 修改 | `internal/httpapi/auth_handlers.go` | 绑定失败回跳 |
| 修改 | `internal/httpapi/admin_handlers.go` | 站点管理员响应补充 |
| 修改 | `internal/httpapi/local_auth_test.go` | 现有本地登录测试适配 |
| 新建 | `internal/httpapi/admin_accounts_test.go` | 建号/重置/守卫/会话来源测试 |
| 修改 | `internal/httpapi/server_test.go` | 本地登录夹具与既有用例改造 |
| 修改 | `internal/store/store_test.go` | 事务建号与冲突测试 |
| 修改 | `internal/config/config.go` | 删除自动超管配置 |
| 修改 | `internal/config/config_test.go` | 同步 |
| 修改 | `cmd/zhupi/main.go` | 构造器、CLI |
| 修改 | `cmd/zhupi/provider_cli_test.go` | 构造器同步 |
| 修改 | `.env.example` | 环境变量说明 |
| 修改 | `admin/src/api/types.ts` | 新类型 |
| 修改 | `admin/src/api/index.ts` | 新接口 |
| 修改 | `admin/src/stores/auth.ts` | 删除外部登录 |
| 修改 | `admin/src/views/LoginView.vue` | 仅本地登录 |
| 新建 | `admin/src/views/AdminsView.vue` | 管理员账号页 |
| 新建 | `admin/src/components/LocalAdminFormDialog.vue` | 新建/开通表单 |
| 修改 | `admin/src/views/UsersView.vue` | 本地账号列与开通 |
| 修改 | `admin/src/views/AccountView.vue` | 绑定/解绑身份 |
| 修改 | `admin/src/components/SiteAdminsTab.vue` | 本地账号状态 |
| 修改 | `admin/src/router.ts` | `/admins` |
| 修改 | `admin/src/layout/AdminLayout.vue` | 侧边栏入口 |
| 修改 | `admin/e2e/console.mjs` | 本地登录 |
| 修改 | 网站文档 7 个文件 + `skills/zhupi-integration/references/self-hosting.md` + `README.md` | 同步行为 |
| 构建 | `internal/adminassets/dist` | `make admin` 刷新 |

## T1 会话来源落库

**文件**：`internal/store/migrations/0003_session_auth_method.sql`、`internal/domain/enums.go`、`internal/domain/entities.go`、`internal/store/session_repo.go`、`internal/service/auth.go`
**依赖**：无

**步骤**
1. 新建迁移：会话表增加 `auth_method TEXT NOT NULL DEFAULT 'legacy'`。
2. `domain` 增加 `AuthMethod` 常量；`Session` 增加 `AuthMethod` 字段。
3. `session_repo` 的列清单、插入、扫描补上该字段；列表查询同样返回来源。
4. `LocalLogin` 创建会话时写 `local`；`completeLogin` 按流程类型写 `github` / `oidc`；`DevLogin` 写 `dev`。
5. 确认升级前旧会话保持 `legacy`。

**验证**
- `go test ./internal/store ./internal/service` 通过。
- 本地登录后查询会话记录，`auth_method=local`；调试登录为 `dev`。

## T2 本地会话中间件与测试夹具

**文件**：`internal/httpapi/middleware.go`、`internal/httpapi/router.go`、`internal/httpapi/server_test.go`、`internal/httpapi/local_auth_test.go`
**依赖**：T1

**步骤**
1. 新增 `requireLocalSession`：会话来源不是 `local` 时返回 403 `admin_local_auth_required`。
2. 所有管理路由统一追加该中间件（`adminSite` 链与超管直连路由都要覆盖）；登录类公开接口除外。
3. 测试夹具新增本地管理员创建与登录辅助：可创建超管/站点管理员、返回本地 token。
4. 将现有访问管理 API 的用例改成本地会话；站点访客与批注用例继续用调试登录。
5. 在本地登录测试中补充反向用例：调试会话调管理 API 返回 403，站点接口仍正常。

**验证**
- `go test ./internal/httpapi` 全部通过。
- 反向用例中调试 token 访问 `/api/v1/admin/overview` 返回 403。

## T3 移除外部直登与自动超管

**文件**：`internal/config/config.go`、`internal/config/config_test.go`、`internal/service/auth.go`、`internal/service/admin.go`、`internal/store/identity_repo.go`、`internal/store/interfaces.go`、`internal/httpapi/admin_auth_handlers.go`、`internal/httpapi/router.go`、`cmd/zhupi/main.go`
**依赖**：T2

**步骤**
1. 删除 `SuperadminGitHubLogins` 字段、解析逻辑与 `config_test` 断言；保留环境变量时不报错。
2. 删除 `applyAutoSuperadmin` 及其调用，GitHub 访客登录不再改角色。
3. 删除 `StartAdminGitHub`、`StartAdminOIDC` 及对应路由/处理函数。
4. `AdminProviders` 仅返回 `{local:true}`；移除 GitHub/站点提供商字段。
5. 删除 `PromoteByGitHubHandle`、`admin promote --github-login` 分支；清理不再使用的 `IdentityByHandle`。
6. 同步 `main.go` 构造与参数解析，确保可编译；更新 providers 相关测试断言。

**验证**
- `go test ./internal/config ./internal/service ./internal/httpapi` 通过。
- `grep -R "SUPERADMIN_GITHUB_LOGINS\|StartAdminGitHub\|StartAdminOIDC\|PromoteByGitHubHandle" internal cmd` 无命中。
- 访问旧后台外部登录地址返回 404。

## T4 用户名校验与本地凭据存储

**文件**：`internal/auth/username.go`、`internal/auth/username_test.go`、`internal/domain/entities.go`、`internal/store/local_credential_repo.go`、`internal/store/interfaces.go`、`internal/store/store_test.go`
**依赖**：T1

**步骤**
1. 实现 `ValidateUsername`：3–32 字符，仅字母、数字、`.`、`_`、`-`。
2. `domain` 增加 `LocalAdminAccount` 读模型。
3. `ListLocalAdmins`：联表查用户名/显示名/状态/超管/强制改密，再聚合站点管理员站点。
4. `CountActiveSuperadminCredentials`：统计未封禁 + 超管 + 有本地凭据的账号。
5. `CreateLocalAdmin`：单事务插入用户、凭据、站点管理员；唯一冲突映射为 `username_taken`。
6. `CreateLocalCredential` 同样映射用户名冲突；扩展 `LocalCredentialRepo` 接口。
7. 补测试：用户名边界、重复用户名不留下孤儿用户、超管计数。

**验证**
- `go test ./internal/auth ./internal/store` 通过。
- 重复用户名建号后用户表行数不变。

## T5 管理员生命周期与守卫

**文件**：`internal/service/admin.go`、`cmd/zhupi/main.go`、`internal/httpapi/server_test.go`、`cmd/zhupi/provider_cli_test.go`
**依赖**：T4

**步骤**
1. `AdminService` 注入本地凭据仓储；移除不再使用的身份仓储依赖。
2. 实现 `ListLocalAdmins`、`CreateLocalAdmin`、`OpenLocalCredential`、`ResetLocalAdminPassword`、`SetSuperadmin`。
3. 校验：仅超管、用户名格式/唯一、密码策略、目标用户存在且未封禁、已有凭据不可重复开通。
4. 建号/开通/重置统一 `mustChangePassword=true`；密码为空时生成并只返回一次。
5. 重置禁止针对自己；成功后吊销目标全部会话。
6. `SetSuperadmin(false)` 与 `GlobalBan` 增加最后超管守卫；`SetSuperadmin(true)` / `PromoteUser` 要求已有本地凭据。
7. 更新 `main.go`、测试构造器与 CLI 服务的依赖注入。

**验证**
- `go build ./...` 通过。
- `go test ./internal/service` 通过。

## T6 管理员管理接口与站点管理员响应

**文件**：`internal/httpapi/admin_account_handlers.go`、`internal/httpapi/router.go`、`internal/httpapi/admin_handlers.go`、`internal/service/admin.go`、`internal/httpapi/admin_accounts_test.go`
**依赖**：T5

**步骤**
1. 新增接口：管理员列表、新建、为已有用户开通、重置密码、授予/撤销超管。
2. 全部挂 `requireAuth → requireLocalSession → requireSuperadmin`；重置接口拒绝自己。
3. 响应包含账号信息；生成密码仅在该次响应返回 `generatedPassword`。
4. 站点管理员列表补充 `displayName`、`localUsername`、`mustChangePassword`。
5. 记录审计日志：建号、开通、重置、角色变更（不含密码）。
6. 增加一条打通「建号 → 本地登录 → 访问管理 API」的集成测试。

**验证**
- `go test ./internal/httpapi` 通过。
- 新建账号首次登录返回 `mustChangePassword=true`，改密后管理 API 可访问。

## T7 后台身份绑定与解绑规则

**文件**：`internal/service/auth.go`、`internal/service/bind_error.go`、`internal/httpapi/admin_auth_handlers.go`、`internal/httpapi/auth_handlers.go`、`internal/httpapi/router.go`
**依赖**：T6

**步骤**
1. 实现 `AdminBindableIdentities`：已绑定身份 + GitHub 可用性 + 已启用站点 OIDC 列表。
2. 实现后台绑定发起：bind 流程，`ReturnTo` 固定为 `${PUBLIC_BASE_URL}/admin/account`；GitHub 流程无站点，OIDC 流程带提供商站点与 PKCE/nonce。
3. `FinishGitHub` / `FinishOIDC` 在 bind 模式下失败时返回携带流程的 `BindError`。
4. 回调处理：识别 `BindError` 并回跳 `#annot_bind=error&message=...`；成功保持 `#annot_bind=ok`。
5. `UnbindIdentity`：有本地凭据允许解绑全部；无本地凭据保留「最后身份」限制。
6. 新增路由：绑定列表、GitHub 绑定、OIDC 绑定；只要求本地会话，不套用强制改密拦截，保证首次改密前也能进入账号页。
7. 补服务层测试：解绑规则、绑定冲突、可绑定列表。

**验证**
- `go test ./internal/service ./internal/httpapi` 通过。
- 有本地凭据的账号可解绑最后一个身份；纯访客账号解绑最后一个身份被拒绝。
- 绑定接口未登录返回 401，访客会话返回 403。

## T8 CLI 调整

**文件**：`cmd/zhupi/main.go`、`cmd/zhupi/provider_cli_test.go`
**依赖**：T5

**步骤**
1. 删除 `admin promote --github-login`，保留 `--user-id`，并说明目标必须先开通本地密码。
2. 新增 `admin users open --user-id <id> --username <name> [--password <pwd>]`；省略密码时生成并打印一次。
3. `admin password reset --username` 保持，可覆盖任意本地管理员。
4. `admin users list` 增加本地用户名列：合并管理员列表后按用户 ID 输出。
5. 更新用法文本与构造器依赖。

**验证**
- `go build ./cmd/zhupi` 通过。
- `go test ./cmd/zhupi` 通过。
- 以临时数据库执行 `admin users open` 后可用该密码本地登录。

## T9 前端类型与 API 客户端

**文件**：`admin/src/api/types.ts`、`admin/src/api/index.ts`
**依赖**：T6

**步骤**
1. 新增类型：`LocalAdminAccount`、`BindableIdentities`、`Identity`、建号/开通/重置输入、生成密码响应。
2. `SiteAdmin` 增加 `displayName`、`localUsername`、`mustChangePassword`。
3. `AdminAuthProviders` 暂时保留 GitHub/OIDC 字段为可选，避免本任务破坏登录页编译。
4. 新增 API：管理员列表/新建/开通/重置/超管开关、身份列表、绑定发起、解绑。
5. 保持现有错误处理方式，不新增拦截器。

**验证**
- `cd admin && npm run typecheck` 通过。
- `npm test` 通过。

## T10 登录页收敛

**文件**：`admin/src/views/LoginView.vue`、`admin/src/stores/auth.ts`
**依赖**：T9

**步骤**
1. 删除 GitHub/站点 OIDC 按钮与相关加载分支，只保留用户名/密码表单。
2. 删除 `loginGitHub`、`loginOIDC` 与 `redirectStart`。
3. 无本地凭据时显示「尚未初始化本地管理员」；登录失败展示服务端提示。
4. `AdminAuthProviders` 收紧为 `{ local: boolean }`。
5. 保留强制改密跳转逻辑。

**验证**
- `npm run typecheck` 通过。
- 登录页只渲染用户名、密码与登录按钮。
- 错误密码显示统一提示，成功后按 `mustChangePassword` 跳转。

## T11 管理员账号页

**文件**：`admin/src/views/AdminsView.vue`（新建）、`admin/src/components/LocalAdminFormDialog.vue`（新建）、`admin/src/router.ts`、`admin/src/layout/AdminLayout.vue`
**依赖**：T10

**步骤**
1. 新增 `/admins` 路由，`meta.superadmin`；侧边栏增加「管理员账号」。
2. 列表展示用户名、显示名、角色、可管理站点、状态、待改密标记。
3. 「新建管理员」与「从已有用户开通」两个入口，共用表单弹窗：
   - 新建：用户名、显示名、初始密码（可生成）、超管开关、站点多选。
   - 开通：用户 ID/选择、用户名、初始密码（可生成）、可选角色。
4. 生成密码弹窗只展示一次，支持复制。
5. 行操作：重置密码、授予/撤销超管；对错误码 `last_superadmin`、`local_credential_required`、`username_taken` 做可读提示。
6. 列表操作后刷新账号与当前会话信息。

**验证**
- `npm run typecheck` 通过。
- 非超管访问 `/admins` 被路由守卫重定向到概览。
- 新建账号成功后弹窗展示生成密码，关闭后无法再次查看。

## T12 用户页与站点管理员页入口

**文件**：`admin/src/views/UsersView.vue`、`admin/src/components/SiteAdminsTab.vue`
**依赖**：T11

**步骤**
1. UsersView 加载管理员列表并建立 `userId → username` 映射，增加「本地账号」列。
2. 未开通本地账号的用户显示「开通本地密码」按钮，打开共用表单弹窗。
3. SiteAdminsTab 使用扩展后的管理员字段，展示「本地账号」状态与待改密标记。
4. 未开通的站点管理员显示「开通」按钮，并提示「当前无法登录后台」。
5. 开通成功后刷新对应表格与提示。

**验证**
- `npm run typecheck` 通过。
- 用户页能区分已开通/未开通；站点管理员页能直接发起开通。

## T13 账号设置页身份绑定

**文件**：`admin/src/views/AccountView.vue`
**依赖**：T11

**步骤**
1. 加载已绑定身份与可绑定提供商（GitHub + 按站点分组的 OIDC）。
2. 已绑定身份列表支持解绑；解绑冲突/最后身份错误用服务端提示展示。
3. 绑定按钮调用后台绑定接口并跳转 `redirectUrl`。
4. 页面加载时解析 `#annot_bind=ok|error&message=...`：
   - 成功/失败分别提示；
   - 清理地址栏 hash；
   - 刷新身份列表。
5. 保留密码修改表单与强制改密提示，两者互不影响。

**验证**
- `npm run typecheck` 通过。
- 模拟 `#annot_bind=ok` 后提示成功并清除 hash。
- 未登录或访客会话调用绑定接口时显示错误，不白屏。

## T14 Go 测试与夹具重构

**文件**：`internal/httpapi/server_test.go`、`internal/httpapi/local_auth_test.go`、`internal/httpapi/admin_accounts_test.go`、`internal/service/auth_bind_test.go`
**依赖**：T2、T5、T6、T7

**步骤**
1. 夹具新增：
   - `localAdmin(t, username, superadmin)`：创建用户与本地凭据（密码已知、不需改密）。
   - `localLogin(t, username, password)`：走本地登录接口返回 token。
2. 将所有访问管理 API 的用例改为本地会话；保持用户 ID 顺序与现有断言一致；站点访客用例继续用调试登录。
3. 新增反向用例：
   - 调试/访客会话调管理 API 返回 403，站点接口正常。
   - `legacy` 会话调管理 API 返回 403。
   - 新建/开通：重复用户名、非法用户名、弱密码、已有凭据重复开通。
   - 重置：旧密码失效、目标全部会话失效、重置自己被拒绝。
   - 最后超管：撤销与封禁均被拒绝；撤销普通超管成功。
   - 站点管理员本地登录后只能访问被授予站点。
   - 绑定冲突与解绑规则（服务层直接调用）。
   - 后台登录方式仅剩本地；旧外部登录地址 404。
4. 同步更新受构造器变化影响的测试。

**验证**
- `go vet ./...` 通过。
- `go test ./...` 全部通过。
- 反向用例断言实际 403 状态码与错误码。

## T15 管理后台 e2e

**文件**：`admin/e2e/console.mjs`
**依赖**：T10、T13

**步骤**
1. 用环境变量提供本地超管初始密码；通过 `/admin/auth/local/login` 获取 token。
2. 若 `mustChangePassword`，先调用改密接口并重新登录。
3. 管理 API 请求与 `localStorage` 注入改用本地 token，不再用调试登录获取超管。
4. 访客待审数据仍用调试登录创建，验证审核链路。
5. 登录页场景改为填写用户名/密码并进入后台。
6. 清理不再使用的 GitHub/OIDC 登录步骤。

**验证**
- 启动测试服务后 `cd admin && npm run e2e` 输出 `ADMIN E2E PASS`。
- 登录页无外部登录按钮；审核、站点编辑、用户封禁流程正常。

## T16 文档同步

**文件**：`website/src/content/docs/sites/authentication.md`、`api/admin-account.md`、`cli/reference.md`、`deployment/configuration.md`、`reference/security.md`、`getting-started/quick-start.md`、`sites/roles-moderation.md`、`development/architecture.md`、`skills/zhupi-integration/references/self-hosting.md`、`README.md`、`.env.example`
**依赖**：T8

**步骤**
1. 删除「后台可用 GitHub/站点 OIDC 登录」与 `SUPERADMIN_GITHUB_LOGINS` 的描述。
2. 说明后台仅本地密码登录、多管理员创建/开通/重置、最后超管保护。
3. 更新管理 API 表：新增管理员与绑定接口，删除外部直登接口。
4. 更新 CLI：删除 `promote --github-login`，新增 `users open`，说明 `password reset` 适用任意本地管理员。
5. 更新升级说明：旧会话不能进后台、旧管理员需超管开通。
6. `.env.example` 删除自动超管变量；补充多管理员说明。
7. 全文检索并清理所有相关旧表述。

**验证**
- `grep -R "SUPERADMIN_GITHUB_LOGINS\|admin/auth/github\|admin/auth/oidc" website skills README.md .env.example` 无遗留（历史 `.specs` 除外）。
- 文档中的接口与命令与实现一致。

## T17 全量验证与资源构建

**文件**：`internal/adminassets/dist`、`internal/sdkassets/dist`（构建产物）
**依赖**：T14、T15、T16

**步骤**
1. 运行 `make admin`（以及需要时的 `make sdk`）刷新嵌入资源。
2. 运行全量检查：
   - `go vet ./...`
   - `go test ./...`
   - `cd admin && npm run typecheck && npm test`
   - `cd sdk && npm run typecheck && npm test`
   - `make server`
3. 启动临时数据库手工冒烟：
   - 本地超管登录 → 新建管理员 → 新账号首登改密 → 进入后台。
   - 调试/访客会话调管理 API 返回 403，站点接口正常。
   - 账号设置页绑定入口展示正常（无 IdP 时仅验证发起与错误回跳）。
4. 记录未覆盖项：真实 OIDC/GitHub 绑定需在具备 IdP 的环境验证。

**验证**
- 上述命令全部退出码为 0。
- 手工冒烟步骤结果符合预期，失败项记录并修复后重跑。

## 执行顺序

```
T1 → T2 → T3
        ↘
T4 → T5 → T6 → T7
                 ↘
T8 ───────────────┴→ T14 → T15 → T16 → T17
T9 → T10 → T11 → T12 → T13
```

- 同一时刻只推进一条主链；前端链可在 T6/T7 接口稳定后并行。
- 每个任务完成并通过验证后再进入下一个；验证失败先修复，不带病推进。
