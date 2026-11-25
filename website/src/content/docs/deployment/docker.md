---
title: Docker 部署
description: 使用 GHCR 镜像或本地构建运行朱批，配置持久化、健康检查与升级回滚。
---

推荐用 Docker Compose 部署。仓库提供了 [docker-compose.yml](https://github.com/Bin-hy/zhupi/blob/main/docker-compose.yml)，默认使用 GitHub Container Registry 上的已发布镜像。

## 准备环境文件

```bash
cp .env.example .env
```

至少修改：

```dotenv
PUBLIC_BASE_URL=https://annot.example.com
SECRET_KEY=<openssl rand -hex 32 的输出>
```

需要 GitHub 登录时再填 `GITHUB_CLIENT_ID` / `GITHUB_CLIENT_SECRET`。

> 服务本身**不会自动读取 `.env`**。Compose 通过 `env_file: .env` 注入；裸机部署请用 systemd `EnvironmentFile` 或 shell `export`。

## 启动

```bash
docker compose pull
docker compose up -d
docker compose logs -f zhupi
```

Compose 做了这些事：

| 配置 | 值 |
| --- | --- |
| 镜像 | `ghcr.io/bin-hy/zhupi:${ZHUPI_TAG:-latest}` |
| 端口 | 仅发布到宿主机 `127.0.0.1:8080` |
| 数据卷 | 命名卷 `zhupi-data` 挂载到 `/data` |
| 数据库 | `/data/zhupi.db` |
| 健康检查 | `GET /healthz` |
| 重启策略 | `unless-stopped` |
| 代理信任 | `TRUST_PROXY=1` |

## 镜像 tag

| tag | 说明 | 架构 |
| --- | --- | --- |
| `latest` | 最新 release | `linux/amd64` + `linux/arm64` |
| `0.2` / `0.2.0` | 具体版本 | `amd64` + `arm64` |
| `edge` | `main` 最新构建 | 仅 `amd64` |

GHCR 包已设为公开，可匿名拉取。切换版本：

```bash
ZHUPI_TAG=0.2.0 docker compose pull
ZHUPI_TAG=0.2.0 docker compose up -d
```

## 创建站点

```bash
docker compose exec zhupi /app/zhupi admin site create \
  --name blog \
  --origin https://blog.example.com
```

输出中的 `key` 就是 siteKey。站点创建命令会直接读写容器内的 SQLite 数据库。

## 健康检查

```bash
curl http://127.0.0.1:8080/healthz   # 进程存活
curl http://127.0.0.1:8080/readyz    # 数据库可用
```

- `/healthz` 恒返回 `{"status":"ok"}`。
- `/readyz` 在数据库不可用时返回 `503` 与 `{"status":"db_unavailable"}`。

## 升级

```bash
docker compose pull
docker compose up -d
```

- 启动时自动执行数据库迁移。
- 建议升级前备份 `/data/zhupi.db` 及 WAL 文件，见 [数据与备份](/deployment/backup/)。
- 修改 `.env` 后同样需要 `up -d` 重建容器才会生效。

## 回滚

把 `ZHUPI_TAG` 改成上一个版本后重新 `pull` + `up -d`。注意：新版本可能已经执行了数据库迁移，旧版本未必能兼容新 schema；回滚前先备份，并在测试环境验证。

## 常用维护命令

```bash
# 查看日志
docker compose logs -f --tail=200 zhupi

# 进入容器执行 CLI
docker compose exec zhupi /app/zhupi admin site list

# 查看数据卷
docker volume inspect zhupi-data

# 停止但保留数据
docker compose down

# 停止并删除数据（破坏性）
docker compose down -v
```

## 本地构建镜像

```bash
docker compose up -d --build
```

Dockerfile 是多阶段构建：Node 22 构建 SDK 与管理后台 → Go 编译 → Alpine 运行时。运行时以非 root 用户（UID 10001）启动，数据目录 `/data` 需要可写。

## 对外暴露

Compose 只把服务发布在 `127.0.0.1:8080`。请用 Caddy/Nginx 提供 HTTPS，不要直接把 8080 暴露到公网。配置见 [反向代理与 HTTPS](/deployment/reverse-proxy/)。
