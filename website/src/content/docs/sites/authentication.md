---
title: 登录与身份
description: 站点访客 GitHub / OIDC 登录、管理后台本地账号密码、身份绑定与会话管理。
---

朱批有两类登录：

1. **站点访客**：全局 GitHub OAuth 或站点级 OIDC，用于批注、评论等站点功能；
2. **管理后台**：只接受本地账号密码，用于站点管理、审核与站点配置。

普通访客首次通过任一提供商登录时自动建号。管理员登录后台后，可以在账号设置页
绑定 GitHub / 站点 OIDC 身份，但绑定只用于身份关联，不能作为后台登录方式。

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

GitHub 仅用于站点访客登录，不会自动授予超级管理员，也不能直接登录管理后台。

## 站点级 OIDC

每个站点可以配置多个 OIDC 提供商（例如 yufu、公司 SSO）。保存时服务端会做一次
Discovery 校验；client secret 使用 AES-256-GCM 加密存储。

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

## 管理后台本地账号

首次启动时，如果数据库中没有本地凭据，会自动创建一个超级管理员：

| 变量 | 默认值 |
| --- | --- |
| `LOCAL_ADMIN_USERNAME` | `admin` |
| `LOCAL_ADMIN_PASSWORD` | `admin` |

默认密码会在首次登录时强制修改，未改密前只能访问账号设置相关接口。

进入后台后，超级管理员可以在「管理员账号」页：

- 新建管理员，指定超级管理员或站点管理员角色；
- 为已有访客用户开通本地密码，保留其历史批注与已绑定身份；
- 重置其他管理员的密码并吊销其全部会话；
- 授予或撤销超级管理员。

新账号与重置后的账号首次登录都必须改密。系统始终保留至少一个可登录的超级管理员，
不能撤销或封禁最后一个。

重置密码（会吊销该账号全部会话）：

```bash
./bin/zhupi admin password reset --username admin
./bin/zhupi admin password reset --username admin --password 'new-password'
```

为已有用户开通本地密码：

```bash
./bin/zhupi admin users open --user-id <id> --username <name>
```

本地登录按 IP 限流，默认每分钟 10 次（`LOCAL_LOGIN_RATE_PER_MIN`）。

### 升级说明

- 后台 GitHub / 站点 OIDC 直登已移除。
- 升级前的旧会话不能访问管理 API；管理员需要重新用本地密码登录。
- 原先只靠外部身份登录后台的管理员，需要由超级管理员在后台或 CLI 为其开通本地密码。

## 身份与会话

- 一个用户可以有多个身份：GitHub 全局身份 + 各站点 OIDC 身份。
- 身份唯一键为 `(provider, siteId, issuer, subject)`；同一邮箱在不同站点不会被自动合并。
- 管理员绑定身份只建立关联：不创建新用户、不创建会话，也不能用于后台登录。
- 绑定已被其他用户占用的身份会返回冲突；朱批不会自动合并账号。
- 绑定入口：SDK 账号中心，或管理后台账号设置页。
- 会话使用 Bearer token；数据库只保存 token 的 SHA-256，并记录登录来源。
- 管理 API 只接受本地密码登录建立的会话；站点公共 API 接受任意有效会话。
- 会话有效期由 `SESSION_TTL_HOURS` 控制（默认 720 小时）。
- 用户可以查看登录设备并单独/全部退出。

## 开发登录

仅本地联调：

```dotenv
DEV_LOGIN=1
```

SDK 初始化时传 `devLogin: true`，即可通过 `/api/v1/auth/dev/login` 创建会话。
调试登录只用于站点侧，不能访问管理 API。生产环境严禁开启。

## 登录流程（站点访客）

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

「绑定身份」走同一套 OAuth/OIDC 流程，但模式是 `bind`，会绑定到当前登录用户，
而不是创建新用户或会话。

## 安全建议

- 生产环境必须修改默认本地管理员密码。
- 只给确实需要管理权限的人开通本地账号；离职或停用时及时封禁。
- 至少保留两个可登录的超级管理员，避免单点无法进入后台。
- OIDC 尽量限制 `allowedEmailDomains`。
- 定期检查登录设备与异常 IP。
- 轮换 `SECRET_KEY` 会导致已保存的 OIDC client secret 不可用，需重新录入。

更多边界见 [安全边界](/reference/security/)。
