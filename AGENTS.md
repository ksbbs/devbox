# DevBox 开发指南

DevBox 是 Go 后端 + Vue 3 前端的镜像代理控制台：代理 npm / pypi / docker / golang / conda 等 17 个镜像源，提供 GitHub 搜索、Release 下载中转、Git clone 加速，带暗色液态玻璃风格的 Web 控制台。

## 环境

- 平台：Windows 11；包管理统一用 `pnpm`（不用 npm）
- 前端技术栈：Vue 3.5（`<script setup>` + TS）+ Vue Router 4 + Vite 8 + Tailwind CSS 4
- **无 UI 组件库、无 icon 库、无动画库** —— 全部手写 SVG / Tailwind 原子类，禁止随意新增 npm 依赖

## 常用命令

```bash
# 前端构建验证（含 vue-tsc 类型检查，改完前端必跑）
cd web && pnpm run build

# 前端本地开发
cd web && pnpm run dev          # http://localhost:5173，/api 代理到 :8080

# 后端本地启动（前端产物需先构建）
go run ./cmd/devbox/ -c configs/devbox.yaml -f web/dist   # http://localhost:8080

# 后端测试
go test ./...
```

## 前端约定

- **设计系统**（`web/src/style.css`）：
  - 背景 `#05070a` + 网格渐变；玻璃卡片用 `glass` / `glass-hover` / `glass-inset`（Tailwind 4 `@utility` 定义，禁止改为普通类后再 `@apply`）
  - 品牌色：紫 `#7c3aed` 系（logo、激活导航、渐变 kicker、sparkline 渐变），青 `cyan-400`（主操作、链接、焦点），语义色 emerald/amber/red
  - 按钮 `btn` / `btn-primary` / `btn-danger` / `btn-brand`；表格用 `table-wrap` + `data-table`；标签 `tag` 系列
- **UI 文案一律使用中文**（按钮、表头、面板标题、状态标签、提示语），不混用英文；命令代码块内容保持原样
- **响应式 grid 必须声明移动端列**：任何带 `sm:/md:/lg:/xl:` 列声明的 grid 都要补 `grid-cols-1`，否则单列 auto 轨道会被代码块/表格的 nowrap 内容撑宽导致移动端横向溢出（历史教训）
- **API 契约以 Go 后端 JSON tag 为准**：如 `has_more`、`per_page`、`bytes_out`、`created_at`，前端不得改成 camelCase
- 通用组件位于 `web/src/components/`：`Panel`（卡片容器）、`CodeBlock`（命令代码块，自带复制）、`CopyButton`、`StatusDot`、`EmptyState`、`Banner`、`Logo`；跨页复用优先用这些组件，不要内联重复
- 新页面沿用：`page-header` + `page-kicker`（渐变紫青）+ `page-title` + `page-subtitle` 结构
- 页面切换过渡在 `App.vue`（`<transition name="page">`），新增交互过渡沿用同一风格（克制不过度）

## 后端约定

- 入口 `cmd/devbox/`，配置 `configs/devbox.yaml`（mirrors / gitproxy / cache / rate_limit 等），SQLite 数据库与缓存默认在 `/data`
- 鉴权：设置 `server.auth_token` 后需登录；`AUTH_TOKEN` 即登录密码，鉴权仅影响控制台，镜像代理与 Git 代理不受影响
- 主要 API：`/status`、`/stats/traffic`、`/stats/logs`、`/config/mirrors`、`/config/ratelimit`、`/config/public`、`/search`、`/release-sources`、`/release-download`

## 开发工作流

- 新功能按分支开发：`refactor/xxx` / `feature/xxx`，验证通过后提 PR，待用户审查后再合并（合并后清理分支）
- 前端视觉验证：`pnpm run dev` + 后端 `go run` 后，用 Edge headless 截图检查（`--headless --screenshot`，Windows 下注意 DPI 缩放会导致 window-size 失效，可用 CDP `Emulation.setDeviceMetricsOverride` 精确模拟 375px）
- 功能变化需同步更新 `README.md`（项目功能清单、Release 下载说明等在 README 有专门章节）
