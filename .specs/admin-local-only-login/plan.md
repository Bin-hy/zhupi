# 管理后台仅本地账号密码登录 · Plan

> 状态：已确认（v1.0）
> 输入：`.specs/admin-local-only-login/spec.md`

## 架构概览

```
管理后台 SPA
  LoginView ── 用户名/密码 ──┐
  AdminsView ─ 管理员账号管理 ─┤
  AccountView ─ 绑定/解绑身份 ─┤
                              ▼
                    /api/v1/admin/*
              requireAuth → requireLocalSession → 角色
                              │
        ┌─────────────────────┼──────────────────────┐
        ▼                     ▼                      ▼
   AuthService           AdminService            Store (SQLite)
 本地登录/改密/重置     建号/开通/重置/角色      users / local_credentials
 绑定流程/解绑          最后超管保护            sessions / identities
                                               site_admins
```

站点访客链路完全不变：

```
SDK → /api/v1/auth/{github,oidc}/... → 会话(auth_method=github/oidc) → 站点 API
```

### 职责划分

- **httpapi**：新增「本地会话」中间件；移除后台外部登录路由；新增管理员账号与绑定接口；
  回调失败时对后台绑定流程回跳账号页。
- **AuthService**：本地登录与会话签发时写入会话来源；提供后台绑定/解绑；维护解绑规则
  所需的本地凭据查询。
- **AdminService**：管理员账号生命周期（新建、为已有用户开通、重置密码、授予/撤销超管、
  最后超管保护、封禁保护）。
- **Store**：会话来源列与迁移；本地凭据列表/计数；用户名唯一冲突映射；事务建号。

## 数据模型变更

**会话（唯一表结构变更）**

- 新增字段 `auth_method`，取值 `local` / `github` / `oidc` / `dev` / `legacy`。
- 新迁移 `0003_session_auth_method.sql`：
  `ALTER TABLE sessions ADD COLUMN auth_method TEXT NOT NULL DEFAULT 'legacy'`。
- 升级前已存在的会话全部落为 `legacy`，站点 API 继续可用，管理 API 一律拒绝。
- 会话来源由服务端在创建会话时写入，客户端不可指定。

**本地凭据**

- 表结构不变（已按 `user_id` 主键、`username` 不区分大小写唯一）。
- 语义从「单个超管」扩展为「管理员可多条」，bootstrap 仍只在零凭据时创建。
- 新增读模型 `LocalAdminAccount`：用户 ID、用户名、显示名、状态、是否超管、是否强制改密、
  站点管理员站点列表、创建时间。由联表/聚合查询生成，不新增表。

**核心不变量**

1. 管理 API 必须同时满足：会话有效 + `auth_method=local` + 角色匹配。
2. 系统中始终至少保留一个「未封禁、有本地凭据、超管角色」的账号。
3. 本地凭据只能由 bootstrap、超管后台操作或 CLI 创建，站点访客登录不会产生本地凭据。
4. 绑定身份不创建用户、不创建会话、不改变角色；冲突不合并。
5. 站点访客的登录、批注、审核数据模型不变。

## 核心数据结构

### domain 层

```go
type AuthMethod string

const (
    AuthMethodLocal  AuthMethod = "local"
    AuthMethodGitHub AuthMethod = "github"
    AuthMethodOIDC   AuthMethod = "oidc"
    AuthMethodDev    AuthMethod = "dev"
    AuthMethodLegacy AuthMethod = "legacy" // 迁移前的旧会话
)

type Session struct {
    // 现有字段...
    AuthMethod AuthMethod `json:"-"`
}

// 管理员账号读模型，不新增表。
type LocalAdminAccount struct {
    UserID             int64
    Username           string
    DisplayName        string
    Status             UserStatus
    IsSuperadmin       bool
    MustChangePassword bool
    SiteIDs            []int64
    CreatedAt          time.Time
}
```

### store 层

