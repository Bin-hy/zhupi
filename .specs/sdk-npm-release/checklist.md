# zhupi-sdk npm 发布 Checklist

> 每一项通过运行命令或观察行为来验证，聚焦系统行为，不依赖代码结构。

## 实现完整性

- [ ] **AC1 许可证**：仓库根与包内各有一份 MIT 许可证
  验证：`head -1 LICENSE` 输出 `MIT License`；`diff LICENSE sdk/LICENSE` 无输出。

- [ ] **AC1 包元数据**：安装者能看到完整的公开信息
  验证：`npm pkg get name version description license repository homepage bugs keywords` 无 `null`，`license` 为 `MIT`。

- [ ] **AC2 入口与导出映射**：通过包名即可获得 ESM 入口与类型
  验证：`npm pkg get main module types exports` 均指向 `dist/` 下的现有文件；从 tarball 安装后 `import { init } from 'zhupi-sdk'` 可解析。

- [ ] **AC2 浏览器入口**：IIFE 产物可被 CDN 直接引用
  验证：`ls dist/zhupi.js` 存在；`npm pkg get unpkg jsdelivr` 均指向 `dist/zhupi.js`。

- [ ] **AC3 公开类型**：消费者能导入领域类型
  验证：最小消费者工程 `import type { Annotation, AnnotationComment, User, EventMap, ThemeMode, CreateAnnotationInput } from 'zhupi-sdk'` 后 `tsc --noEmit` 通过。

- [ ] **AC4 零运行时依赖**：安装后不多装包
  验证：`npm pkg get dependencies` 为 `{}`；临时消费者 `npm ls --omit=dev --depth=0` 只列出 `zhupi-sdk`。

- [ ] **AC5 内容裁剪**：测试声明与 sourcemap 不进入包
  验证：`npm pack --dry-run --json` 的文件清单不含 `.map` 与 `*.test.d.ts`。

- [ ] **AC5/N1 体积**：包体积在上限内
  验证：`npm pack --dry-run --json` 的 `size` ≤ 300KB，`unpackedSize` ≤ 1.5MB。

- [ ] **AC6 SDK 文档**：使用文档与服务端文档分离且完整
  验证：`sdk/README.md` 含安装、快速开始、公开 API、主题/国际化、浏览器支持、许可证六个小节；`npm pack` 清单包含 `README.md`。

- [ ] **AC9 服务端内嵌不变**：既有接入不受影响
  验证：`make build` 后启动服务，`GET /sdk/v1/zhupi.js` 与 `GET /sdk/v1/zhupi.esm.js` 均返回 200，且响应体与 `sdk/dist` 中同名文件一致。

## 集成

- [ ] **AC7 CI 门禁**：PR 流水线覆盖完整发布前检查
  验证：提交 PR 后 Actions 页面依次出现 `npm ci`、`typecheck`、`test`、`build`、`verify-package`，且全部通过。

- [ ] **AC8 发布认证**：0.1.0 完成一次性发布后，npm 侧可配置受信发布者
  验证：`npm view zhupi-sdk version` 输出 `0.1.0`；npm 包设置页显示 `Bin-hy/zhupi` + `publish.yml` + 环境 `npm`。

- [ ] **AC8/N3 tag 与版本一致**：不一致时拒绝发布
  验证：用错误 tag 触发校验步骤，进程退出码非 0；用与版本一致的 tag 触发，校验通过。

- [ ] **AC8 自动发布与 provenance**：新版本由 tag 自动发布
  验证：推送 `v0.2.0` 后 `npm view zhupi-sdk version` 输出 `0.2.0`；npm 页面出现 provenance 徽标。

- [ ] **AC8/N3 不可重复发布**：同一版本不能覆盖
  验证：对已发布版本再次执行发布，npm 返回版本已存在的错误。

- [ ] **N5 构建可复现**：干净环境产出稳定入口集合
  验证：删除 `node_modules` 与 `dist` 后 `npm ci && npm run build`，`dist` 顶部文件集合与 CI 构建一致（哈希 chunk 名可不同，入口 `zhupi.esm.js` / `zhupi.js` / `index.d.ts` 必须一致）。

- [ ] **文档站接入兼容**：go-campus 无需改动即可使用
  验证：本地启动文档站与服务端，浏览器网络面板请求 `/sdk/v1/zhupi.js` 返回 200，批注入口正常渲染。

## 编译与测试

- [ ] `cd sdk && npm run typecheck` 通过，无类型错误。
- [ ] `cd sdk && npm test` 全部用例通过。
- [ ] `cd sdk && npm run build` 成功，`dist` 含 `zhupi.esm.js`、`zhupi.js`、`index.d.ts`。
- [ ] `cd sdk && node scripts/verify-package.mjs` 退出码 0。
- [ ] `go build ./... && go vet ./... && go test ./...` 全部通过，确认内嵌链路未被破坏。

## 端到端场景

- [ ] **场景 1：本地使用者接入**
  打 tarball → 在全新目录 `npm i <tarball>` → 写一个 `init({ siteKey })` 的 ESM 示例 → TypeScript 编译通过 → 浏览器（或 jsdom E2E）加载后批注入口出现。
  预期：无额外依赖安装、无类型报错、入口可交互。

- [ ] **场景 2：从 npm 安装并接入**
  `npm i zhupi-sdk@0.2.0` → 用 `import { init } from 'zhupi-sdk'` 与类型构建示例 → 指向本地自托管服务 → 能创建一条划线批注。
  预期：安装、导入、类型、运行时四个环节全部打通。

- [ ] **场景 3：服务端内嵌回归**
  `make build` → `./bin/zhupi serve` → 请求 `/sdk/v1/zhupi.js` 与 `/sdk/v1/zhupi.esm.js` → 与 `sdk/dist` 对应文件逐字节比对。
  预期：两个路径均 200 且内容一致，文档站既有接入无需改动。
