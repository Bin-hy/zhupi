# Self-hosting reference

Use this when no zhupi service exists yet, or when the service is reachable but needs configuration.

## Requirements

| Path | Needs |
| --- | --- |
| Docker (recommended) | Docker Engine 24+, Compose v2 |
| From source | Go 1.26+, Node.js 22+, `make` |
| Either | `openssl` for `SECRET_KEY` |

The server listens on `:8080`. The bundled Compose file publishes only `127.0.0.1:8080`, so HTTPS termination on the host is expected.

## Docker install

```bash
git clone https://github.com/Bin-hy/zhupi.git
cd zhupi
cp .env.example .env
```

Edit `.env`:

```dotenv
PUBLIC_BASE_URL=https://annot.example.com
SECRET_KEY=<openssl rand -hex 32>
```

Then:

```bash
docker compose pull
docker compose up -d
docker compose logs -f zhupi
curl -fsS http://127.0.0.1:8080/healthz
curl -fsS http://127.0.0.1:8080/readyz
```

Data lives in the `zhupi-data` Docker volume at `/data/zhupi.db` (SQLite, WAL mode). The service does not read `.env` itself; Compose injects those variables.

## Source install

```bash
git clone https://github.com/Bin-hy/zhupi.git
cd zhupi

make build            # SDK + admin console + Go binary -> bin/zhupi

export SECRET_KEY=$(openssl rand -hex 32)
export DB_PATH=./zhupi.db
export PUBLIC_BASE_URL=http://127.0.0.1:8080   # use the external HTTPS URL in production

./bin/zhupi serve
```

`make build` runs `make sdk`, `make admin`, then `make server`; each step can be run alone. The SDK and admin console are embedded into the Go binary, so a single binary serves everything.

## Environment variables

| Variable | Default | Purpose |
| --- | --- | --- |
| `ADDR` | `:8080` | Listen address |
| `DB_PATH` | `./zhupi.db` | SQLite file; `/data/zhupi.db` in Docker |
| `PUBLIC_BASE_URL` | none | External absolute URL used for OAuth/OIDC callbacks |
| `SECRET_KEY` | required | 32-byte hex/base64 key; encrypts stored OIDC secrets |
| `SESSION_TTL_HOURS` | `720` | Session lifetime |
| `GITHUB_CLIENT_ID` / `GITHUB_CLIENT_SECRET` | empty | Global GitHub OAuth |
| `SUPERADMIN_GITHUB_LOGINS` | empty | Comma-separated GitHub logins promoted to superadmin on login |
| `LOCAL_ADMIN_USERNAME` / `LOCAL_ADMIN_PASSWORD` | `admin` / `admin` | Bootstrap local admin for the console; password change is forced |
| `LOCAL_ADMIN_RESET_PASSWORD` | `0` | Set `1` once to reset the local admin password and revoke its sessions |
| `TRUST_PROXY` | `0` in the binary, `1` in Compose | Read the first `X-Forwarded-For` hop |
| `ADMIN_ORIGINS` | empty | Allow cross-origin admin API calls; leave empty when admin and API are same-origin |
| `DEV_LOGIN` | `0` | Enables `/api/v1/auth/dev/login`; local development only |
| `ALLOW_MISSING_ORIGIN` | `0` | Local testing only; do not enable in production |
| `READ_RATE_PER_MIN` | `300` | Per-IP read limit |
| `WRITE_RATE_PER_MIN` | `60` | Per-IP write limit |
| `WRITE_USER_RATE_PER_MIN` | `30` | Per-user write limit |
| `LOCAL_LOGIN_RATE_PER_MIN` | `10` | Local admin login attempts per IP |

## Reverse proxy and HTTPS

`PUBLIC_BASE_URL` must be the external HTTPS URL. The app only sees HTTP behind the proxy, so do not rely on request headers.

Caddy (`deploy/Caddyfile.example`):

```caddyfile
annot.example.com {
	encode zstd gzip
	reverse_proxy 127.0.0.1:8080 {
		header_up X-Forwarded-For {remote_host}
	}
}
```

