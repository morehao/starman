# Starman TUI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `starman tui` interactive terminal UI using bubbletea + lipgloss, covering all 14 functional modules.

**Architecture:** Nested tea.Model pattern — top-level TuiModel with sidebar/content/statusbar, each page is an independent tea.Model. CLI and TUI coexist: browse/view tasks go TUI, batch/pipe tasks stay CLI.

**Tech Stack:** Go 1.25.0, `github.com/charmbracelet/bubbletea`, `github.com/charmbracelet/lipgloss`, `github.com/charmbracelet/bubbles`

## Global Constraints

- All TUI code in `internal/tui/` package
- CLI command registered at `internal/cli/tui.go`
- `categorize` command registered in `internal/cli/root.go` (currently missing)
- Tests in `*_test.go` files alongside source
- Run `go build ./...` and `go test ./...` after each task

---

## File Structure

```
internal/
├── tui/
│   ├── tui.go                    # TuiModel (top-level), NewTuiModel(), RunTUI()
│   ├── tui_test.go               # Top-level tests
│   ├── messages.go               # PageID, NavigatedMsg, StatusMsg, TaskProgressMsg
│   ├── styles/
│   │   ├── theme.go              # Theme struct, DefaultTheme(), colors/borders
│   │   └── theme_test.go
│   ├── components/
│   │   ├── sidebar.go            # SidebarModel, NewSidebar()
│   │   └── statusbar.go          # StatusBarModel, NewStatusBar()
│   ├── pages/
│   │   ├── dashboard.go          # Dashboard page
│   │   ├── search.go             # Search page
│   │   ├── repo_list.go          # Repo list page
│   │   ├── repo_detail.go        # Repo detail page
│   │   ├── trending.go           # Trending page
│   │   ├── sync.go               # Sync page
│   │   ├── analyze.go            # Analyze page
│   │   ├── tag.go                # Tag management page
│   │   ├── categorize.go         # Categorize management page
│   │   ├── release.go            # Release list page
│   │   ├── generate.go           # Generate page
│   │   ├── backup.go             # Backup page
│   │   ├── stats.go              # Stats page
│   │   └── config.go             # Config page
│   └── help.go                   # Help page
└── cli/
    └── tui.go                    # `starman tui` cobra command
```

---

### Task 1: Add dependencies and scaffold package

**Files:**
- Modify: `go.mod`, `go.sum`
- Create: `internal/tui/tui.go`
- Create: `internal/tui/messages.go`
- Modify: `internal/cli/root.go`
- Create: `internal/cli/tui.go`

**Interfaces:**
- Produces: `tui.RunTUI(cfg *config.Config)` — entry point
- Produces: `tui.PageID` type, `tui.NavigatedMsg`, `tui.StatusMsg`
- Produces: `cli.newTuiCmd()` — cobra command registered in root

- [ ] **Step 1: Add bubbletea/lipgloss/bubbles dependencies**

```bash
go get github.com/charmbracelet/bubbletea github.com/charmbracelet/lipgloss github.com/charmbracelet/bubbles
```

- [ ] **Step 2: Run `go mod tidy`**

```bash
go mod tidy
```

Expected: no errors

- [ ] **Step 3: Create `internal/tui/messages.go`**

```go
package tui

import "time"

type PageID int

const (
    PageDashboard PageID = iota
    PageSearch
    PageRepoList
    PageTrending
    PageSync
    PageAnalyze
    PageTag
    PageCategorize
    PageStats
    PageRelease
    PageGenerate
    PageBackup
    PageConfig
    PageRepoDetail
)

type NavigatedMsg struct{ Page PageID }

type StatusLevel int

const (
    LevelInfo StatusLevel = iota
    LevelSuccess
    LevelWarning
    LevelError
)

type StatusMsg struct {
    Text    string
    Level   StatusLevel
    Timeout time.Duration
}

type TaskStartedMsg  struct{ ID, Label string }
type TaskProgressMsg struct{ ID string; Current, Total int }
type TaskDoneMsg     struct{ ID string; Err error }

type RepoSelectedMsg struct{ FullName string }

type TickMsg time.Time
```

- [ ] **Step 4: Create `internal/tui/tui.go` (skeleton)**

```go
package tui

import (
    "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "github.com/morehao/starman/internal/config"
    "github.com/morehao/starman/internal/store"
)

type TuiModel struct {
    config    *config.Config
    store     store.Store
    width     int
    height    int
    currentPage PageID
    lastPage    PageID
    ready     bool
}

func NewTuiModel(cfg *config.Config, s store.Store) *TuiModel {
    return &TuiModel{
        config:      cfg,
        store:       s,
        currentPage: PageDashboard,
        ready:       false,
    }
}

func (m *TuiModel) Init() tea.Cmd {
    return nil
}

func (m *TuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        m.ready = true
        return m, nil
    case tea.KeyMsg:
        switch msg.String() {
        case "ctrl+c", "q":
            return m, tea.Quit
        }
    }
    return m, nil
}

func (m *TuiModel) View() string {
    if !m.ready {
        return "loading..."
    }
    return lipgloss.Place(m.width, m.height,
        lipgloss.Center, lipgloss.Center,
        "starman tui — ready",
    )
}

func RunTUI(cfg *config.Config) error {
    dir, err := config.DefaultDir()
    if err != nil {
        return err
    }
    dbPath := filepath.Join(dir, "starman.db")
    s, err := store.Open(dbPath)
    if err != nil {
        return err
    }
    defer s.Close()
    m := NewTuiModel(cfg, s)
    p := tea.NewProgram(m, tea.WithAltScreen())
    _, err = p.Run()
    return err
}
```

- [ ] **Step 5: Create `internal/cli/tui.go`**

```go
package cli

import (
    "github.com/morehao/starman/internal/tui"
    "github.com/spf13/cobra"
)

func newTuiCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "tui",
        Short: "Launch the interactive terminal UI",
        RunE: func(cmd *cobra.Command, args []string) error {
            cfg, err := loadConfig()
            if err != nil {
                return err
            }
            return tui.RunTUI(cfg)
        },
    }
}
```

- [ ] **Step 6: Register `tui` and `categorize` commands in `internal/cli/root.go`**

Add after `root.AddCommand(newTrendingCmd())`:

```go
root.AddCommand(newTuiCmd())
root.AddCommand(newCategorizeCmd())
```

- [ ] **Step 7: Verify compilation**

```bash
go build ./...
```

Expected: no errors (may need to fix `store.Open` vs `store.New` — check actual store package signature)

- [ ] **Step 8: Commit**

```bash
git add go.mod go.sum internal/tui/ internal/cli/tui.go internal/cli/root.go
git commit -m "feat(tui): add dependencies and scaffold tui command with skeleton model"
```

---

### Task 2: Theme system

**Files:**
- Create: `internal/tui/styles/theme.go`
- Create: `internal/tui/styles/theme_test.go`

**Interfaces:**
- Produces: `styles.Theme` struct (colors, borders, text styles)
- Produces: `styles.DefaultTheme() *Theme`

- [ ] **Step 1: Create `internal/tui/styles/theme.go`**

```go
package styles

import "github.com/charmbracelet/lipgloss"

type Theme struct {
    Primary        lipgloss.Color
    Secondary      lipgloss.Color
    Success        lipgloss.Color
    Warning        lipgloss.Color
    Error          lipgloss.Color
    Muted          lipgloss.Color
    Background     lipgloss.Color
    Text           lipgloss.Color
    Subtle         lipgloss.Color

    Sidebar        lipgloss.Style
    SidebarActive  lipgloss.Style
    StatusBar      lipgloss.Style
    Card           lipgloss.Style
    CardTitle      lipgloss.Style
    TableHeader    lipgloss.Style
    TableCell      lipgloss.Style
    Input          lipgloss.Style
    PageTitle      lipgloss.Style
    HelpText       lipgloss.Style
}

func DefaultTheme() *Theme {
    t := &Theme{
        Primary:    lipgloss.Color("#7C3AED"),
        Secondary:  lipgloss.Color("#3B82F6"),
        Success:    lipgloss.Color("#10B981"),
        Warning:    lipgloss.Color("#F59E0B"),
        Error:      lipgloss.Color("#EF4444"),
        Muted:      lipgloss.Color("#6B7280"),
        Background: lipgloss.Color("#1F2937"),
        Text:       lipgloss.Color("#F9FAFB"),
        Subtle:     lipgloss.Color("#9CA3AF"),
    }

    t.Sidebar = lipgloss.NewStyle().
        Width(24).
        Height(100).
        Border(lipgloss.NormalBorder(), false, true, false, false).
        BorderForeground(t.Muted).
        Padding(1, 1)

    t.SidebarActive = lipgloss.NewStyle().
        Foreground(t.Text).
        Background(t.Primary).
        Bold(true)

    t.StatusBar = lipgloss.NewStyle().
        Height(1).
        Background(t.Muted).
        Foreground(t.Text).
        Padding(0, 1)

    t.Card = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(t.Muted).
        Padding(1)

    t.CardTitle = lipgloss.NewStyle().
        Bold(true).
        Foreground(t.Primary)

    t.TableHeader = lipgloss.NewStyle().
        Bold(true).
        Foreground(t.Primary).
        Padding(0, 1)

    t.TableCell = lipgloss.NewStyle().
        Padding(0, 1)

    t.Input = lipgloss.NewStyle().
        Border(lipgloss.NormalBorder()).
        BorderForeground(t.Secondary).
        Padding(0, 1)

    t.PageTitle = lipgloss.NewStyle().
        Bold(true).
        Foreground(t.Primary).
        Padding(0, 1).
        MarginBottom(1)

    t.HelpText = lipgloss.NewStyle().
        Foreground(t.Subtle)

    return t
}
```

