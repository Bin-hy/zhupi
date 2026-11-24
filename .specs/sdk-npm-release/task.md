# zhupi-sdk npm 发布 Tasks

## 文件清单

| 操作 | 文件 | 职责 |
|------|------|------|
| 新建 | `LICENSE` | 仓库级 MIT 许可证 |
| 新建 | `sdk/LICENSE` | 包内 MIT 许可证 |
| 新建 | `sdk/README.md` | 面向使用者的 SDK 文档 |
| 修改 | `sdk/package.json` | 元数据、exports、依赖归位 |
| 修改 | `sdk/package-lock.json` | 随依赖调整同步 |
| 修改 | `sdk/vite.config.ts` | 关闭 sourcemap、排除测试声明 |
| 修改 | `sdk/src/index.ts` | 公开类型 re-export |
| 新建 | `sdk/scripts/verify-package.mjs` | 打包内容、体积、安装与类型校验 |
| 新建 | `.github/workflows/ci.yml` | PR 发布前门禁 |
| 新建 | `.github/workflows/publish.yml` | tag 触发 OIDC 发布 |

## T1: 仓库级 MIT 许可证

**文件：** `LICENSE`
**依赖：** 无
**步骤：**
1. 写入标准 MIT 文本，版权行使用 `Copyright (c) 2026 Bin-hy`。
2. 文件不含其他许可证片段。

**验证：** `head -1 LICENSE` 输出 `MIT License`，且文件包含 `Permission is hereby granted`。

## T2: 包内 MIT 许可证

**文件：** `sdk/LICENSE`
**依赖：** T1
**步骤：**
1. 复制根 `LICENSE` 到 `sdk/LICENSE`，内容完全一致。

**验证：** `diff LICENSE sdk/LICENSE` 无输出；`npm pack --dry-run`（在 `sdk/` 下）文件清单包含 `LICENSE`。

## T3: SDK 使用者文档

**文件：** `sdk/README.md`
**依赖：** 无
**步骤：**
1. 写入标题、一句话定位与「与 zhupi 服务端的关系」。
2. 写入安装：`npm i zhupi-sdk`。
3. 写入快速开始：`init({ siteKey })`，含 `apiBase`、`selectRoot`、`pageKey` 示例。
4. 写入公开 API 表：`init`、实例方法（`setPage`、`refresh`、`login`、`logout`、`getCurrentUser`、`getAnnotationCount`、`setTheme`、`openDrawer`、`openAccount`、`on`、`destroy`）。
5. 写入事件列表、主题与国际化、浏览器支持（ES2020+ / Shadow DOM）。
6. 结尾写明 MIT 许可证。

**验证：** 文件包含上述六个部分；`npm pack --dry-run` 清单包含 `README.md`。

## T4: 包元数据

**文件：** `sdk/package.json`
**依赖：** T1、T2
**步骤：**
1. 增加 `description`、`keywords`、`license: "MIT"`、`author: "Bin-hy"`。
2. 增加 `repository`（`git+https://github.com/Bin-hy/zhupi.git`，`directory: "sdk"`）、`homepage`、`bugs`。
3. 保持 `name` 为 `zhupi-sdk`、`version` 为 `0.1.0`、`type` 为 `module`。

**验证：** `node -p "const p=require('./sdk/package.json'); [p.description,p.license,p.repository.url,p.homepage,p.bugs.url].join('\\n')"` 输出完整，无 `undefined`。

## T5: 入口与导出映射

**文件：** `sdk/package.json`
**依赖：** T4
**步骤：**
1. `main` 与 `module` 指向 `./dist/zhupi.esm.js`。
2. `types` 指向 `./dist/index.d.ts`。
3. 增加 `exports`：`"."` 含 `types`、`import`、`default` 三个条件；另暴露 `"./package.json"`。
4. 增加 `unpkg` 与 `jsdelivr`，均指向 `./dist/zhupi.js`。
5. 保持 `files: ["dist"]`。

**验证：** `npm pack --dry-run --json` 成功；`node -p "JSON.stringify(require('./sdk/package.json').exports)"` 含 `types`/`import`/`default`。

## T6: 依赖归位

**文件：** `sdk/package.json`、`sdk/package-lock.json`
**依赖：** T4
**步骤：**
1. 把 `@apache-annotator/dom`、`@apache-annotator/selector`、`dom-anchor-text-quote` 从 `dependencies` 移到 `devDependencies`。
2. 在 `sdk/` 下执行 `npm install`，同步 lock 与本地依赖。
3. 确认构建产物仍自包含（三个包已打进 `dist/lib-*.js`）。

**验证：** `npm ls --omit=dev --depth=0` 只列出 `zhupi-sdk`；`npm run build` 后 `grep -l "dom-anchor-text-quote" dist/*.js` 有命中或产物功能测试通过。

## T7: 公开类型契约

**文件：** `sdk/src/index.ts`
**依赖：** 无
**步骤：**
1. 增加 `export type { Annotation, AnnotationComment, User, Author, QuoteSelector, Anchor, AuthProvider, SiteConfig, Identity, SessionView, EventMap } from './events'`。
2. 增加 `export type { CreateAnnotationInput } from './client'`。
3. 增加 `export type { ThemeMode } from './state'`。
4. 不新增运行时代码。

**验证：** `npm run typecheck` 通过；`npm run build` 后 `grep -E "Annotation|CreateAnnotationInput|ThemeMode" dist/index.d.ts` 有输出。

## T8: 构建关闭 sourcemap

**文件：** `sdk/vite.config.ts`
**依赖：** 无
**步骤：**
1. 将 `build.sourcemap` 设为 `false`。

