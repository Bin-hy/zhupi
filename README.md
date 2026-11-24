# 朱批 · zhupi

**多站点、可嵌入的划线批注服务。** 一个部署实例可以托管多个站点（类似 Artalk 的集中管理），
任意 Web 前端通过一段脚本接入，即可获得「选中文字 → 划线 → 批注与回复」的能力。

> **名字取自「朱批」**：古人用朱笔在书页、奏折上圈点批注——红笔划过重点，批语留在页边，
> 与「划线 + 批注」是同一件事。和同门的 **yufu（鱼符）** 一样，用一个有身份的古物指代产品的本质：
> 鱼符管身份，朱批管批注。

- **多站点隔离**：公开 siteKey + 来源白名单，批注/评论按站点隔离
- **可嵌入 SDK**：Shadow DOM UI，不污染宿主样式，支持 ESM / `<script>` 两种引入
- **登录**：全局 GitHub OAuth + 每站点独立 OIDC；首次登录自动建号，支持手动绑定多身份
- **治理**：超管 / 站点管理员两级权限，站点级「先发后审 / 先审后发」
- **自托管友好**：单二进制 + SQLite（WAL）+ 内嵌 SDK，Docker 一键部署

## 架构

```
浏览器 ── SDK（Shadow DOM）──► HTTP API ──► service ──► store ──► SQLite
                                   │
                                   └── GitHub OAuth / 站点 OIDC
```

- 站点级请求必须携带 `X-Site-Key`，服务端校验 `Origin` 是否在该站点的白名单内。
- 会话使用 Bearer token；数据库只存 token 的 SHA-256。
- OIDC clientSecret 使用 AES-256-GCM 加密存储。
- 批注锚点采用 W3C `TextQuoteSelector`（exact/prefix/suffix），前端用
  `dom-anchor-text-quote` 做模糊重定位。

## 本地启动

```bash
# 1) 构建 SDK + 服务
make build            # 产出 bin/zhupi（内嵌 SDK）

# 2) 准备环境
export SECRET_KEY=$(openssl rand -hex 32)
export DB_PATH=./zhupi.db
export PUBLIC_BASE_URL=http://127.0.0.1:8080
export DEV_LOGIN=1

# 3) 创建站点
./bin/zhupi admin site create --name blog --origin http://127.0.0.1:5173

# 4) 启动服务
./bin/zhupi serve
```

打开 `examples/plain-html/index.html`（用任意静态服务器托管），通过
`?siteKey=<上一步的 key>&api=http://127.0.0.1:8080` 访问即可体验。

## 接入 SDK

```html
<script src="https://zhupi.dev/sdk/v1/zhupi.js"></script>
<script>
  ZhupiSDK.init({
    siteKey: 'YOUR_SITE_KEY',
    selectRoot: '.vp-doc',      // 允许划词的范围，默认 document.body
    // apiBase: 'https://zhupi.dev', // 默认从脚本地址推导
    on: {
      ready: () => console.log('annotation ready'),
      created: (annotation) => console.log('created', annotation.id),
    },
  })
</script>
```

SPA 路由切换：

```js
await instance.setPage({ pageKey: location.pathname, pageTitle: document.title })
```

其他 API：`refresh()`、`login(providerId?)`、`logout()`、`getCurrentUser()`、`getAnnotationCount()`、
`setTheme()`、`openDrawer()`、`openAccount()`、`on()`、`destroy()`。

SDK 内置能力：筛选（全部/我的/已解决/待审）、游标分页、账号中心（身份绑定、登录设备、
退出所有设备）、亮/暗/自动主题、中英文、`#annot-<id>` 深链接、orphan 手动重定位。
`statechange` 事件可把登录态与页面批注总数同步给宿主导航；`setPage` 支持 SPA 路由切换。

ESM：

```js
import { init } from 'zhupi-sdk'
const instance = init({ siteKey: 'YOUR_SITE_KEY' })
```

## 站点与管理员

```bash
# 站点列表
./bin/zhupi admin site list

# 轮换 siteKey（旧 key 立即失效）
./bin/zhupi admin site rotate-key --site <siteKey>

# 指定站点管理员（用户需先登录过一次）
./bin/zhupi admin site add-admin --site <siteKey> --user <githubLogin|用户ID>
./bin/zhupi admin site remove-admin --site <siteKey> --user <githubLogin|用户ID>

# 提升超级管理员
./bin/zhupi admin users list
./bin/zhupi admin promote --user-id 1
# 或在环境变量 SUPERADMIN_GITHUB_LOGINS 中配置，登录时自动提升
```

