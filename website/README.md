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
- 生产地址：<https://zhupi.pages.dev>

Cloudflare Pages：Root directory 设为 `website`，构建命令 `npm run build`，输出目录 `dist`，Node.js `22.23.4`。
