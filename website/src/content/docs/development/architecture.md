---
title: 架构与数据模型
description: Go 分层、SQLite 表结构、鉴权请求链、锚点与审核流转、SDK/管理后台内嵌方式。
---

## 分层

```text
cmd/zhupi              命令行入口（serve / admin）
internal/config        环境变量加载与校验
internal/crypto        AES-256-GCM（OIDC client secret）
internal/domain        实体、枚举、对外错误
internal/store         SQLite 仓储与迁移
internal/service       业务规则（站点、批注、审核、认证）
internal/auth          GitHub OAuth / OIDC / 密码哈希
internal/httpapi       路由、中间件、Handler
internal/ratelimit     进程内限流
internal/sdkassets     go:embed SDK 产物
internal/adminassets   go:embed 管理后台产物
sdk/                   TypeScript SDK（Shadow DOM）
admin/                 Vue 3 管理后台
deploy/                Caddy / Nginx / systemd 示例
examples/              纯 HTML 示例
website/               Astro 官方文档（本页）
```

服务只依赖标准库与少量成熟库：`go-oidc`、`oauth2`、`x/crypto`、`modernc.org/sqlite`（纯 Go，无 CGO）。

## 请求链

```text
recoverer → requestID → logging → bodyLimit → cors → mux
  → siteResolve（X-Site-Key + Origin）
  → optionalAuth / requireAuth（Bearer）
  → resolveSiteRole（站点管理员判定）
  → requireSiteNotBanned
  → readLimit / writeLimit
  → handler
  → service
  → store
  → SQLite
```

- 每个请求生成 16 位 hex `requestId`，写入 `X-Request-Id` 与错误体。
- 500 错误只记录日志，不把内部错误暴露给客户端。
- 写请求同时按 IP 与用户限流。

## 数据模型

SQLite 开启 WAL、`busy_timeout(5000)` 与外键。

| 表 | 作用 |
| --- | --- |
| `sites` | 站点：key、名称、状态、审核模式、主题、语言 |
| `site_origins` | 站点来源白名单 |
| `users` | 平台用户：展示名、头像、状态、是否超管 |
| `identities` | 身份：provider / siteId / issuer / subject 唯一 |
| `sessions` | 会话：只存 token 的 SHA-256、UA、IP、过期时间 |
| `local_credentials` | 本地超管：bcrypt 哈希、强制改密标记 |
| `site_admins` | 站点管理员 |
| `site_bans` | 站点封禁 |
| `oidc_providers` | 站点 OIDC：client secret 为 AES-GCM 密文 |
| `auth_flows` | OAuth/OIDC 临时流程：state 哈希、PKCE、nonce |
| `annotations` | 批注：锚点 JSON、引用文本、状态、审核、可见性 |
| `comments` | 评论：挂在批注下，带审核与可见性 |

关键索引：

- `annotations(site_id, page_key, visibility, moderation)`：页面列表；
- `annotations(site_id, moderation, id)`：审核队列；
- `identities(user_id)`、`sessions(user_id)`、`sessions(expires_at)`；
- `auth_flows(expires_at)`。

## 锚点

SDK 在选区上生成 `TextQuoteSelector`（exact/prefix/suffix）与可选字符偏移，服务端存储为 JSON：

```json
{
  "quote": { "type": "TextQuoteSelector", "exact": "…", "prefix": "…", "suffix": "…" },
  "position": { "start": 120, "end": 126 },
  "section": "二级标题"
}
```

渲染时先用 `position` 尝试，再用引用与上下文做模糊匹配；都失败则进入 orphan，交给用户手动重定位。锚点校验在服务端完成（长度、偏移合法性）。

## 鉴权流程

```text
登录：start → IdP → callback → 校验 state/PKCE/nonce
      → 查找或创建 identity/user → 创建 Bearer 会话
绑定：同样的 OAuth/OIDC 流程，但 mode=bind
      → 绑定到当前用户，不创建新用户
```

- `state` 在数据库中只存哈希，并带过期时间，定时清理。
- 会话 token 只在创建时返回一次，数据库只存 SHA-256。
- 全局封禁会在会话认证与本地登录时被拒绝；站点封禁只拦截该站点的写操作。
- 本地超管密码使用 bcrypt；首次登录（`mustChangePassword`）会阻止访问管理 API。

## 审核流转

`pre` 模式：

```text
创建 → pending → approve → approved + visible
                 └ reject → rejected
```

`post` 模式创建即 `approved`。管理员可随时：

- `hide` / `restore` 切换 `visibility`；
- 删除为软删除（`visibility=deleted`）；
- 评论与批注各自独立审核。

## 内嵌资源

- `internal/sdkassets` 与 `internal/adminassets` 用 `go:embed` 把前端产物编进二进制。
- `GET /sdk/v1/*` 直接返回内嵌文件（缓存 5 分钟）。
- `GET /admin/*` 返回 SPA，未命中的路径回退到 `index.html`。
- 因此部署只需要一个二进制 + 一个 SQLite 文件，不需要单独的静态服务器。

## 单实例边界

当前实现面向单实例：

- SQLite 文件锁不适合多进程同时写。
- 限流计数在进程内。
- 会话与 OAuth 流程都在数据库，但没有分布式锁。

仓储层已抽象，替换 PostgreSQL / 多实例属于后续版本工作，V1 不承诺。
