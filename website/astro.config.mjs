// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

// 文档站生产地址；构建时可用 SITE_URL 覆盖（例如预览环境）。
const siteUrl = process.env.SITE_URL ?? 'https://zhupi.pages.dev';

export default defineConfig({
	site: siteUrl,
	integrations: [
		starlight({
			title: '朱批 · zhupi',
			description:
				'多站点、可嵌入的划线批注服务官方文档：自托管部署、SDK 接入、登录与审核、API 与 CLI。',
			logo: {
				src: './src/assets/logo.svg',
				alt: '朱批 zhupi',
			},
			favicon: '/favicon.svg',
			locales: {
				root: { label: '简体中文', lang: 'zh-CN' },
			},
			defaultLocale: 'zh-CN',
			social: [
				{
					icon: 'github',
					label: 'GitHub',
					href: 'https://github.com/Bin-hy/zhupi',
				},
				{
					icon: 'npm',
					label: 'zhupi-sdk',
					href: 'https://www.npmjs.com/package/zhupi-sdk',
				},
			],
			editLink: {
				baseUrl: 'https://github.com/Bin-hy/zhupi/edit/main/website/',
			},
			lastUpdated: true,
			customCss: ['./src/styles/custom.css'],
			components: {
				Head: './src/components/Head.astro',
			},
			head: [
				{ tag: 'meta', attrs: { name: 'theme-color', content: '#c92a2a' } },
			],
			sidebar: [
				{ label: '在线体验', slug: 'playground' },
				{
					label: '开始使用',
					items: [
						{ label: '项目简介', slug: 'getting-started/overview' },
						{ label: '快速开始', slug: 'getting-started/quick-start' },
						{ label: '核心概念', slug: 'getting-started/concepts' },
					],
				},
				{
					label: '部署运维',
					items: [
						{ label: 'Docker 部署', slug: 'deployment/docker' },
						{ label: '源码构建与 systemd', slug: 'deployment/source' },
						{ label: '反向代理与 HTTPS', slug: 'deployment/reverse-proxy' },
						{ label: '环境变量', slug: 'deployment/configuration' },
						{ label: '数据与备份', slug: 'deployment/backup' },
					],
				},
				{
					label: 'SDK 接入',
					items: [
						{ label: '接入指南', slug: 'sdk/integration' },
						{ label: '初始化选项', slug: 'sdk/options' },
						{ label: '方法与事件', slug: 'sdk/methods-events' },
						{ label: '界面、主题与国际化', slug: 'sdk/ui-theme-i18n' },
					],
				},
				{
					label: '站点与权限',
					items: [
						{ label: '站点管理', slug: 'sites/manage' },
						{ label: '登录与身份', slug: 'sites/authentication' },
						{ label: '角色、审核与封禁', slug: 'sites/roles-moderation' },
					],
				},
				{
					label: 'API 与 CLI',
					items: [
						{ label: 'API 概览', slug: 'api/overview' },
						{ label: '批注与评论 API', slug: 'api/annotations' },
						{ label: '管理与账号 API', slug: 'api/admin-account' },
						{ label: 'CLI 参考', slug: 'cli/reference' },
					],
				},
				{
					label: '开发',
					items: [
						{ label: '架构与数据模型', slug: 'development/architecture' },
						{ label: '本地开发与测试', slug: 'development/local' },
						{ label: '发布流程', slug: 'development/release' },
					],
				},
				{
					label: '参考',
					items: [
						{ label: '安全边界', slug: 'reference/security' },
						{ label: '常见问题', slug: 'reference/faq' },
						{ label: 'V1 边界与路线', slug: 'reference/boundaries' },
						{
							label: 'GitHub 仓库 ↗',
							link: 'https://github.com/Bin-hy/zhupi',
						},
					],
				},
			],
		}),
	],
});
