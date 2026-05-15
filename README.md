# ZQ Desktop App

基于 Wails + Vue 3 + TypeScript 的桌面内容管理工具，当前主要覆盖以下能力：

- 站点与租户配置管理
- 租户登录态与本地桌面存储
- 文章、工具、草稿箱与本地采集工作流
- Go 桌面桥接与前端 Wails 调用

## 项目结构

- `main.go` / `app.go`: Wails 应用入口与桌面桥接注册
- `desktop_store.go`: 本地 SQLite 存储，包括站点、租户、登录态、草稿
- `web_collector.go`: 网页采集与正文提取逻辑
- `frontend/src`: Vue 前端源码，实际可维护源码以 `.ts` 和 `.vue` 为准
- `frontend/wailsjs`: Wails 生成的前端桥接代码
- `开发文档.md`: 当前中文开发说明

## 常用命令

在项目根目录执行：

```bash
wails dev
```

前端单独开发：

```bash
cd frontend
npm install
npm run build
```

桌面应用构建：

```bash
wails build
```

## 维护约定

- `frontend/src` 不保留编译产物；源码以 `.ts` / `.vue` 文件为准
- `frontend/dist`、`build/bin`、`*.exe`、`*.db` 属于本地产物，不纳入源码维护
- 一次性调试抓取文件不要放在项目根目录，避免和真实源码混在一起
- 若需新增调试脚本，优先放到独立 `scripts` 或 `tools` 目录，并注明用途

## 当前已清理

- 删除微信页面抓取调试快照 `wechat_raw_debug.html`
- 删除一次性调试脚本 `wechat_fetch_debug.go`
- 删除 `frontend/src` 下与 TypeScript 源码重复的 `.js` / `.js.map` 文件
- 保留构建与运行所需的正式源码、文档与 Wails 生成桥接目录