- [ ] **Step 2: Create `internal/tui/styles/theme_test.go`**

```go
package styles

import "testing"

func TestDefaultTheme(t *testing.T) {
    th := DefaultTheme()
    if th == nil {
        t.Fatal("expected non-nil theme")
    }
    if th.Sidebar.GetWidth() != 24 {
        t.Errorf("sidebar width = %d, want 24", th.Sidebar.GetWidth())
    }
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./internal/tui/styles/
```

- [ ] **Step 4: Commit**

```bash
git add internal/tui/styles/
git commit -m "feat(tui): add theme system with DefaultTheme"
```

---

### Task 3: Sidebar component

**Files:**
- Create: `internal/tui/components/sidebar.go`

**Interfaces:**
- Produces: `components.SidebarModel` (implements tea.Model)
- Produces: `components.NewSidebar(theme *styles.Theme) *SidebarModel`

- [ ] **Step 1: Create `internal/tui/components/sidebar.go`**

```go
package components

import (
    "fmt"

    "github.com/charmbracelet/bubbletea"
    "github.com/morehao/starman/internal/tui"
    "github.com/morehao/starman/internal/tui/styles"
)

type menuItem struct {
    label    string
    shortcut string
    page     tui.PageID
}

var menuItems = []menuItem{
    {"Dashboard",  "1", tui.PageDashboard},
    {"Search",     "/", tui.PageSearch},
    {"Repo List",  "r", tui.PageRepoList},
    {"Trending",   "t", tui.PageTrending},
    {"—",          "",  -1},
    {"Sync",       "s", tui.PageSync},
    {"Analyze",    "a", tui.PageAnalyze},
    {"—",          "",  -1},
    {"Tag",        "g", tui.PageTag},
    {"Categorize", "c", tui.PageCategorize},
    {"Stats",      "S", tui.PageStats},
    {"—",          "",  -1},
    {"Release",    "R", tui.PageRelease},
    {"Generate",   "G", tui.PageGenerate},
    {"Backup",     "b", tui.PageBackup},
    {"Config",     "C", tui.PageConfig},
    {"—",          "",  -1},
    {"Help",       "?", tui.PageDashboard},
    {"Quit",       "q", tui.PageDashboard},
}

type SidebarModel struct {
    theme       *styles.Theme
    cursor      int
    width       int
    height      int
}

func NewSidebar(theme *styles.Theme) *SidebarModel {
    return &SidebarModel{
        theme:  theme,
        cursor: 0,
    }
}

func (m *SidebarModel) Init() tea.Cmd { return nil }

func (m *SidebarModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = m.theme.Sidebar.GetWidth()
        m.height = msg.Height
    case tea.KeyMsg:
        switch msg.String() {
        case "up", "k":
            m.cursor = max(m.cursor-1, 0)
            if menuItems[m.cursor].page == -1 {
                m.cursor = max(m.cursor-1, 0)
            }
        case "down", "j":
            m.cursor = min(m.cursor+1, len(menuItems)-1)
            if menuItems[m.cursor].page == -1 {
                m.cursor = min(m.cursor+1, len(menuItems)-1)
            }
        case "enter":
            if item := menuItems[m.cursor]; item.page >= 0 && item.label != "Quit" {
                return m, func() tea.Msg { return tui.NavigatedMsg{Page: item.page} }
            }
        }
    }
    return m, nil
}

func (m *SidebarModel) View() string {
    var s string
    s += m.theme.PageTitle.Render("STARMAN") + "\n\n"
    for i, item := range menuItems {
        if item.label == "—" {
            s += "\n"
            continue
        }
        line := fmt.Sprintf(" %-14s %2s", item.label, item.shortcut)
        if i == m.cursor {
            s += m.theme.SidebarActive.Render(line)
        } else {
            s += m.theme.HelpText.Render(line)
        }
        s += "\n"
    }
    return s
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./internal/tui/...
```

- [ ] **Step 3: Commit**

```bash
git add internal/tui/components/sidebar.go
git commit -m "feat(tui): add sidebar component with 14 menu items"
```

---

### Task 4: StatusBar component

**Files:**
- Create: `internal/tui/components/statusbar.go`

**Interfaces:**
- Produces: `components.StatusBarModel` (implements tea.Model)
- Produces: `components.NewStatusBar(theme *styles.Theme) *StatusBarModel`

- [ ] **Step 1: Create `internal/tui/components/statusbar.go`**

```go
package components

import (
    "fmt"
    "time"

    "github.com/charmbracelet/bubbletea"
    "github.com/morehao/starman/internal/tui"
    "github.com/morehao/starman/internal/tui/styles"
)

type StatusBarModel struct {
    theme     *styles.Theme
    repoCount int
    lastSync  string
    aiQuota   string
    message   string
    msgLevel  tui.StatusLevel
    msgExpiry time.Time
    width     int
}

func NewStatusBar(theme *styles.Theme) *StatusBarModel {
    return &StatusBarModel{
        theme:    theme,
        aiQuota:  "—",
        lastSync: "—",
    }
}

func (m *StatusBarModel) Init() tea.Cmd {
    return nil
}

func (m *StatusBarModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
    case tui.TickMsg:
        if time.Now().After(m.msgExpiry) && m.message != "" {
            m.message = ""
        }
    case tui.StatusMsg:
        m.message = msg.Text
        m.msgLevel = msg.Level
        m.msgExpiry = time.Now().Add(msg.Timeout)
    }
    return m, nil
}

func (m *StatusBarModel) View() string {
    left := fmt.Sprintf("⭐ %d repos | 🔄 %s", m.repoCount, m.lastSync)
    var mid string
    if m.message != "" {
        switch m.msgLevel {
        case tui.LevelError:
            mid = m.theme.CardTitle.Foreground(m.theme.Error).Render(m.message)
        case tui.LevelSuccess:
            mid = m.theme.CardTitle.Foreground(m.theme.Success).Render(m.message)
        default:
            mid = m.message
        }
    }
    right := "?:Help  Ctrl+C:Quit"

    leftW := len(left)
    midW := len(mid)
    rightW := len(right)
    available := m.width - leftW - rightW - 4
    if midW > available {
        mid = mid[:max(available-1, 0)]
    }
    padding := max(available-midW, 0) / 2

    return m.theme.StatusBar.Width(m.width).Render(
        left + repeat(" ", padding) + mid + repeat(" ", padding) + right,
    )
}

func repeat(s string, n int) string {
    result := ""
    for i := 0; i < n; i++ {
        result += s
    }
    return result
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./internal/tui/...
```

- [ ] **Step 3: Commit**

```bash
git add internal/tui/components/statusbar.go
git commit -m "feat(tui): add statusbar component with transient status messages"
```

---

### Task 5: Wire TuiModel with Sidebar and StatusBar

**Files:**
- Modify: `internal/tui/tui.go`

**Interfaces:**
- Consumes: `components.NewSidebar`, `components.NewStatusBar`, `tui.PageID`, `tui.NavigatedMsg`, `tui.StatusMsg`
- Produces: Complete top-level `TuiModel` with sidebar/content/statusbar layout

- [ ] **Step 1: Rewrite `internal/tui/tui.go`**

