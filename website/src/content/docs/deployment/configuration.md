---
title: 环境变量
description: 朱批全部运行配置项、默认值、安全含义与修改后的生效方式。
---

服务通过环境变量配置。默认值以 `.env.example` 与 `internal/config` 为准；**服务不会自动读取 `.env` 文件**，需要 Compose 的 `env_file`、systemd 的 `EnvironmentFile` 或 shell `export` 注入。

## 基础

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `ADDR` | `:8080` | 监听地址；生产建议 `127.0.0.1:8080` 配合反向代理 |
| `DB_PATH` | `zhupi.db` | SQLite 文件路径；Docker 中为 `/data/zhupi.db` |
| `PUBLIC_BASE_URL` | `http://127.0.0.1:8080` | 对外绝对地址，用于拼接 OAuth/OIDC 回调；必须是 `http(s)://host` |
| `SECRET_KEY` | 无（必填） | 32 字节密钥，hex 或 base64；用于加密 OIDC client secret |
| `SESSION_TTL_HOURS` | `720` | 会话有效期（小时），默认 30 天 |

生成密钥：

```bash
openssl rand -hex 32
```

### SECRET_KEY 的注意事项

- 长度必须是 32 字节（64 个 hex 字符，或 32 字节 base64）。
- 它用于 AES-256-GCM 加密站点 OIDC 的 `clientSecret`。
- **更换 SECRET_KEY 后，已保存的 OIDC client secret 将无法解密**，需要重新录入提供商密钥。
- 不要把 `SECRET_KEY` 提交到仓库或发给前端；轮换时先备份数据库并安排维护窗口。

## GitHub OAuth（全局）

| 变量 | 说明 |
| --- | --- |
| `GITHUB_CLIENT_ID` | GitHub OAuth App 的 Client ID |
| `GITHUB_CLIENT_SECRET` | GitHub OAuth App 的 Client Secret |

GitHub 仅用于站点访客登录，不会自动提升超管，也不能登录管理后台。

未配置 `GITHUB_CLIENT_ID` / `GITHUB_CLIENT_SECRET` 时，GitHub 登录入口不会出现在站点配置中。回调地址：

```text
${PUBLIC_BASE_URL}/api/v1/auth/github/callback
```

## 管理后台本地账号

管理后台只接受本地账号密码；下面的变量只负责首次启动的引导账号。

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `LOCAL_ADMIN_USERNAME` | `admin` | 引导超管用户名 |
| `LOCAL_ADMIN_PASSWORD` | `admin` | 初始密码；等于 `admin` 时首次登录强制改密 |
| `LOCAL_ADMIN_RESET_PASSWORD` | `0` | 设为 `1` 时，启动后用上面的密码重置并吊销会话 |
| `LOCAL_LOGIN_RATE_PER_MIN` | `10` | 本地登录每 IP 每分钟限流 |

- 初始化只在数据库中不存在本地凭据时执行；重启不会覆盖后台改过的密码。
- 更多管理员请由超管在后台「管理员账号」创建，或为已有用户开通本地密码。
- 忘记密码时在服务器上重置：

```bash
./bin/zhupi admin password reset --username admin
./bin/zhupi admin password reset --username admin --password 'new-password'
```

## 代理与来源

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `TRUST_PROXY` | `0` | 为 `1` 时读取 `X-Forwarded-For` 的第一个地址 |
| `ADMIN_ORIGINS` | 空 | 允许跨域访问管理 API 的来源，逗号分隔；留空即关闭 |
| `ALLOW_MISSING_ORIGIN` | `0` | **仅本地测试**：允许缺少 `Origin` 的站点级请求 |

生产环境：

- 前面有可信反向代理时设 `TRUST_PROXY=1`。
- 不要开启 `ALLOW_MISSING_ORIGIN`；它会让非浏览器客户端绕过 Origin 白名单。
- 管理后台与 API 同源时不需要 `ADMIN_ORIGINS`。

## 限流

| 变量 | 默认值 | 作用 |
| --- | --- | --- |
| `READ_RATE_PER_MIN` | `300` | 每 IP 每分钟读请求 |
| `WRITE_RATE_PER_MIN` | `60` | 每 IP 每分钟写请求 |
| `WRITE_USER_RATE_PER_MIN` | `30` | 每登录用户每分钟写请求 |
| `LOCAL_LOGIN_RATE_PER_MIN` | `10` | 本地管理员登录每 IP 每分钟 |

限流是进程内计数，重启后清零；多实例部署时各自独立计数（当前版本面向单实例）。

## 开发专用

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `DEV_LOGIN` | `0` | 开启 `/api/v1/auth/dev/login`，仅用于本地联调 |

`DEV_LOGIN=1` 时任何人都能通过开发登录接口创建会话，**绝不能在生产环境开启**。

## 生效方式

- 修改环境变量后需要**重启服务**（Docker：`docker compose up -d` 重建容器；systemd：`systemctl restart zhupi`）。
- 配置在启动时一次性读取，没有热更新接口。
- 配置错误（例如 `SECRET_KEY` 长度不对）会阻止服务启动，并在日志中给出原因。

## 完整模板

以仓库根目录的 [`.env.example`](https://github.com/Bin-hy/zhupi/blob/main/.env.example) 为准，复制后逐项修改：

```bash
cp .env.example .env
```
