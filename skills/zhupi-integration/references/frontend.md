# Frontend integration reference

Use this when the basic embed from `SKILL.md` needs a framework recipe or a specific option.

## Pick the loading strategy

| Situation | Use | Notes |
| --- | --- | --- |
| Static HTML or server-rendered pages | `<script src="<zhupi>/sdk/v1/zhupi.js">` | Global `ZhupiSDK` |
| Bundler (Vite, webpack, Rspack, Next, Nuxt) | `npm install zhupi-sdk` | Pass `apiBase` explicitly |
| Multi-page site with full reloads | Either | No `setPage` needed |
| Client-side routing (SPA) | Either | Call `setPage` after every route change |

The IIFE build exposes `window.ZhupiSDK` with `init(options)`. The ESM build exports `init` (named) and also has a default export object.

## Recipes

### Plain HTML / static site

```html
<article id="post">...</article>

<script src="https://annot.example.com/sdk/v1/zhupi.js"></script>
<script>
  ZhupiSDK.init({
    siteKey: 'YOUR_SITE_KEY',
    apiBase: 'https://annot.example.com',
    selectRoot: '#post',
    pageKey: location.pathname,
    pageTitle: document.title,
  })
</script>
```

Place the script after the content container so `selectRoot` resolves on first paint. If you must load it in `<head>`, add `defer` to the SDK script and initialize in a `DOMContentLoaded` listener.

### Hexo, Hugo, Jekyll

Add the same snippet to the post layout, immediately before `</body>`. Set `selectRoot` to the generated article wrapper, for example `.post-content` (Hexo), `.post` or `article` (Hugo), `.markdown-body` (Jekyll). Keep `pageKey` as the permalink path so URL changes are explicit.

### Astro and Starlight

```astro
---
// src/layouts/Article.astro
---
<article class="prose"><slot /></article>

<script>
  import { init } from 'zhupi-sdk'

  init({
    siteKey: 'YOUR_SITE_KEY',
    apiBase: 'https://annot.example.com',
    selectRoot: '.prose',
  })
</script>
```

Astro bundles `<script>` as an ES module that runs after the markup exists; no `setPage` is needed for its default MPA navigation. For Starlight, use `selectRoot: '.sl-markdown-content'`. For pages with sticky headers, raise the SDK UI if it is covered: `cssVariables: { '--annotation-drawer-top': '72px' }`.

### VitePress

```ts
// .vitepress/theme/index.ts
import DefaultTheme from 'vitepress/theme'
import { useRouter } from 'vitepress'
import type { AnnotationInstance } from 'zhupi-sdk'

export default {
  extends: DefaultTheme,
  setup() {
    if (import.meta.env.SSR) return

    let annotations: AnnotationInstance | undefined
    const router = useRouter()

    import('zhupi-sdk').then(({ init }) => {
      annotations = init({
        siteKey: 'YOUR_SITE_KEY',
        apiBase: 'https://annot.example.com',
        selectRoot: '.vp-doc',
        pageKey: router.route.path,
        pageTitle: document.title,
      })
    })

    router.onAfterRouteChanged = (to) => {
      void annotations?.setPage({ pageKey: to, pageTitle: document.title })
    }
  },
}
```

Client-only import avoids SSR breakage. `.vp-doc` is the VitePress content container.

### Docusaurus

Wrap the app root so the SDK survives route changes:

```jsx
// src/theme/Root.js
import React, { useEffect, useRef } from 'react'
import { useLocation } from '@docusaurus/router'

export default function Root({ children }) {
  const { pathname } = useLocation()
  const instance = useRef(null)

  useEffect(() => {
    let alive = true
    import('zhupi-sdk').then(({ init }) => {
      if (!alive) return
      instance.current = init({
        siteKey: 'YOUR_SITE_KEY',
        apiBase: 'https://annot.example.com',
        selectRoot: '.markdown',
      })
    })
    return () => {
      alive = false
      instance.current?.destroy()
      instance.current = null
    }
  }, [])

  useEffect(() => {
    instance.current?.setPage({ pageKey: pathname, pageTitle: document.title })
  }, [pathname])

  return children
}
```

`selectRoot: '.markdown'` targets theme-classic docs content. The first `pathname` effect can run before the dynamic import resolves; that is safe because `init` already reads the current path.

### Next.js (App Router)

```tsx
// components/ZhupiAnnotations.tsx
'use client'

import { useEffect, useRef } from 'react'
import { usePathname } from 'next/navigation'
import type { AnnotationInstance } from 'zhupi-sdk'

export function ZhupiAnnotations() {
  const pathname = usePathname()
  const instance = useRef<AnnotationInstance | null>(null)

  useEffect(() => {
    let alive = true
    import('zhupi-sdk').then(({ init }) => {
      if (!alive) return
      instance.current = init({
        siteKey: 'YOUR_SITE_KEY',
        apiBase: 'https://annot.example.com',
        selectRoot: 'article',
      })
    })
    return () => {
      alive = false
      instance.current?.destroy()
      instance.current = null
    }
  }, [])

  useEffect(() => {
    instance.current?.setPage({ pageKey: pathname, pageTitle: document.title })
  }, [pathname])

  return null
}
```

Render `<ZhupiAnnotations />` from the root layout. Keep it in a client component and load the SDK through dynamic import so nothing runs during SSR.

### Nuxt 3

