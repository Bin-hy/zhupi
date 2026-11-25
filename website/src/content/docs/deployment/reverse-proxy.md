---
title: 反向代理与 HTTPS
description: 用 Caddy 或 Nginx 为朱批提供 HTTPS，正确传递客户端 IP、限制请求体并配置回调地址。
---

朱批默认监听 `:8080`，Docker Compose 只把它发布到宿主机 `127.0.0.1:8080`。生产环境应通过 Caddy/Nginx 提供 HTTPS，不要把 8080 直接暴露到公网。

## 必须一致的地址

```dotenv
PUBLIC_BASE_URL=https://annot.example.com
```

它用于拼接 OAuth / OIDC 回调地址，必须是**外部可访问的绝对地址**：

```text
${PUBLIC_BASE_URL}/api/v1/auth/github/callback
${PUBLIC_BASE_URL}/api/v1/auth/oidc/{providerId}/callback
```

代理后的外部协议是 HTTPS，但服务本身只看到 HTTP，因此 `PUBLIC_BASE_URL` 必须写 `https://`，不能依赖请求头推断。

## Caddy

仓库提供 `deploy/Caddyfile.example`：

```caddyfile
annot.example.com {
	encode zstd gzip

	reverse_proxy 127.0.0.1:8080 {
		header_up X-Forwarded-For {remote_host}
	}
}
```

Caddy 会自动申请和续期证书。把域名换成你的子域名，DNS 指向服务器后重载配置：

```bash
caddy reload --config /etc/caddy/Caddyfile
```

## Nginx

仓库提供 `deploy/nginx.conf.example`：

```nginx
server {
    listen 80;
    server_name annot.example.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl;
    http2 on;
    server_name annot.example.com;

    ssl_certificate     /etc/letsencrypt/live/annot.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/annot.example.com/privkey.pem;

    client_max_body_size 128k;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host              $host;
        proxy_set_header X-Real-IP         $remote_addr;
        proxy_set_header X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 30s;
    }
}
```

## 请求体大小

服务端把 JSON 请求体限制在 **128 KiB**（`http.MaxBytesReader`）。Nginx 默认的 1MB 不会造成问题，但按示例设为 `128k` 更贴近实际：

```nginx
client_max_body_size 128k;
```

超过限制会返回 `400`，错误码 `validation_error`。

## 客户端 IP

限流与登录风控按客户端 IP 计数。服务只有在 `TRUST_PROXY=1` 时才会读取 `X-Forwarded-For` 的第一个地址：

```dotenv
TRUST_PROXY=1
```

- **前面有可信代理**：设为 `1`，并确保代理覆盖（而不是追加伪造的）`X-Forwarded-For`。
- **直接暴露服务**：保持 `0`，否则客户端可以伪造 IP 绕过限流。

## 回调与 CORS

- GitHub OAuth App 的回调地址填 `${PUBLIC_BASE_URL}/api/v1/auth/github/callback`。
- 站点 OIDC 的回调地址填 `${PUBLIC_BASE_URL}/api/v1/auth/oidc/{providerId}/callback`。
- 管理后台与 API 同源（`/admin/` 与 `/api/`），通常不需要跨域。
- 只有需要从**其他来源**调用管理 API 时，才配置 `ADMIN_ORIGINS`。

## 健康检查

让负载均衡或监控探测：

```bash
curl -fsS https://annot.example.com/healthz
curl -fsS https://annot.example.com/readyz
```

`/readyz` 会检查数据库；数据库不可用时返回 `503`。

## 常见问题

| 现象 | 检查 |
| --- | --- |
| 登录回调跳到 `http://127.0.0.1:8080` | `PUBLIC_BASE_URL` 没改成外部 HTTPS 地址 |
| 限流总是命中所有人 | `TRUST_PROXY` 配置与代理的 `X-Forwarded-For` 不一致 |
| 批注请求 403 `origin_not_allowed` | 站点 Origin 白名单没包含当前页面的 `scheme://host[:port]` |
| OIDC 回调 404 | 提供商 ID 与回调路径不一致，或代理改写了路径 |
| 大段批注返回 400 | 请求体超过 128 KiB |

更多问题见 [常见问题](/reference/faq/)。
