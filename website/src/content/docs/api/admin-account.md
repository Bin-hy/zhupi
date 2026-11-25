---
title: 管理与账号 API
description: 管理端登录、站点与用户管理、审核队列、封禁、OIDC 提供商，以及当前账号的身份与会话接口。
---

管理 API 使用 `Authorization: Bearer <admin-token>`。站点级管理通过路径中的 `siteID` 限定，不要求 `X-Site-Key`。

## 管理端登录

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/v1/admin/auth/providers` | 可用的登录方式（仅本地） |
| POST | `/api/v1/admin/auth/local/login` | 本地管理员账号密码登录 |

后台只接受本地账号密码。管理 API 只接受由该接口建立的会话；GitHub / 站点 OIDC
会话即使有效也不能访问管理接口。

本地登录：

```http
POST /api/v1/admin/auth/local/login
Content-Type: application/json

{ "username": "admin", "password": "***" }
```

```json
{
  "token": "…",
  "expiresAt": "2026-12-25T12:00:00Z",
  "user": { "id": 1, "displayName": "admin", "isSuperadmin": true },
  "mustChangePassword": true
}
```

`mustChangePassword=true` 时，除账号接口外管理 API 返回 `403 password_change_required`。

## 管理员账号（超管）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/v1/admin/local-admins` | 管理员列表（用户名、角色、站点、状态） |
| POST | `/api/v1/admin/local-admins` | 新建管理员 |
| POST | `/api/v1/admin/local-admins/open` | 为已有用户开通本地密码 |
| POST | `/api/v1/admin/local-admins/{userId}/password/reset` | 重置密码 |
| PUT | `/api/v1/admin/users/{userId}/superadmin` | 授予 / 撤销超级管理员 |

新建示例：

```http
POST /api/v1/admin/local-admins
Content-Type: application/json

{
  "username": "ops",
  "displayName": "运维",
  "initialPassword": "",
  "isSuperadmin": false,
  "siteIds": [1, 2]
}
```

- `initialPassword` 留空时生成随机密码，只在本次响应返回 `generatedPassword`。
- 新账号与重置后的账号首次登录必须改密。
- 用户名 3–32 字符，仅字母、数字、`.`、`_`、`-`，不区分大小写唯一。
- 重置密码会吊销目标账号全部会话；不能重置自己（请到账号设置页改密）。
- 授予超管前，目标用户必须已开通本地密码。
- 不能撤销或封禁最后一个可登录的超级管理员。

## 当前账号

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/v1/me` | 当前用户、`mustChangePassword`、本地用户名 |
| POST | `/api/v1/me/password` | 修改密码（`currentPassword` / `newPassword`） |
| GET | `/api/v1/me/sites` | 当前用户可管理的站点 |
| GET | `/api/v1/me/identities` | 已绑定身份 |
| DELETE | `/api/v1/me/identities/{id}` | 解绑身份 |
| GET | `/api/v1/me/sessions` | 登录设备与当前会话 ID |
| DELETE | `/api/v1/me/sessions/{id}` | 退出指定设备 |
| DELETE | `/api/v1/me/sessions` | 退出所有设备 |
| POST | `/api/v1/me/identities/github/start` | 绑定 GitHub（返回跳转地址） |
| POST | `/api/v1/me/identities/oidc/{providerId}/start` | 绑定 OIDC（返回跳转地址） |
| GET | `/api/v1/admin/auth/identities` | 后台：已绑定身份与可绑定提供商 |
| POST | `/api/v1/admin/auth/identities/github/start` | 后台：发起 GitHub 绑定 |
| POST | `/api/v1/admin/auth/identities/oidc/{providerId}/start` | 后台：发起 OIDC 绑定 |

绑定接口需要同时带站点上下文（`X-Site-Key` 与 `Origin`），因为 OIDC 提供商属于站点；
后台专用绑定接口不需要站点上下文。

绑定只建立身份关联：不创建用户、不创建会话，也不能用于后台登录。绑定失败会回到
`/admin/account#annot_bind=error&message=...`；成功后回到 `#annot_bind=ok`。
已属于其他用户的身份会返回冲突，不做账号合并。

