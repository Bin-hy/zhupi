---
title: 快速开始
description: 用 Docker 或源码启动朱批，创建站点，并在示例页面上完成第一条批注。
---

## 前置要求

| 方式 | 需要 |
| --- | --- |
| Docker（推荐） | Docker Engine 24+、Compose v2 |
| 源码 | Go 1.26+、Node.js 22+、`make` |
| 本地调试 | `openssl`（生成 SECRET_KEY） |

服务默认监听 `:8080`；Docker Compose 只把它发布到宿主机 `127.0.0.1:8080`，由 Caddy/Nginx 对外提供 HTTPS。

## 方式一：Docker Compose

```bash
git clone https://github.com/Bin-hy/zhupi.git
cd zhupi

cp .env.example .env
# 至少修改：
#   PUBLIC_BASE_URL=https://annot.example.com
#   SECRET_KEY=$(openssl rand -hex 32)
# 需要 GitHub 登录时再填 GITHUB_CLIENT_ID / GITHUB_CLIENT_SECRET

docker compose pull
docker compose up -d
docker compose logs -f zhupi
```

健康检查：

```bash
curl http://127.0.0.1:8080/healthz   # {"status":"ok"}
curl http://127.0.0.1:8080/readyz    # {"status":"ready"}
```

> 只在本地体验、又需要 GitHub / OIDC 登录时，把 `PUBLIC_BASE_URL` 设为 `http://127.0.0.1:8080`；否则 OAuth 会回调到公网域名。生产环境再改回外部 HTTPS 地址。

首次启动会自动创建本地超级管理员（默认 `admin` / `admin`），打开 <http://127.0.0.1:8080/admin/> 登录后会强制修改密码。

## 创建第一个站点

站点是批注的隔离单元。创建时会签发公开的 `siteKey`，并配置允许访问的 Origin 白名单。

```bash
docker compose exec zhupi /app/zhupi admin site create \
  --name blog \
  --display-name "我的博客" \
  --origin http://127.0.0.1:5173
```

输出中会包含 `key`（即 siteKey）。后续 SDK 初始化需要它。

> `--origin` 可重复传入多个来源。Origin 的格式是 `scheme://host[:port]`，例如 `https://blog.example.com`，不带路径。

## 打开示例页面

仓库自带一个纯 HTML 示例（`examples/plain-html/`）。用任意静态服务器托管即可：

```bash
# 在仓库根目录
python3 -m http.server 5173

# 浏览器打开（把 siteKey 换成上一步的值）
open 'http://127.0.0.1:5173/examples/plain-html/?siteKey=<siteKey>&api=http://127.0.0.1:8080'
```

选中一段文字 → 点击浮出的「批注」→ 发布。刷新页面后划线仍在；修改正文一两个字后也能重新定位。

如果只用本地开发登录，启动服务时设置 `DEV_LOGIN=1`，示例会使用 SDK 的 `devLogin` 模式。

## 方式二：从源码启动

```bash
git clone https://github.com/Bin-hy/zhupi.git
cd zhupi

make build          # 构建 SDK + 管理后台 + Go 二进制，产物 bin/zhupi

export SECRET_KEY=$(openssl rand -hex 32)
export DB_PATH=./zhupi.db
export PUBLIC_BASE_URL=http://127.0.0.1:8080
export DEV_LOGIN=1   # 仅本地开发

./bin/zhupi admin site create --name blog --origin http://127.0.0.1:5173
./bin/zhupi serve
```

`make build` 会依次执行：

1. `sdk`：`npm ci && npm run build`，把 SDK 产物复制到 `internal/sdkassets/dist/`；
2. `admin`：`npm ci && npm run build`，把管理后台产物复制到 `internal/adminassets/dist/`；
3. `server`：`CGO_ENABLED=0 go build`，把上述资源嵌入二进制。

只想构建其中一部分时，可以执行 `make sdk`、`make admin` 或 `make server`。

## 登录管理后台

- 地址：`http://127.0.0.1:8080/admin/`
- 默认本地超管：`admin` / `admin`（首次登录强制改密）
- 进入「管理员账号」可新建管理员，或为已有用户开通本地密码；GitHub / 站点 OIDC
  只用于站点访客登录与管理员身份绑定，不能直接登录后台。

管理后台包含：概览、站点管理、站点管理员、全局用户封禁、审核工作台、OIDC 提供商。

## 下一步

- 把 SDK 接到自己的网站：[SDK 接入](/sdk/integration/)
- 配置 GitHub / OIDC 登录：[登录与身份](/sites/authentication/)
- 部署到公网：[Docker 部署](/deployment/docker/) · [反向代理与 HTTPS](/deployment/reverse-proxy/)
- 排查接入问题：[常见问题](/reference/faq/)