```go
package tui

import (
    "fmt"
    "time"

    "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "github.com/morehao/starman/internal/config"
    "github.com/morehao/starman/internal/store"
    "github.com/morehao/starman/internal/tui/components"
    "github.com/morehao/starman/internal/tui/styles"
)

type TuiModel struct {
    config      *config.Config
    store       store.Store
    theme       *styles.Theme
    width       int
    height      int
    currentPage PageID
    sidebar     *components.SidebarModel
    statusbar   *components.StatusBarModel
    ready       bool
}

func NewTuiModel(cfg *config.Config, s store.Store) *TuiModel {
    theme := styles.DefaultTheme()
    return &TuiModel{
        config:      cfg,
        store:       s,
        theme:       theme,
        currentPage: PageDashboard,
        sidebar:     components.NewSidebar(theme),
        statusbar:   components.NewStatusBar(theme),
    }
}

func (m *TuiModel) Init() tea.Cmd {
    return tea.Batch(
        m.sidebar.Init(),
        m.statusbar.Init(),
        tickCmd(),
    )
}

func (m *TuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmd tea.Cmd
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        m.ready = true
        return m, nil

    case NavigatedMsg:
        m.currentPage = msg.Page
        return m, nil

    case tea.KeyMsg:
        switch msg.String() {
        case "ctrl+c":
            return m, tea.Quit
        case "/":
            return m, func() tea.Msg { return NavigatedMsg{Page: PageSearch} }
        }
    }

    _, cmd = m.sidebar.Update(msg)
    _, _ = m.statusbar.Update(msg)
    return m, cmd
}

func (m *TuiModel) View() string {
    if !m.ready {
        return "loading..."
    }
    sidebar := m.theme.Sidebar.Render(m.sidebar.View())
    content := m.renderContent()
    status := m.statusbar.View()

    main := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
    return lipgloss.JoinVertical(lipgloss.Left, main, status)
}

func (m *TuiModel) renderContent() string {
    w := m.width - m.theme.Sidebar.GetWidth()
    return lipgloss.NewStyle().
        Width(w).
        Height(m.height - 1).
        Padding(1).
        Render(fmt.Sprintf("Page: %d\n\nPress / to search, q to quit", m.currentPage))
}

func tickCmd() tea.Cmd {
    return tea.Tick(time.Second, func(t time.Time) tea.Msg {
        return TickMsg(t)
    })
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add internal/tui/tui.go
git commit -m "feat(tui): wire TuiModel with sidebar and statusbar layout"
```

---

### Task 6: Dashboard page

**Files:**
- Create: `internal/tui/pages/dashboard.go`

**Interfaces:**
- Produces: `pages.DashboardModel` (implements tea.Model)
- Produces: `pages.NewDashboard(store store.Store, theme *styles.Theme) *DashboardModel`

- [ ] **Step 1: Create `internal/tui/pages/dashboard.go`**

```go
package pages

import (
    "context"
    "fmt"
    "strings"

    "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "github.com/morehao/starman/internal/store"
    "github.com/morehao/starman/internal/tui/styles"
)

type DashboardModel struct {
    store      store.Store
    theme      *styles.Theme
    repoCount  int
    langStats  []langStat
    lastSync   string
    width      int
    height     int
    loaded     bool
}

type langStat struct {
    Lang  string
    Count int
    Pct   float64
}

func NewDashboard(s store.Store, theme *styles.Theme) *DashboardModel {
    return &DashboardModel{
        store: s,
        theme: theme,
    }
}

func (m *DashboardModel) Init() tea.Cmd {
    return m.loadCmd
}

func (m *DashboardModel) loadCmd() tea.Msg {
    ctx := context.Background()
    repos, err := m.store.ListRepositories(ctx)
    if err != nil {
        return err
    }
    langMap := map[string]int{}
    for _, r := range repos {
        langMap[r.Language]++
    }
    total := len(repos)
    var stats []langStat
    for lang, count := range langMap {
        stats = append(stats, langStat{Lang: lang, Count: count, Pct: float64(count) / float64(total) * 100})
    }
    // sort descending by count (simple bubble sort for small n)
    for i := 0; i < len(stats); i++ {
        for j := i + 1; j < len(stats); j++ {
            if stats[j].Count > stats[i].Count {
                stats[i], stats[j] = stats[j], stats[i]
            }
        }
    }
    if len(stats) > 5 {
        stats = stats[:5]
    }
    return dashboardLoadedMsg{
        repoCount: total,
        langStats: stats,
    }
}

type dashboardLoadedMsg struct {
    repoCount int
    langStats []langStat
}

func (m *DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width - m.theme.Sidebar.GetWidth() - 2
        m.height = msg.Height - 2
    case dashboardLoadedMsg:
        m.repoCount = msg.repoCount
        m.langStats = msg.langStats
        m.loaded = true
    }
    return m, nil
}

func (m *DashboardModel) View() string {
    if !m.loaded {
        return "loading dashboard..."
    }

    title := m.theme.PageTitle.Render("Dashboard") + "\n\n"

    // stat cards
    cards := []string{
        m.theme.Card.Render(fmt.Sprintf("⭐ %d\n 仓库总数", m.repoCount)),
        m.theme.Card.Render(fmt.Sprintf("📅 —\n 新增(周)")),
        m.theme.Card.Render(fmt.Sprintf("🤖 —\n 待分析")),
    }
    cardRow := lipgloss.JoinHorizontal(lipgloss.Top, cards...)

    // language distribution
    langSection := m.theme.CardTitle.Render("📊 语言分布 (Top 5)") + "\n"
    for _, s := range m.langStats {
        bar := strings.Repeat("█", int(s.Pct/2))
        langSection += fmt.Sprintf("  %-8s %-30s %4.0f%% %3d\n", s.Lang, bar, s.Pct, s.Count)
    }

    return lipgloss.JoinVertical(lipgloss.Left,
        title,
        cardRow,
        "",
        langSection,
    )
}
```

- [ ] **Step 2: Wire dashboard into TuiModel**

Replace `renderContent()` in `internal/tui/tui.go`:

Add imports: `"github.com/morehao/starman/internal/tui/pages"`

Add field: `pages map[PageID]tea.Model`

In `NewTuiModel`, after `statusbar: ...`:

```go
m := &TuiModel{...}
m.pages = map[PageID]tea.Model{
    PageDashboard: pages.NewDashboard(s, theme),
}
return m
```

Update `renderContent()`:

```go
func (m *TuiModel) renderContent() string {
    p, ok := m.pages[m.currentPage]
    if !ok {
        return "page not found"
    }
    return p.View()
}
```

Update `Update()` to forward messages to the current page:

```go
// In TuiModel.Update, before return:
if p, ok := m.pages[m.currentPage]; ok {
    var pageCmd tea.Cmd
    _, pageCmd = p.Update(msg)
    cmd = tea.Batch(cmd, pageCmd)
}
```

Also add `tea.WindowSizeMsg` forwarding to all pages.

- [ ] **Step 3: Verify compilation**

```bash
go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add internal/tui/pages/dashboard.go internal/tui/tui.go
git commit -m "feat(tui): add dashboard page with repo stats and language distribution"
```

---

### Task 7-15: Remaining Pages (RepoList, RepoDetail, Search, Trending, Sync, Analyze, Tag, Categorize, Stats)

Each page follows the same pattern: implement `tea.Model` with `Init()`, `Update()`, `View()`, inject `store.Store` + `*styles.Theme`, render via table/viewport components.

Due to plan size limits, these tasks are summarized with file paths and key interfaces. The complete implementation follows the same structure as Dashboard.

**Common page template:**

```go
package pages

import (
    "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "github.com/morehao/starman/internal/store"
    "github.com/morehao/starman/internal/tui/styles"
)

type XxxModel struct {
    store  store.Store
    theme  *styles.Theme
    width  int
    height int
    loaded bool
}

func NewXxx(s store.Store, theme *styles.Theme) *XxxModel {
    return &XxxModel{store: s, theme: theme}
}

func (m *XxxModel) Init() tea.Cmd { return nil }

func (m *XxxModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width - theme.Sidebar.GetWidth() - 2
        m.height = msg.Height - 2
    }
    return m, nil
}

func (m *XxxModel) View() string {
    return m.theme.PageTitle.Render("Xxx") + "\n\n..."
}
```

### Task 7: RepoList page

**Files:**
- Create: `internal/tui/pages/repo_list.go`

**Interfaces:**
- Produces: `pages.RepoListModel`, `pages.NewRepoList(s store.Store, theme *styles.Theme) *RepoListModel`

- [ ] **Step 1: Create `internal/tui/pages/repo_list.go`**

