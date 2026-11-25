# 朱批官方文档站

`zhupi.pages.dev` 的源码，基于 **Astro 7 + Starlight 0.43**。

```bash
npm install
npm run dev        # http://localhost:4321
npm run build      # 产物在 dist/
npm run preview
```

- 文档内容：`src/content/docs/`
- 站点配置与侧边栏：`astro.config.mjs`
- 主题：`src/styles/custom.css`
- 全站批注接入：`src/components/Head.astro`
- 生产地址：<https://zhupi.pages.dev>

批注默认连接 `https://zhupi.gataccay.com`。构建时通过环境变量提供
`PUBLIC_ZHUPI_SITE_KEY` 才会注入 SDK；缺失时官网正常构建，但不显示批注入口。
API 与 SDK 地址可用 `PUBLIC_ZHUPI_API_BASE`、`PUBLIC_ZHUPI_SDK_URL` 覆盖。

本地开发可先执行 `cp .env.example .env`，再填入 `PUBLIC_ZHUPI_SITE_KEY`。

Cloudflare Pages：Root directory 设为 `website`，构建命令 `npm run build`，输出目录 `dist`，Node.js `22.23.4`。