**验证：** `npm run build` 后 `find dist -name '*.map'` 无输出。

## T9: 类型声明排除测试

**文件：** `sdk/vite.config.ts`
**依赖：** 无
**步骤：**
1. 在 `dts` 插件配置中增加 `exclude: ['**/*.test.ts']`。

**验证：** `npm run build` 后 `find dist -name '*.test.d.ts'` 无输出，且 `dist/index.d.ts` 仍存在。

## T10: 打包校验脚本

**文件：** `sdk/scripts/verify-package.mjs`
**依赖：** T5、T6、T7、T8、T9
**步骤：**
1. 执行 `npm pack --json`，解析 tarball 路径、文件列表、size 与 unpackedSize。
2. 断言清单不含 `.map` 与 `.test.d.ts`；tarball ≤ 300KB，解包 ≤ 1.5MB。
3. 在临时目录创建消费者工程，`npm init -y` 后安装 tarball 与 `typescript`。
4. 写入 `index.ts`，同时导入 `init` 和公开类型，执行 `npx tsc --noEmit --strict`。
5. 断言消费者的生产依赖只有 `zhupi-sdk`。
6. 结束前清理临时目录；任一断言失败以非零码退出。

**验证：** 在 `sdk/` 下执行 `node scripts/verify-package.mjs`，退出码 0，输出包含体积与依赖检查结果。

## T11: CI 校验工作流

**文件：** `.github/workflows/ci.yml`
**依赖：** T10
**步骤：**
1. 触发条件：`pull_request` 与推送到 `main`。
2. 权限：`contents: read`。
3. 使用 `actions/checkout@v4`、`actions/setup-node@v4`（Node 22，npm 缓存指向 `sdk/package-lock.json`）。
4. 默认工作目录 `sdk`，依次执行 `npm ci`、`npm run typecheck`、`npm test`、`npm run build`、`node scripts/verify-package.mjs`。

**验证：** 推到 GitHub 后 PR 检查变绿；本地按相同顺序执行全部命令均通过。

## T12: tag 发布工作流

**文件：** `.github/workflows/publish.yml`
**依赖：** T11
**步骤：**
1. 触发条件：推送 `v*` tag。
2. 权限：`contents: read`、`id-token: write`。
3. 设置 `environment: npm` 与 `concurrency`（同组不并发）。
4. `actions/setup-node@v4`（Node 22，`registry-url: https://registry.npmjs.org`）。
5. 执行 `npm install -g npm@^11.5.1`。
6. `npm ci`、`npm run build`。
7. 校验 `${GITHUB_REF_NAME}` 等于 `v` + `sdk/package.json` 的 `version`，不等则退出 1。
8. 在 `sdk/` 下执行 `npm publish --provenance --access public`。

**验证：** YAML 可被 GitHub 解析（推送后 Actions 页面无语法错误）；版本校验逻辑用 `GITHUB_REF_NAME=v0.1.1` 与错误 tag 各跑一次，前者通过、后者失败。

## T13: 创建远端并推送

**文件：** 无（git 操作）
**依赖：** T1–T12 全部提交
**步骤：**
1. 将当前 feature 分支合并为 `main`。
2. 执行 `gh repo create Bin-hy/zhupi --public --source=. --remote=origin --push`。
3. 确认默认分支为 `main`，工作区干净。

**验证：** `git remote -v` 指向 `Bin-hy/zhupi`；`git ls-remote --heads origin` 含 `main`；GitHub 页面可见仓库与 CI。

## T14: 0.1.0 手动 bootstrap 发布

**文件：** 无
**依赖：** T13
**步骤：**
1. 在 `sdk/` 下 `npm run build`。
2. 使用 npm 账号登录（或一次性 granular token）执行 `npm publish --access public`。
3. 发布后立即撤销本次使用的 token（若使用 token）。

**验证：** `npm view zhupi-sdk version` 输出 `0.1.0`；`npm view zhupi-sdk dist.tarball` 可访问。

## T15: 配置受信发布者

**文件：** 无（npm 后台操作）
**依赖：** T14
**步骤：**
1. 打开 npm 上 `zhupi-sdk` 的 Settings → Trusted Publisher。
2. 填写 GitHub 仓库 `Bin-hy/zhupi`、工作流文件名 `publish.yml`、环境 `npm`。
3. 保存并确认列表中出现该配置。

**验证：** npm 包设置页显示受信发布者；不再保留长期发布 token。

## T16: 用 0.1.1 验证 OIDC 发布链

**文件：** `sdk/package.json`
**依赖：** T15
**步骤：**
1. 将版本改为 `0.2.0`（体现发布链验证，也避免与 bootstrap 版本混淆）。
2. 提交并推送 `main`。
3. 创建并推送 tag `v0.2.0`。
4. 观察 `publish.yml` 运行结果。
5. 在临时目录执行 `npm i zhupi-sdk@0.2.0`，运行 `tsc --noEmit` 的消费者示例。

**验证：** Actions 成功；`npm view zhupi-sdk version` 输出 `0.2.0`；npm 页面出现 provenance 徽标；临时消费者安装与类型检查通过。

## 执行顺序

```
T1 ──┐
T2 ──┼─→ T4 ─→ T5 ─┐
T3 ──┘         T6 ─┼─→ T10 ─→ T11 ─→ T12 ─→ T13 ─→ T14 ─→ T15 ─→ T16
T7 ────────────────┤
T8 ────────────────┤
T9 ────────────────┘
```

T1–T9 可并行推进，但建议每完成一组（许可证、元数据、构建、类型）提交一次；T10 起进入发布链，必须串行。