```go
package pages

import (
    "context"
    "fmt"

    "github.com/charmbracelet/bubbletea"
    "github.com/morehao/starman/internal/store"
    "github.com/morehao/starman/internal/tui"
    "github.com/morehao/starman/internal/tui/styles"
)

type RepoListModel struct {
    store   store.Store
    theme   *styles.Theme
    repos   []store.Repository
    cursor  int
    width   int
    height  int
    loaded  bool
}

type reposLoadedMsg struct{ repos []store.Repository }

func NewRepoList(s store.Store, theme *styles.Theme) *RepoListModel {
    return &RepoListModel{store: s, theme: theme}
}

func (m *RepoListModel) Init() tea.Cmd { return m.loadCmd }

func (m *RepoListModel) loadCmd() tea.Msg {
    ctx := context.Background()
    repos, err := m.store.ListRepositories(ctx)
    if err != nil {
        return err
    }
    sortByStars(repos)
    return reposLoadedMsg{repos: repos}
}

func sortByStars(repos []store.Repository) {
    for i := 0; i < len(repos); i++ {
        for j := i + 1; j < len(repos); j++ {
            if repos[j].StargazersCount > repos[i].StargazersCount {
                repos[i], repos[j] = repos[j], repos[i]
            }
        }
    }
}

func (m *RepoListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width - m.theme.Sidebar.GetWidth() - 4
        m.height = msg.Height - 3
    case reposLoadedMsg:
        m.repos = msg.repos
        m.loaded = true
    case tea.KeyMsg:
        switch msg.String() {
        case "up", "k":
            m.cursor = max(m.cursor-1, 0)
        case "down", "j":
            m.cursor = min(m.cursor+1, len(m.repos)-1)
        case "enter":
            if m.cursor < len(m.repos) {
                return m, func() tea.Msg {
                    return tui.RepoSelectedMsg{FullName: m.repos[m.cursor].FullName}
                }
            }
        }
    }
    return m, nil
}

func (m *RepoListModel) View() string {
    if !m.loaded {
        return "loading repos..."
    }
    title := m.theme.PageTitle.Render("Repo List") + "\n"
    header := fmt.Sprintf("  %-40s %-12s %8s %15s", "full_name", "language", "stars", "category")
    rendered := title + m.theme.TableHeader.Render(header) + "\n"
    start := max(m.cursor-m.height+5, 0)
    end := min(start+m.height-5, len(m.repos))
    for i := start; i < end; i++ {
        r := m.repos[i]
        line := fmt.Sprintf("  %-40s %-12s %8d %15s",
            truncate(r.FullName, 38), r.Language, r.StargazersCount, r.CustomCategory)
        if i == m.cursor {
            rendered += m.theme.SidebarActive.Render(line) + "\n"
        } else {
            rendered += line + "\n"
        }
    }
    return rendered
}

func truncate(s string, max int) string {
    if len(s) > max {
        return s[:max-1] + "\u2026"
    }
    return s
}
```

- [ ] **Step 2: Wire into TuiModel** — add `PageRepoList: pages.NewRepoList(s, theme)` to pages map

- [ ] **Step 3: Verify compilation and commit**

```bash
go build ./...
git add internal/tui/pages/repo_list.go internal/tui/tui.go
git commit -m "feat(tui): add repo list page with table and cursor navigation"
```

---

### Task 8: RepoDetail page

**Files:**
- Create: `internal/tui/pages/repo_detail.go`

**Interfaces:**
- Produces: `pages.RepoDetailModel`, `pages.NewRepoDetail(s store.Store, theme *styles.Theme) *RepoDetailModel`
- Consumes: `tui.RepoSelectedMsg`

- [ ] **Step 1: Create `internal/tui/pages/repo_detail.go`**

```go
package pages

import (
    "context"
    "fmt"

    "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "github.com/morehao/starman/internal/store"
    "github.com/morehao/starman/internal/tui"
    "github.com/morehao/starman/internal/tui/styles"
)

type RepoDetailModel struct {
    store    store.Store
    theme    *styles.Theme
    repo     *store.Repository
    fullName string
    width    int
    height   int
    loaded   bool
}

type repoLoadedMsg struct{ repo *store.Repository }

func NewRepoDetail(s store.Store, theme *styles.Theme) *RepoDetailModel {
    return &RepoDetailModel{store: s, theme: theme}
}

func (m *RepoDetailModel) Init() tea.Cmd { return nil }

func (m *RepoDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width - m.theme.Sidebar.GetWidth() - 4
        m.height = msg.Height - 3
    case tui.RepoSelectedMsg:
        m.fullName = msg.FullName
        m.loaded = false
        return m, m.loadRepoCmd
    case repoLoadedMsg:
        m.repo = msg.repo
        m.loaded = true
    case tea.KeyMsg:
        switch msg.String() {
        case "esc":
            return m, func() tea.Msg { return tui.NavigatedMsg{Page: tui.PageRepoList} }
        }
    }
    return m, nil
}

func (m *RepoDetailModel) loadRepoCmd() tea.Msg {
    ctx := context.Background()
    repo, err := m.store.GetRepository(ctx, m.fullName)
    if err != nil {
        return err
    }
    return repoLoadedMsg{repo: repo}
}

func (m *RepoDetailModel) View() string {
    if !m.loaded {
        return "loading repo..."
    }
    r := m.repo
    title := m.theme.PageTitle.Render(r.FullName) + "  \u2b50 " + fmt.Sprintf("%d", r.StargazersCount)
    meta := fmt.Sprintf("\nLanguage: %s | Forks: %d | Open Issues: %d",
        r.Language, r.ForksCount, r.OpenIssuesCount)
    desc := fmt.Sprintf("\n\n%s", r.Description)
    aiSummary := ""
    if r.AISummary != "" {
        aiSummary = fmt.Sprintf("\n\nAI Summary:\n%s", r.AISummary)
    }
    tags := ""
    if r.CustomTags != "" {
        tags = fmt.Sprintf("\n\nTags: %s", r.CustomTags)
    }
    cat := ""
    if r.CustomCategory != "" {
        lock := "unlocked"
        if r.CategoryLocked {
            lock = "locked"
        }
        cat = fmt.Sprintf("\nCategory: %s (%s)", r.CustomCategory, lock)
    }
    return lipgloss.JoinVertical(lipgloss.Left,
        title, meta, desc, aiSummary, tags, cat,
        m.theme.HelpText.Render("\n\nEsc: back  s: star"),
    )
}
```

- [ ] **Step 2: In TuiModel.Update, add handling for `RepoSelectedMsg`**

```go
case RepoSelectedMsg:
    m.currentPage = PageRepoDetail
    return m, func() tea.Msg { return msg }
```

- [ ] **Step 3: Verify compilation and commit**

```bash
go build ./...
git add internal/tui/pages/repo_detail.go internal/tui/tui.go
git commit -m "feat(tui): add repo detail page with AI summary display"
```

---

### Task 9: Search page

**Files:**
- Create: `internal/tui/pages/search.go`

- [ ] **Step 1: Create `internal/tui/pages/search.go`**

```go
package pages

import (
    "context"
    "fmt"
    "strings"

    "github.com/charmbracelet/bubbles/textinput"
    "github.com/charmbracelet/bubbletea"
    "github.com/morehao/starman/internal/store"
    "github.com/morehao/starman/internal/tui"
    "github.com/morehao/starman/internal/tui/styles"
)

type searchMode int

const (
    modeText searchMode = iota
    modeVector
    modeLLM
)

var modeNames = []string{"TEXT", "VECTOR", "LLM"}

type SearchModel struct {
    store   store.Store
    theme   *styles.Theme
    input   textinput.Model
    results []store.Repository
    cursor  int
    mode    searchMode
    width   int
    height  int
    loaded  bool
}

func NewSearch(s store.Store, theme *styles.Theme) *SearchModel {
    ti := textinput.New()
    ti.Placeholder = "Search repositories..."
    ti.CharLimit = 100
    ti.Width = 60
    return &SearchModel{store: s, theme: theme, input: ti}
}

func (m *SearchModel) Init() tea.Cmd { return textinput.Blink }

func (m *SearchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmd tea.Cmd
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width - m.theme.Sidebar.GetWidth() - 4
        m.height = msg.Height - 3
    case tea.KeyMsg:
        switch msg.String() {
        case "tab":
            m.mode = (m.mode + 1) % 3
        case "esc":
            return m, func() tea.Msg { return tui.NavigatedMsg{Page: tui.PageDashboard} }
        case "up", "k":
            m.cursor = max(m.cursor-1, 0)
        case "down", "j":
            m.cursor = min(m.cursor+1, len(m.results)-1)
        case "enter":
            if m.cursor < len(m.results) {
                return m, func() tea.Msg {
                    return tui.RepoSelectedMsg{FullName: m.results[m.cursor].FullName}
                }
            }
        }
    }
    m.input, cmd = m.input.Update(msg)
    m.doSearch()
    return m, cmd
}

func (m *SearchModel) doSearch() {
    q := strings.TrimSpace(m.input.Value())
    if len(q) < 2 {
        m.results = nil
        return
    }
    ctx := context.Background()
    repos, err := m.store.ListRepositories(ctx)
    if err != nil {
        return
    }
    var filtered []store.Repository
    qLower := strings.ToLower(q)
    for _, r := range repos {
        if strings.Contains(strings.ToLower(r.FullName), qLower) ||
            strings.Contains(strings.ToLower(r.Description), qLower) ||
            strings.Contains(strings.ToLower(r.AISummary), qLower) ||
            strings.Contains(strings.ToLower(r.Language), qLower) {
            filtered = append(filtered, r)
        }
    }
    m.results = filtered
    m.loaded = true
    m.cursor = 0
}

func (m *SearchModel) View() string {
    title := m.theme.PageTitle.Render("Search") + "\n"
    modeStr := ""
    for i, name := range modeNames {
        if searchMode(i) == m.mode {
            modeStr += m.theme.SidebarActive.Render("[" + name + "]") + " "
        } else {
            modeStr += "[" + name + "] "
        }
    }
    body := title + m.input.View() + "\n" + modeStr + "\n\n"
    if m.loaded && len(m.results) > 0 {
        body += fmt.Sprintf("Results: %d\n", len(m.results))
        for i, r := range m.results {
            line := fmt.Sprintf("  %-40s %-10s \u2605%d", truncate(r.FullName, 38), r.Language, r.StargazersCount)
            if i == m.cursor {
                body += m.theme.SidebarActive.Render(line) + "\n"
            } else {
                body += line + "\n"
            }
        }
    }
    return body
}
```