Nginx (`deploy/nginx.conf.example`) essentials: proxy to `127.0.0.1:8080`, set `Host`, `X-Real-IP`, `X-Forwarded-For`, `X-Forwarded-Proto`, and set `client_max_body_size 128k` (the app rejects bodies over 128 KiB).

Set `TRUST_PROXY=1` only when a trusted proxy overwrites `X-Forwarded-For`. Otherwise clients can spoof IPs and bypass rate limits.

## Admin console and local admin

- Console: `https://annot.example.com/admin/`
- First start creates the local superadmin (`admin` / `admin` by default); first login forces a password change.
- Reset later: `./bin/zhupi admin password reset --username admin` (prints a random password once) or `--password '...'`. Resetting revokes that account's sessions.

## Authentication

### GitHub OAuth (global)

1. Create a GitHub OAuth App.
2. Callback URL: `${PUBLIC_BASE_URL}/api/v1/auth/github/callback`.
3. Set `GITHUB_CLIENT_ID` / `GITHUB_CLIENT_SECRET` and restart.

Unset credentials hide the GitHub provider from `GET /api/v1/site-config`, so the SDK shows no GitHub login entry.

Promote an account:

```bash
./bin/zhupi admin promote --github-login <login>
./bin/zhupi admin promote --user-id <id>
```

### Site-level OIDC

```bash
./bin/zhupi admin provider add \
  --site <siteKey or ID> \
  --name "Company SSO" \
  --issuer https://sso.example.com \
  --client-id zhupi \
  --client-secret-file /run/secrets/oidc
```

Register the callback in the IdP:

```text
${PUBLIC_BASE_URL}/api/v1/auth/oidc/{providerId}/callback
```

`providerId` is zhupi's numeric provider ID, not the client ID. Add `--allowed-email-domains example.com` to restrict access. Discover document validation runs on create and when the issuer changes.

## Site CLI

```bash
# create; output JSON contains "key" (the siteKey)
./bin/zhupi admin site create --name blog --display-name "Blog" --origin https://blog.example.com

# list
./bin/zhupi admin site list

# rotate; old key stops working immediately
./bin/zhupi admin site rotate-key --site <siteKey or ID>

# site admins must have logged in at least once
./bin/zhupi admin site add-admin --site <siteKey or ID> --user <user ID or GitHub login>
./bin/zhupi admin site remove-admin --site <siteKey or ID> --user <user ID or GitHub login>
```

In Docker, prefix with `docker compose exec zhupi /app/zhupi ...` so the same database and env are used. CLI commands also require `SECRET_KEY`, `DB_PATH`, and a valid `PUBLIC_BASE_URL`.

## Backups

All business data is one SQLite database. WAL mode means `zhupi.db`, `zhupi.db-wal`, and `zhupi.db-shm` belong together.

Online backup (host has `sqlite3`):

```bash
sqlite3 /path/to/zhupi.db ".backup 'backup/zhupi-$(date +%F).db'"
```

Docker volume variant:

```bash
docker run --rm -v zhupi-data:/data -v "$PWD/backup":/backup alpine:3.22 \
  sh -c 'apk add --no-cache sqlite >/dev/null && sqlite3 /data/zhupi.db ".backup /backup/zhupi-$(date +%F).db"'
```

Also back up `SECRET_KEY` separately: losing it makes stored OIDC client secrets undecryptable. Stop the service before copying raw files.

## Upgrades

```bash
docker compose pull
docker compose up -d
```

Migrations run automatically on start. Back up first; older binaries may not read a newer schema. The current architecture is single-instance with SQLite: do not run multiple writers against the same database file.

## Production checklist

- `PUBLIC_BASE_URL` is the external HTTPS URL.
- `SECRET_KEY` is random, backed up, and never committed.
- `DEV_LOGIN=0`, `ALLOW_MISSING_ORIGIN=0`.
- Local admin password changed; a trusted GitHub account promoted to superadmin.
- Rate limits and `TRUST_PROXY` match the proxy setup.
- Automated backup plus a tested restore path.
- Each site's Origin allowlist contains only real page origins.
