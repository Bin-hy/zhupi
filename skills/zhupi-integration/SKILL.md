---
name: zhupi-integration
description: Add zhupi (朱批) text-selection highlights and inline comments to a web project end to end - run the self-hosted zhupi server, create a siteKey with an Origin allowlist, embed zhupi-sdk through a script tag, npm/ESM, or a framework, and wire SPA route changes. Use when asked to add annotated passages, inline review comments, highlight-and-comment, or 划线批注 to a blog, docs site, wiki, knowledge base, or web app.
---

# Integrate zhupi annotations

zhupi is a self-hosted annotation service plus an embeddable browser SDK (npm `zhupi-sdk`, MIT). Readers select text, highlight it, and comment inline. The UI runs in a Shadow DOM, so host page CSS is untouched. Anchors use the W3C `TextQuoteSelector` model (exact quote plus prefix/suffix), so highlights survive small content edits and can be manually re-anchored after larger rewrites.

Three parts must line up:

| Part | What it is | Where it lives |
| --- | --- | --- |
| Server | One Go binary + SQLite; serves the API, admin console, and SDK asset | Your infrastructure |
| Site | A `siteKey` plus an Origin allowlist | Created on the server |
| SDK | `zhupi-sdk` bundle initialized on the page | Your front end |

## Inputs to settle before editing code

| Input | If unknown | Why it matters |
| --- | --- | --- |
| zhupi base URL | Ask whether a service already exists; otherwise self-host | SDK `apiBase` and OAuth callbacks |
| Page origins | Ask for every environment (prod, preview, local) | Wrong origins fail with `origin_not_allowed` |
| Framework and router | Inspect the repo | Decides script vs ESM and whether `setPage` is needed |
| Content container selector | Inspect the rendered page | `selectRoot` should exclude nav, sidebar, footer |
| Page identity | Default to `location.pathname` | `pageKey` must be stable per document |

Never guess production origins. `http://localhost:5173` and `http://127.0.0.1:5173` are different origins, and ports are part of the origin.

## 1. Get a reachable service

If a base URL already exists, verify it:

```bash
curl -fsS https://annot.example.com/healthz   # {"status":"ok"}
curl -fsS https://annot.example.com/readyz    # {"status":"ready"}
```

Otherwise self-host with Docker:

```bash
git clone https://github.com/Bin-hy/zhupi.git && cd zhupi
cp .env.example .env
# edit .env: PUBLIC_BASE_URL=https://annot.example.com
#            SECRET_KEY=$(openssl rand -hex 32)
docker compose pull && docker compose up -d
```

Required env: `PUBLIC_BASE_URL` (external absolute URL, used to build OAuth/OIDC callbacks), `SECRET_KEY` (32-byte hex or base64), `DB_PATH`. Compose publishes only `127.0.0.1:8080`; put HTTPS (Caddy/Nginx) in front. Full runbook: [references/self-hosting.md](references/self-hosting.md).

## 2. Create the site and capture the siteKey

```bash
docker compose exec zhupi /app/zhupi admin site create \
  --name blog \
  --display-name "My blog" \
  --origin https://blog.example.com
```

- The JSON output contains `key`; that is the `siteKey`. It is a public identifier, not a secret.
- `--origin` is repeatable and must be exactly `scheme://host[:port]`, no path, no wildcard.
- Optional: `--moderation post|pre` (default `post`), `--theme "#087f5b"`, `--locale zh-CN|en`.
- Later: `admin site list`; `admin site rotate-key --site <key>` revokes the old key immediately.

## 3. Embed the SDK

### Script tag (asset served by your zhupi instance, no bundler)

```html
<script src="https://annot.example.com/sdk/v1/zhupi.js"></script>
<script>
  const annotations = ZhupiSDK.init({
    siteKey: 'YOUR_SITE_KEY',
    apiBase: 'https://annot.example.com',
    selectRoot: 'article',
    pageKey: location.pathname,
    pageTitle: document.title,
    theme: 'auto',
    on: { error: (err) => console.error('[zhupi]', err) },
  })
</script>
```

### npm / ESM

```bash
npm install zhupi-sdk
```

```ts
import { init } from 'zhupi-sdk'

const annotations = init({
  siteKey: 'YOUR_SITE_KEY',
  apiBase: 'https://annot.example.com',
  selectRoot: '.article-body',
})
```