```go
type LocalCredentialRepo interface {
    // 现有方法保持不变
    CreateLocalCredential(ctx, c domain.LocalCredential) error
    LocalCredentialByUsername(ctx, username string) (domain.LocalCredential, error)
    LocalCredentialByUserID(ctx, userID int64) (domain.LocalCredential, error)
    CountLocalCredentials(ctx) (int, error)
    UpdateLocalCredentialPassword(ctx, userID int64, hash string, mustChange bool) error

    // 新增
    ListLocalAdmins(ctx) ([]domain.LocalAdminAccount, error)
    CountActiveSuperadminCredentials(ctx) (int, error)
    CreateLocalAdmin(ctx, u domain.User, c domain.LocalCredential, siteIDs []int64) (domain.User, error)
}
```

- `CreateLocalAdmin` 在单个事务内插入用户、本地凭据与可选站点管理员，避免用户名冲突后
  留下孤儿用户。
- `CreateLocalCredential` 将唯一键冲突映射为 `username_taken` 冲突错误；用户名预检仍由
  service 做友好提示。
- `CountActiveSuperadminCredentials` 统计「未封禁 + 超管角色 + 有本地凭据」的账号数，
  用于最后超管保护。

### service 层

**AuthService 新增/调整**

```go
// 本地登录、GitHub/OIDC 登录、调试登录在创建会话时写入对应 AuthMethod。

func (a *AuthService) StartAdminBindGitHub(ctx, userID int64, returnTo string) (string, error)
func (a *AuthService) StartAdminBindOIDC(ctx, userID, providerID int64, returnTo string) (string, error)
func (a *AuthService) AdminBindableIdentities(ctx, userID int64) (BindableIdentities, error)

type BindableIdentities struct {
    GitHub    bool                `json:"github"`
    Providers []AdminProviderView `json:"providers"`
}
```

- 后台绑定复用现有 OAuth/OIDC 流程，模式为 `bind`，但不签发会话。
- 绑定/回调失败时，把已消费的流程与错误一起返回（`BindError`），HTTP 层据此回跳账号页；
  未取得流程的极端情况仍返回 JSON。
- `UnbindIdentity` 先查本地凭据：有本地密码则允许解绑全部身份；无本地凭据仍要求至少
  保留一个身份。

**AdminService 新增/调整**

```go
type CreateLocalAdminInput struct {
    Username        string
    DisplayName     string
    InitialPassword string // 为空则生成
    IsSuperadmin    bool
    SiteIDs         []int64
}

type OpenLocalCredentialInput struct {
    UserID          int64
    Username        string
    InitialPassword string // 为空则生成
    IsSuperadmin    bool
    SiteIDs         []int64
}

type LocalAdminResult struct {
    Account           domain.LocalAdminAccount `json:"account"`
    GeneratedPassword string                   `json:"generatedPassword,omitempty"`
}

func (a *AdminService) ListLocalAdmins(ctx, p domain.Principal) ([]domain.LocalAdminAccount, error)
func (a *AdminService) CreateLocalAdmin(ctx, p, in CreateLocalAdminInput) (LocalAdminResult, error)
func (a *AdminService) OpenLocalCredential(ctx, p, in OpenLocalCredentialInput) (LocalAdminResult, error)
func (a *AdminService) ResetLocalAdminPassword(ctx, p, userID int64, password string) (string, error)
func (a *AdminService) SetSuperadmin(ctx, p, userID int64, v bool) error
```

- 全部方法要求 `p.IsSuperadmin`；`ResetLocalAdminPassword` 返回生成密码（为空时）。
- 重置自己走账号设置页，管理接口拒绝。
- `SetSuperadmin(true)` 要求目标已有本地凭据；`SetSuperadmin(false)` 若目标是最后一个
  可登录超管则拒绝。
- 新账号一律 `MustChangePassword=true`；重置密码吊销全部会话。
- `CreateLocalAdmin` 创建用户 + 凭据 + 站点管理员；`OpenLocalCredential` 不改用户历史
  数据，只补凭据与可选角色。

**AdminService 守卫调整**

- `GlobalBan`：目标是「未封禁 + 超管 + 有本地凭据」且计数 ≤ 1 时拒绝。
- `PromoteUser`：目标必须已有本地凭据；CLI 与后台共用同一守卫。
- 删除 `PromoteByGitHubHandle` 与 `IdentityByHandle` 的调用。

