---
title: 站点管理
description: 创建站点、配置 Origin 白名单、轮换 siteKey、分配站点管理员与停用站点。
---

站点是朱批的数据隔离单元。一个实例可以创建任意多个站点，每个站点有独立的 siteKey、来源白名单、登录提供商、审核队列与管理员。

## 创建站点

```bash
./bin/zhupi admin site create \
  --name blog \
  --display-name "我的博客" \
  --origin https://blog.example.com \
  --origin https://www.example.com \
  --moderation post \
  --theme "#087f5b" \
  --locale zh-CN
```

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `--name` | 必填 | 内部唯一名称，1–64 字符 |
| `--display-name` | 等于 `name` | 展示名 |
| `--origin` | 必填 | 允许来源，可重复；至少一个 |
| `--moderation` | `post` | `post`（先发后审）/ `pre`（先审后发） |
| `--theme` | `#087f5b` | 默认主题色 |
| `--locale` | `zh-CN` | 默认语言 |

输出包含站点 `id`、`key`（siteKey）、`origins`、`moderationMode` 等字段。

## 查看站点

```bash
./bin/zhupi admin site list
```

输出列为 `ID / 名称 / siteKey / 状态 / 来源`。

## Origin 白名单

Origin 会规范化为小写的 `scheme://host[:port]`：

```text
https://blog.example.com
http://127.0.0.1:5173
```

- 协议、域名、端口任一不同就是不同来源。
- 不带路径、查询或末尾斜杠。
- 新增域名（包括预览域名）后必须同步加入，否则请求返回 `origin_not_allowed`。
- `ALLOW_MISSING_ORIGIN` 只用于本地测试，生产不要开启。

## 轮换 siteKey

```bash
./bin/zhupi admin site rotate-key --site <siteKey 或 ID>
```

旧 key **立即失效**，所有接入页面都需要更新为新 key。轮换前先确认你能同时更新所有前端。

## 站点管理员

站点管理员只能进入被分配站点的审核、封禁与 OIDC 配置。用户需要先登录过一次：

```bash
./bin/zhupi admin site add-admin --site <siteKey> --user <GitHub 登录名或用户 ID>
./bin/zhupi admin site remove-admin --site <siteKey> --user <GitHub 登录名或用户 ID>
```

也可以由超级管理员在管理后台的站点详情页操作。站点管理员还需要开通本地密码才能
登录后台，可在「管理员账号」页或执行：

```bash
./bin/zhupi admin users open --user-id <id> --username <name>
```

## 修改站点设置

由**超级管理员**在管理后台 `/admin/sites/{id}` 或管理 API 中可以修改：

- `displayName` 展示名
- `origins` 来源白名单
- `moderationMode` 审核模式
- `themeColor` 主题色
- `defaultLocale` 默认语言
- `status` 站点状态（`active` / `disabled`）

站点管理员只能处理被分配站点的审核、封禁与 OIDC 提供商，不能修改站点设置。

站点停用后，所有站点级请求返回 `site_disabled`，但数据不会删除。重新启用即可恢复。

## 多站点建议

| 场景 | 建议 |
| --- | --- |
| 同一产品的多个域名 | 放在同一站点，把多个 Origin 加入白名单 |
| 不同品牌 / 不同团队 | 拆成不同站点，独立管理员与审核队列 |
| 测试与生产 | 拆成不同站点，避免测试数据进入生产审核队列 |
| 预览域名 | 加入对应站点的 Origin，或单独建预览站点 |

## 相关

- [登录与身份](/sites/authentication/)
- [角色、审核与封禁](/sites/roles-moderation/)
- [CLI 参考](/cli/reference/)