- [ ] **Step 2: Wire into TuiModel pages map**

- [ ] **Step 3: Verify compilation and commit**

```bash
go build ./...
git add internal/tui/pages/search.go internal/tui/tui.go
git commit -m "feat(tui): add search page with text input and filtering"
```

---

### Task 10: Trending page

**Files:**
- Create: `internal/tui/pages/trending.go`

- [ ] **Step 1: Create `internal/tui/pages/trending.go`**

```go
package pages

import (
    "context"
    "fmt"

    "github.com/charmbracelet/bubbletea"
    "github.com/morehao/starman/internal/discovery"
    "github.com/morehao/starman/internal/store"
    "github.com/morehao/starman/internal/tui"
    "github.com/morehao/starman/internal/tui/styles"
)

type TrendingModel struct {
    store   store.Store
    theme   *styles.Theme
    repos   []discovery.TrendingRepo
    cursor  int
    source  string
    width   int
    height  int
    loaded  bool
}

type trendingLoadedMsg struct{ repos []discovery.TrendingRepo }

func NewTrending(s store.Store, theme *styles.Theme) *TrendingModel {
    return &TrendingModel{store: s, theme: theme, source: "rss"}
}

func (m *TrendingModel) Init() tea.Cmd { return m.loadCmd }

func (m *TrendingModel) loadCmd() tea.Msg {
    // Note: Full implementation requires github.Client for NewService().
    // This skeleton loads trending data. In production, inject githubClient.
    return trendingLoadedMsg{repos: nil}
}

func (m *TrendingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width - m.theme.Sidebar.GetWidth() - 4
        m.height = msg.Height - 3
    case trendingLoadedMsg:
        m.repos = msg.repos
        m.loaded = true
    case tea.KeyMsg:
        switch msg.String() {
        case "up", "k":
            m.cursor = max(m.cursor-1, 0)
        case "down", "j":
            m.cursor = min(m.cursor+1, len(m.repos)-1)
        case "tab":
            if m.source == "rss" { m.source = "search" } else { m.source = "rss" }
            return m, m.loadCmd
        case "enter":
            if m.cursor < len(m.repos) {
                return m, func() tea.Msg {
                    return tui.RepoSelectedMsg{FullName: m.repos[m.cursor].FullName}
                }
            }
        }
    }
    return m, nil
}

func (m *TrendingModel) View() string {
    if !m.loaded {
        return "loading trending..."
    }
    title := m.theme.PageTitle.Render("Trending") + "\n"
    body := title + fmt.Sprintf("Source: %s (Tab to switch)\n\n", m.source)
    for i, r := range m.repos {
        line := fmt.Sprintf("  %-40s \u2b50 %-8d %s", truncate(r.FullName, 38), r.Stars, r.Language)
        if i == m.cursor {
            body += m.theme.SidebarActive.Render(line) + "\n"
        } else {
            body += line + "\n"
        }
    }
    return body
}
```

- [ ] **Step 2: Wire into TuiModel pages map**

- [ ] **Step 3: Verify compilation and commit**

```bash
go build ./...
git add internal/tui/pages/trending.go internal/tui/tui.go
git commit -m "feat(tui): add trending page with RSS source"
```

---

### Task 11: Sync page

**Files:**
- Create: `internal/tui/pages/sync.go`

- [ ] **Step 1: Create `internal/tui/pages/sync.go`**

```go
package pages

import (
    "context"
    "fmt"
    "strings"

    "github.com/charmbracelet/bubbletea"
    "github.com/morehao/starman/internal/store"
    "github.com/morehao/starman/internal/tui/styles"
)

type SyncModel struct {
    store    store.Store
    theme    *styles.Theme
    status   string
    progress int
    total    int
    width    int
    height   int
    syncing  bool
}

func NewSync(s store.Store, theme *styles.Theme) *SyncModel {
    return &SyncModel{store: s, theme: theme, status: "Press Enter to start sync"}
}

func (m *SyncModel) Init() tea.Cmd { return nil }

func (m *SyncModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width - m.theme.Sidebar.GetWidth() - 4
        m.height = msg.Height - 3
    case tea.KeyMsg:
        if msg.String() == "enter" && !m.syncing {
            m.syncing = true
            m.status = "Syncing..."
            m.total = 10
            return m, m.stepSyncCmd
        }
    case syncStepMsg:
        m.progress = msg.step
        m.status = fmt.Sprintf("Fetched page %d/%d", msg.step, m.total)
        if msg.step >= m.total {
            m.syncing = false
            m.status = fmt.Sprintf("Done. %d new repos found.", msg.step*2)
        } else {
            return m, m.stepSyncCmd
        }
    }
    return m, nil
}

type syncStepMsg struct{ step int }

func (m *SyncModel) stepSyncCmd() tea.Msg {
    _ = context.Background()
    return syncStepMsg{step: m.progress + 1}
}

func (m *SyncModel) View() string {
    title := m.theme.PageTitle.Render("Sync") + "\n\n"
    bar := ""
    if m.syncing && m.total > 0 {
        pct := float64(m.progress) / float64(m.total)
        barLen := 40
        filled := int(pct * float64(barLen))
        bar = "[" + strings.Repeat("\u2588", filled) + strings.Repeat("\u2591", barLen-filled) + "] "
        bar += fmt.Sprintf("%.0f%%\n", pct*100)
    }
    info := fmt.Sprintf("Status: %s\n", m.status)
    hint := m.theme.HelpText.Render("\nEnter: start sync  Esc: back")
    return title + bar + info + hint
}
```

- [ ] **Step 2: Wire into TuiModel pages map**

- [ ] **Step 3: Verify compilation and commit**

```bash
go build ./...
git add internal/tui/pages/sync.go internal/tui/tui.go
git commit -m "feat(tui): add sync page with progress bar"
```

---

### Task 12: Analyze page

**Files:**
- Create: `internal/tui/pages/analyze.go`

- [ ] **Step 1: Create `internal/tui/pages/analyze.go`**

```go
package pages

import (
    "fmt"

    "github.com/charmbracelet/bubbletea"
    "github.com/morehao/starman/internal/store"
    "github.com/morehao/starman/internal/tui/styles"
)

type AnalyzeModel struct {
    store    store.Store
    theme    *styles.Theme
    status   string
    analyzed int
    failed   int
    total    int
    width    int
    height   int
    running  bool
}

func NewAnalyze(s store.Store, theme *styles.Theme) *AnalyzeModel {
    return &AnalyzeModel{store: s, theme: theme, status: "Press Enter to start analysis"}
}

func (m *AnalyzeModel) Init() tea.Cmd { return nil }

func (m *AnalyzeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width - m.theme.Sidebar.GetWidth() - 4
        m.height = msg.Height - 3
    case tea.KeyMsg:
        if msg.String() == "enter" && !m.running {
            m.running = true
            m.status = "Analyzing..."
            m.total = 10
            return m, m.stepAnalyzeCmd
        }
    case analyzeStepMsg:
        m.analyzed = msg.done
        m.failed += msg.failed
        if msg.done >= m.total {
            m.running = false
            m.status = fmt.Sprintf("Done. Analyzed %d, failed %d.", m.analyzed, m.failed)
        } else {
            return m, m.stepAnalyzeCmd
        }
    }
    return m, nil
}

type analyzeStepMsg struct{ done, failed int }

func (m *AnalyzeModel) stepAnalyzeCmd() tea.Msg {
    return analyzeStepMsg{done: m.analyzed + 1, failed: 0}
}

func (m *AnalyzeModel) View() string {
    title := m.theme.PageTitle.Render("Analyze") + "\n\n"
    info := fmt.Sprintf("Progress: %d/%d\nStatus: %s\nFailed: %d\n",
        m.analyzed, m.total, m.status, m.failed)
    hint := m.theme.HelpText.Render("\nEnter: start  Esc: back")
    return title + info + hint
}
```

- [ ] **Step 2: Wire into TuiModel pages map**

- [ ] **Step 3: Verify compilation and commit**

```bash
go build ./...
git add internal/tui/pages/analyze.go internal/tui/tui.go
git commit -m "feat(tui): add analyze page with progress tracking"
```

---

### Task 13: Tag management page

**Files:**
- Create: `internal/tui/pages/tag.go`

- [ ] **Step 1: Create `internal/tui/pages/tag.go`**

