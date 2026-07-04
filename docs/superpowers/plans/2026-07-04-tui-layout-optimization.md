# TUI 布局优化 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 优化 Starman TUI 布局：sidebar 比例可配置、列宽弹性扩展、footer 去重、tabs 视觉区分

**Architecture:** 改动集中在 `internal/tui/` 层和 `internal/config/` 默认值，不涉及业务包。最小化改动面，利用现有 Column 结构体加 Flex 字段驱动列宽扩展。

**Tech Stack:** Go 1.22+, lipgloss/v2, bubbletea/v2, modernc.org/sqlite

## Global Constraints

- Go 格式化使用 `go fmt`
- 错误使用 `fmt.Errorf("context: %w", err)` 包装
- 颜色只走 theme token，禁止硬编码 `lipgloss.Color`
- 运行 `make lint` 和 `go test ./...` 验证所有改动
- 禁止修改不相干代码和注释

---

### Task 1: Column 结构体增加 Flex 字段 + fitColumns 扩展逻辑

**Files:**
- Modify: `internal/tui/components/listviewport/listviewport.go:14-17` (Column struct)
- Modify: `internal/tui/components/listviewport/listviewport.go:246-277` (fitColumns)

**Interfaces:**
- Consumes: none
- Produces: `Column{Flex bool}` — 调用方在列定义中设置 `Flex: true` 标记弹性列；`fitColumns()` 自动在空间富余时扩展弹性列宽度

- [ ] **Step 1: 修改 Column 结构体**

在 `listviewport.go` 第 14-17 行，Column 结构体增加 `Flex` 字段：

```go
type Column struct {
	Title string
	Width int
	Flex  bool
}
```

- [ ] **Step 2: 修改 fitColumns 增加扩展逻辑**

在 `listviewport.go` 第 246-277 行，在现有 `fitColumns` 函数中，缩放逻辑之后（第 258 行 `return m.columns, totalW` 之前）、缩放逻辑的 `return fitted, used` 之后，增加扩展逻辑。

将第 256-258 行的早返回改为继续向下执行：

```go
func (m Model) fitColumns(availWidth int) ([]Column, int) {
	if availWidth <= 0 {
		return m.columns, 0
	}
	gap := len(m.columns) - 1
	prefixW := 2
	totalW := prefixW + gap
	for _, c := range m.columns {
		totalW += c.Width
	}
	if totalW <= availWidth {
		// 空间富余：先复制一份，再扩展弹性列
		fitted := make([]Column, len(m.columns))
		copy(fitted, m.columns)
		used := prefixW
		for i := range fitted {
			used += fitted[i].Width
			if i < len(m.columns)-1 {
				used++
			}
		}
		extra := availWidth - used
		if extra > 0 {
			for i := range fitted {
				if m.columns[i].Flex {
					fitted[i].Width += extra
					used += extra
					break
				}
			}
		}
		return fitted, used
	}
	scale := float64(availWidth-prefixW-gap) / float64(totalW-prefixW-gap)
	if scale < 0.3 {
		scale = 0.3
	}
	fitted := make([]Column, len(m.columns))
	used := prefixW
	for i, c := range m.columns {
		w := int(float64(c.Width) * scale)
		if w < 3 {
			w = 3
		}
		fitted[i] = Column{Title: c.Title, Width: w, Flex: c.Flex}
		used += w
		if i < len(m.columns)-1 {
			used++
		}
	}
	return fitted, used
}
```

- [ ] **Step 3: 运行测试验证**

```bash
go test ./internal/tui/components/listviewport/ -v
```

- [ ] **Step 4: Commit**

```bash
git add internal/tui/components/listviewport/listviewport.go
git commit -m "feat: add Flex field to Column, expand flex columns when space available"
```

---

### Task 2: 各 section 列定义增加 Flex: true

**Files:**
- Modify: `internal/tui/components/starssection/starssection.go:23-28`
- Modify: `internal/tui/components/trendingsection/trendingsection.go:26-31`
- Modify: `internal/tui/components/categoriessection/categoriessection.go:13-20`
- Modify: `internal/tui/components/releasessection/releasessection.go:18-23`

