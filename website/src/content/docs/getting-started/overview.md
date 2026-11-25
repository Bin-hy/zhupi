---
title: 项目简介
description: 朱批是什么、能做什么、由哪些部分组成，以及它不做什么。
---

朱批（zhupi）是一个**多站点、可嵌入的自托管划线批注服务**。你在自己的服务器上运行一个实例，就可以为博客、文档站、知识库或任意 Web 页面提供「选中文字 → 划线 → 批注与回复」的能力。

它解决的是「评论系统只能对整篇文章发言」的问题：读者可以精确地选中一段话，把批注挂在文字上，而不是挂在页面底部。

## 主要能力

| 能力 | 说明 |
| --- | --- |
| 多站点隔离 | 一个实例托管多个站点；公开 `siteKey` + Origin 白名单；批注、评论、管理员、审核队列按站点隔离 |
| 可嵌入 SDK | Shadow DOM UI，不污染宿主样式；支持 `<script>` 与 ESM；支持 SPA 路由切换 |
| 登录 | 全局 GitHub OAuth、每站点独立 OIDC、本地超级管理员兜底；首次登录自动建号，可手动绑定多身份 |
| 审核治理 | 超管 / 站点管理员两级权限；站点级「先发后审 / 先审后发」；批量审核、封禁、隐藏、恢复 |
| 账号中心 | 身份绑定、登录设备管理、退出全部设备；主题与语言切换 |
| 自托管 | 单 Go 二进制 + SQLite（WAL）+ 内嵌 SDK 与管理后台；Docker 一键部署 |
| 开放 API | 公共批注/评论 API、账号 API、管理 API、健康检查；统一错误结构与请求 ID |

## 组成

```text
宿主页面
  └── zhupi SDK（Shadow DOM）
        │  X-Site-Key + Origin + Authorization: Bearer
        ▼
Go HTTP API ──► service ──► store ──► SQLite（WAL）
  │
  ├── GitHub OAuth / 站点级 OIDC
  ├── GET /sdk/v1/*   内嵌 SDK 静态资源
  └── GET /admin/*    内嵌管理后台（SPA）
```

- **服务端**：Go 标准库 `net/http`，单二进制部署。
- **存储**：SQLite 单文件，开启 WAL、`busy_timeout` 与外键约束。
- **SDK**：TypeScript + Shadow DOM，构建为 ESM 与 IIFE 两种产物，直接嵌入 Go 二进制。
- **管理后台**：Vue 3 + Element Plus，构建后同样嵌入二进制，挂在 `/admin/`。

## 适合谁

- 有博客 / 文档站，想要「划线批注」而不是普通评论区。
- 有多个站点，希望用一套服务集中管理登录、审核与用户封禁。
- 需要自托管、数据留在自己服务器上的团队。
- 需要给内部知识库、课程页面、产品手册加批注能力的开发者。

## 它不做什么

- **不提供托管服务**：你必须自己部署。项目不包含 SaaS 计费、多租户隔离之外的运营能力。
- **不是通用评论系统**：评论必须挂在批注上，不提供文章级独立评论区。
- **不兼容旧单站点 API**：V1 不迁移旧数据、不保留旧接口。
- **不承诺多实例**：当前存储层面向单实例 SQLite；PostgreSQL / 多实例留给后续版本。

完整的 V1 边界见 [V1 边界与路线](/reference/boundaries/)。

## 许可

服务端、SDK 与管理后台均为 **MIT License**。SDK 同时发布在 npm：`zhupi-sdk`。

## 下一步

- 想马上跑起来：[快速开始](/getting-started/quick-start/)
- 想先理解概念：[核心概念](/getting-started/concepts/)
- 想接入自己的网站：[SDK 接入](/sdk/integration/)
- 想部署到服务器：[Docker 部署](/deployment/docker/)
