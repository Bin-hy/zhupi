---
title: API 概览
description: 基础地址、站点请求头、Bearer 鉴权、CORS、错误结构、限流与分页约定。
---

API 统一挂在 `/api/v1` 下，请求与响应均为 JSON（除 OAuth 跳转与静态资源）。

## 请求头

### 站点级接口

```http
GET /api/v1/annotations?page=/posts/1
X-Site-Key: <siteKey>
Origin: https://blog.example.com
Authorization: Bearer <token>   # 可选；写入接口必须
```

- `X-Site-Key`：站点公开标识，SDK 自动携带。
- `Origin`：必须命中站点白名单；**非浏览器客户端也要显式发送**。
- `Authorization`：读取可匿名；写入必须登录。

### 管理接口

```http
Authorization: Bearer <admin-token>
```

管理接口通过路径里的站点 ID 限定范围，不要求 `X-Site-Key`。跨域访问管理 API 需要配置 `ADMIN_ORIGINS`。

## CORS

来源命中站点白名单时，服务返回：

```http
Access-Control-Allow-Origin: <origin>
Access-Control-Allow-Headers: Authorization, Content-Type, X-Site-Key
Access-Control-Allow-Methods: GET, POST, PATCH, DELETE, OPTIONS
Access-Control-Max-Age: 600
Vary: Origin
```

预检请求（`OPTIONS`）不带 `X-Site-Key` 时，只校验来源是否属于任一启用站点；真正的请求会再次严格校验 siteKey 与来源。

## 错误结构

所有错误统一为：

```json
{
  "error": {
    "code": "origin_not_allowed",
    "message": "该来源不允许访问此站点",
    "requestId": "9f2c1a7b3d4e5f60"
  }
}
```

- `code` 机器可读，`message` 面向人，`requestId` 用于排查。
- 响应头同样带 `X-Request-Id`。
- 500 错误不会暴露内部细节；日志里按 `requestId` 关联。

常见错误码：

| code | HTTP | 含义 |
| --- | --- | --- |
| `site_key_required` | 400 | 缺少 `X-Site-Key` |
| `site_not_found` | 404 | siteKey 不存在或已轮换 |
| `site_disabled` | 403 | 站点已停用 |
| `origin_required` | 403 | 缺少 `Origin`（且未开启本地放行） |
| `origin_not_allowed` | 403 | 来源不在白名单 |
| `unauthorized` | 401 | 未登录或会话失效 |
| `forbidden` | 403 | 已登录但无权限 |
| `site_banned` | 403 | 被该站点封禁 |
| `user_banned` | 403 | 账号被全局封禁 |
| `rate_limited` | 429 | 触发限流 |
| `validation_error` | 400 | 参数不合法或请求体过大 |
| `password_change_required` | 403 | 本地管理员首次登录未改密 |

## 限流

| 类型 | 默认 | 配置 |
| --- | --- | --- |
| 读请求（按 IP） | 300 / 分钟 | `READ_RATE_PER_MIN` |
| 写请求（按 IP） | 60 / 分钟 | `WRITE_RATE_PER_MIN` |
| 写请求（按用户） | 30 / 分钟 | `WRITE_USER_RATE_PER_MIN` |
| 本地登录（按 IP） | 10 / 分钟 | `LOCAL_LOGIN_RATE_PER_MIN` |

计数在进程内，重启清零。触发后返回 `429` 与 `rate_limited`。

## 请求体与分页

- JSON 请求体上限 **128 KiB**，超过返回 `400`。
- 批注列表使用游标分页：`cursor` 是上一页最后一条的 ID，`nextCursor=0` 表示没有更多。
- 列表默认 `limit=100`；超过 200 时回退为 100。

## 接口分组

| 分组 | 路径 | 文档 |
| --- | --- | --- |
| 站点公共 | `/api/v1/site-config` · `/api/v1/annotations` · `/api/v1/comments` | [批注与评论 API](/api/annotations/) |
| 登录 | `/api/v1/auth/*` | [登录与身份](/sites/authentication/) |
| 账号 | `/api/v1/me*` | [管理与账号 API](/api/admin-account/) |
| 管理 | `/api/v1/admin/*` | [管理与账号 API](/api/admin-account/) |
| 健康检查 | `/healthz` · `/readyz` | 本页 |

## 健康检查

```http
GET /healthz
```

```json
{ "status": "ok" }
```

```http
GET /readyz
```

数据库可用时：

```json
{ "status": "ready" }
```

数据库不可用时返回 `503` 与 `{"status":"db_unavailable"}`。

## 版本

路径版本为 `/api/v1`。V1 不承诺向后兼容旧单站点 API；字段会以追加方式演进，破坏性变更会升版本。