### httpapi 层

```go
func (s *Server) requireLocalSession(next http.Handler) // 403 admin_local_auth_required
```

路由变化：

| 动作 | 设计 |
| --- | --- |
| 后台本地登录 | 保留 `POST /api/v1/admin/auth/local/login` |
| 后台可用登录方式 | `GET /api/v1/admin/auth/providers` 仅返回 `{local:true}` |
| 后台 GitHub/OIDC 直登 | 删除路由与处理函数 |
| 管理员列表 | `GET /api/v1/admin/local-admins` |
| 新建管理员 | `POST /api/v1/admin/local-admins` |
| 为已有用户开通 | `POST /api/v1/admin/local-admins/open` |
| 重置密码 | `POST /api/v1/admin/local-admins/{userID}/password/reset` |
| 授予/撤销超管 | `PUT /api/v1/admin/users/{userID}/superadmin` |
| 绑定列表 | `GET /api/v1/admin/auth/identities` |
| 开始绑定 | `POST /api/v1/admin/auth/identities/{github|oidc/{id}}/start` |
| 解绑 | 复用 `DELETE /api/v1/me/identities/{id}`，规则按本地凭据判断 |

- 所有管理路由统一链：`requireAuth → requireLocalSession → requirePasswordChanged → 角色`。
- 站点管理员列表响应补 `displayName`、`localUsername`、`mustChangePassword`，供后台标注
  与开通入口。
- 绑定回调在 `bind` 模式下成功回跳 `#annot_bind=ok`，失败回跳
  `#annot_bind=error&message=...`。

## 模块交互

### 本地登录与鉴权

```
LoginView → POST /admin/auth/local/login
  → LocalLogin：查用户名 → 等价耗时校验 → bcrypt 比对 → 检查封禁
  → 创建会话（auth_method=local）→ 返回 token + mustChangePassword
后续管理请求：Bearer → AuthenticateSession → requireLocalSession
  → auth_method != local ⇒ 403 admin_local_auth_required
  → mustChangePassword ⇒ 仅账号相关接口可用
```

### 新建管理员 / 为已有用户开通

```
AdminsView / UsersView / SiteAdminsTab
  → POST /admin/local-admins 或 /admin/local-admins/open
  → AdminService 校验：超管、用户名格式与唯一、密码策略、目标用户存在且未封禁
  → bcrypt 哈希 → 单事务写入用户 + 本地凭据 + 可选站点管理员
  → 返回账号信息；若密码为系统生成，仅本次响应返回 generatedPassword
  → 前端弹窗展示一次并要求复制，账号 mustChangePassword=true
```

### 重置密码

```
AdminsView → POST /admin/local-admins/{userID}/password/reset
  → 禁止重置自己（引导到账号设置页改密）
  → 校验或生成新密码 → 更新哈希、mustChangePassword=true
  → 吊销该用户全部会话 → 返回 generatedPassword（如生成）
```

### 最后超管保护

```
撤销超管 / 封禁用户
  → 目标是否为「未封禁 + 超管 + 有本地凭据」
  → 是且计数 ≤ 1 ⇒ 409 last_superadmin，拒绝操作
授予超管
  → 目标无本地凭据 ⇒ 409 local_credential_required
```

### 身份绑定

```
AccountView → GET /admin/auth/identities
  → 已绑定身份 + GitHub 是否可用 + 已启用站点 OIDC 列表
点击绑定 → POST /admin/auth/identities/{github|oidc/{id}}/start
  → 创建 bind 流程，ReturnTo 固定为 /admin/account
  → 跳转 IdP → 回调 /api/v1/auth/{github|oidc}/callback
  → 成功：绑定身份，回跳 /admin/account#annot_bind=ok
  → 失败：带回流程上下文，回跳 #annot_bind=error&message=...
  → 已属于其他用户 ⇒ 409，不合并、不重复建号
```

### 升级后恢复

```
升级 → 迁移加列 → 旧会话 auth_method=legacy → 管理 API 403
  → 超管用本地密码登录 → 在「管理员账号」为原站点管理员开通本地密码
  → 站点管理员重新登录后台
```

