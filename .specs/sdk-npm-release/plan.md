# zhupi-sdk npm 发布 Plan

## 架构概览

本次不改变 SDK 运行时结构，只在「构建 → 打包 → 校验 → 发布」链路上补齐发布工程。整体分为四块：

1. **包描述层**：`sdk/package.json` 成为唯一的发布契约，声明入口、导出映射、类型入口、元数据与依赖。
2. **类型契约层**：`sdk/src/index.ts` 作为唯一公开入口，显式 re-export 领域类型；类型声明由 `vite-plugin-dts` 产出。
3. **产物裁剪层**：`sdk/vite.config.ts` 关闭 sourcemap、排除测试声明，确保 `dist` 只含运行必需文件。
4. **发布流水线层**：GitHub Actions 承担 PR 校验与 tag 发布；首次 0.1.0 手动 bootstrap 后切换到 OIDC 受信发布。

服务端内嵌链路不变：`make sdk` 仍把 `sdk/dist` 复制到 `internal/sdkassets/dist`，Go 服务继续以 `/sdk/v1/zhupi.js` 与 `/sdk/v1/zhupi.esm.js` 提供。

## 核心数据结构

### 包描述（`sdk/package.json`）

关键字段及取值：

| 字段 | 取值 |
|------|------|
| `name` | `zhupi-sdk`（不变） |
| `version` | `0.1.0`（首发） |
| `description` | 可嵌入的划线批注 SDK（朱批 zhupi） |
| `keywords` | annotation / highlight / comments / web-annotation / sdk / zhupi |
| `license` | `MIT` |
| `repository` | `git+https://github.com/Bin-hy/zhupi.git`，`directory: "sdk"` |
| `homepage` | `https://github.com/Bin-hy/zhupi/tree/main/sdk#readme` |
| `bugs` | `https://github.com/Bin-hy/zhupi/issues` |
| `author` | `Bin-hy` |
| `type` | `module`（不变） |
| `main` / `module` | `./dist/zhupi.esm.js` |
| `types` | `./dist/index.d.ts` |
| `exports` | `"."` → `{ types, import, default }`；`"./package.json"` |
| `unpkg` / `jsdelivr` | `./dist/zhupi.js`（IIFE，CDN 直引） |
| `files` | `["dist"]` |
| `dependencies` | 空（现有三个打包依赖移到 `devDependencies`） |

`exports` 不包含 CJS 条件：SDK 是浏览器 ESM 包，不承诺 `require`。

### 公开类型契约（`sdk/src/index.ts`）

在现有 `InitOptions`、`AnnotationInstance`、`init` 之外，新增类型 re-export：

- 来自 `events.ts`：`Annotation`、`AnnotationComment`、`User`、`Author`、`QuoteSelector`、`Anchor`、`AuthProvider`、`SiteConfig`、`Identity`、`SessionView`、`EventMap`
- 来自 `client.ts`：`CreateAnnotationInput`
- 来自 `state.ts`：`ThemeMode`（`InitOptions.theme` 的取值类型）

只做 `export type`，不新增运行时代码。

### 打包校验脚本（`sdk/scripts/verify-package.mjs`）

输入：执行 `npm pack --json` 得到的 tarball 与文件清单。
输出：进程退出码。

校验项：

1. 文件清单不含 `.map` 与 `*.test.d.ts`；
2. tarball ≤ 300KB，解包体积 ≤ 1.5MB；
3. 在临时目录初始化消费者工程，安装 tarball 与 TypeScript；
4. 写入同时导入 `init` 与公开类型的示例文件，执行 `tsc --noEmit`；
5. 校验 `npm ls --omit=dev` 的运行时依赖为空。

## 模块设计

### 包描述（`sdk/package.json`）

**职责：** 定义安装者看到的包契约。
**对外接口：** 包名、入口、exports、类型、元数据。
**依赖：** 无。

### 构建配置（`sdk/vite.config.ts`）

**职责：** 产出 ESM、IIFE 与类型声明。
**改动：**
- `build.sourcemap` 改为 `false`；
- `vite-plugin-dts` 增加 `exclude: ['**/*.test.ts']`，避免 `*.test.d.ts` 进入 `dist`。
**依赖：** 现有 Vite、vite-plugin-dts。

### 类型入口（`sdk/src/index.ts`）

**职责：** 唯一公开 API 面。
**改动：** 追加类型 re-export，不改变任何运行时导出。
**依赖：** `events.ts`、`client.ts`、`state.ts`。

### 发布校验脚本（`sdk/scripts/verify-package.mjs`）

**职责：** 在本地与 CI 中证明「包能装、能导入、类型能解析、体积达标」。
**对外接口：** `node scripts/verify-package.mjs`。
**依赖：** Node 内置模块、npm、TypeScript（临时安装）。

### CI 校验（`.github/workflows/ci.yml`）

**职责：** PR 与 main 分支的发布前门禁。
**步骤：** checkout → setup-node 22 → `npm ci` → `npm run typecheck` → `npm test` → `npm run build` → `node scripts/verify-package.mjs`。
**权限：** `contents: read`。