**Interfaces:**
- Consumes: `listviewport.Column{Flex bool}` from Task 1
- Produces: each section's defaultColumns now marks the primary variable-length column as Flex

- [ ] **Step 1: starssection 列定义**

```go
var defaultColumns = []listviewport.Column{
	{Title: "repo", Width: 40, Flex: true},
	{Title: "stars", Width: 7},
	{Title: "lang", Width: 12},
	{Title: "cat", Width: 12},
}
```

- [ ] **Step 2: trendingsection 列定义**

```go
var defaultColumns = []listviewport.Column{
	{Title: "repo", Width: 38, Flex: true},
	{Title: "stars", Width: 7},
	{Title: "lang", Width: 12},
	{Title: "desc", Width: 20},
}
```

- [ ] **Step 3: categoriessection 列定义**

```go
var defaultColumns = []listviewport.Column{
	{Title: "ID", Width: 16},
	{Title: "Name", Width: 14, Flex: true},
	{Title: "Keywords", Width: 22},
	{Title: "Repos", Width: 6},
	{Title: "Sort", Width: 5},
	{Title: "Type", Width: 8},
}
```

- [ ] **Step 4: releasessection 列定义**

```go
var defaultColumns = []listviewport.Column{
	{Title: "repo", Width: 35, Flex: true},
	{Title: "version", Width: 16},
	{Title: "date", Width: 12},
	{Title: "status", Width: 7},
}
```

- [ ] **Step 5: 运行测试验证**

```bash
go test ./internal/tui/components/starssection/ ./internal/tui/components/trendingsection/ ./internal/tui/components/categoriessection/ ./internal/tui/components/releasessection/ -v
```

- [ ] **Step 6: Commit**

```bash
git add internal/tui/components/starssection/starssection.go internal/tui/components/trendingsection/trendingsection.go internal/tui/components/categoriessection/categoriessection.go internal/tui/components/releasessection/releasessection.go
git commit -m "feat: mark flex columns in all section column definitions"
```

---

### Task 3: 修改 sidebar 宽度默认值

**Files:**
- Modify: `internal/config/config.go:104` (PreviewConfig.Width 默认值)

**Interfaces:**
- Consumes: none
- Produces: `PreviewConfig.Width` 默认值从 0.38 改为 0.42

- [ ] **Step 1: 修改默认值**

在 `config.go` 第 104 行，将 `Width: 0.38` 改为 `Width: 0.42`：

```go
TUI: TUIConfig{
	Preview: PreviewConfig{
		Open:     true,
		Position: "right",
		Width:    0.42,
		Height:   0.4,
	},
```

- [ ] **Step 2: 运行测试验证**

```bash
go test ./internal/config/ -v
```

- [ ] **Step 3: Commit**

```bash
git add internal/config/config.go
git commit -m "feat: change default sidebar width ratio from 0.38 to 0.42"
```

---

### Task 4: recalcLayout() 读取配置中的 sidebar 比例

**Files:**
- Modify: `internal/tui/ui.go:779-784` (recalcLayout 中 sidebar 宽度计算)

**Interfaces:**
- Consumes: `m.ctx.TUICfg.Preview.Width` (float64) from config
- Produces: sidebar 宽度由配置驱动，不再硬编码

- [ ] **Step 1: 修改 recalcLayout**

在 `ui.go` 第 779-784 行，将硬编码的 `0.38` 改为读取配置：

```go
case "right":
	ratio := 0.42
	if m.ctx.TUICfg != nil && m.ctx.TUICfg.Preview.Width > 0 {
		ratio = m.ctx.TUICfg.Preview.Width
	}
	sidebarWidth := max(28, int(float64(w)*ratio))
	m.ctx.DynamicPreviewWidth = sidebarWidth
	m.ctx.DynamicPreviewHeight = mainHeight
	m.ctx.MainContentWidth = w - sidebarWidth
	m.sidebar.SetSize(m.ctx.DynamicPreviewWidth, mainHeight)
```