## 管理后台界面

- **LoginView**：仅保留用户名/密码表单；无本地凭据时提示「尚未初始化本地管理员」；
  删除 GitHub/OIDC 按钮与相关 store 方法。
- **新增 AdminsView**（`/admins`，仅超管，侧边栏「管理员账号」）：
  - 列表：用户名、显示名、角色、可管理站点、状态、是否待改密。
  - 操作：重置密码、授予/撤销超管。
  - 「新建管理员」：用户名、显示名、初始密码（可生成）、超管开关、站点多选。
  - 「从已有用户开通」：选择用户、用户名、初始密码（可生成）、可选角色。
  - 生成密码用一次性弹窗展示，支持复制，关闭后不可再查看。
- **UsersView**：增加「本地账号」列；未开通的用户提供「开通本地密码」入口，复用同一弹窗。
- **SiteAdminsTab**：展示站点管理员是否已开通本地密码；未开通时显示「开通」按钮并提示
  其当前无法登录后台。
- **AccountView**：新增「登录身份」区：
  - 列出已绑定身份，支持解绑。
  - 绑定按钮：GitHub（已配置时）与按站点分组的 OIDC 提供商。
  - 处理 `#annot_bind=ok|error` 回跳提示，随后清理地址栏 hash 并刷新身份列表。

## CLI

- 删除 `admin promote --github-login`；`--user-id` 保留，但目标必须已有本地凭据。
- 新增 `admin users open --user-id <id> --username <name> [--password <pwd>]`：
  为已有用户开通本地密码，省略密码时生成并打印一次。
- `admin password reset --username <name>` 继续可用，覆盖任意本地管理员。
- `admin user list` 增加本地用户名列，便于离线排查。

## 文件组织

```
internal/domain/
  entities.go              Session.AuthMethod；新增 LocalAdminAccount
  enums.go                 AuthMethod 常量

internal/store/
  migrations/0003_session_auth_method.sql   新增 auth_method 列
  session_repo.go          读写会话来源
  local_credential_repo.go 管理员列表、超管计数、事务建号、唯一冲突映射
  interfaces.go            LocalCredentialRepo 扩展

internal/service/
  auth.go                  会话来源、后台绑定、解绑规则、移除自动超管
  bind_error.go            绑定失败携带流程上下文
  admin.go                 管理员生命周期、最后超管守卫、移除 GitHub 按名提升

internal/auth/
  username.go              用户名校验
  username_test.go

internal/httpapi/
  router.go                路由增删与统一管理链
  middleware.go            requireLocalSession
  admin_auth_handlers.go   仅保留本地登录；新增后台绑定
  admin_account_handlers.go 新增：管理员列表/建号/开通/重置/超管开关
  auth_handlers.go         绑定失败回跳
  admin_handlers.go        站点管理员响应补充本地账号状态
  local_auth_test.go       扩充
  admin_accounts_test.go   新增：建号、重置、守卫、绑定冲突
  server_test.go           测试夹具新增本地登录辅助

internal/config/
  config.go                删除 SuperadminGitHubLogins
  config_test.go

cmd/zhupi/
  main.go                  AdminService 依赖、CLI 增删
  provider_cli_test.go     构造器同步
```

```
admin/src/
  api/types.ts             新类型：LocalAdminAccount、BindableIdentities、Identity
  api/index.ts             管理员与绑定接口
  stores/auth.ts           删除 loginGitHub/loginOIDC
  views/LoginView.vue      只保留本地登录
  views/AdminsView.vue     新增：管理员账号管理
  views/UsersView.vue      本地账号列与开通入口
  views/AccountView.vue    登录身份绑定/解绑与回跳提示
  components/SiteAdminsTab.vue       本地账号状态与开通按钮
  components/LocalAdminFormDialog.vue 新增：新建/开通共用表单
  router.ts                /admins 路由
  layout/AdminLayout.vue   侧边栏入口
admin/e2e/console.mjs      改为本地登录准备
```

