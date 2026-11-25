---
title: CLI 参考
description: zhupi 命令行的全部子命令、参数、环境依赖与典型用法。
---

CLI 与服务使用同一个二进制：

```bash
zhupi [serve]
zhupi admin <subcommand> [...]
```

不带子命令时等价于 `serve`。管理子命令会加载与服务相同的配置（`SECRET_KEY`、`DB_PATH` 等），并直接打开 SQLite 数据库。

> Docker 部署时用 `docker compose exec zhupi /app/zhupi ...` 执行；容器内已注入 `.env` 中的环境变量。

## serve

```bash
zhupi serve
```

启动 HTTP 服务。需要：

- `SECRET_KEY`（必填）
- `DB_PATH`
- `PUBLIC_BASE_URL`

启动时自动执行数据库迁移，并初始化本地超管（仅当不存在凭据时）。

## admin migrate

```bash
zhupi admin migrate
```

显式执行数据库迁移。`serve` 启动时也会自动执行，通常不需要手动调用。

## admin site

### create

```bash
zhupi admin site create \
  --name blog \
  --display-name "我的博客" \
  --origin https://blog.example.com \
  --origin https://www.example.com \
  --moderation post \
  --theme "#087f5b" \
  --locale zh-CN
```

| 参数 | 默认 | 说明 |
| --- | --- | --- |
| `--name` | 必填 | 内部唯一名称 |
| `--display-name` | 等于 name | 展示名 |
| `--origin` | 必填，可重复 | 允许来源 |
| `--moderation` | `post` | `post` / `pre` |
| `--theme` | `#087f5b` | 主题色 |
| `--locale` | `zh-CN` | 默认语言 |

输出 JSON，包含 `key`（siteKey）。

### list

```bash
zhupi admin site list
```

### rotate-key

```bash
zhupi admin site rotate-key --site <siteKey 或 ID>
```

打印新的 siteKey。旧 key 立即失效。

### add-admin / remove-admin

```bash
zhupi admin site add-admin --site <siteKey 或 ID> --user <用户 ID 或 GitHub 登录名>
zhupi admin site remove-admin --site <siteKey 或 ID> --user <用户 ID 或 GitHub 登录名>
```

用户需要先登录过一次，才能按 GitHub 登录名找到。

## admin provider

管理站点级 OIDC 提供商。

### list

```bash
zhupi admin provider list
zhupi admin provider list --site <siteKey 或 ID>
```

输出 JSON，包含提供商的站点名。

### add

```bash
zhupi admin provider add \
  --site <siteKey 或 ID> \
  --name "公司 SSO" \
  --issuer https://sso.example.com \
  --client-id zhupi \
  --client-secret-file /run/secrets/oidc
```

| 参数 | 默认 | 说明 |
| --- | --- | --- |
| `--site` | 必填 | 站点 |
| `--name` | 必填 | 展示名 |
| `--issuer` | 必填 | OIDC issuer，保存前会做 Discovery 校验 |
| `--client-id` | 必填 | OIDC client id |
| `--client-secret` | — | client secret |
| `--client-secret-file` | — | 从文件读取，优先于 `--client-secret` |
| `--scopes` | `openid,profile,email` | 逗号分隔 |
| `--allowed-email-domains` | 空 | 逗号分隔 |
| `--enabled` | `true` | 是否启用 |

### update

```bash
zhupi admin provider update --id 3 --enabled false
zhupi admin provider update --id 3 --client-secret-file /run/secrets/oidc
```

留空的字段保持原值。仅当 issuer 变化时才做 Discovery 校验；停用、改名等操作可离线执行。

### delete

```bash
zhupi admin provider delete --id 3
```

## admin promote

```bash
zhupi admin promote --github-login <login>
zhupi admin promote --user-id <id>
```

把用户提升为超级管理员。

## admin users list

```bash
zhupi admin users list
zhupi admin users list --json
```

输出用户 ID、名称、状态与是否超管。

## admin password reset

```bash
# 生成随机密码并打印一次
zhupi admin password reset --username admin

# 指定新密码
zhupi admin password reset --username admin --password 'new-password'
```

重置会吊销该账号的全部会话。若该账号不是本地凭据账号，会返回错误。

## 环境依赖

| 变量 | 用途 |
| --- | --- |
| `SECRET_KEY` | 必填；即使是 `site list` 也会加载 |
| `DB_PATH` | 目标数据库 |
| `PUBLIC_BASE_URL` | 配置校验需要合法绝对地址 |

在 Docker 中执行时，最不容易出错的方式是：

```bash
docker compose exec zhupi /app/zhupi admin site list
```

## 退出码

| 码 | 含义 |
| --- | --- |
| `0` | 成功 |
| `1` | 配置错误、数据库错误或命令执行失败 |
| `2` | 用法错误（缺少子命令） |

## 注意事项

- CLI 直接写 SQLite；避免在服务高写入时执行批量变更。
- 创建/修改 OIDC 提供商需要服务器能访问 issuer 的 Discovery 文档。
- 轮换 siteKey、重置密码、删除提供商都是即时生效的操作，执行前确认影响范围。
