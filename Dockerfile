# syntax=docker/dockerfile:1

# 1) 构建可嵌入 SDK
FROM node:22-alpine AS sdk
WORKDIR /src/sdk
COPY sdk/package.json sdk/package-lock.json ./
RUN npm ci
COPY sdk/ ./
RUN npm run build

# 1.1) 构建管理后台
FROM node:22-alpine AS admin
WORKDIR /src/admin
COPY admin/package.json admin/package-lock.json ./
RUN npm ci
COPY admin/ ./
RUN npm run build

# 2) 编译 Go 服务，并把 SDK / 管理后台产物 embed 进去
FROM golang:1.27.3-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=sdk /src/sdk/dist ./internal/sdkassets/dist
COPY --from=admin /src/admin/dist ./internal/adminassets/dist
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
    -o /out/zhupi ./cmd/zhupi

# 3) 运行时
FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata \
 && addgroup -S -g 10001 app \
 && adduser -S -D -H -u 10001 -G app app \
 && mkdir -p /data \
 && chown app:app /data

USER 10001:10001
WORKDIR /app
COPY --from=build /out/zhupi /app/zhupi

ENV ADDR=:8080 \
    DB_PATH=/data/zhupi.db

EXPOSE 8080
VOLUME ["/data"]

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8080/healthz >/dev/null || exit 1

ENTRYPOINT ["/app/zhupi"]
CMD ["serve"]