- [ ] **Step 2: 运行测试验证**

```bash
go test ./internal/tui/ -v -run TestRecalc
```

- [ ] **Step 3: Commit**

```bash
git add internal/tui/ui.go
git commit -m "feat: read sidebar width ratio from config instead of hardcoding"
```

---

### Task 5: Tabs 视图行与 section 行间加分割线

**Files:**
- Modify: `internal/tui/constants/constants.go:4` (TabsHeight)
- Modify: `internal/tui/components/tabs/tabs.go:93-115` (View 方法)
- Modify: `internal/tui/components/tabs/tabs.go:117-132` (renderSectionTabs)

**Interfaces:**
- Consumes: none
- Produces: tabs.View() 在视图行和 section 行之间输出一条 `─` 分割线；TabsHeight 从 2 改为 3

- [ ] **Step 1: 修改 TabsHeight 常量**

在 `constants/constants.go` 第 4 行：

```go
const (
	TabsHeight         = 3
	FooterHeight       = 1
	TableHeaderHeight  = 2
	SidebarPagerHeight = 1
)
```

- [ ] **Step 2: 修改 tabs.View() 插入分割线**

在 `tabs.go` 第 93-115 行，在 viewRow 和 sectionRow 之间插入分割线：

```go
func (m Model) View() string {
	theme := m.ctx.Theme
	faintStyle := lipgloss.NewStyle().Foreground(theme.FaintText)
	activeStyle := lipgloss.NewStyle().Foreground(theme.PrimaryText).Bold(true)
	bgStyle := lipgloss.NewStyle().Background(theme.SelectedBackground).Width(m.ctx.ScreenWidth)
	separatorStyle := lipgloss.NewStyle().Foreground(theme.FaintBorder).Width(m.ctx.ScreenWidth)

	var titleParts []string
	for i, t := range m.titles {
		if i == m.active {
			titleParts = append(titleParts, activeStyle.Render(t))
		} else {
			titleParts = append(titleParts, faintStyle.Render(t))
		}
	}
	viewRow := bgStyle.Render(strings.Join(titleParts, " | "))

	if len(m.sectionTabs) == 0 {
		return viewRow
	}

	sectionRow := m.renderSectionTabs()
	separator := separatorStyle.Render(strings.Repeat("─", m.ctx.ScreenWidth))
	return lipgloss.JoinVertical(lipgloss.Top, viewRow, separator, sectionRow)
}
```

- [ ] **Step 3: 运行测试验证**

```bash
go test ./internal/tui/components/tabs/ -v
```

- [ ] **Step 4: Commit**

```bash
git add internal/tui/constants/constants.go internal/tui/components/tabs/tabs.go
git commit -m "feat: add separator line between view tabs and section tabs"
```

---

### Task 6: Footer 重构 — 去掉 viewSwitcher，帮助文案常显左侧

**Files:**
- Modify: `internal/tui/components/footer/footer.go:34-139` (View 方法及 ViewSwitcherAtX)

**Interfaces:**
- Consumes: none
- Produces: footer 左侧显示快捷键帮助文案，右侧显示 task 状态+分页；移除 ViewSwitcherAtX 方法

- [ ] **Step 1: 重写 footer.View()**

完全替换 `footer/footer.go` 的第 34-109 行：

