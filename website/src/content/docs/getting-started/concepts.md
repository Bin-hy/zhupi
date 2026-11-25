---
title: 核心概念
description: 站点、siteKey、页面、批注锚点、评论、审核状态、用户身份与会话之间的关系。
---

理解下面几个对象，后面的部署、接入与审核都会顺很多。

## 对象关系

```text
Site（站点）
 ├── Origins（来源白名单）
 ├── Admins（站点管理员）
 ├── Bans（站点封禁）
 ├── Providers（OIDC 登录提供商）
 └── Annotations（批注）
       ├── Anchor（TextQuoteSelector 锚点）
       └── Comments（评论）

User（用户）
 ├── Identities（GitHub / OIDC / 本地身份）
 └── Sessions（Bearer 会话）
```

## 站点（Site）

一个站点代表一个被批注的网站，是数据隔离的边界。

| 字段 | 说明 |
| --- | --- |
| `key` / siteKey | 公开标识，SDK 初始化时使用；可轮换 |
| `name` | 内部唯一名称，CLI 创建时必填 |
| `displayName` | 展示名，默认等于 `name` |
| `status` | `active` / `disabled`；停用后站点级请求被拒绝 |
| `moderationMode` | `post`（先发后审）/ `pre`（先审后发） |
| `themeColor` | 默认主题色，SDK 读取 |
| `defaultLocale` | 默认语言，`zh-CN` / `en` |
| `origins` | 允许访问该站点的来源列表 |

### siteKey 与 Origin

站点级请求必须同时满足：

1. 携带 `X-Site-Key`（SDK 自动加）；
2. 请求的 `Origin` 在该站点的白名单内。

Origin 会规范化为小写的 `scheme://host[:port]`，例如：

```text
http://127.0.0.1:5173
https://blog.example.com
```

协议、域名或端口任一不同都视为不同来源。`https://example.com` 与 `https://www.example.com` 需要分别加入。

`siteKey` 是公开标识，不是密钥；真正的访问控制靠 Origin 白名单。轮换 `siteKey` 后旧 key 立即失效，所有接入页面都需要更新。

### 审核模式

| 模式 | 含义 | 用户体验 |
| --- | --- | --- |
| `post` | 先发后审 | 发布后立即可见，管理员可事后隐藏/删除 |
| `pre` | 先审后发 | 新批注/评论进入待审队列，通过后才对普通读者可见 |

## 页面（Page）

批注按 `pageKey` 归属。默认取 `location.pathname`，也可以由 `setPage` 显式指定。

- 同一路径不同查询参数默认视为同一页面，除非你在 `pageKey` 中显式包含 `location.search`。
- `pageTitle` 只用于展示和审核上下文，不参与归属。

## 批注（Annotation）

一条批注由「锚点 + 正文 + 状态 + 审核状态」组成。

### 锚点（Anchor）

锚点采用 W3C Web Annotation 的 **TextQuoteSelector** 思路：

```json
{
  "quote": {
    "type": "TextQuoteSelector",
    "exact": "被选中的原文",
    "prefix": "前文若干字符",
    "suffix": "后文若干字符"
  },
  "position": { "start": 120, "end": 146 },
  "section": "二级标题"
}
```

- `exact` 是被选中的文字，`prefix` / `suffix` 用于在正文变化后做模糊重定位。
- `position` 是辅助信息，正文改动后可能失效。
- 页面大改导致无法自动重定位时，SDK 会把批注标记为 orphan，允许手动重新指定位置。

### 状态与可见性

| 字段 | 取值 | 含义 |
| --- | --- | --- |
| `status` | `open` / `resolved` | 是否已解决；作者与管理员可切换 |
| `moderation` | `pending` / `approved` / `rejected` | 审核状态；`post` 模式默认 `approved` |
| `visibility` | `visible` / `hidden` / `deleted` | 对读者是否可见；删除为软删除 |

普通读者只能看到 `visible` 且 `approved` 的批注；站点管理员可以在审核上下文中看到待审与隐藏内容。

## 评论（Comment）

评论必须挂在某条批注下，不支持文章级独立评论。评论同样有 `moderation` 与 `visibility`，并支持轻量 Markdown（行内代码、粗体、斜体、删除线、链接、换行）。

## 用户、身份与会话

| 对象 | 说明 |
| --- | --- |
| User | 平台用户；首次通过任一提供商登录时自动创建 |
| Identity | 用户在某个提供商下的身份；GitHub 为全局身份，OIDC 按站点隔离 |
| Session | Bearer 会话；数据库只保存 token 的 SHA-256，不存明文 |
| LocalCredential | 本地超级管理员账号密码；密码只保存 bcrypt 哈希 |

一个用户可以有多个身份（例如 GitHub + 公司 SSO），在账号中心手动绑定。解绑最后一个身份前请确认还有可用的登录方式。

## 角色

| 角色 | 能力 |
| --- | --- |
| 超级管理员 | 所有站点、全局用户、站点管理员、OIDC 提供商、审核 |
| 站点管理员 | 被分配站点的审核、封禁、OIDC 配置与站点设置 |
| 普通用户 | 创建/编辑/删除自己的批注与评论，标记已解决 |

详见 [角色、审核与封禁](/sites/roles-moderation/)。