## 站点与用户（超管）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/v1/admin/overview` | 概览统计 |
| GET | `/api/v1/admin/sites` | 站点列表 |
| POST | `/api/v1/admin/sites` | 创建站点 |
| PATCH | `/api/v1/admin/sites/{id}` | 修改站点设置 |
| POST | `/api/v1/admin/sites/{id}/rotate-key` | 轮换 siteKey |
| GET | `/api/v1/admin/sites/{id}/admins` | 站点管理员列表 |
| POST | `/api/v1/admin/sites/{id}/admins` | 添加站点管理员（`userId`） |
| DELETE | `/api/v1/admin/sites/{id}/admins/{userId}` | 移除站点管理员 |
| GET | `/api/v1/admin/users` | 全局用户列表 |
| POST | `/api/v1/admin/users/{id}/ban` | 全局封禁 |
| DELETE | `/api/v1/admin/users/{id}/ban` | 解除全局封禁 |

站点设置可修改：`displayName`、`origins`、`moderationMode`、`themeColor`、`defaultLocale`、`status`。

## 审核（站点管理员 / 超管）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/v1/admin/sites/{siteID}/queue` | 审核队列 |
| GET | `/api/v1/admin/sites/{siteID}/annotations/{id}` | 批注详情与完整讨论 |
| POST | `/api/v1/admin/sites/{siteID}/annotations/{id}/moderate` | 审核批注 |
| POST | `/api/v1/admin/sites/{siteID}/comments/{id}/moderate` | 审核评论 |
| GET | `/api/v1/admin/sites/{siteID}/bans` | 站点封禁列表 |
| POST | `/api/v1/admin/sites/{siteID}/bans/{userID}` | 站点封禁（`reason` 可选） |
| DELETE | `/api/v1/admin/sites/{siteID}/bans/{userID}` | 解除站点封禁 |

### 队列参数

| 参数 | 默认 | 说明 |
| --- | --- | --- |
| `status` | `pending` | `pending` / `approved` / `rejected` |
| `type` | `all` | `all` / `annotation` / `comment` |
| `q` | 空 | 关键词，最多 100 字 |
| `annotationCursor` | `0` | 批注游标 |
| `commentCursor` | `0` | 评论游标 |
| `limit` | `50` | 每类返回数量 |

响应中同时返回批注、评论、下一游标与各状态计数；评论会带 `parent` 批注摘要。

### 审核动作

```http
POST /api/v1/admin/sites/1/annotations/123/moderate
Content-Type: application/json

{ "action": "approve" }
```

| action | 效果 |
| --- | --- |
| `approve` | 通过，并设为可见 |
| `reject` | 拒绝 |
| `hide` | 隐藏（保留审核状态） |
| `restore` | 恢复可见 |

评论使用同样的 action 语义。

## OIDC 提供商（站点管理员 / 超管）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/v1/admin/sites/{siteID}/providers` | 提供商列表 |
| POST | `/api/v1/admin/sites/{siteID}/providers` | 新建 |
| PATCH | `/api/v1/admin/sites/{siteID}/providers/{providerID}` | 修改 |
| DELETE | `/api/v1/admin/sites/{siteID}/providers/{providerID}` | 删除 |

请求字段：

```json
{
  "name": "公司 SSO",
  "issuer": "https://sso.example.com",
  "clientId": "zhupi",
  "clientSecret": "***",
  "scopes": ["openid", "email", "profile"],
  "allowedEmailDomains": ["example.com"],
  "enabled": true
}
```

- 保存时会做 OIDC Discovery 校验。
- `clientSecret` 使用 AES-256-GCM 加密存储，响应中不会回显。
- 修改时 `clientSecret` 留空表示保持原值。
- 更推荐用 CLI 的 `--client-secret-file`，避免出现在 shell history。

## 示例

```bash
TOKEN='<admin token>'

curl -s http://127.0.0.1:8080/api/v1/admin/sites \
  -H "Authorization: Bearer $TOKEN"

curl -s 'http://127.0.0.1:8080/api/v1/admin/sites/1/queue?status=pending&type=all' \
  -H "Authorization: Bearer $TOKEN"

curl -s -X POST http://127.0.0.1:8080/api/v1/admin/sites/1/annotations/123/moderate \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"action":"approve"}'
```

## 与 CLI 的分工

- **HTTP API**：适合管理后台、自动化脚本与远程操作。
- **CLI**：适合服务器初始化、站点创建、管理员引导与密码重置；需要直接访问数据库文件。

两者最终操作同一套 service/store。详见 [CLI 参考](/cli/reference/)。
