# 本地管理员账号密码登录 · Spec

> 状态：已确认（v1.0）
> 范围：管理后台本地超级管理员账号密码登录、强制改密、账号设置页与 CLI 重置
> 关联：`.specs/admin-console`（管理后台）、GitHub / 站点 OIDC 登录

## 背景

管理后台目前只支持 GitHub 与站点 OIDC 登录。全新部署或 GitHub 不可用时，
没有任何方式进入后台，无法创建站点、配置 OIDC，形成引导死锁。

需要一个**仅用于超级管理员**的本地账号密码登录作为兜底入口。站点管理员仍通过
GitHub / 站点 OIDC 登录，本地账号不参与站点级权限体系。

## 目标

- 服务启动时自动初始化一个本地超级管理员账号，凭据落库（bcrypt 哈希）。
- 默认账号密码为 `admin` / `admin`，首次登录强制修改密码。
- 支持通过环境变量指定初始账号密码，以及显式重置密码。
- 管理后台提供账号设置页，可修改本地账号密码。
- 提供 CLI 重置命令，用于忘记密码时离线恢复。
- 本地账号密码登录失败有基础限流，错误信息不泄露账号是否存在。
- 不破坏现有 GitHub / OIDC 登录与权限模型。

## 角色与边界

- **本地超级管理员**：唯一使用本地账号密码登录的角色，拥有全部超管权限。
- 本地账号只允许存在一个（单行凭据表），不支持多本地账号。
- 站点管理员、普通用户仍只能通过 GitHub / 站点 OIDC 登录。
- 本地账号可以同时绑定 GitHub / OIDC 身份，但绑定不是本期目标。

## 功能需求

### F1 初始化与配置

- 新增环境变量：
  - `LOCAL_ADMIN_USERNAME`：本地账号用户名，默认 `admin`。
  - `LOCAL_ADMIN_PASSWORD`：初始密码，默认 `admin`。
  - `LOCAL_ADMIN_RESET_PASSWORD`：设为 `1` 时，服务启动后用上面的密码重置本地账号。
- 服务启动时执行 bootstrap：
  - 若不存在任何本地凭据：创建超级管理员用户 + 本地凭据。
  - 若已存在本地凭据且未开启重置：**不覆盖**后台已修改的密码。
  - 若已存在且开启重置：更新密码哈希，吊销该用户全部会话，并在日志中告警。
- 密码使用 bcrypt（cost 12）哈希后存储，任何日志与 API 都不返回哈希。
- 初始密码等于 `admin` 时标记 `must_change_password = 1`；否则为 `0`。

### F2 本地登录

- 新增公开接口 `POST /api/v1/admin/auth/local/login`：
  - 请求：`{"username": "...", "password": "..."}`。
  - 成功：返回 `token`、`expiresAt`、`user`、`mustChangePassword`。
  - 失败：统一返回 401「用户名或密码错误」，不区分账号不存在与密码错误。
  - 被封禁用户返回 403。
- 登录接口按 IP 限流（默认 10 次/分钟），超限返回 429。
- 登录成功创建普通 session，与 GitHub / OIDC 登录共用会话体系。
- `GET /api/v1/admin/auth/providers` 增加 `local: true`，登录页据此展示表单。

### F3 强制改密

- `GET /api/v1/me` 增加 `username` 与 `mustChangePassword` 字段。
- `mustChangePassword = true` 时，除以下接口外，所有管理 API 返回
  403 `password_change_required`：
  - `GET /api/v1/me`
  - `POST /api/v1/me/password`
  - `POST /api/v1/auth/logout`
  - `GET /api/v1/me/sessions`、`DELETE /api/v1/me/sessions*`
  - `GET /api/v1/me/identities`、`DELETE /api/v1/me/identities/{id}`
- 管理后台路由守卫在检测到 `mustChangePassword` 时强制跳转 `/account`。

### F4 修改密码

- 新增认证接口 `POST /api/v1/me/password`：
  - 请求：`{"currentPassword": "...", "newPassword": "..."}`。
  - 校验当前密码；新密码长度 8–72 字节，且不能与当前密码相同。
  - 成功后更新哈希、清除 `must_change_password`、吊销**其他**会话，保留当前会话。
  - 当前密码错误返回 400 `invalid_current_password`。
- 管理后台新增「账号设置」页 `/account`：
  - 展示用户名、角色。
  - 修改密码表单（当前密码、新密码、确认新密码）。
  - 强制改密模式下提示「首次登录必须修改密码」，成功后回到概览。

### F5 CLI 重置

- 新增命令：`zhupi admin password reset --username <name> [--password <pwd>]`。
- 未提供 `--password` 时生成 16 位随机密码并打印一次。
- 重置成功后吊销该用户全部会话；密码为 `admin` 时仍要求下次登录改密。
- 本地凭据不存在时报错并提示先启动一次服务完成初始化。

### F6 兼容与安全

- 本地登录与现有 session / Bearer token 完全兼容。
- 不新增 Cookie；不改变 CORS 与 CSRF 现状。
- 密码字段不进入 `domain.User`，避免用户列表等接口意外泄露。
- 登录失败与改密失败均记录必要的审计日志（不含明文密码）。

## 不做的事

- 不做多本地账号、角色分配、账号注册、找回密码邮件。
- 不做 TOTP / WebAuthn 二次验证。
- 不做密码历史、复杂度评分、定期过期。
- 不做本地账号与 GitHub / OIDC 的绑定管理界面。
- 不改变站点管理员的登录方式与权限模型。

## 验收标准

- **AC1** 全新数据库启动后，`local_credentials` 中存在一条 `admin` 记录，
  对应用户为超级管理员，密码哈希为 bcrypt。
- **AC2** `admin` / `admin` 可登录，返回 `mustChangePassword: true`。
- **AC3** 强制改密状态下访问 `/api/v1/admin/sites` 返回 403
  `password_change_required`，访问 `/api/v1/me` 正常。
- **AC4** 调用 `POST /api/v1/me/password` 修改密码后，旧密码登录失败、新密码登录成功，
  `mustChangePassword` 变为 `false`。
- **AC5** 修改密码会吊销其他会话，当前会话仍可用。
- **AC6** 重启服务且未设置 `LOCAL_ADMIN_RESET_PASSWORD=1` 时，后台修改的密码不被覆盖。
- **AC7** 设置 `LOCAL_ADMIN_RESET_PASSWORD=1` 重启后，密码恢复为环境变量值，会话被吊销。
- **AC8** `zhupi admin password reset --username admin` 可生成新密码并成功登录。
- **AC9** 登录页展示本地账号表单，登录成功后进入后台；强制改密时只能进入账号设置页。
- **AC10** `go vet ./...`、`go test ./...`、`cd admin && npm run typecheck && npm test` 全部通过。
