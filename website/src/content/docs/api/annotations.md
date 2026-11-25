---
title: 批注与评论 API
description: 站点公共接口：站点配置、批注列表、创建/更新/删除、评论与回复。
---

站点级接口需要 `X-Site-Key` 与合法 `Origin`。读取可匿名，写入需要 Bearer 会话。

## 站点配置

```http
GET /api/v1/site-config
X-Site-Key: <siteKey>
Origin: https://blog.example.com
```

```json
{
  "siteKey": "…",
  "displayName": "我的博客",
  "themeColor": "#087f5b",
  "locale": "zh-CN",
  "moderationMode": "post",
  "providers": [
    { "id": 0, "type": "github", "name": "GitHub" },
    { "id": 3, "type": "oidc", "name": "公司 SSO", "issuer": "https://sso.example.com" }
  ]
}
```

SDK 用它渲染登录方式、主题与语言。

## 列出批注

```http
GET /api/v1/annotations?page=/posts/1&filter=all&sort=position&q=缓存&cursor=0&limit=100
```

| 参数 | 默认 | 说明 |
| --- | --- | --- |
| `page` | 必填 | 页面键，1–512 字符 |
| `filter` | `all` | `all` / `mine` / `resolved` / `pending` |
| `sort` | `position` | `position`（文档顺序）/ `newest` |
| `q` | 空 | 关键词，最多 100 字 |
| `cursor` | `0` | 上一页的 `nextCursor` |
| `limit` | `100` | 有效范围 1–200；超过 200 时按 100 处理 |

`mine` 与 `pending` 需要登录。站点管理员/超管可以额外看到未通过审核的内容。

响应：

```json
{
  "annotations": [
    {
      "id": 123,
      "pageKey": "/posts/1",
      "pageTitle": "一篇可以被批注的文章",
      "anchor": {
        "quote": {
          "type": "TextQuoteSelector",
          "exact": "被选中的原文",
          "prefix": "前文",
          "suffix": "后文"
        },
        "position": { "start": 120, "end": 126 },
        "section": "二级标题"
      },
      "quoteExact": "被选中的原文",
      "section": "二级标题",
      "status": "open",
      "moderation": "approved",
      "visibility": "visible",
      "createdAt": "2026-11-25T12:00:00Z",
      "updatedAt": "2026-11-25T12:00:00Z",
      "author": { "id": 7, "name": "Bin", "avatarUrl": "…" },
      "comments": []
    }
  ],
  "nextCursor": 0,
  "total": 1,
  "counts": { "all": 1, "mine": 0, "resolved": 0, "pending": 0 }
}
```

## 创建批注

```http
POST /api/v1/annotations
X-Site-Key: <siteKey>
Origin: https://blog.example.com
Authorization: Bearer <token>
Content-Type: application/json
```

```json
{
  "pageKey": "/posts/1",
  "pageTitle": "一篇可以被批注的文章",
  "content": "这里的设计值得展开讲讲。",
  "anchor": {
    "quote": {
      "type": "TextQuoteSelector",
      "exact": "被选中的原文",
      "prefix": "前文",
      "suffix": "后文"
    },
    "position": { "start": 120, "end": 126 },
    "section": "二级标题"
  }
}
```

校验：

| 字段 | 约束 |
| --- | --- |
| `pageKey` | 1–512 字符 |
| `content` | 1–4000 字 |
| `quote.exact` | 1–2000 字 |
| `quote.prefix` / `suffix` | 各 0–500 字 |
| `section` | 0–200 字 |

成功返回 `201` 与完整批注。`pre` 审核模式下 `moderation` 为 `pending`，普通读者暂不可见。

## 更新批注

```http
PATCH /api/v1/annotations/{id}
Authorization: Bearer <token>
```

```json
{ "status": "resolved" }
```

或重定位锚点：

```json
{ "anchor": { "quote": { "exact": "新的原文", "prefix": "", "suffix": "" } } }
```

只有作者本人、站点管理员或超管可以更新。成功返回更新后的批注。

## 删除批注

```http
DELETE /api/v1/annotations/{id}
```

成功返回 `204`。这是软删除，数据仍保留在数据库中。

## 发表评论

```http
POST /api/v1/annotations/{id}/comments
Authorization: Bearer <token>
Content-Type: application/json
```

```json
{ "content": "同意，另外补充一点……" }
```

- 内容 1–4000 字。
- `pre` 模式下评论进入 `pending`。
- 不能评论不可见的批注（管理员除外）。

成功返回 `201` 与评论对象。

## 删除评论

```http
DELETE /api/v1/comments/{id}
Authorization: Bearer <token>
```

成功返回 `204`。作者本人、站点管理员或超管可删。

## 常见错误

| 场景 | 返回 |
| --- | --- |
| 未登录写入 | `401 unauthorized` |
| 来源不在白名单 | `403 origin_not_allowed` |
| 被站点封禁 | `403 site_banned` |
| 修改他人批注 | `403 forbidden` |
| 内容过长 / 锚点非法 | `400 validation_error` |
| 批注不存在或不可见 | `404 not_found` |

完整错误码见 [API 概览](/api/overview/)。
