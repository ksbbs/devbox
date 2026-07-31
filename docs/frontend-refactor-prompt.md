# DevBox 前端重构任务提示词

> 用法：新开会话后，把本文件全文粘贴给 opencode 作为第一条指令（可附上"请先读一遍仓库里的 docs/frontend-refactor-prompt.md 再开始"）。请全程使用中文交流。

## 角色与背景

你是 DevBox 项目的资深前端工程师。DevBox 是一个 Go 后端 + Vue 3 前端的**镜像代理控制台**：代理 npm / pypi / docker / golang / conda 等 17 个镜像源，提供 GitHub 搜索、Release 下载中转、Git clone 加速，带一个暗色极客风的 Web Dashboard。项目仓库根目录为 `E:\code\devbox`（Windows，包管理用 pnpm）。

当前任务：在**保持全部现有功能**的前提下，对前端做一次完整的 UI/UX 重构，让界面更精致、统一、有高级感。

## 技术栈与硬性约束

- Vue 3.5（`<script setup>` + TS）+ Vue Router 4 + Vite 8 + Tailwind CSS 4
- **无 UI 组件库、无 icon 库、无动画库** —— 全部手写 SVG / Tailwind 原子类
- 代码位置：`web/src/views/*.vue`（7 个页面）、`web/src/components/StatusCard.vue`、`web/src/style.css`、`web/src/api/client.ts`、`web/src/router/index.ts`、`web/src/App.vue`
- **不要改动后端 API 契约**；前端参数名须与后端 JSON tag 对齐（例如搜索接口是 `has_more`、`per_page`，不是 camelCase）
- 不新增任何 npm 依赖
- 构建验证：在 `web/` 目录下 `pnpm run build`（含 vue-tsc 类型检查）

## 现有设计资产（必读）

1. `web/public/favicon.svg` —— 新的品牌图标：等轴测紫渐变立方体 + 青色终端提示符 `>_`。重构时把它的视觉语言延伸到整个 UI
2. 品牌色系：紫 `#7c3aed / #9455f5 / #6528d4 / #bd8bff / #f2e4ff`，青色强调 `#67e8f9`
3. 根目录 `view.png` —— 旧版 Dashboard 全貌截图（只作**信息架构**参考，不作视觉参考）
4. 现有风格特征：`#05070a` 深色背景、slate 灰文字层级、cyan 强调、等宽字体（ui-monospace）、细边框（border-slate-800 类）、渐变网格背景

## 重构范围

| 文件                        | 内容                                               | 优先级 |
| --------------------------- | -------------------------------------------------- | ------ |
| `App.vue`                   | 导航布局、header logo（用新 SVG 立方体）、页面过渡 | 高     |
| `views/Dashboard.vue`       | 状态卡片、流量趋势图（自绘 sparkline）、访问日志   | 高     |
| `views/Mirrors.vue`         | 17 镜像启停 / upstream / cache TTL 配置            | 高     |
| `views/Search.vue`          | 7 源聚合搜索 + 安装命令生成/复制                   | 中     |
| `views/Releases.vue`        | GitHub Release 源持久化 + 中转下载                 | 中     |
| `views/Settings.vue`        | 运行信息、限流配置（rate/interval/白黑名单）       | 中     |
| `views/GitProxy.vue`        | git clone/archive/raw 命令生成与复制               | 低     |
| `views/Login.vue`           | 登录（token 输入）                                 | 低     |
| `components/StatusCard.vue` | 仪表盘统计卡                                       | 低     |

## 设计方向（必须遵守）

1. **延续暗色终端极客风**，但提升精致度：统一卡片、面板、表格、表单的视觉语言（spacing 刻度、圆角、边框、hover/active/disabled 状态一致）
2. **品牌色系统**：紫为主色、青为强调色、slate 做文字层级；交互元素（链接、激活导航、主按钮）统一用 cyan/紫渐变，保证 WCAG AA 对比度
3. **导航**：顶部 sticky header；移动端（375px）导航可用；logo 用新 SVG 立方体 + "devbox" 字标
4. **表格**：统一 `data-table` 样式（表头小号大写、行分隔、状态 tag、空态/加载态）
5. **微交互**：保留 App.vue 现有页面切换 `<transition>`；卡片/按钮 hover 反馈统一；列表重排/状态切换可加与主题一致的过渡（遵循流体动效原则，克制不过度）
6. **组件化**：跨页复用的视觉模式（Panel 卡片、Tag、StatusDot、EmptyState、CopyButton 等）提取到 `web/src/components/`，不要每个页面重复内联
7. **功能语义不变**：保留每个页面的全部功能与现有文案语义，不删减、不篡改行为
8. **渐进式**：先做 App.vue + Dashboard（确立设计系统），与用户确认视觉方向后再铺开其余页面，不要一次性全部改完再让用户看

## 推荐的设计 tokens（起点，可优化）

- 背景：`#05070a` 保持，面板 `slate-950/60` + `border-slate-800`
- 强调：主操作 `cyan-400` 系渐变，品牌点缀紫 `#7c3aed` 系渐变
- 圆角：卡片 10px（rounded-lg 偏大），标签/按钮 6px
- 文字：标题 slate-100，正文 slate-300，次要 slate-500，标签 uppercase tracking-widest text-[10px]（沿用现有 feel）

## 工作流程

1. 先读代码：全部 views、components、style.css、api/client.ts、router/index.ts、index.html；用 `web/../view.png`（或 agent-vision 分析）了解信息架构
2. 拟定设计系统（色板/间距/圆角/组件清单）摘要，**先向用户确认方向**，再动手写代码
3. 逐页重构；每个页面完成后 `pnpm run build` 验证类型与构建
4. 全部完成后启动本地 dev（`nohup pnpm run dev > /tmp/dev.log 2>&1 &`，后台进程），用 Edge headless 截图 + agent-vision 逐页检查视觉效果并修复；完成后停掉后台进程
5. 最终 `pnpm run build` 通过后：新建分支 `refactor/frontend-redesign`，提交，推送，创建 PR（PR 描述写明设计变更点）；PR 创建后等用户审查，**不要自行合并**
6. 若涉及 README 功能描述变化，同步更新 README.md

## 验收标准

- [ ] `pnpm run build` 通过（vue-tsc 无类型错误）
- [ ] 登录、Dashboard（状态/流量/日志）、Mirrors（启停/改 upstream/TTL）、Search（搜索/复制命令）、Releases（源管理/下载）、Settings（限流保存）全部可用
- [ ] 视觉统一、无样式冲突；375px 移动端导航可用
- [ ] 后端 API 契约未变（若前端字段名与后端不一致，以后端为准：`has_more`、`per_page` 等）
- [ ] 无新增依赖；无 console 报错

## 环境注意事项（重要）

- 本机安装了 opencode-vibeguard 插件：**读取文件时若看到 `__VG_IPV4_xxx__`、`__VG_EMAIL_xxx__` 等占位符是插件的显示层脱敏，磁盘文件是完好的**，不要据此"修复"文件
- 用 Edit 工具写 YAML/TS 文件可能被插件意外重排引号风格；若发现整个文件 diff 异常，改用 python 脚本写文件
- 本机 docker 命令不在 git-bash PATH 中；涉及 Docker 的操作在 PowerShell 中执行
