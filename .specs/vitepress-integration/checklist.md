# VitePress 站内接入打磨 · Checklist

> 状态：草案 v0.1（等待评审）
> 输入：`.specs/vitepress-integration/spec.md`、`plan.md`、`task.md`
> 原则：每一项通过运行命令或观察行为验证，不逐行读代码下结论。

## A. 构建与测试

- [ ] Go 格式干净（验证：`gofmt -l .` 无输出）
- [ ] Go 静态检查通过（验证：`go vet ./...`）
- [ ] Go 单测通过（验证：`go test ./...`）
- [ ] SDK 类型检查通过（验证：`cd sdk && npm run typecheck`）
- [ ] SDK 单测通过（验证：`npm test`）
- [ ] SDK 可构建（验证：`npm run build`）
- [ ] `make build` 成功（SDK + 后台 + Go）
- [ ] 博客构建通过（验证：`go-campus && npm run docs:build`）
- [ ] SDK IIFE gzip 体积未明显回退（记录实际值）

## B. 后端 total

- [ ] 列表响应包含 `total` 字段
- [ ] 匿名用户 total = 公开可见批注数
- [ ] 作者 total 包含自己的待审内容
- [ ] 站点管理员/超管 total 包含该页全部非删除内容
- [ ] `filter=mine` 时 total 与 all 相同
- [ ] `filter=resolved` 时 total 不变
- [ ] `filter=pending` 时 total 不变
- [ ] 翻页（cursor）时 total 不变
- [ ] 空页面 total=0
- [ ] 删除批注后 total 减少
- [ ] 跨站点计数互不影响

## C. SDK 增补

- [ ] `getAnnotationCount()` 返回页面总数
- [ ] `statechange` 在登录/退出后触发，载荷 user/loggedIn 正确
- [ ] `statechange` 在列表加载/创建/删除后触发，annotationCount 正确
- [ ] `setTheme('dark'|'light'|'auto')` 立即改变 SDK UI 主题
- [ ] `openDrawer()` 可打开抽屉
- [ ] `openAccount()` 可打开账号/登录面板
- [ ] `selectRoot` 为选择器时，`setPage` 后能在新的 `.vp-doc` 节点高亮
- [ ] `selectRoot` 元素被移除时回退到 body，不抛异常
- [ ] `cssVariables` 可覆盖抽屉 top/height
- [ ] `cssVariables` 可覆盖 FAB bottom/right
- [ ] 现有 `init({ siteKey })` 与事件语义不变
- [ ] SDK 单测覆盖 total / statechange / selectRoot / cssVariables

## D. 顶部导航入口

- [ ] 文档页顶部显示「批注 N」与登录/头像
- [ ] 首页不显示入口
- [ ] 404 不显示入口
- [ ] 计数 N 与后端 total 一致
- [ ] 切换「我的/已解决/待审」时 N 不变
- [ ] 创建批注后 N +1
- [ ] 删除批注后 N -1
- [ ] 未登录显示「登录」按钮
- [ ] 点击登录打开 SDK 登录面板
- [ ] 登录后显示头像或首字母 + 名称
- [ ] 点击头像打开账号面板
- [ ] 退出登录后恢复「登录」按钮
- [ ] 导航按钮支持 Tab/Enter，焦点可见
- [ ] 移动端隐藏用户名，仅保留头像与计数，不挤压原导航

## E. 暗色主题同步

- [ ] 首次加载按 VitePress 当前主题初始化 SDK
- [ ] 切换 VitePress 暗色开关后，SDK host `data-theme` 变化
- [ ] 暗色下高亮、抽屉、按钮、徽章可读
- [ ] 亮色下同样可读
- [ ] SDK 账号内主题切换仍可用
- [ ] VitePress 下一次暗色切换会覆盖 SDK 内部选择

## F. 生命周期与 SPA

- [ ] 首次进入文档页只创建一个 SDK 实例
- [ ] 文档页 A → B 调用 `setPage`，不重新创建实例
- [ ] A → B 后旧页高亮被清除，新页批注加载
- [ ] 快速连续切换路由不出现串页/重复 UI
- [ ] 文档页 → 首页：SDK UI 移除、入口隐藏
- [ ] 首页 → 文档页：重新初始化并恢复入口
- [ ] 同一时刻最多一个 `.annotation-sdk-host`
- [ ] SDK 脚本只加载一次
- [ ] `destroy` 后事件监听与样式不残留

## G. 布局与移动端

- [ ] 桌面端抽屉从 VitePress 顶栏下方开始
- [ ] 抽屉打开时顶栏仍可见/可点击（不重叠）
- [ ] FAB 不遮挡右下角回到顶部等控件
- [ ] FAB 位置符合 CSS 变量配置
- [ ] 移动端抽屉为底部面板/全宽
- [ ] 移动端导航入口可用且不溢出
- [ ] 选中文字时工具条不被顶栏/导航遮挡