站点创建时默认 `--moderation post`（先发后审）；需要先审后发时加 `--moderation pre`。

## 登录配置

### GitHub（全局）

配置 `GITHUB_CLIENT_ID` / `GITHUB_CLIENT_SECRET`，OAuth App 回调地址：

```
${PUBLIC_BASE_URL}/api/v1/auth/github/callback
```

### OIDC（每站点）

站点管理员或超管通过管理 API 配置，服务端会在保存时做一次 Discovery 校验：

```bash
curl -X POST https://zhupi.dev/api/v1/admin/sites/1/providers \
  -H "Authorization: Bearer <site-admin-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "公司 SSO",
    "issuer": "https://sso.example.com",
    "clientId": "zhupi",
    "clientSecret": "***",
    "scopes": ["openid", "email", "profile"],
    "allowedEmailDomains": ["example.com"],
    "enabled": true
  }'
```

在 IdP 中注册的回调地址：

```
${PUBLIC_BASE_URL}/api/v1/auth/oidc/{providerId}/callback
```

## 管理后台

后台内嵌在 Go 服务的 `/admin/`，与 API 同源：

- 登录：全局 GitHub 或任意启用站点的 OIDC 提供商；
- 超管：概览、站点管理、站点管理员、全局用户封禁、审核、OIDC 提供商；
- 站点管理员：仅被分配站点的审核、封禁与 OIDC 配置；
- 构建：`make admin`（仅后台）或 `make build`（SDK + 后台 + Go）；
- 本地开发：`cd admin && npm run dev`，Vite 已代理 `/api` 到 `127.0.0.1:8080`。

首次使用需要把某个 GitHub 用户提升为超管：

```bash
./bin/zhupi admin promote --github-login <你的 GitHub 登录名>
```

## API 概览

公共 / SDK（需要 `X-Site-Key`）：

| 方法 | 路径 |
|---|---|
| GET | `/api/v1/site-config` |
| GET | `/api/v1/annotations?page=&cursor=&limit=` |
| POST | `/api/v1/annotations` |
| PATCH / DELETE | `/api/v1/annotations/{id}` |
| POST | `/api/v1/annotations/{id}/comments` |
| DELETE | `/api/v1/comments/{id}` |

账号：`/api/v1/me`、`/api/v1/me/identities`、`/api/v1/me/sessions`。

管理：`/api/v1/admin/sites`、`/admin/sites/{id}/queue`、
`/admin/sites/{id}/annotations/{id}/moderate`、`/admin/sites/{id}/providers` 等。

错误统一为：

```json
{ "error": { "code": "site_not_found", "message": "站点不存在", "requestId": "..." } }
```

## 部署

Docker（多阶段：Node 构建 SDK → Go embed → Alpine）：

```bash
cp .env.example .env
# 修改 PUBLIC_BASE_URL / SECRET_KEY / GITHUB_* 等
docker compose up -d --build
docker compose logs -f zhupi
```

服务只监听宿主机 `127.0.0.1:8080`，由 Caddy/Nginx 提供 HTTPS：

- [deploy/Caddyfile.example](./deploy/Caddyfile.example)
- [deploy/nginx.conf.example](./deploy/nginx.conf.example)
- [deploy/zhupi.service](./deploy/zhupi.service)（裸机 systemd）

数据存放在命名卷 `zhupi-data`。

## 验证

```bash
go vet ./... && go test ./...
cd sdk && npm run typecheck && npm test && npm run build
make build
```

## V1 边界

- 管理后台仅内嵌 `/admin`，不做独立子域名部署、审计日志与图表分析。
- 无邮箱密码 / 找回密码 / 邮件通知 / Webhook。
- 无敏感词、举报、自动反垃圾。
- 无 PostgreSQL / 多实例（仓储层已抽象，后续可替换）。
- 不兼容旧单站点 API，不迁移旧数据。
- SDK 登录使用整页跳转；弹窗 + postMessage 留到后续版本。