### 发布流水线（`.github/workflows/publish.yml`）

**职责：** tag 触发的自动发布。
**步骤：**
1. checkout；
2. setup-node 22，registry 指向 npmjs；
3. 升级 npm 到 ≥ 11.5.1（Trusted Publishing 要求）；
4. `npm ci`、`npm run build`；
5. 校验 `v<package.json version>` 与 tag 完全一致，否则退出；
6. `npm publish --provenance --access public`。
**权限：** `contents: read`、`id-token: write`。
**环境：** 绑定 GitHub Environment `npm`，供 npm 侧受信发布者配置引用。

### 许可证与文档

- 根 `LICENSE`：MIT，覆盖整个仓库；
- `sdk/LICENSE`：同一份 MIT，确保 npm 包页面展示许可证；
- `sdk/README.md`：安装、快速开始、公开 API、配置、事件、主题/国际化、浏览器支持、与自托管服务的关系、许可证。

## 模块交互

```
开发机 / PR
  npm ci → typecheck → test → build → verify-package
                                      ├─ npm pack 内容与体积
                                      ├─ 临时消费者安装
                                      └─ tsc --noEmit

发布
  git tag v0.1.0 → publish.yml
                     ├─ 校验 tag == sdk/package.json version
                     ├─ npm ci → build
                     └─ npm publish (OIDC) → npm registry + provenance

服务端
  make sdk → sdk/dist → internal/sdkassets/dist → go:embed
           → /sdk/v1/zhupi.js, /sdk/v1/zhupi.esm.js（不变）
```

首次发布例外：npm 要求包已存在才能配置受信发布者，0.1.0 先在本地用一次性令牌执行 `npm publish --access public`，随后在 npm 设置中把受信发布者指向 `Bin-hy/zhupi` + `publish.yml` + 环境 `npm`，之后的版本才走 OIDC。

## 文件组织

```
zhupi/
├── LICENSE                                   — 新建，MIT（仓库级）
├── .github/
│   └── workflows/
│       ├── ci.yml                            — 新建，PR 校验
│       └── publish.yml                       — 新建，tag 发布
└── sdk/
    ├── LICENSE                               — 新建，MIT（包内）
    ├── README.md                             — 新建，使用者文档
    ├── package.json                          — 修改，元数据 / exports / 依赖归位
    ├── package-lock.json                     — 修改，随依赖移动更新
    ├── vite.config.ts                        — 修改，sourcemap 与 dts exclude
    ├── scripts/
    │   └── verify-package.mjs                — 新建，打包与类型校验
    └── src/
        └── index.ts                          — 修改，公开类型 re-export
```

## 技术决策

| 决策点 | 选择 | 理由 |
|--------|------|------|
| 构建工具 | 保留 Vite library 模式 | 现状可用，迁移到 tsup/unbuild 无收益 |
| 依赖策略 | 三个打包依赖移入 devDependencies | 产物已自包含，消费者不应重复安装 |
| 类型出口 | 在 `index.ts` 显式 re-export | 单一入口、契约稳定；避免消费者抄内部路径 |
| sourcemap | 生产构建关闭 | 直接满足体积目标；调试可用本地 dev 构建 |
| 测试声明 | dts 插件排除 `*.test.ts` | 包内只保留公开声明 |
| IIFE 暴露 | `unpkg` / `jsdelivr` 字段 | CDN 可直接引用；不塞进 `exports` 的 ESM 条件 |
| 发布认证 | 0.1.0 手动 bootstrap → Trusted Publishing | npm 限制：包需先存在才能配置 OIDC |
| CI 运行时 | Node 22 + npm ≥ 11.5.1 | 受信发布的最低要求 |
| 许可证 | MIT（根 + 包内各一份） | 用户选择；npm 包需自带许可证文件 |
| 版本 | 0.1.0 | 未对外验证 API，不宣称 1.0 稳定 |

## Spec 覆盖检查

| Spec 需求 | Plan 归属 |
|-----------|-----------|
| F1 发布元数据 | 包描述 |
| F2 入口与导出映射 | 包描述（main/module/types/exports/unpkg） |
| F3 类型契约 | 公开类型契约、类型入口 |
| F4 依赖归位 | 包描述、技术决策 |
| F5 内容裁剪 | 构建配置、校验脚本 |
| F6 许可证与文档 | 许可证与文档 |
| F7 CI 校验 | CI 校验工作流 |
| F8 发布流水线 | 发布流水线、模块交互 |
| F9 服务端内嵌兼容 | 架构概览、模块交互 |
| N1 包体积 | 校验脚本 |
| N2 类型可用性 | 校验脚本第 3–4 步 |
| N3 发布可审计 | 发布流水线（tag 校验、provenance） |
| N4 零新增运行时依赖 | 依赖归位 |
| N5 构建可复现 | CI 从 `npm ci` 开始 |