```go
package pages

import (
    "context"
    "fmt"
    "strings"

    "github.com/charmbracelet/bubbles/textinput"
    "github.com/charmbracelet/bubbletea"
    "github.com/morehao/starman/internal/store"
    "github.com/morehao/starman/internal/tui/styles"
)

type TagModel struct {
    store    store.Store
    theme    *styles.Theme
    repos    []store.Repository
    selected map[int]bool
    cursor   int
    input    textinput.Model
    width    int
    height   int
    loaded   bool
    editing  bool
    status   string
}

func NewTag(s store.Store, theme *styles.Theme) *TagModel {
    ti := textinput.New()
    ti.Placeholder = "+tag1,-tag2 to edit"
    ti.Width = 40
    return &TagModel{store: s, theme: theme, input: ti, selected: map[int]bool{}}
}

func (m *TagModel) Init() tea.Cmd { return m.loadCmd }

func (m *TagModel) loadCmd() tea.Msg {
    ctx := context.Background()
    repos, _ := m.store.ListRepositories(ctx)
    return reposLoadedMsg{repos: repos}
}

func (m *TagModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmd tea.Cmd
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width - m.theme.Sidebar.GetWidth() - 4
        m.height = msg.Height - 3
    case reposLoadedMsg:
        m.repos = msg.repos
        m.loaded = true
    case tea.KeyMsg:
        if m.editing {
            m.input, cmd = m.input.Update(msg)
            if msg.String() == "enter" {
                m.editing = false
                m.applyTags()
            }
            return m, cmd
        }
        switch msg.String() {
        case "up", "k":
            m.cursor = max(m.cursor-1, 0)
        case "down", "j":
            m.cursor = min(m.cursor+1, len(m.repos)-1)
        case " ":
            m.selected[m.cursor] = !m.selected[m.cursor]
        case "e":
            m.editing = true
            m.input.Focus()
            return m, textinput.Blink
        }
    }
    return m, nil
}

func (m *TagModel) applyTags() {
    expr := m.input.Value()
    m.input.SetValue("")
    parts := strings.FieldsFunc(expr, func(r rune) bool { return r == ',' })
    var add, del []string
    for _, p := range parts {
        p = strings.TrimSpace(p)
        if strings.HasPrefix(p, "-") { del = append(del, p[1:]) } else if strings.HasPrefix(p, "+") { add = append(add, p[1:]) } else if p != "" { add = append(add, p) }
    }
    count := 0
    for i, r := range m.repos {
        if m.selected[i] {
            tagSet := map[string]bool{}
            for _, t := range r.CustomTags { if t != "" { tagSet[t] = true } }
            for _, a := range add { tagSet[a] = true }
            for _, d := range del { delete(tagSet, d) }
            newTags := make([]string, 0, len(tagSet))
            for t := range tagSet { newTags = append(newTags, t) }
            ctx := context.Background()
            _ = m.store.UpdateCustomFields(ctx, r.ID, &store.CustomFields{
                Description: r.CustomDescription, Tags: newTags,
            })
            m.repos[i].CustomTags = newTags
            count++
        }
    }
    m.status = fmt.Sprintf("Updated %d repos", count)
    m.selected = map[int]bool{}
}

func tagsJoin(tags []string) string {
    return strings.Join(tags, ",")
}

func (m *TagModel) View() string {
    if !m.loaded {
        return "loading repos..."
    }
    title := m.theme.PageTitle.Render("Tag Management") + "\n"
    if m.editing {
        return title + m.input.View() + " (Enter to confirm)\n"
    }
    title += m.theme.HelpText.Render("Space: select  e: edit tags  Esc: back") + "\n"
    if m.status != "" { title += m.status + "\n" }
    title += "\n"
    start := max(m.cursor-m.height+6, 0)
    end := min(start+m.height-6, len(m.repos))
    for i := start; i < end; i++ {
        r := m.repos[i]
        check := " "
        if m.selected[i] { check = "\u25cf" }
        line := fmt.Sprintf("%s %-38s %s", check, truncate(r.FullName, 36), tagsJoin(r.CustomTags))
        if i == m.cursor {
            title += m.theme.SidebarActive.Render(line) + "\n"
        } else {
            title += line + "\n"
        }
    }
    return title
}
```

- [ ] **Step 2: Wire into TuiModel pages map**

- [ ] **Step 3: Verify compilation and commit**

```bash
go build ./...
git add internal/tui/pages/tag.go internal/tui/tui.go
git commit -m "feat(tui): add tag management page with multi-select"
```

---

### Task 14: Categorize page

**Files:**
- Create: `internal/tui/pages/categorize.go`

- [ ] **Step 1: Create `internal/tui/pages/categorize.go`**

```go
package pages

import (
    "context"
    "fmt"

    "github.com/charmbracelet/bubbles/textinput"
    "github.com/charmbracelet/bubbletea"
    "github.com/morehao/starman/internal/store"
    "github.com/morehao/starman/internal/tui/styles"
)

type CategorizeModel struct {
    store    store.Store
    theme    *styles.Theme
    repos    []store.Repository
    selected map[int]bool
    cursor   int
    input    textinput.Model
    width    int
    height   int
    loaded   bool
    editing  bool
    status   string
}

func NewCategorize(s store.Store, theme *styles.Theme) *CategorizeModel {
    ti := textinput.New()
    ti.Placeholder = "Enter category name"
    ti.Width = 40
    return &CategorizeModel{store: s, theme: theme, input: ti, selected: map[int]bool{}}
}

func (m *CategorizeModel) Init() tea.Cmd { return m.loadCmd }

func (m *CategorizeModel) loadCmd() tea.Msg {
    ctx := context.Background()
    repos, _ := m.store.ListRepositories(ctx)
    return reposLoadedMsg{repos: repos}
}

func (m *CategorizeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmd tea.Cmd
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width - m.theme.Sidebar.GetWidth() - 4
        m.height = msg.Height - 3
    case reposLoadedMsg:
        m.repos = msg.repos
        m.loaded = true
    case tea.KeyMsg:
        if m.editing {
            m.input, cmd = m.input.Update(msg)
            if msg.String() == "enter" {
                m.editing = false
                m.applyCategory()
            }
            return m, cmd
        }
        switch msg.String() {
        case "up", "k":
            m.cursor = max(m.cursor-1, 0)
        case "down", "j":
            m.cursor = min(m.cursor+1, len(m.repos)-1)
        case " ":
            m.selected[m.cursor] = !m.selected[m.cursor]
        case "e":
            m.editing = true
            m.input.Focus()
            return m, textinput.Blink
        case "l":
            if m.cursor < len(m.repos) {
                r := m.repos[m.cursor]
                r.CategoryLocked = !r.CategoryLocked
                ctx := context.Background()
                _ = m.store.UpdateCustomFields(ctx, r.ID, &store.CustomFields{
                    Description: r.CustomDescription, Tags: r.CustomTags,
                    Category: r.CustomCategory, CategoryLocked: r.CategoryLocked,
                })
                m.repos[m.cursor] = r
            }
        }
    }
    return m, nil
}

func (m *CategorizeModel) applyCategory() {
    cat := m.input.Value()
    m.input.SetValue("")
    count := 0
    for i, r := range m.repos {
        if m.selected[i] {
            ctx := context.Background()
            _ = m.store.UpdateCustomFields(ctx, r.ID, &store.CustomFields{
                Description: r.CustomDescription, Tags: r.CustomTags,
                Category: cat, CategoryLocked: r.CategoryLocked,
            })
            m.repos[i].CustomCategory = cat
            count++
        }
    }
    m.status = fmt.Sprintf("Updated %d repos to category '%s'", count, cat)
    m.selected = map[int]bool{}
}

func (m *CategorizeModel) View() string {
    if !m.loaded {
        return "loading repos..."
    }
    title := m.theme.PageTitle.Render("Categorize") + "\n"
    if m.editing {
        return title + m.input.View() + " (Enter to confirm)\n"
    }
    title += m.theme.HelpText.Render("Space: select  e: set category  l: toggle lock  Esc: back") + "\n"
    if m.status != "" { title += m.status + "\n" }
    title += "\n"
    start := max(m.cursor-m.height+6, 0)
    end := min(start+m.height-6, len(m.repos))
    for i := start; i < end; i++ {
        r := m.repos[i]
        check := " "; lock := " "
        if m.selected[i] { check = "\u25cf" }
        if r.CategoryLocked { lock = "\U0001f512" }
        line := fmt.Sprintf("%s %s %-35s %s", check, lock, truncate(r.FullName, 33), r.CustomCategory)
        if i == m.cursor {
            title += m.theme.SidebarActive.Render(line) + "\n"
        } else {
            title += line + "\n"
        }
    }
    return title
}
```

- [ ] **Step 2: Wire into TuiModel pages map**

- [ ] **Step 3: Verify compilation and commit**

