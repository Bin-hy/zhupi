---
title: 本地开发与测试
description: 源码依赖、make 目标、三个开发服务、测试与当前 CI 覆盖范围。
---

## 依赖

| 工具 | 版本 |
| --- | --- |
| Go | 1.26+（以 `go.mod` 为准） |
| Node.js | 22+ |
| npm | 随 Node 提供 |
| make | 任意 |
| SQLite CLI（可选） | 备份/排查 |

## 仓库结构

```text
cmd/zhupi/        Go 入口与 CLI
internal/         服务端
sdk/              TypeScript SDK
admin/            Vue 3 管理后台
deploy/           反向代理与 systemd 示例
examples/         纯 HTML 接入示例
website/          Astro 官方文档
```

## 常用 make 目标

```bash
make sdk      # 构建 SDK 并复制到 internal/sdkassets/dist
make admin    # 构建管理后台并复制到 internal/adminassets/dist
make server   # 编译 bin/zhupi
make build    # 依次执行 sdk → admin → server
make test     # go vet + go test + SDK typecheck/test
make clean    # 清理 bin 与前端产物
```

首次构建需要安装前端依赖，`make sdk` / `make admin` 内部使用 `npm ci`。

## 启动服务

```bash
make server
export SECRET_KEY=$(openssl rand -hex 32)
export DB_PATH=./zhupi.db
export PUBLIC_BASE_URL=http://127.0.0.1:8080
export DEV_LOGIN=1
./bin/zhupi serve
```

没有构建前端产物时，`/sdk/v1/*` 与 `/admin/*` 会返回 404；先执行 `make sdk` / `make admin` 或完整 `make build`。

## 管理后台开发

```bash
cd admin
npm ci
npm run dev
```

Vite 已把 `/api` 代理到 `127.0.0.1:8080`。开发时同时运行 Go 服务即可。

常用脚本：

```bash
npm run typecheck
npm test
npm run e2e
```

## SDK 开发

```bash
cd sdk
npm ci
npm run build
npm run typecheck
npm test
npm run verify
```

构建产物在 `sdk/dist/`。要把新产物嵌入服务，重新执行 `make sdk && make server`。

E2E（需要 Playwright 环境）：

```bash
npm run e2e:plain
npm run e2e:vitepress
npm run e2e:account
npm run e2e:relocate
npm run e2e:comments
```

## 测试

```bash
# Go
go vet ./...
go test ./...

# SDK
cd sdk
npm run typecheck
npm test

# 管理后台
cd admin
npm run typecheck
npm test
```

### 当前 CI 覆盖范围

`.github/workflows/ci.yml` 目前只运行 SDK 的 `typecheck`、`test`、`build` 与 `verify`。**Go 与管理后台测试需要在本地执行**，或在 PR 前手动跑上面的命令。

## 文档站

```bash
cd website
npm install
npm run dev        # http://localhost:4321
npm run build      # 产物 dist/
```

Cloudflare Pages 配置：

| 配置 | 值 |
| --- | --- |
| Root directory | `website` |
| Build command | `npm run build` |
| Output directory | `dist` |
| Node.js | `22.23.4` |

## 新增功能的位置

| 改动 | 位置 |
| --- | --- |
| 环境变量 | `internal/config` + `.env.example` + 文档 |
| 数据表 | `internal/store/migrations/` + 仓储 |
| 业务规则 | `internal/service` |
| HTTP 路由/参数 | `internal/httpapi` |
| SDK 能力 | `sdk/src`（UI 在 `sdk/src/ui`） |
| 管理后台页面 | `admin/src/views` + `admin/src/router.ts` |
| 文档 | `website/src/content/docs` |

## 调试技巧

- `ADDR=127.0.0.1:18090 ./bin/zhupi serve` 换端口，避免与已有实例冲突。
- `DEV_LOGIN=1` + SDK `devLogin: true` 跳过 OAuth 调试登录。
- 请求失败时用响应里的 `requestId` 在服务日志中检索。
- SQLite 可用 `sqlite3 zhupi.db` 直接查看表；生产环境谨慎操作。
- 管理后台是 SPA，刷新 404 通常意味着二进制里没有嵌入管理后台产物。