```ts
// plugins/zhupi.client.ts
import { init } from 'zhupi-sdk'

export default defineNuxtPlugin((nuxtApp) => {
  const annotations = init({
    siteKey: 'YOUR_SITE_KEY',
    apiBase: 'https://annot.example.com',
    selectRoot: '.prose',
  })

  nuxtApp.hook('page:finish', () => {
    const route = useRoute()
    void annotations.setPage({ pageKey: route.path, pageTitle: document.title })
  })
}
```

The `.client.ts` suffix keeps it browser-only.

### React or Vue SPA (generic)

Initialize once in the app shell, then update on router changes and destroy on unmount:

```ts
// React example
useEffect(() => {
  let instance: AnnotationInstance | undefined
  let alive = true

  import('zhupi-sdk').then(({ init }) => {
    if (!alive) return
    instance = init({ siteKey: 'YOUR_SITE_KEY', apiBase: 'https://annot.example.com' })
  })

  return () => {
    alive = false
    instance?.destroy()
  }
}, [])

useEffect(() => {
  void instance?.setPage({ pageKey: location.pathname, pageTitle: document.title })
}, [location.pathname])
```

In Vue, do the same in `onMounted` / `onBeforeUnmount` and hook `router.afterEach`. Do not re-init inside a render function or a watcher that fires often.

### iframes

The SDK does not cross frame boundaries. If the article body is inside an iframe, load and initialize the SDK inside that iframe's document with the same `siteKey`; its Origin must be in the site allowlist.

## Options

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| `siteKey` | `string` | required | Public site identifier |
| `apiBase` | `string` | script/module origin, then page origin | Trailing slashes are stripped |
| `selectRoot` | `HTMLElement \| string` | `document.body` | Text selection scope; falls back to body |
| `container` | `HTMLElement \| string` | `document.body` | Where the drawer, button, and account UI mount |
| `pageKey` | `string` | `location.pathname` | Annotation ownership key |
| `pageTitle` | `string` | `document.title` | Display and moderation only |
| `locale` | `'zh-CN' \| 'en'` | `'zh-CN'` | Site default applies when omitted |
| `theme` | `'auto' \| 'light' \| 'dark'` | `'auto'` | Explicit value overrides stored preference |
| `cssVariables` | `Record<string, string>` | none | Overrides Shadow DOM variables |
| `devLogin` | `boolean` | `false` | Requires server `DEV_LOGIN=1`; local only |
| `on` | event handler map | none | Same as `instance.on(...)` |

## Methods

| Method | Returns | Use |
| --- | --- | --- |
| `setPage({ pageKey?, pageTitle? })` | `Promise<void>` | After SPA navigation |
| `refresh()` | `Promise<void>` | Re-fetch current page annotations |
| `login(providerId?)` | `void` | Full-page OAuth redirect; provider chooses site OIDC |
| `logout()` | `void` | End session |
| `getCurrentUser()` | `User \| null` | Current user |
| `getAnnotationCount()` | `number` | Current page count |
| `setTheme(mode)` | `void` | `auto` / `light` / `dark` |
| `openDrawer()` | `void` | Open the annotation drawer |
| `openAccount()` | `void` | Open account center |
| `on(event, handler)` | `() => void` | Subscribe; returns unsubscribe |
| `destroy()` | `void` | Remove UI, listeners, timers; instance is dead |

## Events

| Event | Payload | When |
| --- | --- | --- |
| `ready` | `{ siteKey }` | Initial load finished |
| `login` | `User` | Login succeeded |
| `logout` | `undefined` | Logged out |
| `created` | `Annotation` | Annotation created |
| `updated` | `Annotation` | Annotation or anchor updated |
| `deleted` | `{ id }` | Annotation deleted |
| `error` | `Error` | Network, permission, or quota failure |
| `pagechange` | `{ pageKey }` | Page context changed |
| `statechange` | `{ user, annotationCount, loggedIn }` | Login state or count changed |

Host nav integration:

```ts
const off = annotations.on('statechange', ({ loggedIn, annotationCount }) => {
  nav.setLoggedIn(loggedIn)
  nav.setAnnotationCount(annotationCount)
})

document.querySelector('#annotations')?.addEventListener('click', () => {
  annotations.openDrawer()
})
```

## Theming and i18n

- Theme preference is stored per `apiBase` plus `siteKey`; passing `theme` overrides it.
- Site `themeColor` and `defaultLocale` act as site defaults; `init.locale` wins.
- Useful CSS variables: `--annot-accent`, `--annot-bg`, `--annot-fg`, `--annot-muted`, `--annot-border`, `--annot-soft`, `--annot-danger`, `--annotation-fab-right`, `--annotation-fab-bottom`, `--annotation-drawer-top`.
- Dark mode overrides some colors; set both light and dark values or pin `theme`.

## Deep links

Every annotation has a stable fragment: `https://blog.example.com/post#annot-123`. On startup the SDK parses `#annot-<id>`, scrolls to it, and highlights it. If content changed enough that automatic relocation fails, the annotation becomes orphaned and the UI offers a manual re-anchor flow.

## Content Security Policy

If the host sets CSP, allow the zhupi origin in `script-src` (script tag path) and `connect-src` (API calls). The Shadow DOM UI injects inline styles, so a strict `style-src` also needs `'unsafe-inline'` or a compatible nonce strategy. Test with the browser console open; violations otherwise look like a silent no-op.
