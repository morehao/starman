# TUI 布局优化设计

日期：2026-07-04
状态：已确认

## 目标

优化 Starman TUI 的布局，提升空间利用率和信息密度：
1. 左右栏比例可配置，默认更平衡（38% → 42%）
2. 左栏列宽弹性扩展，充分利用可用空间
3. Footer 去重：移除视图切换器，帮助文案常显
4. 顶部视图 tabs 与 section tabs 视觉区分

---

## Section 1：Sidebar 比例可配置

### 现状

`recalcLayout()` 中硬编码 `sidebarWidth := max(28, int(float64(w)*0.38))`，无法通过配置调整。

80宽终端：sidebar=30，左栏=50
120宽终端：sidebar=45，左栏=75

### 改动

1. **`config/config.go`**：默认值 `PreviewConfig.Width` 从 0.38 → 0.42
2. **`internal/tui/ui.go` `recalcLayout()`**：读取 `m.ctx.TUICfg.Preview.Width` 替代硬编码 `0.38`

```go
// 之前
sidebarWidth := max(28, int(float64(w)*0.38))

// 之后
ratio := m.ctx.TUICfg.Preview.Width
if ratio <= 0 {
    ratio = 0.42
}
sidebarWidth := max(28, int(float64(w)*ratio))
```

### 效果

| 终端宽度 | 改前 sidebar | 改后 sidebar |
|----------|-------------|-------------|
| 80       | 30          | 33          |
| 100      | 38          | 42          |
| 120      | 45          | 50          |

---

## Section 2：列宽弹性扩展

### 现状

`listviewport.fitColumns()` 只在空间不足时等比缩放列宽，空间富余时不扩展，列保持硬编码默认值（repo=40, stars=7, lang=12, cat=12），导致宽屏下右半截空白。

### 改动

**Column 结构体新增 `Flex bool` 字段**（`listviewport/listviewport.go`）：

```go
type Column struct {
    Title string
    Width int
    Flex  bool  // true = 弹性列，占满剩余空间
}
```

**`fitColumns()` 新增扩展逻辑**（在缩放逻辑之后）：

```go
// 空间富余时扩展弹性列
if used < availWidth {
    extra := availWidth - used
    for i := range fitted {
        if m.columns[i].Flex {
            fitted[i].Width += extra
            used += extra
            break
        }
    }
}
```

**各 section 列定义示例**（`starssection/starssection.go`）：

```go
var defaultColumns = []listviewport.Column{
    {Title: "repo", Width: 40, Flex: true},
    {Title: "stars", Width: 7},
    {Title: "lang", Width: 12},
    {Title: "cat", Width: 12},
}
```

### 效果

80宽终端，sidebar=33，左栏可用≈47：repo 列从 40 自动扩展到 ~47
120宽终端，sidebar=50，左栏可用≈70：repo 列从 40 自动扩展到 ~63

---

## Section 3：Footer 重构

### 现状

Footer 水平三段布局：左侧 viewSwitcher（⭐ Stars | 📂 Categories | ...）+ 中间 spacer + 右侧 task+pager+?help。

问题：viewSwitcher 与顶部 tabs 重复占用空间；帮助文案通过 `?` 键 toggle overlay 显示，不常显。

### 改动

**3.1 顶部 tabs 加图标**（`ui.go`）：

```go
// 之前
tabModel.SetTitles([]string{"Stars", "Categories", "Trending", "Releases", "Stats"})

// 之后
tabModel.SetTitles([]string{"⭐ Stars", "📂 Categories", "📈 Trending", "📦 Releases", "📊 Stats"})
```

**3.2 Footer 布局调整**（`footer/footer.go`）：

左侧改为常显帮助文案，右侧保持 task + 分页：

```
j/k move  g/G first/last  h/l tab  p sidebar  / search  : cmd  q quit    [spacer]    ⠏ syncing...  12/50
```

- 移除 `viewSwitcher` 相关代码（views 数组、viewParts 拼接、ViewSwitcherAtX()）
- 帮助文案构造为 footer 左侧内容
- 移除右侧 `?help` 提示文字
- 窄屏（<80）时帮助文案截断

**3.3 移除 `?` toggle overlay**（`ui.go`）：

- 删除 `showHelp bool` 字段
- 删除 `?` 键的 toggle 处理（第 374-375、468-469 行）
- 删除 `helpLine` 渲染代码（第 864-872 行）
- 删除 `extraLines` 中的 `showHelp` 相关逻辑（第 883-885 行）

### 效果

Footer 空间利用率提升，帮助信息始终可见，无需按键切换。

---

## Section 4：Tabs 视图行与 Section 行视觉区分

### 现状

`tabs.View()` 直接 `JoinVertical(viewRow, sectionRow)` 拼接，无视觉分隔。视图行有背景色，section 行无背景，但毗邻时区分不明显。

### 改动

**4.1 添加水平分割线**（`tabs/tabs.go` `View()`）：

在两行之间插入一条 `─` 分割线，颜色使用 `theme.FaintBorder`：

```go
separator := lipgloss.NewStyle().
    Foreground(theme.FaintBorder).
    Width(m.ctx.ScreenWidth).
    Render(strings.Repeat("─", m.ctx.ScreenWidth))

return lipgloss.JoinVertical(lipgloss.Top, viewRow, separator, sectionRow)
```

**4.2 样式区分**：

- 视图行：保持 `SelectedBackground` + Bold
- section 行：`renderSectionTabs()` 中选中项用 `Background(theme.SelectedBackground)`，非选中项保持 `FaintText`

### 效果

```
⭐ Stars | 📂 Categories | 📈 Trending          ← 粗体 + 深色背景
───────────────────────────────────────────────  ← 分割线（浅色）
 🔍 Search | All | Pinned                       ← 选中项高亮背景
```

---

## 涉及文件

| 文件 | 改动 |
|------|------|
| `internal/config/config.go` | `PreviewConfig.Width` 默认值 0.38 → 0.42 |
| `internal/tui/ui.go` | `recalcLayout()` 读配置比例；tab titles 加图标；移除 `showHelp` 相关代码 |
| `internal/tui/components/footer/footer.go` | 移除 viewSwitcher，帮助文案常显左侧 |
| `internal/tui/components/listviewport/listviewport.go` | Column 加 Flex 字段，`fitColumns()` 扩展逻辑 |
| `internal/tui/components/starssection/starssection.go` | 列定义加 Flex: true |
| `internal/tui/components/*section/`（其他 section） | 各 section 列定义加 Flex: true |
| `internal/tui/components/tabs/tabs.go` | 视图行与 section 行间加分割线 |
| `internal/tui/components/tabs/tabs_test.go` | 更新 titles 期望值（加图标）；分割线渲染验证 |
| `internal/tui/commands_test.go` | 移除 `showHelp` 相关断言 |

## 未涉及

- 右栏纵向划分（用户明确取消）
- 响应式断点逻辑不变
- Stats 视图（无 sidebar 视图，不受影响）
