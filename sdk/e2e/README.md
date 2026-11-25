# SDK 端到端测试

这些脚本使用 `playwright-core` 驱动本机 Chrome，不随包下载浏览器。
运行前需要：

1. 启动批注服务，并创建一个允许目标来源的站点；
2. 启动示例页静态服务器或 VitePress 开发服务器；
3. 设置环境变量。

## 环境变量

| 变量 | 说明 |
| --- | --- |
| `SITE_KEY` | 必填，站点 siteKey |
| `ANNOTATION_API` | 服务地址，默认 `http://127.0.0.1:18090` |
| `PAGE` | 被测页面地址；plain 默认自动拼接 |
| `SITE_ORIGIN` | 页面来源，默认 `http://127.0.0.1:15174` |
| `CHROME_PATH` | Chrome/Chromium 可执行文件路径，默认 macOS 的 Google Chrome |

## 命令

```bash
# 纯 HTML 示例：登录、划线、发布、刷新恢复
SITE_KEY=xxx CHROME_PATH=/path/to/chrome npm run e2e:plain

# VitePress 页面
SITE_KEY=xxx PAGE=http://127.0.0.1:15174/某篇文章 CHROME_PATH=/path/to/chrome npm run e2e:vitepress

# 账号、会话撤销、移动端、键盘焦点
SITE_KEY=xxx PAGE=http://127.0.0.1:15174/某篇文章 CHROME_PATH=/path/to/chrome npm run e2e:account

# orphan 重定位与筛选
SITE_KEY=xxx CHROME_PATH=/path/to/chrome npm run e2e:relocate

# 评论面板：服务端搜索、排序、回复与草稿
SITE_KEY=xxx CHROME_PATH=/path/to/chrome npm run e2e:comments
```

> 脚本会创建临时用户与批注，建议使用独立的测试数据库。
