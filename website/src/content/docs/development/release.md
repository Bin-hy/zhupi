---
title: 发布流程
description: 版本号、tag、GHCR Docker 镜像、npm SDK 发布与发布前检查。
---

朱批通过 Git tag 驱动发布：推送 `v*` tag 会构建多架构 Docker 镜像并发布 SDK 到 npm。

## 发布物

| 发布物 | 位置 | 触发 |
| --- | --- | --- |
| Docker 镜像 | `ghcr.io/bin-hy/zhupi` | `main` 推送 → `edge`；`v*` tag → `latest` + 版本 tag（amd64 + arm64） |
| SDK | npm `zhupi-sdk` | `v*` tag，且 tag 必须等于 `v` + `sdk/package.json` 的 version |
| Go 二进制 | Docker 镜像内 | 多阶段构建 |

当前没有单独的 GitHub Release 附件流程；需要裸机二进制时从源码 `make server` 构建。

## 版本号

SDK 使用语义化版本：

```bash
cd sdk
npm version patch   # 或 minor / major；会改 package.json 与 lock
```

服务端没有单独的版本文件，Docker 镜像 tag 直接来自 Git tag。发布前确认：

- `sdk/package.json` 的 `version` 与准备推送的 tag 一致；
- 管理后台与 Go 服务的变更已包含在同一提交/合并中；
- 数据库迁移可向后兼容（升级会立即执行）。

## 发布步骤

```bash
# 1. 确认 main 为最新且 CI 通过
git checkout main
git pull

# 2. 更新 SDK 版本
cd sdk
npm version patch
cd ..

# 3. 提交并推送
git add sdk/package.json sdk/package-lock.json
git commit -m "chore(release): v0.2.1"
git push

# 4. 打 tag 并推送
git tag v0.2.1
git push origin v0.2.1
```

推送 tag 后：

1. **Docker workflow** 构建并推送 `ghcr.io/bin-hy/zhupi:0.2.1`、`:0.2`、`:latest`（多架构）。
2. **Publish workflow** 校验 `v0.2.1` 与 SDK 版本一致，然后 `npm publish --provenance --access public`。

任一校验失败，发布停止；修正后重新打 tag。

## Docker tags

| tag | 来源 | 架构 |
| --- | --- | --- |
| `edge` | `main` 最新提交 | amd64 |
| `latest` | 最新 `v*` tag | amd64 + arm64 |
| `X.Y.Z` | tag 版本 | amd64 + arm64 |
| `X.Y` | major.minor | amd64 + arm64 |

镜像包含 OCI labels、provenance 与 SBOM；GHCR 包当前为 public。

## npm 发布

Publish workflow 使用 npm trusted publishing（`id-token: write`），并依赖仓库的 `npm` environment。发布前会：

```bash
npm ci
npm run build
```

然后校验 tag 与版本一致。SDK 的 `files` 只包含 `dist/`，不会把源码或测试发布出去。

## 发布前检查

```bash
# Go
go vet ./...
go test ./...

# SDK
cd sdk
npm ci
npm run typecheck
npm test
npm run build
npm run verify

# 管理后台
cd ../admin
npm ci
npm run typecheck
npm test
npm run build

# 完整构建
cd ..
make build
```

另外确认：

- [ ] `.env.example` 与 `internal/config` 一致；
- [ ] 数据库迁移已在旧数据副本上验证；
- [ ] 文档（`website/`）中的命令、默认值与实际一致；
- [ ] 没有把 `.env`、`*.db`、密钥或构建产物加入提交；
- [ ] CHANGELOG / 提交信息说明了破坏性变更（如有）。

## 回滚

- **Docker**：把 `ZHUPI_TAG` 改回上一个版本并重新 `pull` + `up -d`。
- **npm**：npm 不支持覆盖已发布版本；发布新补丁版本修复，必要时 `npm deprecate` 旧版本。
- **数据库**：回滚二进制前先确认旧版本能读新 schema；不能确定时用升级前备份恢复。

回滚同样需要走一次发布流程（新 tag），不要删除或强制移动已发布的 tag。
