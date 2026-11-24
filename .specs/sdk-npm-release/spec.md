# zhupi-sdk npm 发布 Spec

## 背景

zhupi 的 SDK 与自托管服务同仓。SDK 以 Vite library 模式构建，产出 ESM、IIFE 与
TypeScript 声明，并由 Go 服务内嵌托管在 `/sdk/v1/zhupi.js`，供任意 Web 前端接入。

SDK 的运行时能力已经完整，但发布工程缺失：

- 包名 `zhupi-sdk` 在 npm 上空闲，尚未发布；
- `npm pack` 当前包含 40 个文件、解包 2.85MB，混入测试声明文件与 sourcemap；
- `package.json` 缺发布元数据（描述、许可证、仓库、关键词等）；
- 仓库无 LICENSE、无 CI、无 GitHub 远端；
- 运行时依赖已被构建打进 bundle，但仍声明为 dependencies，消费者会被多装包。

仓库将整体以 MIT 开源，SDK 作为独立 npm 包对外发布。

## 目标

- `zhupi-sdk@0.1.0` 可被公开安装；`npm i zhupi-sdk` 后，ESM 与 TypeScript 开箱可用。
- 后续版本由 git tag 触发自动发布，并带 npm provenance。
- 包内容、元数据、许可证与文档达到可对外发布的基线。
- 不改变 SDK 运行时行为，也不影响服务端内嵌托管。

## 功能需求

- **F1 发布元数据**：包具备完整的公开信息——名称、版本、描述、关键词、许可证（MIT）、
  源码仓库、主页、问题反馈地址、作者、运行环境声明。
- **F2 入口与导出映射**：ESM 作为主入口；浏览器 IIFE 入口可被 CDN（unpkg / jsdelivr）
  直接引用；类型入口指向正确；使用方通过包名即可导入，不依赖内部文件路径。
- **F3 类型契约**：公开所有消费者会接触到的领域类型，包括批注、评论、用户、作者、
  事件映射、锚点、站点配置、身份、会话与创建输入；使用方无需深入内部模块即可获得完整类型。
- **F4 依赖归位**：已被打进产物的运行时依赖不再作为安装依赖；消费者安装后运行时依赖为空。
- **F5 发布内容裁剪**：发布内容只包含运行所需产物；测试声明文件与 sourcemap 不进入包
  （或作为显式的可选策略）；包体积达到非功能需求中的上限。
- **F6 许可证与使用文档**：仓库根提供 MIT `LICENSE`；SDK 有面向使用者的 README，
  覆盖安装、快速开始、公开 API、主题与国际化、浏览器支持、与自托管服务的关系。
- **F7 CI 校验**：PR 流水线执行类型检查、单元测试、构建与打包预演，任一失败即阻断。
- **F8 发布流水线**：发布由版本 tag 驱动，并校验 tag 与包版本一致；因 npm 要求包已存在
  才能配置受信发布者，0.1.0 允许一次性手动/令牌 bootstrap，随后立即切换为
  Trusted Publishing，后续版本不再使用长期令牌。
- **F9 服务端内嵌兼容**：服务端构建仍能取得 SDK 产物，`/sdk/v1/zhupi.js` 与
  `/sdk/v1/zhupi.esm.js` 的路径与行为保持不变。

## 非功能需求

- **N1 包体积**：tarball ≤ 300KB，解包体积 ≤ 1.5MB。
- **N2 类型可用性**：在一个独立的最小消费者工程中，ESM 导入与公开类型解析均通过类型检查。
- **N3 发布可审计**：自动发布带 provenance；同一版本不可重复发布；tag 与版本不一致时拒绝发布。
- **N4 零新增运行时依赖**：实现本 spec 不引入新的运行时依赖。
- **N5 构建可复现**：在干净环境中执行依赖安装与构建可得到相同入口文件集合。

## 不做的事

- 不发布 `zhupi-admin`（private，随服务端内嵌交付）。
- 不拆分仓库、不引入 monorepo / workspaces。
- 不做 React / Vue 等框架封装。
- 不发布 Go module、Docker 镜像或 GitHub Release 资产（另立 spec）。
- 不提供 npm CDN 托管或版本别名策略。
- 不改变 SDK 运行时行为、事件语义或登录流程。
- 不补邮件通知、反垃圾、审计日志等功能能力。

## 验收标准

- **AC1**（F1/F6）：`package.json` 元数据齐全且 `license` 为 MIT；仓库根存在 MIT `LICENSE`；
  `npm pack --dry-run` 输出中可见描述、版本与许可证信息。
- **AC2**（F2）：从打包产物安装后，ESM 导入 `init` 成功；IIFE 产物可通过包内路径或 CDN
  字段直接引用。
- **AC3**（F3）：最小消费者工程能 `import type` 批注、评论、用户、事件等公开类型，
  且 `tsc --noEmit` 通过。
- **AC4**（F4）：安装后运行时依赖列表为空，无多余传递依赖。
- **AC5**（F5/N1）：`npm pack --dry-run` 不含 `*.test.d.ts` 与 sourcemap；
  tarball ≤ 300KB，解包 ≤ 1.5MB。
- **AC6**（F6）：`sdk/README.md` 覆盖安装、快速开始、公开 API、主题/国际化、浏览器支持，
  并与服务端 README 区分。
- **AC7**（F7）：PR 流水线依次通过 typecheck、test、build、pack 校验。
- **AC8**（F8/N3）：0.1.0 完成一次性发布后，npm 上可查询到 `zhupi-sdk@0.1.0`；
  配置受信发布者后，推送新版本 tag 能自动发布；provenance 可查；
  tag 与版本不一致时发布被拒绝。
- **AC9**（F9）：服务端 `make build` 后启动，`/sdk/v1/zhupi.js` 与
  `/sdk/v1/zhupi.esm.js` 返回 200，内容与新包一致；既有接入方式无需修改。
