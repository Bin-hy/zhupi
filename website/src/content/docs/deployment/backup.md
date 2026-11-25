---
title: 数据与备份
description: SQLite 文件、WAL 模式、在线备份与恢复、Docker 卷备份，以及需要一并保存的密钥。
---

朱批的全部业务数据都在一个 SQLite 文件里，默认路径 `/data/zhupi.db`（Docker）或 `./zhupi.db`（源码运行）。

## 数据文件

SQLite 开启 WAL 模式，运行时目录下会出现：

```text
zhupi.db          # 主数据库
zhupi.db-wal      # 预写日志
zhupi.db-shm      # 共享内存索引
```

备份时必须把这三个文件视为一个整体。只复制 `zhupi.db` 而不带 WAL，可能丢失最近提交的事务。

## 备份前先保存密钥

`SECRET_KEY` 用于解密数据库中保存的站点 OIDC `clientSecret`。恢复数据库但丢失原 `SECRET_KEY` 时，OIDC 提供商需要重新录入密钥。

建议把以下内容一起备份到安全位置：

- `zhupi.db`（及 WAL/SHM，或在停止服务后只备份主文件）
- `SECRET_KEY`
- GitHub OAuth App 的 Client ID / Secret
- 反向代理与域名配置

不要把密钥提交到仓库。

## 方式一：在线备份（推荐）

如果宿主机装了 `sqlite3`，可以在服务运行时做一致性备份：

```bash
mkdir -p backup
sqlite3 /path/to/zhupi.db ".backup 'backup/zhupi-$(date +%F).db'"
```

`.backup` 使用 SQLite 的在线备份 API，会生成一个完整、可独立打开的文件，不依赖 WAL。

Docker 运行时可从宿主机对卷内文件执行：

```bash
docker run --rm \
  -v zhupi-data:/data \
  -v "$PWD/backup":/backup \
  alpine:3.22 \
  sh -c 'apk add --no-cache sqlite >/dev/null && sqlite3 /data/zhupi.db ".backup /backup/zhupi-$(date +%F).db"'
```

## 方式二：停止服务后复制

最稳妥、最通用：

```bash
# Docker
docker compose stop zhupi
docker run --rm -v zhupi-data:/data -v "$PWD/backup":/backup alpine:3.22 \
  sh -c 'cp /data/zhupi.db* /backup/'
docker compose start zhupi

# 源码 / systemd
sudo systemctl stop zhupi
sudo cp /var/lib/zhupi/zhupi.db* /backup/
sudo systemctl start zhupi
```

## Docker 卷整体备份

```bash
docker run --rm \
  -v zhupi-data:/data \
  -v "$PWD/backup":/backup \
  alpine:3.22 \
  tar czf /backup/zhupi-data-$(date +%F).tgz -C /data .
```

恢复：

```bash
docker compose down

docker run --rm \
  -v zhupi-data:/data \
  -v "$PWD/backup":/backup \
  alpine:3.22 \
  sh -c 'rm -rf /data/* && tar xzf /backup/zhupi-data-2026-11-25.tgz -C /data'

docker compose up -d
```

> 恢复会覆盖现有数据。执行前确认卷名与备份文件正确；`rm -rf /data/*` 属于破坏性操作。

## 恢复步骤

1. 停止服务（`docker compose stop zhupi` 或 `systemctl stop zhupi`）。
2. 备份当前数据库（即使准备覆盖）。
3. 用备份替换 `/data/zhupi.db`；如果使用在线备份文件，删除旧的 `-wal` / `-shm`。
4. 确认 `SECRET_KEY` 与备份时一致。
5. 启动服务；启动时自动执行数据库迁移。
6. 打开 `/admin/` 与一个接入页面，确认站点、批注与 OIDC 提供商正常。

## 迁移与版本

- 服务启动时自动执行 `internal/store/migrations/` 下的迁移。
- 升级前先备份；降级到旧版本时，新 schema 可能不被旧代码兼容。
- 当前只有 `0001_init` 与 `0002_local_credentials` 两个迁移，后续会继续追加。

## 单实例约束

当前版本面向**单实例 + SQLite**：

- 不要让多个进程同时写同一个数据库文件（SQLite 会锁，但性能与行为不适合多实例）。
- 需要水平扩展时，应先替换存储层（代码中仓储层已抽象，但 V1 未提供 PostgreSQL 实现）。

## 建议

- 每天至少一次自动备份，保留 7 ~ 30 天。
- 备份文件与 `SECRET_KEY` 分开存放，但都要可恢复。
- 定期做一次恢复演练，而不是只验证备份文件存在。