```go
func (m Model) View() string {
	theme := m.ctx.Theme
	bgStyle := lipgloss.NewStyle().
		Background(theme.SelectedBackground).
		Width(m.ctx.ScreenWidth)
	faintStyle := lipgloss.NewStyle().
		Foreground(theme.FaintText).
		Background(theme.SelectedBackground)
	successStyle := lipgloss.NewStyle().
		Foreground(theme.SuccessText).
		Background(theme.SelectedBackground)
	errorStyle := lipgloss.NewStyle().
		Foreground(theme.ErrorText).
		Background(theme.SelectedBackground)

	helpText := "j/k move  g/G first/last  h/l tab  p sidebar  / search  : cmd  q quit"
	helpLeft := lipgloss.NewStyle().
		Foreground(theme.FaintText).
		Background(theme.SelectedBackground).
		Render(helpText)

	var rightParts []string

	if m.task != nil {
		switch m.task.Status {
		case 0:
			frame := spinnerFrames[m.task.SpinnerIdx%len(spinnerFrames)]
			rightParts = append(rightParts, frame+" "+m.task.Message)
		case 1:
			rightParts = append(rightParts, successStyle.Render("✅ "+m.task.Message))
		case 2:
			msg := m.task.Message
			if m.task.Err != nil {
				msg = m.task.Message + ": " + m.task.Err.Error()
			}
			rightParts = append(rightParts, errorStyle.Render("❌ "+msg))
		}
	}

	if m.pager != "" {
		rightParts = append(rightParts, m.pager)
	}

	rightStr := strings.Join(rightParts, "  ")

	spacerWidth := m.ctx.ScreenWidth - lipgloss.Width(helpLeft) - lipgloss.Width(rightStr)
	if spacerWidth < 1 {
		spacerWidth = 1
	}

	return bgStyle.Render(lipgloss.JoinHorizontal(
		lipgloss.Top,
		helpLeft,
		lipgloss.NewStyle().
			Background(theme.SelectedBackground).
			Width(spacerWidth).
			Render(""),
		lipgloss.NewStyle().Background(theme.SelectedBackground).Render(rightStr),
	))
}
```

- [ ] **Step 2: 移除 ViewSwitcherAtX 方法**

删除 `footer/footer.go` 的第 128-139 行（ViewSwitcherAtX 方法和 getViewKey 函数），以及不再需要的 `views` 相关变量和 `activeStyle`/`inactiveStyle`。

- [ ] **Step 3: 运行测试验证**

```bash
go test ./internal/tui/components/footer/ -v
```

- [ ] **Step 4: Commit**

```bash
git add internal/tui/components/footer/footer.go
git commit -m "feat: replace footer viewSwitcher with persistent help text"
```

---

### Task 7: 顶部 tabs 加图标

**Files:**
- Modify: `internal/tui/ui.go:81` (SetTitles)

**Interfaces:**
- Consumes: none
- Produces: tab titles 带图标，与 footer 原图标一致

- [ ] **Step 1: 修改 SetTitles 调用**

在 `ui.go` 第 81 行：

```go
tabModel.SetTitles([]string{"⭐ Stars", "📂 Categories", "📈 Trending", "📦 Releases", "📊 Stats"})
```

- [ ] **Step 2: 运行测试验证**

```bash
go test ./internal/tui/ -v -run TestTabs
```

- [ ] **Step 3: Commit**

```bash
git add internal/tui/ui.go
git commit -m "feat: add icons to top tab titles"
```

---

### Task 8: 移除 showHelp toggle overlay

**Files:**
- Modify: `internal/tui/ui.go:65` (showHelp 字段)
- Modify: `internal/tui/ui.go:374-375` (? 键 toggle)
- Modify: `internal/tui/ui.go:468-471` (:help 命令 toggle)
- Modify: `internal/tui/ui.go:864-872` (helpLine 渲染)
- Modify: `internal/tui/ui.go:883-885` (extraLines 中的 showHelp)
- Modify: `internal/tui/commands_test.go:78-91` (showHelp 相关测试)

**Interfaces:**
- Consumes: from Task 6 (help now in footer)
- Produces: showHelp 状态和渲染代码全部移除

- [ ] **Step 1: 删除 showHelp 字段**

在 `ui.go` 第 65 行，删除 `showHelp bool`：

```go
// 之前
showHelp    bool

// 之后
// (删除此行)
```

- [ ] **Step 2: 删除 ? 键 toggle 处理**

在 `ui.go` 第 374-375 行，删除：

```go
// 删除这两行：
// case key.Matches(typed, m.ctx.Keys.Help):
//     m.showHelp = !m.showHelp
```

- [ ] **Step 3: 删除 :help 命令 toggle 处理**

在 `ui.go` 第 468-471 行，删除：

```go
// 删除这三行：
// if cmd == "help" {
//     m.showHelp = !m.showHelp
//     return nil
// }
```

