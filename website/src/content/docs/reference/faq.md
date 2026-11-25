---
title: 常见问题
description: 接入、登录、审核、部署、备份与开发中的高频问题。
---

## SDK 不显示 / 无法批注

### 页面右下角没有悬浮按钮

按顺序检查：

1. `<script>` 是否加载成功（Network 面板）；
2. `ZhupiSDK.init` 是否执行、`siteKey` 是否传入；
3. `selectRoot` 是否存在于页面；
4. 控制台是否有 `error` 事件或 CORS 错误。

### 选中文字没有「批注」按钮

- 选区必须在 `selectRoot` 内；
- 页面可能有全局 `user-select: none` 或事件拦截；
- `container` 指定的挂载节点不存在时，SDK 会回退到 `document.body`。

### 403 `origin_not_allowed`

页面实际来源没有加入站点白名单。Origin 是 `scheme://host[:port]`：

```text
https://blog.example.com
http://127.0.0.1:5173
```

协议、www、端口任一不同都要分别加入。

### 403 `origin_required`

请求没有 `Origin`。浏览器会正常发送；用 curl / 服务端脚本调用时需要手动带：

```bash
curl -H 'Origin: https://blog.example.com' -H 'X-Site-Key: …' …
```

本地测试可以临时 `ALLOW_MISSING_ORIGIN=1`，生产不要开启。

### 404 `site_not_found`

siteKey 错误或已轮换。用 `admin site list` 确认当前 key。

### 刷新后批注消失 / SPA 页面串数据

SPA 路由变化后没有调用 `setPage`：

```js
await instance.setPage({ pageKey: location.pathname, pageTitle: document.title })
```

### `apiBase` 推导错误

从 CDN 加载 SDK 时，默认 apiBase 是 CDN origin。显式指定：

```js
init({ siteKey: '…', apiBase: 'https://annot.example.com' })
```

## 登录

### 登录后跳回 `127.0.0.1:8080`

`PUBLIC_BASE_URL` 还是本地地址。生产改成外部 HTTPS 地址并重启。

### 看不到 GitHub 登录

没有配置 `GITHUB_CLIENT_ID` / `GITHUB_CLIENT_SECRET`，或回调地址不对：

```text
${PUBLIC_BASE_URL}/api/v1/auth/github/callback
```

### OIDC Discovery 校验失败

- issuer 必须是可访问的 HTTPS 地址；
- 服务器需要能访问 `${issuer}/.well-known/openid-configuration`；
- 修改 issuer 后需要重新保存并校验。

### OIDC 回调 404

IdP 中注册的回调地址应为：

```text
${PUBLIC_BASE_URL}/api/v1/auth/oidc/{providerId}/callback
```

`providerId` 是朱批内的数字 ID，不是 client ID。

### 登录后仍然 401

- token 已被“退出所有设备”吊销；
- 会话超过 `SESSION_TTL_HOURS`；
- 账号被全局封禁；
- 换站点后使用了另一个 API/siteKey 下的 token（token 按 API + siteKey 存储）。

### 本地超管密码忘了

```bash
./bin/zhupi admin password reset --username admin
# 或
docker compose exec zhupi /app/zhupi admin password reset --username admin
```

会吊销该账号全部会话。

## 审核

### 新批注立刻可见，没有待审

站点是 `post`（先发后审）模式。需要先审后发时：

```bash
./bin/zhupi admin site create --name blog --moderation pre --origin https://blog.example.com
```

已有站点由超级管理员在后台修改审核模式。

### 站点管理员看不到设置页

站点管理员只能处理审核、封禁与 OIDC 提供商；Origin、审核模式、站点状态只能由超级管理员修改。

### 封禁后还能读

站点封禁只拦截写操作；全局封禁才会拒绝登录与会话认证。

## 部署

### `/sdk/v1/zhupi.js` 或 `/admin/` 返回 404

二进制里没有嵌入前端产物。执行完整构建：

```bash
make build
```

只跑 `make server` 不会自动构建 SDK 与管理后台。

### 容器健康但无法登录后台

- 检查 `PUBLIC_BASE_URL` 与访问地址是否一致；
- 检查反向代理是否传递了正确的 Host/协议；
- 本地超管默认 `admin/admin`，首次登录必须改密。

### 修改 `.env` 不生效

环境变量只在启动时读取：

```bash
docker compose up -d
# 或
sudo systemctl restart zhupi
```

### 端口冲突

服务默认 `8080`；示例页面常用 `5173`。确认本机没有其他进程占用，或修改 `ADDR` / Compose 端口映射。

### SQLite `database is locked`

- 确认只运行一个实例；
- 避免多个进程同时写同一个 DB 文件；
- 备份/维护时尽量避开写入高峰。

## 数据与备份

### 能只复制 `zhupi.db` 吗

运行中复制主文件可能丢失 WAL 中未合并的事务。推荐：

```bash
sqlite3 zhupi.db ".backup 'backup.db'"
```

或停止服务后复制 `zhupi.db*` 三个文件。

### 恢复后 OIDC 登录失效

`SECRET_KEY` 与备份时不一致，已保存的 client secret 无法解密。使用原密钥，或重新录入提供商 secret。

### 如何清空全部数据

没有内置重置命令。停止服务后删除数据库文件即可：

```bash
docker compose down
docker volume rm zhupi-data   # 破坏性
```

重新启动会自动建表并创建本地超管。

## 开发

### CI 没跑 Go 测试

当前 `ci.yml` 只跑 SDK；Go 与管理后台测试需要本地执行，见 [本地开发与测试](/development/local/)。

### `make build` 很慢

它包含 SDK 与管理后台的 `npm ci` 与两次前端构建。只改后端时用 `make server`（前提是前端产物已存在）。

### 管理后台刷新 404

二进制中缺少 `internal/adminassets/dist` 产物，重新 `make admin` 并编译。

## 其他

### 支持多实例 / PostgreSQL 吗

V1 不支持，当前面向单实例 SQLite。见 [V1 边界与路线](/reference/boundaries/)。

### 可以把 SDK 放到 CDN 吗

可以。服务端也托管同一份产物（`/sdk/v1/zhupi.js`）。放 CDN 时记得显式传 `apiBase`。

### 有审计日志 / 2FA / 邮件通知吗

V1 没有。需要时在反向代理或周边系统补足。