Snippet checklist:

- Pass `apiBase` unless the script is loaded from the zhupi server itself. Resolution order is: option, script/module origin, page origin. Bundled builds lose the server origin.
- `selectRoot` is the article body; without it the whole `document.body` is selectable. A missing element falls back to `body` and is re-resolved on `setPage`.
- `pageKey` defaults to `location.pathname`. Pages that differ only by query string need an explicit key.
- Initialize only in the browser. The SDK touches `document` and `localStorage` at init time.
- `init` returns synchronously and loads data in the background; listen for `ready` before assuming annotations are loaded.

## 4. Update page context on route changes (SPA)

Call `setPage` after every route change, or new routes keep writing to the previous `pageKey`:

```ts
await annotations.setPage({ pageKey: location.pathname, pageTitle: document.title })
```

It reloads annotations, clears selection and search, and switches drafts. On teardown (micro-frontend switch, HMR, tests) call `annotations.destroy()`; the instance is unusable afterwards.

## 5. Authentication (optional; required to post comments)

Login is a full-page OAuth redirect via `instance.login(providerId?)` and returns to the current page. Configure one of:

- Global GitHub OAuth: `GITHUB_CLIENT_ID` / `GITHUB_CLIENT_SECRET`, callback `${PUBLIC_BASE_URL}/api/v1/auth/github/callback`.
- Site-level OIDC: `admin provider add --site <key> --issuer ... --client-id ... --client-secret-file ...`, callback `${PUBLIC_BASE_URL}/api/v1/auth/oidc/{providerId}/callback`.
- Local development only: server `DEV_LOGIN=1` plus SDK `devLogin: true`. Never enable this in production.

## 6. Verify

- Selecting text inside `selectRoot` shows the floating annotate button.
- Posting works, and a reload keeps the highlight.
- Editing a few characters of the source text keeps relocation working; larger edits show the orphan/re-anchor entry.
- Network requests carry `X-Site-Key: <siteKey>` and return 2xx; no `origin_not_allowed`.
- `getAnnotationCount()` and the `statechange` event update the host nav if you wired one.

## Rules that prevent most breakage

- Do not proxy the zhupi API through the host backend. The browser calls `apiBase` directly; access is authorized by Origin plus `siteKey`.
- Keep `siteKey` in client code. Keep `SECRET_KEY`, OAuth secrets, and OIDC secrets server-side only.
- Do not reuse one `pageKey` across unrelated documents. Changing URL slugs changes page identity for existing annotations.
- Initialize once per page; do not re-init on every render.
- Under a strict CSP, allow the zhupi origin in `script-src` and `connect-src`, and allow inline styles for the Shadow DOM UI.
- Text inside iframes needs a separate SDK instance inside that iframe; selections do not cross frames.

## Common failures

| Symptom | Likely cause | Fix |
| --- | --- | --- |
| `origin_not_allowed` | Page origin missing from allowlist | Add exact `scheme://host[:port]` to the site |
| `site_key_required` | `siteKey` not passed | Check `init` options |
| `not_found` on every call | Wrong or rotated `siteKey` | `admin site list` / re-create or update key |
| CORS preflight fails | Origin mismatch or site disabled | Check site status and allowlist |
| Login returns to the wrong host | `PUBLIC_BASE_URL` wrong | Set the external HTTPS URL and restart |
| Highlight disappears after navigation | SPA route changed without `setPage` | Call `setPage` in the router hook |
| No annotate button on selection | Selection outside `selectRoot` or UI hidden | Verify selector; check `container`/z-index |
| `window.ZhupiSDK` undefined | Script blocked/404 or init before load | Check network, CSP, script order |

Full diagnosis: [references/troubleshooting.md](references/troubleshooting.md).

## References

- [references/frontend.md](references/frontend.md): VitePress, Docusaurus, Astro/Starlight, Next.js, Nuxt, Vue/React SPA recipes; option, method, and event tables; theming; deep links.
- [references/self-hosting.md](references/self-hosting.md): Docker/source install, env vars, reverse proxy, auth, CLI, backups, upgrades.
- [references/troubleshooting.md](references/troubleshooting.md): symptom-to-fix table and debugging steps.

Upstream docs: https://zhupi.pages.dev (getting started, SDK integration, options, API).