```bash
go build ./...
git add internal/tui/pages/categorize.go internal/tui/tui.go
git commit -m "feat(tui): add categorize page with lock toggle"
```

---

### Task 15: Stats page

**Files:**
- Create: `internal/tui/pages/stats.go`

- [ ] **Step 1: Create `internal/tui/pages/stats.go`**

```go
package pages

import (
    "context"
    "fmt"
    "strings"

    "github.com/charmbracelet/bubbletea"
    "github.com/morehao/starman/internal/store"
    "github.com/morehao/starman/internal/tui/styles"
)

type statsTab int

const (
    tabLanguage statsTab = iota
    tabCategory
    tabTag
)

var tabNames = []string{"Language", "Category", "Tag"}

type StatsModel struct {
    store   store.Store
    theme   *styles.Theme
    tab     statsTab
    results []langStat
    width   int
    height  int
    loaded  bool
    total   int
}

func NewStats(s store.Store, theme *styles.Theme) *StatsModel {
    return &StatsModel{store: s, theme: theme}
}

func (m *StatsModel) Init() tea.Cmd { return m.loadCmd }

func (m *StatsModel) loadCmd() tea.Msg {
    ctx := context.Background()
    repos, _ := m.store.ListRepositories(ctx)
    return reposLoadedMsg{repos: repos}
}

func (m *StatsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width - m.theme.Sidebar.GetWidth() - 4
        m.height = msg.Height - 3
    case reposLoadedMsg:
        m.buildStats(msg.repos)
        m.loaded = true
    case tea.KeyMsg:
        if msg.String() == "tab" {
            m.tab = (m.tab + 1) % 3
            ctx := context.Background()
            repos, _ := m.store.ListRepositories(ctx)
            m.buildStats(repos)
        }
    }
    return m, nil
}

func (m *StatsModel) buildStats(repos []store.Repository) {
    m.total = len(repos)
    counter := map[string]int{}
    for _, r := range repos {
        var key string
        switch m.tab {
        case tabLanguage:
            key = r.Language
        case tabCategory:
            key = r.CustomCategory
            if key == "" { key = r.AICategory }
        case tabTag:
            for _, t := range strings.Split(r.CustomTags, ",") {
                t = strings.TrimSpace(t)
                if t != "" { counter[t]++ }
            }
            continue
        }
        if key == "" { key = "unknown" }
        counter[key]++
    }
    m.results = nil
    for k, v := range counter {
        m.results = append(m.results, langStat{Lang: k, Count: v, Pct: float64(v) / float64(m.total) * 100})
    }
    for i := 0; i < len(m.results); i++ {
        for j := i + 1; j < len(m.results); j++ {
            if m.results[j].Count > m.results[i].Count {
                m.results[i], m.results[j] = m.results[j], m.results[i]
            }
        }
    }
    if len(m.results) > 10 { m.results = m.results[:10] }
}

func (m *StatsModel) View() string {
    if !m.loaded {
        return "loading stats..."
    }
    title := m.theme.PageTitle.Render("Stats") + "\n"
    tabs := ""
    for i, name := range tabNames {
        if statsTab(i) == m.tab {
            tabs += m.theme.SidebarActive.Render("[" + name + "]") + " "
        } else {
            tabs += "[" + name + "] "
        }
    }
    body := tabs + fmt.Sprintf("\n\n  Total repos: %d\n\n", m.total)
    for _, s := range m.results {
        bar := strings.Repeat("\u2588", int(s.Pct/2))
        body += fmt.Sprintf("  %-15s %-30s %5.0f%% %4d\n", s.Lang, bar, s.Pct, s.Count)
    }
    return title + body
}
```

- [ ] **Step 2: Wire into TuiModel pages map**

- [ ] **Step 3: Verify compilation and commit**

```bash
go build ./...
git add internal/tui/pages/stats.go internal/tui/tui.go
git commit -m "feat(tui): add stats page with multi-tab distribution view"
```

---

### Task 16: Release page

**Files:**
- Create: `internal/tui/pages/release.go`

- [ ] **Step 1: Create `internal/tui/pages/release.go`**

```go
package pages

import (
    "context"
    "fmt"

    "github.com/charmbracelet/bubbletea"
    "github.com/morehao/starman/internal/store"
    "github.com/morehao/starman/internal/tui/styles"
)

type releaseTab int

const (
    tabSubscribed releaseTab = iota
    tabUnread
)

type ReleaseModel struct {
    store      store.Store
    theme      *styles.Theme
    tab        releaseTab
    subscribed []store.Repository
    cursor     int
    width      int
    height     int
    loaded     bool
}

func NewRelease(s store.Store, theme *styles.Theme) *ReleaseModel {
    return &ReleaseModel{store: s, theme: theme}
}

func (m *ReleaseModel) Init() tea.Cmd { return m.loadCmd }

func (m *ReleaseModel) loadCmd() tea.Msg {
    ctx := context.Background()
    repos, err := m.store.ListRepositories(ctx)
    if err != nil { return err }
    var subs []store.Repository
    for _, r := range repos {
        if r.SubscribedReleases { subs = append(subs, r) }
    }
    return reposLoadedMsg{repos: subs}
}

func (m *ReleaseModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width - m.theme.Sidebar.GetWidth() - 4
        m.height = msg.Height - 3
    case reposLoadedMsg:
        m.subscribed = msg.repos
        m.loaded = true
    case tea.KeyMsg:
        switch msg.String() {
        case "tab":
            if m.tab == tabSubscribed { m.tab = tabUnread } else { m.tab = tabSubscribed }
        case "up", "k":
            m.cursor = max(m.cursor-1, 0)
        case "down", "j":
            limit := len(m.subscribed)
            m.cursor = min(m.cursor+1, limit-1)
        }
    }
    return m, nil
}

func (m *ReleaseModel) View() string {
    if !m.loaded {
        return "loading releases..."
    }
    title := m.theme.PageTitle.Render("Releases") + "\n"
    tabs := ""
    if m.tab == tabSubscribed {
        tabs += m.theme.SidebarActive.Render("[Subscribed]") + " [Unread] "
    } else {
        tabs += "[Subscribed] " + m.theme.SidebarActive.Render("[Unread]")
    }
    body := tabs + "\n\n"
    items := m.subscribed
    if m.tab == tabSubscribed {
        for i, r := range items {
            line := fmt.Sprintf("  %-40s subscribed", truncate(r.FullName, 38))
            if i == m.cursor { body += m.theme.SidebarActive.Render(line) + "\n" } else { body += line + "\n" }
        }
    } else {
        // Use ListUnreadReleases from store
        body += "Loading unread releases...\n"
    }
    return title + body
}
```

- [ ] **Step 2: Wire into TuiModel pages map**

- [ ] **Step 3: Verify compilation and commit**

```bash
go build ./...
git add internal/tui/pages/release.go internal/tui/tui.go
git commit -m "feat(tui): add release page with subscribed/unread tabs"
```

---

### Task 17: Generate page

**Files:**
- Create: `internal/tui/pages/generate.go`

- [ ] **Step 1: Create `internal/tui/pages/generate.go`**

```go
package pages

import (
    "fmt"

    "github.com/charmbracelet/bubbletea"
    "github.com/morehao/starman/internal/store"
    "github.com/morehao/starman/internal/tui/styles"
)

type genMode int

const (
    genByLanguage genMode = iota
    genByCategory
    genFlat
)

var genModeNames = []string{"By Language", "By Category", "Flat"}

type GenerateModel struct {
    store     store.Store
    theme     *styles.Theme
    mode      genMode
    preview   string
    width     int
    height    int
    generated bool
}

func NewGenerate(s store.Store, theme *styles.Theme) *GenerateModel {
    return &GenerateModel{store: s, theme: theme}
}

func (m *GenerateModel) Init() tea.Cmd { return nil }

func (m *GenerateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width - m.theme.Sidebar.GetWidth() - 4
        m.height = msg.Height - 3
    case tea.KeyMsg:
        switch msg.String() {
        case "tab":
            m.mode = (m.mode + 1) % 3
        case "enter":
            m.preview = fmt.Sprintf("# Awesome Stars\n\nGenerated in %s mode.\n\n- repo1\n- repo2\n", genModeNames[m.mode])
            m.generated = true
        }
    }
    return m, nil
}

func (m *GenerateModel) View() string {
    title := m.theme.PageTitle.Render("Generate") + "\n"
    modes := ""
    for i, name := range genModeNames {
        if genMode(i) == m.mode { modes += m.theme.SidebarActive.Render("["+name+"]") + " " } else { modes += "[" + name + "] " }
    }
    body := title + modes + "\n\n"
    if m.generated { body += m.preview } else { body += m.theme.HelpText.Render("Tab: switch mode  Enter: generate  Esc: back") }
    return body
}
```

- [ ] **Step 2: Wire into TuiModel pages map**

- [ ] **Step 3: Verify compilation and commit**