## H. 错误与降级

- [ ] SDK 脚本加载失败时，顶部入口显示不可用
- [ ] 不可用状态有可读提示（title/文本）
- [ ] 站点配置/接口失败时 SDK Toast 提示，博客不重复弹窗
- [ ] 失败后进入下一篇文档可重试初始化
- [ ] 失败不破坏正文阅读与 VitePress 导航
- [ ] 控制台只记录必要错误，不产生未捕获异常

## I. 深链接与登录

- [ ] `#annot-<id>` 打开页面后自动定位到批注
- [ ] 登录回跳后保留当前路由
- [ ] 登录回跳后保留深链接
- [ ] token 被消费后 URL 中不残留 `annot_token`
- [ ] 登录回调后导航头像/名称更新

## J. 回归

- [ ] 博客 `npm run docs:build` 通过
- [ ] Artalk 评论区正常加载
- [ ] Viewer.js 图片放大正常
- [ ] mermaid「放大查看」正常
- [ ] 本地搜索可用
- [ ] 侧边栏/目录/导航行为不变
- [ ] 未配置 `VITE_ANNOTATION_*` 时不加载 SDK、不显示入口

## K. 端到端场景

- [ ] 场景 1：文档页入口与计数
  - 进入文档页 → 顶部出现「批注 N」→ N 与 API total 一致。
- [ ] 场景 2：文档页切换
  - 文档 A → 文档 B → 只有一个 SDK host → 旧高亮清除 → B 的批注与计数正确。
- [ ] 场景 3：首页与返回
  - 文档页 → 首页 → 入口隐藏、SDK 移除 → 返回文档页 → 入口恢复。
- [ ] 场景 4：暗色同步
  - 切换 VitePress 暗色 → SDK host `data-theme` 变化 → 抽屉与高亮可读。
- [ ] 场景 5：登录态
  - 注入 token → 导航显示头像/名称 → 点击打开账号 → 退出后恢复登录按钮。
- [ ] 场景 6：移动端
  - 390×844 → 导航不溢出 → 抽屉全宽 → 工具条可用。
- [ ] 场景 7：错误降级
  - 阻断 SDK 脚本请求 → 入口禁用 → 恢复网络并切换路由 → 初始化成功。
- [ ] 场景 8：深链接
  - 打开 `#annot-<id>` → 自动定位并高亮。

## L. 验收证据要求

- 每条通过项记录：命令 / 操作步骤 + 实际输出或观察结果。
- 端到端场景保留 Playwright 报告或截图。
- 未通过项记录预期、实际、修复方案，修复后重新执行。
- 记录 SDK gzip 体积与 VitePress 首屏可交互时间。

## M. 验收记录（VitePress 接入打磨，2026-11-24）

### 已通过

- 后端：
  - `gofmt -l .` 无输出；`go vet ./...`、`go test ./...` 全部通过。
  - 新增 `TestPageTotal`：匿名/作者/管理员 total、筛选与分页不变、待审站点口径、删除后减少。
- SDK：
  - `npm run typecheck` 通过；Vitest 10 文件 / 23 用例全部通过。
  - IIFE gzip **71,449 字节（约 69.8KB）**。
  - 新增 `total`、`statechange`、`setTheme`、`openDrawer`、`openAccount`、
    `getAnnotationCount`、`selectRoot` 重新解析、`cssVariables`。
- 博客接入 E2E（`sdk/e2e/vitepress.mjs`，10 步全部通过）：
  1. 顶部导航入口渲染；
  2. 导航计数 = API total；
  3. 账号面板与登录态生效，抽屉 top=64（从顶栏下方开始）；
  4. 创建批注后计数 +1；
  5. 暗色同步 light → dark；
  6. 文档页切换使用 setPage，SDK 实例数=1；
  7. 首页隐藏入口并销毁实例；
  8. 返回文档页恢复实例；
  9. 390×844 移动端导航与抽屉正常；
  10. 阻断 SDK 脚本时入口显示「批注不可用」。
- 回归：
  - `make build` 通过；博客 `npm run docs:build` 通过。
  - `e2e:plain` 与 `admin/e2e` 均通过。

### 响应式取舍

- 批注入口在 768–1279px 隐藏，使用右下角 FAB，避免挤压已很满的导航。
- 1280–1439px 将 VitePress 的外观开关/社交链接收进其自带「…」菜单，
  压紧导航项内边距；1440px 以上恢复顶层显示。
- 移动端导航隐藏用户名，仅保留「批注 N」与头像。

### 未验证 / 待补

- 真实 GitHub / OIDC 登录回跳（E2E 使用 dev token 注入）。
- 1280–1439px 下「…」菜单内暗色开关的浏览器级断言（通过 1440px 验证同步，响应式布局人工检查）。
- FAB 与 VitePress 回到顶部按钮的逐像素检查（按 80px 上移，未做元素重叠断言）。
- 暗色对比度逐项审计。
