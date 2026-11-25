# Troubleshooting reference

Work top to bottom. Most failures are an origin allowlist mismatch, a missing `apiBase`, or a SPA page that never called `setPage`.

## 60-second triage

Open the browser console and the Network panel, then confirm:

1. `window.ZhupiSDK` is defined after the script loads. If not, the script 404'd, was blocked, or ran after your init code.
2. A request to `<apiBase>/api/v1/site-config` (or similar) returned 2xx and carried `X-Site-Key`.
3. The page did not emit a 403 `origin_not_allowed` or a CORS error.
4. `document.querySelector(selectRoot)` matches exactly one visible content container.
5. The `error` event fires: attach `on: { error: (e) => console.error(e) }` or `instance.on('error', ...)`.

Capture the response's `X-Request-Id` header before retrying; it appears in server logs.

## API error codes

| Code | Meaning | Fix |
| --- | --- | --- |
| `site_key_required` | No `X-Site-Key` / no `siteKey` option | Pass it to `init` |
| `origin_not_allowed` | Page Origin not in the site allowlist | Add the exact `scheme://host[:port]` |
| `site_disabled` | Site disabled in the console | Re-enable the site |
| `site_banned` | Site blocked | Check global bans / site status |
| `not_found` | Unknown site key or resource | Wrong or rotated `siteKey`; verify with `admin site list` |
| `unauthorized` | No or expired session | Log in again; check `SESSION_TTL_HOURS` |
| `forbidden` | Authenticated but not allowed | Role, ownership, or moderation rule |
| `rate_limited` | Read/write/IP limit exceeded | Raise `READ_RATE_PER_MIN`, `WRITE_RATE_PER_MIN`, `WRITE_USER_RATE_PER_MIN` |
| `validation_error` | Bad payload, or body over 128 KiB | Shorter comment or fix inputs |

## Symptom to cause

| Symptom | Cause | Fix |
| --- | --- | --- |
| `window.ZhupiSDK` undefined | Script 404, CSP block, ad blocker, or init before load | Load in order; allow the origin in CSP; check Network |
| Floating button never appears | Selection outside `selectRoot`, or content not rendered yet | Fix selector; init after content mounts |
| Button covers host UI or is hidden | Host z-index, overflow, or transformed ancestor | Set `container` to a top-level node; adjust `--annotation-fab-*` |
| Annotate works but posting fails | Not logged in, or provider not configured | Configure GitHub OAuth or site OIDC |
| Login redirects to `127.0.0.1`/wrong host | `PUBLIC_BASE_URL` is wrong | Set the external HTTPS URL and restart |
| OIDC callback 404 | Callback path uses client ID instead of provider ID, or proxy rewrote the path | Use `${PUBLIC_BASE_URL}/api/v1/auth/oidc/{providerId}/callback` |
| First annotation after login lands on the wrong page | Login redirect dropped the path | Verify the redirect returns to the full page URL, including query string |
| Highlights vanish after navigating | SPA route change without `setPage` | Call `setPage` in the router hook |
| Two drawers / two buttons | SDK initialized twice (HMR, Strict Mode, double mount) | Init once; destroy on unmount |
| Annotations attach to the wrong article | Unstable `pageKey` or shared key across documents | Use a stable per-document key |
| Highlight off by a few words | Content edited; fuzzy relocation matched | Expected; user can re-anchor orphaned annotations |
| Orphan state everywhere | Large content rewrite or different rendering (templates, ads) | Re-anchor; avoid wrapping text in changing markup |
| Same text highlighted in two places | Repeated identical quotes | Prefix/suffix usually disambiguates; check the stored anchor |
| Requests fail only in preview deployments | Preview origin not allowlisted | Add each preview origin, or use a wildcard-free stable host |
| 400 on long comments | Body over 128 KiB | Shorten content; do not raise the limit casually |
| 429 under normal use | Shared IP or limit too low | Tune rate-limit env values |
| SSR crash in VitePress/Next/Nuxt | SDK ran on the server | Client-only import or plugin |

## Layer-by-layer checks

### Script and init

- IIFE path is `<apiBase>/sdk/v1/zhupi.js`. In Docker/source builds the SDK is embedded; if the server returns "SDK 尚未构建" (not built), run `make sdk`.
- If the host uses a strict CSP, allow the zhupi origin in `script-src`, `connect-src`, and inline styles for the Shadow DOM.
- `init` is synchronous; data loads in the background. Wait for `ready` before reading counts.

### API and CORS

- The browser must send an `Origin` header. Same-origin deployment works; cross-origin needs the exact origin in the site allowlist.
- Preflight requests do not carry `X-Site-Key`; the server authorizes them by checking whether the Origin belongs to any enabled site. A failed preflight usually means the site is disabled or the origin is wrong.
- `ALLOW_MISSING_ORIGIN=1` is a local-only escape hatch; do not ship it.

### Content and anchoring

- Anchors are quote-based. Identical passages, heavy templating, or text that changes on every render make matching harder.
- Content must live in the main document. If it is inside an iframe, initialize inside that iframe.
- Text rendered in canvas, images, or closed shadow roots cannot be selected.

### SPA and page identity

- If `pageKey` changes between visits (slugs, locale prefixes, query parameters), the same article looks new.
- `setPage` clears search, selection, and unread state for the previous page; that is expected.
- After `destroy()`, the instance cannot be reused; call `init` again.

### Auth

- Login is a full-page redirect, not a popup. If a test harness stubs navigation, finish the OAuth flow first.
- GitHub provider appears only when `GITHUB_CLIENT_ID`/`SECRET` are set.
- OIDC provider create/update validates the issuer's discovery document; the server needs outbound network access to the IdP.
- `SECRET_KEY` rotation breaks stored OIDC client secrets; re-enter them.

## When escalating

Include: SDK version, `apiBase`, page URL and origin, `pageKey`, framework, the failing request's `X-Request-Id`, the response code/message, and console errors. Never include `SECRET_KEY`, OAuth secrets, or session tokens.