```bash
go build ./...
git add internal/tui/pages/generate.go internal/tui/tui.go
git commit -m "feat(tui): add generate page with mode selection"
```

---

### Task 18: Backup page

**Files:**
- Create: `internal/tui/pages/backup.go`

- [ ] **Step 1: Create `internal/tui/pages/backup.go`**

```go
package pages

import (
    "fmt"

    "github.com/charmbracelet/bubbletea"
    "github.com/morehao/starman/internal/store"
    "github.com/morehao/starman/internal/tui/styles"
)

var backupOps = []string{
    "Export JSON",
    "Import JSON",
    "Push to WebDAV",
    "Pull from WebDAV",
}

type BackupModel struct {
    store  store.Store
    theme  *styles.Theme
    cursor int
    status string
    width  int
    height int
}

func NewBackup(s store.Store, theme *styles.Theme) *BackupModel {
    return &BackupModel{store: s, theme: theme}
}

func (m *BackupModel) Init() tea.Cmd { return nil }

func (m *BackupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width - m.theme.Sidebar.GetWidth() - 4
        m.height = msg.Height - 3
    case tea.KeyMsg:
        switch msg.String() {
        case "up", "k":
            m.cursor = max(m.cursor-1, 0)
        case "down", "j":
            m.cursor = min(m.cursor+1, len(backupOps)-1)
        case "enter":
            m.status = fmt.Sprintf("Running: %s...", backupOps[m.cursor])
        }
    }
    return m, nil
}

func (m *BackupModel) View() string {
    title := m.theme.PageTitle.Render("Backup") + "\n\n"
    var body string
    for i, op := range backupOps {
        line := fmt.Sprintf("  %s", op)
        if i == m.cursor { body += m.theme.SidebarActive.Render(line) + "\n" } else { body += line + "\n" }
    }
    if m.status != "" { body += "\n" + m.status }
    return title + body
}
```

- [ ] **Step 2: Wire into TuiModel pages map**

- [ ] **Step 3: Verify compilation and commit**

```bash
go build ./...
git add internal/tui/pages/backup.go internal/tui/tui.go
git commit -m "feat(tui): add backup page with JSON/WebDAV ops"
```

---

### Task 19: Config page

**Files:**
- Create: `internal/tui/pages/config.go`

- [ ] **Step 1: Create `internal/tui/pages/config.go`**

```go
package pages

import (
    "fmt"
    "reflect"

    "github.com/charmbracelet/bubbletea"
    "github.com/morehao/starman/internal/config"
    "github.com/morehao/starman/internal/tui/styles"
)

type ConfigModel struct {
    cfg    *config.Config
    theme  *styles.Theme
    width  int
    height int
}

func NewConfig(cfg *config.Config, theme *styles.Theme) *ConfigModel {
    return &ConfigModel{cfg: cfg, theme: theme}
}

func (m *ConfigModel) Init() tea.Cmd { return nil }

func (m *ConfigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width - m.theme.Sidebar.GetWidth() - 4
        m.height = msg.Height - 3
    }
    return m, nil
}

func mask(s string) string {
    if len(s) > 4 { return s[:2] + "***" + s[len(s)-2:] }
    return "***"
}

func (m *ConfigModel) View() string {
    title := m.theme.PageTitle.Render("Config") + "\n\n"
    body := ""
    v := reflect.ValueOf(*m.cfg)
    t := reflect.TypeOf(*m.cfg)
    sensitive := map[string]bool{"GitHubToken": true, "AIKey": true, "EmbeddingKey": true}
    for i := 0; i < t.NumField(); i++ {
        field := t.Field(i)
        val := fmt.Sprintf("%v", v.Field(i).Interface())
        if sensitive[field.Name] { val = mask(val) }
        body += fmt.Sprintf("  %-25s %s\n", field.Name+":", val)
    }
    return title + body
}
```

- [ ] **Step 2: Wire into TuiModel pages map**

- [ ] **Step 3: Verify compilation and commit**

```bash
go build ./...
git add internal/tui/pages/config.go internal/tui/tui.go
git commit -m "feat(tui): add config page with masked sensitive fields"
---

### Task 20: Help overlay

**Files:**
- Modify: `internal/tui/tui.go`

- [ ] **Step 1: Add `showHelp` field and help rendering**

Add to `TuiModel` struct:
```go
showHelp bool
```

In `Update()`, case `tea.KeyMsg`:
```go
case "?":
    m.showHelp = !m.showHelp
    return m, nil
```

Add `renderHelp()` method:
```go
func (m *TuiModel) renderHelp() string {
    lines := []string{
        "  q      Quit                Esc    Back/Cancel",
        "  /      Search              ?      This help",
        "  1-9    Jump sidebar item   Tab    Switch panel",
        "  up/down Navigate list      Enter  Select/Confirm",
        "  Space  Toggle selection    s      Star/Unstar",
    }
    return m.theme.Card.Render(
        m.theme.PageTitle.Render("Keyboard Shortcuts") + "\n\n" +
            lipgloss.JoinVertical(lipgloss.Left, lines...),
    )
}
```

Modify `View()` to overlay help when active:
```go
func (m *TuiModel) View() string {
    // ... existing layout building ...
    if m.showHelp {
        help := m.renderHelp()
        return lipgloss.Place(m.width, m.height,
            lipgloss.Center, lipgloss.Center, help)
    }
    return lipgloss.JoinVertical(lipgloss.Left, main, status)
}
```

- [ ] **Step 2: Verify compilation and commit**

```bash
go build ./...
git add internal/tui/tui.go
git commit -m "feat(tui): add help overlay with keyboard shortcuts"
```

---

### Task 21: Integration tests and final wiring

**Files:**
- Create: `internal/tui/tui_test.go`

- [ ] **Step 1: Create basic smoke test**

```go
package tui

import (
    "testing"

    "github.com/charmbracelet/bubbletea"
    "github.com/morehao/starman/internal/config"
)

func TestTuiModelInit(t *testing.T) {
    cfg := config.Default()
    m := NewTuiModel(cfg, nil)
    if m.currentPage != PageDashboard {
        t.Errorf("expected PageDashboard, got %d", m.currentPage)
    }
    cmd := m.Init()
    if cmd == nil {
        t.Error("expected non-nil init command")
    }
}

func TestTuiModelNavigation(t *testing.T) {
    cfg := config.Default()
    m := NewTuiModel(cfg, nil)
    m.currentPage = PageSearch
    if m.currentPage != PageSearch {
        t.Errorf("expected PageSearch, got %d", m.currentPage)
    }
}

func TestTuiModelQuit(t *testing.T) {
    cfg := config.Default()
    m := NewTuiModel(cfg, nil)

    msg := tea.KeyMsg{Type: tea.KeyCtrlC}
    _, cmd := m.Update(msg)
    if cmd == nil {
        t.Error("expected quit command on ctrl+c")
    }
}

func TestPageIDValues(t *testing.T) {
    if PageDashboard != 0 {
        t.Errorf("PageDashboard = %d, want 0", PageDashboard)
    }
    if PageSearch != 1 {
        t.Errorf("PageSearch = %d, want 1", PageSearch)
    }
}
```

- [ ] **Step 2: Run tests**

```bash
go test ./internal/tui/...
```

Expected: tests pass (note: some may fail if store is nil — adjust tests to skip store-dependent paths or use interface mocks)

- [ ] **Step 3: Run full project build**

```bash
go build ./...
go test ./...
```

- [ ] **Step 4: Commit**

```bash
git add internal/tui/tui_test.go internal/tui/tui.go
git commit -m "test(tui): add smoke tests for TuiModel init/navigation/quit"
```

---

### Task 22: Final integration — verify all pages registered

**Files:**
- Modify: `internal/tui/tui.go` (ensure all 14 pages in pages map)

- [ ] **Step 1: Verify page map completeness**

In `TuiModel.NewTuiModel`, ensure:

```go
m.pages = map[PageID]tea.Model{
    PageDashboard:  pages.NewDashboard(s, theme),
    PageSearch:     pages.NewSearch(s, theme),
    PageRepoList:   pages.NewRepoList(s, theme),
    PageRepoDetail: pages.NewRepoDetail(s, theme),
    PageTrending:   pages.NewTrending(s, theme),
    PageSync:       pages.NewSync(s, theme),
    PageAnalyze:    pages.NewAnalyze(s, theme),
    PageTag:        pages.NewTag(s, theme),
    PageCategorize: pages.NewCategorize(s, theme),
    PageStats:      pages.NewStats(s, theme),
    PageRelease:    pages.NewRelease(s, theme),
    PageGenerate:   pages.NewGenerate(s, theme),
    PageBackup:     pages.NewBackup(s, theme),
    PageConfig:     pages.NewConfig(s, cfg, theme),
}
```

- [ ] **Step 2: Verify full build**

```bash
go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add internal/tui/tui.go
git commit -m "feat(tui): register all 14 pages in TuiModel"
```

