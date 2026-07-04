# Sidebar Tab 合并设计

> 将 Stars/Trending 视图 sidebar 从「三 tab 切换」合并为「单一滚动详情面板」

**版本：** v1.0
**日期：** 2026-07-04

## 1. 背景与动机

当前 Stars 视图的 sidebar（右侧预览面板）有三个 tab：

| Tab | 内容 | 问题 |
|---|---|---|
| Overview | 仓库基础信息 + AI 摘要 | — |
| README | AI 摘要（glamour 渲染） | 和 Overview 底部 AI Summary 重复 |
| Releases | 订阅状态（两行文字） | 信息量极少，单独占一个 tab 过度设计 |

用户需要按 `]]`/`[[` 在三个 tab 间切换，增加了认知和操作负担。README 和 Releases 的内容量不足以撑起独立 tab，合并为一个连续滚动面板更简洁。

Trending 视图使用相同的 `repoview` 组件，自动受益于此改动。

## 2. 设计方案

### 2.1 重命名 `repoview` → `repodetail`

- 路径：`internal/tui/components/repoview/` → `internal/tui/components/repodetail/`
- 原因：不再有 tab 切换，`repoview` 名不副实

### 2.2 删除的内容

- `activeTab` 字段、`tabTitles` 切片
- `NextTab()`、`PrevTab()`、`ActiveTab()` 方法
- `renderTabs()` — top tab 栏渲染
- `renderReadme()` — 独立的 README tab 渲染
- `renderReleases()` — 独立的 Releases tab 渲染

### 2.3 合并后的渲染结构

`renderDetail()` 统一输出，从上到下：

```
仓库名称              ⭐xxx  🍴xxx  Language
────────────────────────────────────────
Description（如有）
Language:       xxx
Category:       xxx  （含 🔒 锁定标记）
Platform:       cli, web, ...
Starred:        2025-01-15
Topics:         topic1, topic2
Tags:           tag1, tag2, tag3

▶ AI Summary
（glamour 渲染的 AI 摘要，宽度 = width - 4，最低 20）

────────────────────────────────────────
Release: Subscribed（或 Not subscribed）
  Last fetched: 2026-01-01（如有）
```

关键点：
- AI 摘要使用 glamour 渲染（继承原 README tab 的渲染质量）
- Release 状态缩为一行，放在底部
- 空值字段不渲染，避免空白行

### 2.4 `ui.go` 变更

- import 路径 `repoview` → `repodetail`
- `repo` 字段类型 `*repoview.Model` → `*repodetail.Model`
- `PrevSection`/`NextSection` 键位处理中，删除 `m.repo.PrevTab()`/`m.repo.NextTab()` 调用
  - Stars 视图中 `PrevSection`/`NextSection` 不再调用 `m.repo.*` 方法
  - Stats 视图保留 `m.stats.PrevTab()`/`m.stats.NextTab()`
- `syncSidebar()` 中 `m.repo.View()` 调用不变

### 2.5 不受影响的部分

- Stars 顶部 section tabs（All/Language/Category/Tag）— 控制主表格分组，与 sidebar 无关
- Trending 视图 — 共享同一个 `repodetail`，自动受益
- sidebar 容器本身 — 只负责渲染和滚动，不感知内容变化
- Stats 视图的 `PrevTab()`/`NextTab()` — Stats 有独立的 week/month/overall tab

## 3. 影响范围

| 文件 | 操作 |
|---|---|
| `internal/tui/components/repodetail/repodetail.go` | 新建 |
| `internal/tui/components/repodetail/repodetail_test.go` | 新建 |
| `internal/tui/components/repoview/repoview.go` | 删除 |
| `internal/tui/components/repoview/repoview_test.go` | 删除 |
| `internal/tui/ui.go` | 修改 import + tab 切换逻辑 + 类型引用 |
