---
title: 登录与身份
description: GitHub OAuth、站点级 OIDC、本地超级管理员、身份绑定与会话管理。
---

朱批有三类登录方式：

1. **全局 GitHub OAuth**：部署级配置，所有站点可用；
2. **站点级 OIDC**：每个站点独立配置，适合公司 SSO；
3. **本地超级管理员**：账号密码兜底，仅用于管理后台。

普通用户首次通过任一提供商登录时自动建号；之后可以在账号中心绑定更多身份。

## GitHub OAuth（全局）

1. 在 GitHub 创建 OAuth App；
2. 回调地址填：

```text
${PUBLIC_BASE_URL}/api/v1/auth/github/callback
```

3. 配置环境变量并重启：

```dotenv
GITHUB_CLIENT_ID=...
GITHUB_CLIENT_SECRET=...
```

未配置时，`GET /api/v1/site-config` 不会返回 GitHub 提供商，SDK 也不显示 GitHub 登录入口。

### 自动提升超管

```dotenv
SUPERADMIN_GITHUB_LOGINS=your-github-login,teammate
```

命中名单的用户登录后自动成为超级管理员。也可以之后用 CLI 提升：

```bash
./bin/zhupi admin promote --github-login <login>
./bin/zhupi admin promote --user-id <id>
```

## 站点级 OIDC

每个站点可以配置多个 OIDC 提供商（例如 yufu、公司 SSO）。保存时服务端会做一次 Discovery 校验；client secret 使用 AES-256-GCM 加密存储。

### 在管理后台配置

超级管理员或站点管理员进入站点详情 → OIDC 提供商，填写：

- 展示名
- Issuer（例如 `https://sso.example.com`）
- Client ID / Client Secret
- Scopes（默认 `openid,profile,email`）
- 允许的邮箱域（可选）
- 是否启用

### 用 CLI 配置

```bash
# 添加
./bin/zhupi admin provider add \
  --site <siteKey 或 ID> \
  --name "公司 SSO" \
  --issuer https://sso.example.com \
  --client-id zhupi \
  --client-secret '***' \
  --scopes openid,profile,email \
  --allowed-email-domains example.com

# 列表 / 更新 / 删除
./bin/zhupi admin provider list --site <siteKey>
./bin/zhupi admin provider update --id <providerId> --enabled false
./bin/zhupi admin provider delete --id <providerId>
```

也可以把 secret 放进文件，避免出现在 shell history：

```bash
./bin/zhupi admin provider add ... --client-secret-file /run/secrets/oidc
```

### 回调地址

在 IdP 中注册：

```text
${PUBLIC_BASE_URL}/api/v1/auth/oidc/{providerId}/callback
```

`providerId` 是朱批内的提供商 ID，不是 client ID。

## 本地超级管理员

首次启动时，如果数据库中没有本地凭据，会自动创建：

| 变量 | 默认值 |
| --- | --- |
| `LOCAL_ADMIN_USERNAME` | `admin` |
| `LOCAL_ADMIN_PASSWORD` | `admin` |

默认密码会在首次登录时强制修改，未改密前只能访问账号设置相关接口。

重置密码（会吊销该账号全部会话）：

```bash
./bin/zhupi admin password reset --username admin
./bin/zhupi admin password reset --username admin --password 'new-password'
```

本地登录按 IP 限流，默认每分钟 10 次（`LOCAL_LOGIN_RATE_PER_MIN`）。

## 身份与会话

- 一个用户可以有多个身份：GitHub 全局身份 + 各站点 OIDC 身份。
- 身份唯一键为 `(provider, siteId, issuer, subject)`；同一邮箱在不同站点不会被自动合并。
- 绑定入口：SDK 账号中心，或管理后台账号页。
- 会话使用 Bearer token；数据库只保存 token 的 SHA-256。
- 会话有效期由 `SESSION_TTL_HOURS` 控制（默认 720 小时）。
- 用户可以查看登录设备并单独/全部退出。

## 开发登录

仅本地联调：

```dotenv
DEV_LOGIN=1
```

SDK 初始化时传 `devLogin: true`，即可通过 `/api/v1/auth/dev/login` 创建会话。生产环境严禁开启。

## 登录流程

```text
用户点击登录
  → GET /api/v1/auth/{github|oidc/...}/start
  → 跳转 IdP
  → 回调 ${PUBLIC_BASE_URL}/api/v1/auth/...
  → 校验 state / PKCE / nonce
  → 创建或匹配身份
  → 建立 Bearer 会话
  → 回跳宿主机页面（校验站点 Origin）
```

“绑定身份”走同一套 OAuth/OIDC 流程，但模式是 `bind`，会绑定到当前登录用户，而不是创建新用户。

## 安全建议

- 生产只开需要的登录方式；本地超管不要把默认密码留在生产。
- 设置 `SUPERADMIN_GITHUB_LOGINS` 或尽快提升一个可信账号，避免只依赖本地密码。
- OIDC 尽量限制 `allowedEmailDomains`。
- 定期检查登录设备与异常 IP。
- 轮换 `SECRET_KEY` 会导致已保存的 OIDC client secret 不可用，需重新录入。

更多边界见 [安全边界](/reference/security/)。
