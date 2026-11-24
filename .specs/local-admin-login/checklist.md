# 本地管理员账号密码登录 · Checklist

> 状态：已确认（v1.0）
> 输入：`.specs/local-admin-login/spec.md`、`plan.md`、`task.md`
> 原则：每项通过命令或实际行为验证。

## A. 构建与静态检查

- [ ] `gofmt -l .` 无输出
- [ ] `go vet ./...` 通过
- [ ] `go test ./...` 全部通过
- [ ] `cd admin && npm run typecheck` 通过
- [ ] `cd admin && npm test` 通过
- [ ] `cd admin && npm run build` 成功
- [ ] `make server` 成功产出二进制

## B. 初始化与配置

- [ ] 全新数据库启动后自动创建本地超管
- [ ] `local_credentials.password_hash` 为 bcrypt 格式（`$2a$` / `$2b$`）
- [ ] 默认 `admin` / `admin` 登录成功且 `mustChangePassword = true`
- [ ] 自定义 `LOCAL_ADMIN_USERNAME` / `LOCAL_ADMIN_PASSWORD` 首次启动生效
- [ ] 重启且未设置重置标志时，后台修改的密码不被覆盖
- [ ] `LOCAL_ADMIN_RESET_PASSWORD=1` 重启后密码恢复为环境变量值
- [ ] 重置后该用户全部会话被吊销

## C. 登录与安全

- [ ] 错误密码返回 401，消息不区分账号是否存在
- [ ] 不存在账号与错误密码响应一致
- [ ] 登录接口超过限流返回 429
- [ ] 被封禁用户无法本地登录
- [ ] 登录响应不包含密码哈希
- [ ] `GET /api/v1/admin/auth/providers` 返回 `local: true`

## D. 强制改密

- [ ] `GET /api/v1/me` 返回 `username` 与 `mustChangePassword`
- [ ] 强制改密时 `GET /api/v1/admin/sites` 返回 403 `password_change_required`
- [ ] 强制改密时 `GET /api/v1/me` 正常
- [ ] 强制改密时 `POST /api/v1/me/password` 正常
- [ ] 前端登录后被强制跳转 `/account`
- [ ] 强制改密时手动访问其他后台路由被重定向 `/account`

## E. 修改密码

- [ ] 当前密码错误返回 400 `invalid_current_password`
- [ ] 新密码少于 8 字节返回 400
- [ ] 新密码超过 72 字节返回 400
- [ ] 新旧密码相同返回 400
- [ ] 修改成功后旧密码登录失败、新密码登录成功
- [ ] 修改成功后 `mustChangePassword = false`
- [ ] 修改成功后其他会话失效、当前会话仍可用

## F. CLI

- [ ] `zhupi admin password reset --username admin` 生成随机密码并打印
- [ ] `--password` 指定密码可用
- [ ] 不存在的用户名返回明确错误
- [ ] 重置后旧密码失效

## G. 端到端

- [ ] 浏览器打开 `/admin/` 可见本地账号表单
- [ ] 默认账号登录 → 强制改密页 → 修改成功 → 概览可访问
- [ ] 退出后新密码可再次登录
- [ ] GitHub / OIDC 登录入口仍正常展示
- [ ] 站点管理员登录后不出现本地账号相关入口

## H. 部署

- [ ] 服务器 `.env` 增加 `LOCAL_ADMIN_*` 配置
- [ ] 容器 healthy
- [ ] `https://zhupi.gataccay.com/admin/` 可完成首次登录与改密
- [ ] 服务器日志无密码明文
