<div align="center">

<img src="website/src/assets/logo.svg" width="96" alt="朱批 zhupi" />

# 朱批 · zhupi

**让讨论回到原文本身**

为博客、文档站与知识库打造的划线批注基础设施。选中一段文字即可划线、评论与回复，无需改造站点后端，也不污染宿主页面样式。

Self-hosted, embeddable text annotation and inline comments for any web page.

[![文档](https://img.shields.io/badge/文档-zhupi.pages.dev-c92a2a?style=flat-square)](https://zhupi.pages.dev)
[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?style=flat-square&logo=go&logoColor=white)](go.mod)
[![SDK](https://img.shields.io/badge/SDK-v0.2.0-3178C6?style=flat-square&logo=typescript&logoColor=white)](sdk/package.json)
[![skills.sh](https://skills.sh/b/Bin-hy/zhupi)](https://skills.sh/Bin-hy/zhupi)
[![Docker](https://img.shields.io/badge/GHCR-ghcr.io%2Fbin--hy%2Fzhupi-2496ED?style=flat-square&logo=docker&logoColor=white)](https://github.com/Bin-hy/zhupi/pkgs/container/zhupi)
[![License](https://img.shields.io/badge/License-MIT-2f9e44?style=flat-square)](LICENSE)

[官方文档](https://zhupi.pages.dev) · [快速开始](https://zhupi.pages.dev/getting-started/quick-start/) · [SDK 接入](https://zhupi.pages.dev/sdk/integration/) · [GitHub](https://github.com/Bin-hy/zhupi)

</div>

---

## 为什么是朱批

传统评论区围绕「这篇文章」组织讨论，但很多真正有价值的问题，只针对其中的一句话、一个概念或一段代码。

朱批把讨论单位从**页面**缩小到**原文片段**：

- 读者选中文字即可创建批注，讨论始终贴着上下文；
- 批注使用 W3C `TextQuoteSelector` 保存引用，正文轻微变化后仍可模糊重定位；
- 博主、文档作者和读者可以在正文里直接澄清问题，不必反复描述「我说的是哪一段」。

> **一句话理解：Artalk 让读者讨论「这篇文章」，朱批让读者讨论「这句话」。**

## 核心能力

| 能力 | 说明 |
| --- | --- |
| **正文级锚点** | 保存原文、前后文与位置信息；内容变化后自动重定位，失效时可手动重新锚定 |
| **可嵌入 SDK** | 支持 `<script>` 与 ESM；Shadow DOM 隔离 UI，不覆盖宿主 CSS |
| **多框架与 SPA** | 适用于静态博客、文档站和客户端路由；路由切换用 `setPage()` 更新页面 |
| **多站点与治理** | `siteKey` + Origin 白名单；站点级审核、封禁与管理员权限 |
| **登录与账号** | 站点访客：GitHub OAuth、站点级 OIDC；管理后台：本地账号密码；身份绑定与会话管理 |
| **自托管** | 单 Go 二进制 + SQLite（WAL），内嵌 SDK 与管理后台；Docker 一键部署 |
| **开放 API** | 批注、评论、账号与管理 API；统一错误结构与请求 ID |

## 和 Artalk 怎么选

朱批不试图替代完整评论区，而是补上「贴着原文讨论」这一层。

| 维度 | Artalk | 朱批 |
| --- | --- | --- |
| **讨论对象** | 页面与评论线程 | 选中的文本片段 |
| **主要场景** | 博客留言、社区互动 | 精读、纠错、审阅、知识讨论 |
| **组合方式** | 独立承担页面评论 | 可与 Artalk 共存，专注原文批注 |

完整对比与适用场景见 [项目简介](https://zhupi.pages.dev/getting-started/overview/)。

## 快速开始

```bash
git clone https://github.com/Bin-hy/zhupi.git
cd zhupi

cp .env.example .env
# 至少修改 PUBLIC_BASE_URL；SECRET_KEY 用 openssl rand -hex 32 生成
docker compose pull
docker compose up -d

# 创建第一个站点，输出中的 key 即 siteKey
docker compose exec zhupi /app/zhupi admin site create \
  --name blog \
  --origin http://127.0.0.1:5173
```

管理后台：<http://127.0.0.1:8080/admin/>，首次启动默认本地管理员 `admin` / `admin`，登录后强制改密。

源码启动、示例页面和完整配置见 [快速开始](https://zhupi.pages.dev/getting-started/quick-start/)。

## 接入 SDK

服务端已内嵌 SDK，在正文页面加载脚本并初始化即可：

```html
<script src="https://annot.example.com/sdk/v1/zhupi.js"></script>
<script>
  ZhupiSDK.init({
    siteKey: 'YOUR_SITE_KEY',
    selectRoot: 'article',
    pageKey: location.pathname,
    pageTitle: document.title,
    theme: 'auto',
  })
</script>
```

也支持 `npm install zhupi-sdk` 与 ESM。适用于 Hexo、Hugo、Jekyll、Astro、VitePress、Docusaurus、Starlight、Vue、React、Next.js、Nuxt 及任意 Web 页面；SPA 路由切换时调用 `instance.setPage(...)`。

- [SDK 接入指南](https://zhupi.pages.dev/sdk/integration/)
- [初始化选项](https://zhupi.pages.dev/sdk/options/)
- [方法与事件](https://zhupi.pages.dev/sdk/methods-events/)

### 让 AI 编码代理接入

```bash
npx skills add Bin-hy/zhupi --skill zhupi-integration
```

Skill 覆盖部署、创建站点、嵌入 SDK、SPA 路由与排错，安装与更新由 [skills.sh](https://skills.sh/Bin-hy/zhupi) 管理。

## 文档

| 主题 | 内容 |
| --- | --- |
| [项目简介](https://zhupi.pages.dev/getting-started/overview/) | 产品定位、组成、适用场景与 Artalk 对比 |
| [快速开始](https://zhupi.pages.dev/getting-started/quick-start/) | Docker / 源码启动、创建站点、示例页 |
| [接入指南](https://zhupi.pages.dev/sdk/integration/) | script / ESM、SPA、VitePress 示例 |
| [登录与身份](https://zhupi.pages.dev/sites/authentication/) | 访客 GitHub/OIDC、后台本地账号、身份绑定 |
| [站点与权限](https://zhupi.pages.dev/sites/roles-moderation/) | 多站点、角色、审核、封禁、后台 |
| [部署](https://zhupi.pages.dev/deployment/docker/) | Docker、反向代理、备份与环境变量 |
| [API 与 CLI](https://zhupi.pages.dev/api/overview/) | [公共 API](https://zhupi.pages.dev/api/overview/)、[管理 API](https://zhupi.pages.dev/api/admin-account/)、[CLI 参考](https://zhupi.pages.dev/cli/reference/) |
| [架构与开发](https://zhupi.pages.dev/development/architecture/) | 分层、数据模型、本地开发与发布 |
| [安全与边界](https://zhupi.pages.dev/reference/security/) | 安全模型、[V1 边界](https://zhupi.pages.dev/reference/boundaries/)、[常见问题](https://zhupi.pages.dev/reference/faq/) |

## 开发

```bash
make build   # SDK + 管理后台 + Go 二进制
make test    # Go + SDK 测试
```

开发环境、仓库结构、测试与发布流程见 [本地开发与测试](https://zhupi.pages.dev/development/local/) 与 [发布流程](https://zhupi.pages.dev/development/release/)。

## 许可与致谢

[MIT License](LICENSE) © 2026 Bin-hy

SDK 同时发布在 npm：[`zhupi-sdk`](https://www.npmjs.com/package/zhupi-sdk)。

朱批的文本锚定与高亮能力使用了以下开源项目：

- [Apache Annotator](https://github.com/apache/incubator-annotator)（`@apache-annotator/dom`，[Apache-2.0](https://www.apache.org/licenses/LICENSE-2.0)）：DOM 文本高亮与批注工具。
- [dom-anchor-text-quote](https://github.com/tilgovi/dom-anchor-text-quote)（MIT，Randall Leeds）：DOM `Range` 与 `TextQuoteSelector` 转换与近似匹配。