- [ ] **Step 4: 删除 helpLine 渲染代码**

在 `ui.go` 第 864-872 行，删除 helpLine 变量及其赋值：

```go
// 删除这整段：
// helpLine := ""
// if m.showHelp {
//     helpText := "j/k move  g/G first/last  h/l prev/next tab  "
//     if m.tabs.HasSectionTabs() {
//         helpText += "[ / ] prev/next section  "
//     }
//     helpText += "p sidebar  m actions  / search  : cmd  Tab view  ? help  q quit"
//     helpLine = "\n" + common.RenderPreviewHeader(theme, m.ctx.ScreenWidth, helpText)
// }
```

- [ ] **Step 5: 删除 extraLines 中的 showHelp 逻辑**

在 `ui.go` 第 883-885 行，删除：

```go
// 删除这三行：
// if m.showHelp {
//     extraLines++
// }
```

- [ ] **Step 6: 更新 View() 中的 contentOutput**

在 `ui.go` 第 893 行附近，`contentOutput` 拼接去掉 `helpLine`：

```go
// 之前：
contentOutput := adjustedMainArea + searchLine + m.renderErrorBar() + helpLine

// 之后：
contentOutput := adjustedMainArea + searchLine + m.renderErrorBar()
```

- [ ] **Step 7: 更新 commands_test.go**

在 `commands_test.go` 第 78-91 行，删除 `TestHandleCommandMode_HelpTogglesShowHelp` 函数或修改为验证 help 输出抽屉行为。由于 help 不再 toggle，移除该测试函数：

```go
// 删除第 78-91 行的整个 TestHandleCommandMode_HelpTogglesShowHelp 函数
```

- [ ] **Step 8: 运行测试验证**

```bash
go test ./internal/tui/ -v
```

- [ ] **Step 9: Commit**

```bash
git add internal/tui/ui.go internal/tui/commands_test.go
git commit -m "feat: remove showHelp toggle overlay, help text now always in footer"
```

---

### Task 9: 更新鼠标点击处理

**Files:**
- Modify: `internal/tui/ui.go:396` (section tab 点击 y 坐标)
- Modify: `internal/tui/ui.go:411-416` (footer viewSwitcher 点击)

**Interfaces:**
- Consumes: Task 5 (TabsHeight 变为 3，section tabs 在 y=2), Task 6 (remove ViewSwitcherAtX)
- Produces: 鼠标点击坐标适配新布局

- [ ] **Step 1: 更新 section tab 点击 y 坐标**

在 `ui.go` 第 396 行，因分割线占用了 y=1 行，section tabs 变为 y=2：

```go
// 之前：
if y == 1 && m.tabs.HasSectionTabs() {

// 之后：
if y == 2 && m.tabs.HasSectionTabs() {
```

- [ ] **Step 2: 移除 footer 视图切换器鼠标点击处理**

在 `ui.go` 第 411-416 行，删除 footer 鼠标点击切换视图的逻辑：

```go
// 删除这整段：
// if y == m.ctx.ScreenHeight-1 {
//     if viewIdx := m.footer.ViewSwitcherAtX(msg.X); viewIdx >= 0 {
//         m.switchView(viewIdx - m.tabs.Active())
//     }
//     return nil
// }
```

- [ ] **Step 3: 运行测试验证**

```bash
go test ./internal/tui/ -v
```

- [ ] **Step 4: Commit**

```bash
git add internal/tui/ui.go
git commit -m "fix: update mouse click coordinates for new layout"
```

---

### Task 10: 全量验证

- [ ] **Step 1: 运行全部单元测试**

```bash
go test ./... -count=1
```

确认所有测试通过。

- [ ] **Step 2: 运行 lint**

```bash
make lint
```

确认无 lint 错误。

- [ ] **Step 3: 运行构建**

```bash
make build
```

确认成功编译。

- [ ] **Step 4: 最终 Commit（如有遗漏文件）**

```bash
git status
git add -A
git commit -m "chore: final verification after TUI layout optimization"
```