```
website/src/content/docs/sites/authentication.md
website/src/content/docs/api/admin-account.md
website/src/content/docs/cli/reference.md
website/src/content/docs/deployment/configuration.md
website/src/content/docs/reference/security.md
website/src/content/docs/getting-started/quick-start.md
website/src/content/docs/sites/roles-moderation.md
website/src/content/docs/development/architecture.md      按需更新
skills/zhupi-integration/references/self-hosting.md
README.md                                                按需更新
```

## 技术决策

| 决策点 | 选择 | 理由 |
| --- | --- | --- |
| 如何区分后台会话与访客会话 | 会话表加 `auth_method` 列 | 一次登录一次写入，鉴权随会话读取，不增加额外查询；站点链路零改动 |
| 旧会话处理 | 默认 `legacy`，管理 API 拒绝 | 避免升级时批量吊销全体访客会话，同时直接切断旧后台入口 |
| 后台限制位置 | 服务端中间件 | 仅隐藏前端按钮无法阻止访客 Bearer token 调管理 API |
| 本地账号存储 | 复用现有 `local_credentials` 表 | 表本身已支持多用户，避免无意义迁移 |
| 建号原子性 | 用户 + 凭据（+ 站点角色）单事务 | 用户名冲突时不留下孤儿用户 |
| 最后超管判定 | 未封禁 + 超管 + 有本地凭据 | 角色单独存在无法登录后台，不能算可用超管 |
| 绑定是否可登录 | 否，仅身份关联 | 严格符合「后台仅密码登录」，避免第二条登录路径 |
| 管理员绑定入口 | 新增后台专用绑定接口 | 现有绑定接口要求站点上下文，后台无站点上下文；固定回跳地址也避免开放重定向 |
| GitHub 自动超管 | 完全移除 | 角色本身会带来站点侧审核权限，保留即留下旁路 |
| 管理员创建入口 | 新页面 + 用户页/站点管理员页快捷入口 | 集中管理，同时覆盖「新建」和「从已有用户开通」两条路径 |
| 生成密码 | 只在响应中返回一次，不落库明文 | 符合现有 CLI 行为，降低泄露面 |
| 测试策略 | 新增本地登录夹具，替换管理 API 用例 | 访客会话与管理会话语义已分离，测试需显式区分 |

## 部署与存储

- 不新增外部服务或依赖；仍为单机 SQLite。
- 唯一数据库变更是会话表加列，属于增量迁移，启动时自动执行。
- 无新增必填环境变量；`SUPERADMIN_GITHUB_LOGINS` 被忽略且不导致启动失败。
- 文档需说明：升级后原有 OIDC/GitHub 后台入口失效，管理员须由超管开通本地密码。

## 主要风险与取舍

- **升级即锁死旧管理员**：接受直接切换；超管本地账号仍可登录，随后为其他人开通。
  文档与登录页提示「请联系超管开通本地密码」。
- **身份绑定冲突**：已存在的访客账号占用同一 OIDC 身份时无法绑定到新管理员账号；
  应改为对该已有用户「开通本地密码」，文档中说明。
- **测试改动面较大**：约十余处管理 API 用例需要换成本地会话；通过统一夹具函数控制
  改动范围。
- **新增后台页面**：只做账号管理与身份绑定，不扩展用户资料、删除、合并等功能。

## 与 Spec 的覆盖关系

| Spec | Plan 落点 |
| --- | --- |
| F1 登录入口收敛 | 路由表删除外部直登；LoginView 仅本地表单 |
| F2 会话来源与访问控制 | `auth_method` 列 + `requireLocalSession` + 统一管理链 |
| F3 多管理员本地账号 | LocalAdminAccount、AdminService 生命周期、最后超管守卫 |
| F4 强制改密 | 建号/开通/重置置 `MustChangePassword`，沿用现有拦截 |
| F5 身份绑定 | 后台专用绑定接口、BindError 回跳、按凭据调整解绑规则 |
| F6 移除自动超管 | 删除配置、`applyAutoSuperadmin`、按 GitHub 名提升 |
| F7 访客侧不变 | 站点路由、SDK 与身份表均不改 |
| F8 升级与 CLI | 迁移加列、CLI open/reset、旧会话 legacy |
