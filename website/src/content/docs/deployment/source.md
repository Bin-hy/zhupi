---
title: 源码构建与 systemd
description: 不用 Docker 时，如何构建单二进制、准备目录、配置环境变量并用 systemd 托管。
---

## 前置要求

| 工具 | 版本 | 用途 |
| --- | --- | --- |
| Go | 1.26+（以 `go.mod` 为准） | 编译服务端 |
| Node.js | 22+ | 构建 SDK 与管理后台 |
| make | 任意 | 串联构建步骤 |
| npm | 随 Node 提供 | 安装前端依赖 |

## 构建

```bash
git clone https://github.com/Bin-hy/zhupi.git
cd zhupi
make build
```

产物是 `bin/zhupi`。`make build` 依次执行：

| 目标 | 动作 |
| --- | --- |
| `make sdk` | `cd sdk && npm ci && npm run build`，复制产物到 `internal/sdkassets/dist/` |
| `make admin` | `cd admin && npm ci && npm run build`，复制产物到 `internal/adminassets/dist/` |
| `make server` | `CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/zhupi ./cmd/zhupi` |

只改后端时可以只跑 `make server`；但前提是 `internal/sdkassets/dist/` 与 `internal/adminassets/dist/` 里已有构建产物。

## 运行

```bash
export SECRET_KEY=$(openssl rand -hex 32)
export DB_PATH=/var/lib/zhupi/zhupi.db
export PUBLIC_BASE_URL=https://annot.example.com
export TRUST_PROXY=1

./bin/zhupi serve
```

服务默认监听 `:8080`。可以用 `ADDR=127.0.0.1:8080` 把监听限制在本机，再由反向代理对外。

## 目录与权限

推荐布局：

```text
/opt/zhupi/zhupi          # 二进制
/var/lib/zhupi/           # 数据目录（DB + WAL）
/etc/zhupi.env            # 环境变量
```

创建一个专用用户并授权：

```bash
sudo useradd --system --home /var/lib/zhupi --shell /usr/sbin/nologin zhupi
sudo mkdir -p /var/lib/zhupi
sudo chown -R zhupi:zhupi /var/lib/zhupi
```

## systemd

仓库提供了 `deploy/zhupi.service` 示例：

```ini
[Unit]
Description=Zhupi annotation service
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=zhupi
Group=zhupi
WorkingDirectory=/opt/zhupi
EnvironmentFile=/etc/zhupi.env
ExecStart=/opt/zhupi/zhupi
Restart=on-failure
RestartSec=3

NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/zhupi

[Install]
WantedBy=multi-user.target
```

安装：

```bash
sudo cp bin/zhupi /opt/zhupi/zhupi
sudo cp deploy/zhupi.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now zhupi
sudo systemctl status zhupi
```

环境变量写在 `/etc/zhupi.env`（参考 `.env.example`），`DB_PATH` 指向 `/var/lib/zhupi/zhupi.db`。

## 升级

```bash
make build
sudo systemctl stop zhupi
sudo cp bin/zhupi /opt/zhupi/zhupi
sudo systemctl start zhupi
```

启动时会自动执行数据库迁移。升级前先备份数据。

## 验证

```bash
go vet ./...
go test ./...
cd sdk && npm run typecheck && npm test && npm run build
cd ../admin && npm run typecheck && npm test && npm run build
make build
```
