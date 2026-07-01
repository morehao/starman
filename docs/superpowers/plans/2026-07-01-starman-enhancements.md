# starman 功能增强实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 starman CLI 新增 7 个功能：completion、stats、搜索增强、info、tag/categorize、trending、sync --watch，完善 star 仓库管理体验。

**Architecture:** 遵循现有分层（cli → 业务包 → store）。新增 `internal/discovery` 包承载 trending 逻辑。GitHub 客户端扩展 3 个方法。不新增 store 接口方法，不引入新依赖。

**Tech Stack:** Go 1.25+、cobra/pflag、go-github/v71、httpcache、modernc.org/sqlite、encoding/xml（标准库，RSS 解析）

## Global Constraints

- 模块路径：`github.com/morehao/starman`
- Go 版本：`go 1.25`
- 无 CGO 依赖，无新增第三方依赖
- 无注释（遵循项目 AGENTS.md 约定），除非用户要求
- 测试用标准库 `testing` + `net/http/httptest`，mock 外部 API，不依赖网络
- 日志中不输出 API key/token 等敏感信息
- 敏感字段优先从环境变量读取：`STARMAN_GITHUB_TOKEN`/`GITHUB_TOKEN`、`STARMAN_AI_API_KEY`
- 数据目录：`~/.starman/`（含 `config.yaml` 与 `starman.db`）
- 设计文档：`docs/superpowers/specs/2026-07-01-starman-enhancements-design.md`

## File Structure

```
internal/
  cli/
    root.go              # 修改：注册 7 个新命令
    search.go            # 修改：新增 --json/--limit/--lang/--category/--sort flags
    sync.go              # 修改：新增 --watch/--interval flags
    stats.go             # 新增：统计命令
    info.go              # 新增：仓库详情命令
    trending.go          # 新增：趋势发现命令
    tag.go               # 新增：标签管理命令
    categorize.go        # 新增：分类管理命令
    completion.go        # 新增：shell 补全命令
    stats_test.go        # 新增：聚合逻辑测试
    search_test.go       # 新增：过滤排序测试
    tag_test.go          # 新增：标签表达式解析测试
    categorize_test.go   # 新增：分类逻辑测试
  github/
    operations.go        # 修改：新增 ListReadmeVariants/GetContentFile/SearchRepositories
    operations_test.go   # 修改：新增 3 个方法的测试
  discovery/
    trending.go          # 新增：RSS 解析 + search fallback + 字段补充
    trending_test.go     # 新增：RSS 解析与 trending 逻辑测试
```

---

### Task 1: completion Shell 补全命令

**Files:**
- Create: `internal/cli/completion.go`
- Modify: `internal/cli/root.go` (注册命令)

**Interfaces:**
- Consumes: `cli.NewRootCmd` 返回的 `*cobra.Command`（用于生成补全脚本）
- Produces: `newCompletionCmd(root *cobra.Command) *cobra.Command`

- [ ] **Step 1: 编写 completion 命令**

`internal/cli/completion.go`:
```go
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newCompletionCmd(root *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:       "completion <shell>",
		Short:     "Generate shell completion script",
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return root.GenBashCompletion(os.Stdout)
			case "zsh":
				return root.GenZshCompletion(os.Stdout)
			case "fish":
				return root.GenFishCompletion(os.Stdout)
			case "powershell":
				return root.GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return fmt.Errorf("unsupported shell: %s", args[0])
			}
		},
	}
	return cmd
}
```

- [ ] **Step 2: 在 root.go 中注册命令**

Modify `internal/cli/root.go`, 在 `return root` 前添加：

```go
	root.AddCommand(newCompletionCmd(root))
```

- [ ] **Step 3: 验证构建与运行**

Run:
```bash
go build -o /tmp/starman ./cmd/starman && /tmp/starman completion bash | head -5
```
Expected: 输出 bash 补全脚本前 5 行（非空）

Run:
```bash
/tmp/starman completion zsh | head -5
```
Expected: 输出 zsh 补全脚本前 5 行

- [ ] **Step 4: 提交**

```bash
git add internal/cli/completion.go internal/cli/root.go
git commit -m "feat: add shell completion command"
```

---

### Task 2: stats 统计命令

**Files:**
- Create: `internal/cli/stats.go`
- Create: `internal/cli/stats_test.go`
- Modify: `internal/cli/root.go` (注册命令)

**Interfaces:**
- Consumes: `store.Store.ListRepositories`、`store.Repository` 字段
- Produces: `newStatsCmd() *cobra.Command`、`aggregateStats(repos []*store.Repository, by string) []StatItem`、`StatItem` 类型

- [ ] **Step 1: 编写失败测试**

`internal/cli/stats_test.go`:
```go
package cli

import (
	"testing"

	"github.com/morehao/starman/internal/store"
)

func TestAggregateStatsByLanguage(t *testing.T) {
	repos := []*store.Repository{
		{FullName: "a/b", Language: "Go"},
		{FullName: "c/d", Language: "Go"},
		{FullName: "e/f", Language: "TypeScript"},
		{FullName: "g/h", Language: ""},
	}
	items := aggregateStats(repos, "language")
	if len(items) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(items))
	}
	if items[0].Name != "Go" || items[0].Count != 2 {
		t.Fatalf("expected Go count 2 first, got %+v", items[0])
	}
	if items[2].Name != "Others" || items[2].Count != 1 {
		t.Fatalf("expected Others count 1 last, got %+v", items[2])
	}
}

func TestAggregateStatsByCategory(t *testing.T) {
	repos := []*store.Repository{
		{FullName: "a/b", AICategory: "dev-tools"},
		{FullName: "c/d", CustomCategory: "dev-tools"},
		{FullName: "e/f"},
	}
	items := aggregateStats(repos, "category")
	if len(items) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(items))
	}
	if items[0].Count != 2 {
		t.Fatalf("expected count 2 for top category, got %d", items[0].Count)
	}
	if items[1].Name != "其他" || items[1].Count != 1 {
		t.Fatalf("expected 其他 count 1 second, got %+v", items[1])
	}
}

func TestAggregateStatsByTag(t *testing.T) {
	repos := []*store.Repository{
		{FullName: "a/b", AITags: []string{"cli", "go"}, CustomTags: []string{"awesome"}},
		{FullName: "c/d", AITags: []string{"cli"}, CustomTags: []string{}},
		{FullName: "e/f", AITags: []string{}, CustomTags: []string{"web"}},
	}
	items := aggregateStats(repos, "tag")
	if len(items) != 4 {
		t.Fatalf("expected 4 tags, got %d", len(items))
	}
	if items[0].Name != "cli" || items[0].Count != 2 {
		t.Fatalf("expected cli count 2 first, got %+v", items[0])
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `go test ./internal/cli/ -run TestAggregateStats -v`
Expected: FAIL with `aggregateStats not defined`

- [ ] **Step 3: 编写 stats 命令与聚合逻辑**

`internal/cli/stats.go`:
```go
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/morehao/starman/internal/store"
	"github.com/spf13/cobra"
)

type StatItem struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func newStatsCmd() *cobra.Command {
	var by string
	var top int
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show statistics of synced repositories",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore()
			if err != nil {
				return err
			}
			defer s.Close()
			repos, err := s.ListRepositories(context.Background())
			if err != nil {
				return err
			}
			items := aggregateStats(repos, by)
			if top > 0 && top < len(items) {
				items = items[:top]
			}
			if jsonOut {
				return outputStatsJSON(items)
			}
			outputStatsTable(items, by)
			return nil
		},
	}
	cmd.Flags().StringVar(&by, "by", "language", "dimension: language|category|tag")
	cmd.Flags().IntVar(&top, "top", 10, "show top N items (0=all)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output as JSON")
	return cmd
}

func aggregateStats(repos []*store.Repository, by string) []StatItem {
	counts := make(map[string]int)
	switch by {
	case "language":
		for _, r := range repos {
			lang := r.Language
			if lang == "" {
				lang = "Others"
			}
			counts[lang]++
		}
	case "category":
		for _, r := range repos {
			cat := r.CustomCategory
			if cat == "" {
				cat = r.AICategory
			}
			if cat == "" {
				cat = "其他"
			}
			counts[cat]++
		}
	case "tag":
		for _, r := range repos {
			seen := make(map[string]bool)
			for _, tag := range r.AITags {
				tag = strings.ToLower(tag)
				if !seen[tag] {
					counts[tag]++
					seen[tag] = true
				}
			}
			seen = make(map[string]bool)
			for _, tag := range r.CustomTags {
				tag = strings.ToLower(tag)
				if !seen[tag] {
					counts[tag]++
					seen[tag] = true
				}
			}
		}
	}
	items := make([]StatItem, 0, len(counts))
	for name, count := range counts {
		items = append(items, StatItem{Name: name, Count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count != items[j].Count {
			return items[i].Count > items[j].Count
		}
		return items[i].Name < items[j].Name
	})
	return items
}

func outputStatsTable(items []StatItem, by string) {
	header := strings.ToUpper(by)
	fmt.Fprintf(os.Stdout, "%-30s %s\n", header, "COUNT")
	for _, item := range items {
		fmt.Fprintf(os.Stdout, "%-30s %d\n", item.Name, item.Count)
	}
}

func outputStatsJSON(items []StatItem) error {
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, string(data))
	return nil
}
```

- [ ] **Step 4: 在 root.go 中注册命令**

Modify `internal/cli/root.go`, 添加：
```go
	root.AddCommand(newStatsCmd())
```

- [ ] **Step 5: 运行测试验证通过**

Run: `go test ./internal/cli/ -run TestAggregateStats -v`
Expected: 全部 PASS

- [ ] **Step 6: 验证构建**

Run: `go build -o /tmp/starman ./cmd/starman && /tmp/starman stats --help`
Expected: 输出 stats 命令帮助

- [ ] **Step 7: 提交**

```bash
git add internal/cli/stats.go internal/cli/stats_test.go internal/cli/root.go
git commit -m "feat: add stats command for repository statistics"
```

---

### Task 3: 搜索增强

**Files:**
- Modify: `internal/cli/search.go`
- Create: `internal/cli/search_test.go`

**Interfaces:**
- Consumes: `ai.Service.Search`（现有）、`store.Repository` 字段
- Produces: `filterAndSortHits(hits []*ai.SearchHit, opts searchOpts) []*ai.SearchHit`、`searchOpts` 类型

- [ ] **Step 1: 编写失败测试**

`internal/cli/search_test.go`:
```go
package cli

import (
	"testing"

	"github.com/morehao/starman/internal/ai"
	"github.com/morehao/starman/internal/store"
)

func TestFilterAndSortHitsByLang(t *testing.T) {
	hits := []*ai.SearchHit{
		{Repo: &store.Repository{FullName: "a/b", Language: "Go", StargazersCount: 10}, Score: 5},
		{Repo: &store.Repository{FullName: "c/d", Language: "Python", StargazersCount: 20}, Score: 8},
		{Repo: &store.Repository{FullName: "e/f", Language: "Go", StargazersCount: 30}, Score: 3},
	}
	opts := searchOpts{Lang: "Go"}
	filtered := filterAndSortHits(hits, opts)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 Go repos, got %d", len(filtered))
	}
	for _, h := range filtered {
		if h.Repo.Language != "Go" {
			t.Fatalf("expected only Go repos, got %s", h.Repo.Language)
		}
	}
}

func TestFilterAndSortHitsByCategory(t *testing.T) {
	hits := []*ai.SearchHit{
		{Repo: &store.Repository{FullName: "a/b", AICategory: "dev-tools"}, Score: 5},
		{Repo: &store.Repository{FullName: "c/d", CustomCategory: "dev-tools"}, Score: 8},
		{Repo: &store.Repository{FullName: "e/f", AICategory: "web-app"}, Score: 3},
	}
	opts := searchOpts{Category: "dev-tools"}
	filtered := filterAndSortHits(hits, opts)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 dev-tools repos, got %d", len(filtered))
	}
}

func TestFilterAndSortHitsByStars(t *testing.T) {
	hits := []*ai.SearchHit{
		{Repo: &store.Repository{FullName: "a/b", StargazersCount: 10}, Score: 5},
		{Repo: &store.Repository{FullName: "c/d", StargazersCount: 50}, Score: 3},
		{Repo: &store.Repository{FullName: "e/f", StargazersCount: 30}, Score: 8},
	}
	opts := searchOpts{Sort: "stars"}
	sorted := filterAndSortHits(hits, opts)
	if len(sorted) != 3 {
		t.Fatalf("expected 3 repos, got %d", len(sorted))
	}
	if sorted[0].Repo.StargazersCount != 50 {
		t.Fatalf("expected 50 stars first, got %d", sorted[0].Repo.StargazersCount)
	}
	if sorted[1].Repo.StargazersCount != 30 {
		t.Fatalf("expected 30 stars second, got %d", sorted[1].Repo.StargazersCount)
	}
}

func TestFilterAndSortHitsLimit(t *testing.T) {
	hits := []*ai.SearchHit{
		{Repo: &store.Repository{FullName: "a/b"}, Score: 5},
		{Repo: &store.Repository{FullName: "c/d"}, Score: 3},
		{Repo: &store.Repository{FullName: "e/f"}, Score: 8},
	}
	opts := searchOpts{Limit: 2}
	limited := filterAndSortHits(hits, opts)
	if len(limited) != 2 {
		t.Fatalf("expected 2 repos after limit, got %d", len(limited))
	}
	if limited[0].Score != 8 {
		t.Fatalf("expected score 8 first, got %f", limited[0].Score)
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `go test ./internal/cli/ -run TestFilterAndSort -v`
Expected: FAIL with `filterAndSortHits not defined`

- [ ] **Step 3: 修改 search.go，新增 flags 与过滤排序逻辑**

Replace the full content of `internal/cli/search.go`:
```go
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/morehao/starman/internal/ai"
	"github.com/morehao/starman/internal/config"
	"github.com/spf13/cobra"
)

type searchOpts struct {
	Lang     string
	Category string
	Sort     string
	Limit    int
}

func newSearchCmd() *cobra.Command {
	var jsonOut bool
	var limit int
	var lang string
	var category string
	var sortBy string
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search repos by AI-translated keywords",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, err := loadConfig(cmd)
			if err != nil {
				return err
			}
			aiKey := config.ResolveAIKey(cfg, "")
			if aiKey == "" {
				return fmt.Errorf("AI api_key required")
			}
			s, err := openStore()
			if err != nil {
				return err
			}
			defer s.Close()
			ctx := context.Background()
			repos, err := s.ListRepositories(ctx)
			if err != nil {
				return err
			}
			aiClient := ai.NewClient(cfg.AI.BaseURL, aiKey, cfg.AI.Model)
			svc := ai.NewService(aiClient, nil)
			hits, err := svc.Search(ctx, args[0], repos)
			if err != nil {
				return err
			}
			opts := searchOpts{Lang: lang, Category: category, Sort: sortBy, Limit: limit}
			hits = filterAndSortHits(hits, opts)
			if jsonOut {
				return outputSearchJSON(hits)
			}
			outputSearchTable(hits)
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output as JSON")
	cmd.Flags().IntVar(&limit, "limit", 0, "limit number of results (0=no limit)")
	cmd.Flags().StringVar(&lang, "lang", "", "filter by language")
	cmd.Flags().StringVar(&category, "category", "", "filter by category")
	cmd.Flags().StringVar(&sortBy, "sort", "score", "sort by: score|stars|updated|name")
	return cmd
}

func filterAndSortHits(hits []*ai.SearchHit, opts searchOpts) []*ai.SearchHit {
	var filtered []*ai.SearchHit
	for _, h := range hits {
		if opts.Lang != "" && !strings.EqualFold(h.Repo.Language, opts.Lang) {
			continue
		}
		if opts.Category != "" {
			cat := h.Repo.CustomCategory
			if cat == "" {
				cat = h.Repo.AICategory
			}
			if !strings.EqualFold(cat, opts.Category) {
				continue
			}
		}
		filtered = append(filtered, h)
	}
	switch opts.Sort {
	case "stars":
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].Repo.StargazersCount > filtered[j].Repo.StargazersCount
		})
	case "name":
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].Repo.FullName < filtered[j].Repo.FullName
		})
	case "updated":
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].Repo.StarredAt > filtered[j].Repo.StarredAt
		})
	default:
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].Score > filtered[j].Score
		})
	}
	if opts.Limit > 0 && opts.Limit < len(filtered) {
		filtered = filtered[:opts.Limit]
	}
	return filtered
}

func outputSearchTable(hits []*ai.SearchHit) {
	fmt.Fprintf(os.Stdout, "%-6s %-40s %s\n", "SCORE", "REPO", "DESCRIPTION")
	for _, h := range hits {
		desc := h.Repo.Description
		if len(desc) > 50 {
			desc = desc[:50] + "..."
		}
		fmt.Fprintf(os.Stdout, "%-6.0f %-40s %s\n", h.Score, h.Repo.FullName, desc)
	}
}

func outputSearchJSON(hits []*ai.SearchHit) error {
	type jsonHit struct {
		Score    float64 `json:"score"`
		FullName string  `json:"full_name"`
		Language string  `json:"language"`
		Stars    int     `json:"stars"`
		Category string  `json:"category"`
		Summary  string  `json:"summary"`
	}
	var out []jsonHit
	for _, h := range hits {
		cat := h.Repo.CustomCategory
		if cat == "" {
			cat = h.Repo.AICategory
		}
		out = append(out, jsonHit{
			Score:    h.Score,
			FullName: h.Repo.FullName,
			Language: h.Repo.Language,
			Stars:    h.Repo.StargazersCount,
			Category: cat,
			Summary:  h.Repo.AISummary,
		})
	}
	if out == nil {
		out = []jsonHit{}
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, string(data))
	return nil
}
```

- [ ] **Step 4: 运行测试验证通过**

Run: `go test ./internal/cli/ -run TestFilterAndSort -v`
Expected: 全部 PASS

- [ ] **Step 5: 验证构建**

Run: `go build -o /tmp/starman ./cmd/starman && /tmp/starman search --help`
Expected: 帮助中显示 `--json`、`--limit`、`--lang`、`--category`、`--sort` flags

- [ ] **Step 6: 提交**

```bash
git add internal/cli/search.go internal/cli/search_test.go
git commit -m "feat: enhance search with filters, sorting, and JSON output"
```

---

### Task 4: GitHub 客户端新增方法

**Files:**
- Modify: `internal/github/operations.go`
- Modify: `internal/github/operations_test.go`

**Interfaces:**
- Consumes: `github.Client`（现有）、go-github `Repositories.GetContents`、`Search.Repositories`
- Produces: `(*Client).ListReadmeVariants(ctx, owner, repo) ([]string, error)`、`(*Client).GetContentFile(ctx, owner, repo, path) (string, error)`、`(*Client).SearchRepositories(ctx, query) ([]*Repository, error)`

- [ ] **Step 1: 编写失败测试**

Append to `internal/github/operations_test.go` (add `"strings"` to imports if missing):
```go
func TestListReadmeVariants(t *testing.T) {
	server, c := mockOpsServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/owner/repo/contents" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]*gh.RepositoryContent{
				{Name: gh.Ptr("README.md"), Type: gh.Ptr("file")},
				{Name: gh.Ptr("README_zh.md"), Type: gh.Ptr("file")},
				{Name: gh.Ptr("main.go"), Type: gh.Ptr("file")},
				{Name: gh.Ptr("docs"), Type: gh.Ptr("dir")},
			})
			return
		}
		http.Error(w, "not found", 404)
	})
	defer server.Close()
	variants, err := c.ListReadmeVariants(context.Background(), "owner", "repo")
	if err != nil {
		t.Fatal(err)
	}
	if len(variants) != 2 {
		t.Fatalf("expected 2 README variants, got %d: %v", len(variants), variants)
	}
}

func TestGetContentFile(t *testing.T) {
	content := base64.StdEncoding.EncodeToString([]byte("# Hello World"))
	server, c := mockOpsServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&gh.RepositoryContent{
			Content:  gh.Ptr(content),
			Encoding: gh.Ptr("base64"),
		})
	})
	defer server.Close()
	got, err := c.GetContentFile(context.Background(), "owner", "repo", "README_zh.md")
	if err != nil {
		t.Fatal(err)
	}
	if got != "# Hello World" {
		t.Fatalf("expected '# Hello World', got %s", got)
	}
}

func TestSearchRepositories(t *testing.T) {
	server, c := mockOpsServer(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/search/repositories") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(&gh.RepositoriesSearchResult{
				Repositories: []*gh.Repository{
					{ID: gh.Ptr(int64(1)), FullName: gh.Ptr("owner/repo1"), Name: gh.Ptr("repo1"), StargazersCount: gh.Ptr(100)},
					{ID: gh.Ptr(int64(2)), FullName: gh.Ptr("owner/repo2"), Name: gh.Ptr("repo2"), StargazersCount: gh.Ptr(50)},
				},
			})
			return
		}
		http.Error(w, "not found", 404)
	})
	defer server.Close()
	repos, err := c.SearchRepositories(context.Background(), "stars:>1000")
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(repos))
	}
	if repos[0].FullName != "owner/repo1" {
		t.Fatalf("expected owner/repo1, got %s", repos[0].FullName)
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `go test ./internal/github/ -run "TestListReadmeVariants|TestGetContentFile|TestSearchRepositories" -v`
Expected: FAIL with methods not defined

- [ ] **Step 3: 实现三个方法**

Append to `internal/github/operations.go` (add `"encoding/base64"`, `"regexp"`, `"strings"` to imports if missing):
```go
var readmeVariantRe = regexp.MustCompile(`(?i)^readme([._-]?([a-z]{2})(?:-[a-z]{2})?)?\.(md|txt|markdown|rst)$`)

func (c *Client) ListReadmeVariants(ctx context.Context, owner, repo string) ([]string, error) {
	_, contents, _, err := c.client.Repositories.GetContents(ctx, owner, repo, "", nil)
	if err != nil {
		return nil, fmt.Errorf("list contents %s/%s: %w", owner, repo, err)
	}
	var variants []string
	for _, content := range contents {
		name := content.GetName()
		if isReadmeVariant(name) {
			variants = append(variants, name)
		}
	}
	return variants, nil
}

func isReadmeVariant(name string) bool {
	lower := strings.ToLower(name)
	if lower == "readme" {
		return true
	}
	return readmeVariantRe.MatchString(lower)
}

func (c *Client) GetContentFile(ctx context.Context, owner, repo, path string) (string, error) {
	content, _, _, err := c.client.Repositories.GetContents(ctx, owner, repo, path, nil)
	if err != nil {
		return "", fmt.Errorf("get content %s/%s/%s: %w", owner, repo, path, err)
	}
	decoded, err := base64.StdEncoding.DecodeString(content.GetContent())
	if err != nil {
		return content.GetContent(), nil
	}
	return string(decoded), nil
}

func (c *Client) SearchRepositories(ctx context.Context, query string) ([]*Repository, error) {
	opts := &gh.SearchOptions{
		Sort:        "stars",
		Order:       "desc",
		ListOptions: gh.ListOptions{PerPage: 50},
	}
	result, _, err := c.client.Search.Repositories(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("search repositories: %w", err)
	}
	var repos []*Repository
	for _, r := range result.Repositories {
		repos = append(repos, convertRepo(r))
	}
	return repos, nil
}
```

- [ ] **Step 4: 运行测试验证通过**

Run: `go test ./internal/github/ -run "TestListReadmeVariants|TestGetContentFile|TestSearchRepositories" -v`
Expected: 全部 PASS

- [ ] **Step 5: 运行全部 github 包测试确保不破坏**

Run: `go test ./internal/github/ -v`
Expected: 全部 PASS

- [ ] **Step 6: 提交**

```bash
git add internal/github/operations.go internal/github/operations_test.go
git commit -m "feat: add ListReadmeVariants, GetContentFile, SearchRepositories to github client"
```

---

### Task 5: info 仓库详情命令

**Files:**
- Create: `internal/cli/info.go`
- Modify: `internal/cli/root.go` (注册命令)

**Interfaces:**
- Consumes: `store.Store.GetRepository`、`github.Client.GetReadme`/`ListReadmeVariants`/`GetContentFile`（Task 4）
- Produces: `newInfoCmd() *cobra.Command`

- [ ] **Step 1: 编写 info 命令**

`internal/cli/info.go`:
```go
package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/morehao/starman/internal/github"
	"github.com/morehao/starman/internal/store"
	"github.com/spf13/cobra"
)

func newInfoCmd() *cobra.Command {
	var readme bool
	var readmeVariant string
	cmd := &cobra.Command{
		Use:   "info <fullName>",
		Short: "Show details of a repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore()
			if err != nil {
				return err
			}
			defer s.Close()
			ctx := context.Background()
			repo, err := s.GetRepository(ctx, args[0])
			if err != nil {
				return fmt.Errorf("repository %s not found in local DB, run 'starman sync' first: %w", args[0], err)
			}
			printRepoInfo(repo)
			if readme {
				cfg, _, err := loadConfig(cmd)
				if err != nil {
					return err
				}
				token := resolveGitHubToken(cmd, cfg)
				gh := github.New(token)
				parts := strings.SplitN(args[0], "/", 2)
				if readmeVariant != "" {
					content, err := gh.GetContentFile(ctx, parts[0], parts[1], readmeVariant)
					if err != nil {
						return fmt.Errorf("fetch %s: %w", readmeVariant, err)
					}
					fmt.Fprintf(os.Stdout, "\n--- README (%s) ---\n%s\n", readmeVariant, content)
				} else {
					content, err := gh.GetReadme(ctx, parts[0], parts[1])
					if err != nil {
						return fmt.Errorf("fetch README: %w", err)
					}
					fmt.Fprintf(os.Stdout, "\n--- README ---\n%s\n", content)
					variants, _ := gh.ListReadmeVariants(ctx, parts[0], parts[1])
					if len(variants) > 1 {
						fmt.Fprintf(os.Stdout, "\nAvailable README variants: %s\n", strings.Join(variants, ", "))
						fmt.Fprintf(os.Stdout, "Use --readme-variant <filename> to view a specific variant.\n")
					}
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&readme, "readme", false, "fetch and display README")
	cmd.Flags().StringVar(&readmeVariant, "readme-variant", "", "specific README variant file (e.g. README_zh.md)")
	return cmd
}

func printRepoInfo(r *store.Repository) {
	fmt.Fprintf(os.Stdout, "Repository:  %s\n", r.FullName)
	fmt.Fprintf(os.Stdout, "URL:         %s\n", r.URL)
	fmt.Fprintf(os.Stdout, "Language:    %s\n", r.Language)
	fmt.Fprintf(os.Stdout, "Stars:       %d    Forks: %d\n", r.StargazersCount, r.ForksCount)
	if len(r.Topics) > 0 {
		fmt.Fprintf(os.Stdout, "Topics:      %s\n", strings.Join(r.Topics, ", "))
	}
	if r.StarredAt != "" {
		fmt.Fprintf(os.Stdout, "Starred At:  %s\n", r.StarredAt[:10])
	}
	fmt.Fprintf(os.Stdout, "\n")
	if r.AISummary != "" {
		fmt.Fprintf(os.Stdout, "AI Summary:  %s\n", r.AISummary)
		fmt.Fprintf(os.Stdout, "AI Tags:     %s\n", strings.Join(r.AITags, ", "))
		fmt.Fprintf(os.Stdout, "AI Category: %s\n", r.AICategory)
		if r.AnalyzedAt != nil {
			fmt.Fprintf(os.Stdout, "Analyzed At: %s\n", r.AnalyzedAt.Format("2006-01-02"))
		}
	} else {
		fmt.Fprintf(os.Stdout, "AI Summary:  (not analyzed, run 'starman analyze --repo %s')\n", r.FullName)
	}
	fmt.Fprintf(os.Stdout, "\n")
	if r.CustomCategory != "" {
		fmt.Fprintf(os.Stdout, "Custom Category: %s\n", r.CustomCategory)
	} else {
		fmt.Fprintf(os.Stdout, "Custom Category: (none)\n")
	}
	if len(r.CustomTags) > 0 {
		fmt.Fprintf(os.Stdout, "Custom Tags:     %s\n", strings.Join(r.CustomTags, ", "))
	} else {
		fmt.Fprintf(os.Stdout, "Custom Tags:     (none)\n")
	}
	fmt.Fprintf(os.Stdout, "Category Locked: %v\n", r.CategoryLocked)
}
```

- [ ] **Step 2: 在 root.go 中注册命令**

Modify `internal/cli/root.go`, 添加：
```go
	root.AddCommand(newInfoCmd())
```

- [ ] **Step 3: 验证构建**

Run: `go build -o /tmp/starman ./cmd/starman && /tmp/starman info --help`
Expected: 输出 info 命令帮助，显示 `--readme` 和 `--readme-variant` flags

- [ ] **Step 4: 提交**

```bash
git add internal/cli/info.go internal/cli/root.go
git commit -m "feat: add info command for repository details"
```

---

### Task 6: tag 标签管理命令

**Files:**
- Create: `internal/cli/tag.go`
- Create: `internal/cli/tag_test.go`
- Modify: `internal/cli/root.go` (注册命令)

**Interfaces:**
- Consumes: `store.Store.ListRepositories`、`store.Store.GetRepository`、`store.Store.UpdateCustomFields`、`store.Repository.CustomTags`
- Produces: `newTagCmd() *cobra.Command`、`parseTagExpr(expr string) (add, remove []string)`、`applyTags(current, add, remove []string) []string`

- [ ] **Step 1: 编写失败测试**

`internal/cli/tag_test.go`:
```go
package cli

import (
	"reflect"
	"testing"
)

func TestParseTagExpr(t *testing.T) {
	add, remove := parseTagExpr("+cli,+go,-old")
	if !reflect.DeepEqual(add, []string{"cli", "go"}) {
		t.Fatalf("expected add [cli go], got %v", add)
	}
	if !reflect.DeepEqual(remove, []string{"old"}) {
		t.Fatalf("expected remove [old], got %v", remove)
	}
}

func TestParseTagExprEmpty(t *testing.T) {
	add, remove := parseTagExpr("")
	if len(add) != 0 || len(remove) != 0 {
		t.Fatalf("expected empty add and remove, got %v / %v", add, remove)
	}
}

func TestApplyTags(t *testing.T) {
	current := []string{"go", "old", "web"}
	add := []string{"cli", "web"}
	remove := []string{"old"}
	result := applyTags(current, add, remove)
	expected := []string{"go", "web", "cli"}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestApplyTagsEmptyCurrent(t *testing.T) {
	add := []string{"cli", "go"}
	remove := []string{}
	result := applyTags(nil, add, remove)
	expected := []string{"cli", "go"}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `go test ./internal/cli/ -run "TestParseTagExpr|TestApplyTags" -v`
Expected: FAIL with functions not defined

- [ ] **Step 3: 编写 tag 命令**

`internal/cli/tag.go`:
```go
package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/morehao/starman/internal/store"
	"github.com/spf13/cobra"
)

func newTagCmd() *cobra.Command {
	var lang string
	var category string
	var addTags string
	var removeTags string
	cmd := &cobra.Command{
		Use:   "tag [fullName] [tagExpr]",
		Short: "Manage custom tags on repositories",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore()
			if err != nil {
				return err
			}
			defer s.Close()
			ctx := context.Background()
			if lang != "" || category != "" {
				return batchTag(ctx, s, lang, category, addTags, removeTags)
			}
			if len(args) < 1 {
				return fmt.Errorf("fullName required for single-repo mode, or use --lang/--category for batch mode")
			}
			if len(args) < 2 && addTags == "" && removeTags == "" {
				return fmt.Errorf("tagExpr required for single-repo mode, or use --add/--remove")
			}
			return singleTag(ctx, s, args, addTags, removeTags)
		},
	}
	cmd.Flags().StringVar(&lang, "lang", "", "batch mode: filter by language")
	cmd.Flags().StringVar(&category, "category", "", "batch mode: filter by category")
	cmd.Flags().StringVar(&addTags, "add", "", "batch mode: tags to add (comma-separated)")
	cmd.Flags().StringVar(&removeTags, "remove", "", "batch mode: tags to remove (comma-separated)")
	return cmd
}

func singleTag(ctx context.Context, s store.Store, args []string, addFlag, removeFlag string) error {
	repo, err := s.GetRepository(ctx, args[0])
	if err != nil {
		return fmt.Errorf("repository %s not found: %w", args[0], err)
	}
	var add, remove []string
	if len(args) >= 2 {
		add, remove = parseTagExpr(args[1])
	} else {
		add = parseCSV(addFlag)
		remove = parseCSV(removeFlag)
	}
	newTags := applyTags(repo.CustomTags, add, remove)
	if err := s.UpdateCustomFields(ctx, repo.ID, &store.CustomFields{
		Description:    repo.CustomDescription,
		Tags:           newTags,
		Category:       repo.CustomCategory,
		CategoryLocked: repo.CategoryLocked,
	}); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "Updated tags for %s: %s\n", repo.FullName, strings.Join(newTags, ", "))
	return nil
}

func batchTag(ctx context.Context, s store.Store, lang, category, addStr, removeStr string) error {
	repos, err := s.ListRepositories(ctx)
	if err != nil {
		return err
	}
	add := parseCSV(addStr)
	remove := parseCSV(removeStr)
	count := 0
	for _, r := range repos {
		if lang != "" && !strings.EqualFold(r.Language, lang) {
			continue
		}
		if category != "" {
			cat := r.CustomCategory
			if cat == "" {
				cat = r.AICategory
			}
			if !strings.EqualFold(cat, category) {
				continue
			}
		}
		newTags := applyTags(r.CustomTags, add, remove)
		if err := s.UpdateCustomFields(ctx, r.ID, &store.CustomFields{
			Description:    r.CustomDescription,
			Tags:           newTags,
			Category:       r.CustomCategory,
			CategoryLocked: r.CategoryLocked,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to update %s: %v\n", r.FullName, err)
			continue
		}
		count++
	}
	fmt.Fprintf(os.Stdout, "Updated tags for %d repositories\n", count)
	return nil
}

func parseTagExpr(expr string) (add, remove []string) {
	parts := strings.Split(expr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.HasPrefix(part, "+") {
			tag := strings.TrimSpace(part[1:])
			if tag != "" {
				add = append(add, tag)
			}
		} else if strings.HasPrefix(part, "-") {
			tag := strings.TrimSpace(part[1:])
			if tag != "" {
				remove = append(remove, tag)
			}
		} else {
			add = append(add, part)
		}
	}
	return add, remove
}

func applyTags(current, add, remove []string) []string {
	set := make(map[string]bool)
	for _, t := range current {
		set[t] = true
	}
	for _, t := range add {
		set[t] = true
	}
	for _, t := range remove {
		delete(set, t)
	}
	result := make([]string, 0, len(set))
	for t := range set {
		result = append(result, t)
	}
	return result
}

func parseCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
```

- [ ] **Step 4: 在 root.go 中注册命令**

Modify `internal/cli/root.go`, 添加：
```go
	root.AddCommand(newTagCmd())
```

- [ ] **Step 5: 运行测试验证通过**

Run: `go test ./internal/cli/ -run "TestParseTagExpr|TestApplyTags" -v`
Expected: 全部 PASS

- [ ] **Step 6: 验证构建**

Run: `go build -o /tmp/starman ./cmd/starman && /tmp/starman tag --help`
Expected: 输出 tag 命令帮助

- [ ] **Step 7: 提交**

```bash
git add internal/cli/tag.go internal/cli/tag_test.go internal/cli/root.go
git commit -m "feat: add tag command for custom tag management"
```

---

### Task 7: categorize 分类管理命令

**Files:**
- Create: `internal/cli/categorize.go`
- Create: `internal/cli/categorize_test.go`
- Modify: `internal/cli/root.go` (注册命令)

**Interfaces:**
- Consumes: `store.Store.ListRepositories`、`store.Store.GetRepository`、`store.Store.UpdateCustomFields`、`store.Repository.CustomCategory`/`CategoryLocked`
- Produces: `newCategorizeCmd() *cobra.Command`

- [ ] **Step 1: 编写命令**

`internal/cli/categorize.go`:
```go
package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/morehao/starman/internal/store"
	"github.com/spf13/cobra"
)

func newCategorizeCmd() *cobra.Command {
	var lang string
	var catFilter string
	var lock bool
	var unlock bool
	cmd := &cobra.Command{
		Use:   "categorize [fullName] <category>",
		Short: "Manage custom category on repositories",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore()
			if err != nil {
				return err
			}
			defer s.Close()
			ctx := context.Background()
			if lock && unlock {
				return fmt.Errorf("cannot use --lock and --unlock together")
			}
			if lang != "" || catFilter != "" {
				if len(args) < 1 {
					return fmt.Errorf("category required as positional argument")
				}
				return batchCategorize(ctx, s, args[0], lang, catFilter, lock, unlock)
			}
			if len(args) < 2 {
				return fmt.Errorf("fullName and category required for single-repo mode, or use --lang for batch mode")
			}
			return singleCategorize(ctx, s, args[0], args[1], lock, unlock)
		},
	}
	cmd.Flags().StringVar(&lang, "lang", "", "batch mode: filter by language")
	cmd.Flags().StringVar(&catFilter, "cat-filter", "", "batch mode: filter by existing category")
	cmd.Flags().BoolVar(&lock, "lock", false, "lock category (prevent analyze from overwriting)")
	cmd.Flags().BoolVar(&unlock, "unlock", false, "unlock category")
	return cmd
}

func singleCategorize(ctx context.Context, s store.Store, fullName, cat string, lock, unlock bool) error {
	repo, err := s.GetRepository(ctx, fullName)
	if err != nil {
		return fmt.Errorf("repository %s not found: %w", fullName, err)
	}
	locked := repo.CategoryLocked
	if lock {
		locked = true
	}
	if unlock {
		locked = false
	}
	if err := s.UpdateCustomFields(ctx, repo.ID, &store.CustomFields{
		Description:    repo.CustomDescription,
		Tags:           repo.CustomTags,
		Category:       cat,
		CategoryLocked: locked,
	}); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "Set category '%s' for %s (locked=%v)\n", cat, fullName, locked)
	return nil
}

func batchCategorize(ctx context.Context, s store.Store, cat, lang, catFilter string, lock, unlock bool) error {
	repos, err := s.ListRepositories(ctx)
	if err != nil {
		return err
	}
	count := 0
	for _, r := range repos {
		if lang != "" && !strings.EqualFold(r.Language, lang) {
			continue
		}
		if catFilter != "" {
			existing := r.CustomCategory
			if existing == "" {
				existing = r.AICategory
			}
			if !strings.EqualFold(existing, catFilter) {
				continue
			}
		}
		locked := r.CategoryLocked
		if lock {
			locked = true
		}
		if unlock {
			locked = false
		}
		if err := s.UpdateCustomFields(ctx, r.ID, &store.CustomFields{
			Description:    r.CustomDescription,
			Tags:           r.CustomTags,
			Category:       cat,
			CategoryLocked: locked,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to update %s: %v\n", r.FullName, err)
			continue
		}
		count++
	}
	fmt.Fprintf(os.Stdout, "Set category '%s' for %d repositories\n", cat, count)
	return nil
}
```

- [ ] **Step 2: 在 root.go 中注册命令**

Modify `internal/cli/root.go`, 添加：
```go
	root.AddCommand(newCategorizeCmd())
```

- [ ] **Step 3: 验证构建**

Run: `go build -o /tmp/starman ./cmd/starman && /tmp/starman categorize --help`
Expected: 输出 categorize 命令帮助

- [ ] **Step 4: 验证 --lock 与 --unlock 互斥**

Run: `/tmp/starman categorize owner/repo "dev-tools" --lock --unlock 2>&1`
Expected: 报错 `cannot use --lock and --unlock together`

- [ ] **Step 5: 提交**

```bash
git add internal/cli/categorize.go internal/cli/root.go
git commit -m "feat: add categorize command for custom category management"
```

---

### Task 8: discovery 包 — trending 后端逻辑

**Files:**
- Create: `internal/discovery/trending.go`
- Create: `internal/discovery/trending_test.go`

**Interfaces:**
- Consumes: `github.Client.GetRepository`（Task 4）、`github.Client.SearchRepositories`（Task 4）
- Produces: `discovery.Service`、`discovery.NewService(gh *github.Client) *Service`、`discovery.TrendingRepo`、`discovery.TrendingOpts`、`(*Service).Trending(ctx, opts) ([]*TrendingRepo, error)`、`parseRSS(xmlData []byte) ([]*TrendingRepo, error)`

- [ ] **Step 1: 编写失败测试**

`internal/discovery/trending_test.go`:
```go
package discovery

import (
	"context"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"testing"
)

const sampleRSS = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>GitHub Trending</title>
    <item>
      <title>owner1/repo1</title>
      <link>https://github.com/owner1/repo1</link>
      <description>⭐ 1,234 | 🍴 56 | A CLI tool for managing stars</description>
    </item>
    <item>
      <title>owner2/repo2</title>
      <link>https://github.com/owner2/repo2</link>
      <description>⭐ 5,678 | 🍴 90 | A web framework</description>
    </item>
  </channel>
</rss>`

func TestParseRSS(t *testing.T) {
	repos, err := parseRSS([]byte(sampleRSS))
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(repos))
	}
	if repos[0].FullName != "owner1/repo1" {
		t.Fatalf("expected owner1/repo1, got %s", repos[0].FullName)
	}
	if repos[0].Stars != 1234 {
		t.Fatalf("expected 1234 stars, got %d", repos[0].Stars)
	}
	if repos[0].Forks != 56 {
		t.Fatalf("expected 56 forks, got %d", repos[0].Forks)
	}
	if repos[1].Stars != 5678 {
		t.Fatalf("expected 5678 stars, got %d", repos[1].Stars)
	}
}

func TestParseRSSEmpty(t *testing.T) {
	repos, err := parseRSS([]byte(`<?xml version="1.0"?><rss><channel></channel></rss>`))
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 0 {
		t.Fatalf("expected 0 repos, got %d", len(repos))
	}
}

func TestTrendingViaRSS(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(sampleRSS))
	}))
	defer srv.Close()
	svc := &Service{rssURL: srv.URL, http: &http.Client{}}
	repos, err := svc.trendingViaRSS(context.Background(), TrendingOpts{Since: "weekly", Top: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(repos))
	}
	if repos[0].Rank != 1 {
		t.Fatalf("expected rank 1, got %d", repos[0].Rank)
	}
}

func TestTrendingRSSFallbackOnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", 500)
	}))
	defer srv.Close()
	svc := &Service{rssURL: srv.URL, http: &http.Client{}}
	_, err := svc.trendingViaRSS(context.Background(), TrendingOpts{Since: "weekly", Top: 10})
	if err == nil {
		t.Fatal("expected error on RSS failure")
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `go test ./internal/discovery/ -v`
Expected: FAIL with package not found / functions not defined

- [ ] **Step 3: 编写 discovery 包**

`internal/discovery/trending.go`:
```go
package discovery

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/morehao/starman/internal/github"
)

type TrendingRepo struct {
	Rank        int
	FullName    string
	Description string
	URL         string
	Stars       int
	Forks       int
	Language    string
	Topics      []string
}

type TrendingOpts struct {
	Since  string
	Lang   string
	Top    int
	Source string
}

type Service struct {
	gh     *github.Client
	http   *http.Client
	rssURL string
}

func NewService(gh *github.Client) *Service {
	return &Service{
		gh:   gh,
		http: &http.Client{Timeout: 30 * time.Second},
	}
}

var rssURLMap = map[string]string{
	"daily":   "https://mshibanami.github.io/GitHubTrendingRSS/daily/all.xml",
	"weekly":  "https://mshibanami.github.io/GitHubTrendingRSS/weekly/all.xml",
	"monthly": "https://mshibanami.github.io/GitHubTrendingRSS/monthly/all.xml",
}

func (s *Service) Trending(ctx context.Context, opts TrendingOpts) ([]*TrendingRepo, error) {
	if opts.Source == "search" {
		return s.trendingViaSearch(ctx, opts)
	}
	return s.trendingViaRSS(ctx, opts)
}

type rssFeed struct {
	XMLName xml.Name    `xml:"rss"`
	Channel rssChannel  `xml:"channel"`
}

type rssChannel struct {
	Items []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
}

var (
	starsRe = regexp.MustCompile(`⭐\s*([\d,]+)`)
	forksRe = regexp.MustCompile(`🍴\s*([\d,]+)`)
	linkRe  = regexp.MustCompile(`github\.com/([^/]+)/([^/?#]+)`)
)

func parseRSS(xmlData []byte) ([]*TrendingRepo, error) {
	var feed rssFeed
	if err := xml.Unmarshal(xmlData, &feed); err != nil {
		return nil, fmt.Errorf("parse RSS XML: %w", err)
	}
	var repos []*TrendingRepo
	for i, item := range feed.Channel.Items {
		repo := parseRSSItem(item, i+1)
		if repo != nil {
			repos = append(repos, repo)
		}
	}
	return repos, nil
}

func parseRSSItem(item rssItem, rank int) *TrendingRepo {
	match := linkRe.FindStringSubmatch(item.Link)
	if len(match) < 3 {
		match = linkRe.FindStringSubmatch(item.Title)
		if len(match) < 3 {
			return nil
		}
	}
	owner := match[1]
	repoName := match[2]
	desc := cleanDescription(item.Description)
	stars := extractNumber(starsRe, item.Description)
	forks := extractNumber(forksRe, item.Description)
	return &TrendingRepo{
		Rank:        rank,
		FullName:    owner + "/" + repoName,
		Description: desc,
		URL:         item.Link,
		Stars:       stars,
		Forks:       forks,
	}
}

func cleanDescription(desc string) string {
	desc = strings.TrimSpace(desc)
	desc = regexp.MustCompile(`⭐\s*[\d,]+\s*\|\s*🍴\s*[\d,]+\s*\|?\s*`).ReplaceAllString(desc, "")
	return strings.TrimSpace(desc)
}

func extractNumber(re *regexp.Regexp, text string) int {
	match := re.FindStringSubmatch(text)
	if len(match) < 2 {
		return 0
	}
	num := 0
	for _, c := range match[1] {
		if c >= '0' && c <= '9' {
			num = num*10 + int(c-'0')
		}
	}
	return num
}

func (s *Service) trendingViaRSS(ctx context.Context, opts TrendingOpts) ([]*TrendingRepo, error) {
	rssURL := s.rssURL
	if rssURL == "" {
		rssURL = rssURLMap[opts.Since]
		if rssURL == "" {
			rssURL = rssURLMap["weekly"]
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rssURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create RSS request: %w", err)
	}
	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml")
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch RSS: %w (try --source search as fallback)", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("RSS fetch failed: %d (try --source search as fallback)", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read RSS body: %w", err)
	}
	repos, err := parseRSS(data)
	if err != nil {
		return nil, err
	}
	if opts.Top > 0 && opts.Top < len(repos) {
		repos = repos[:opts.Top]
	}
	if err := s.enrichRepos(ctx, repos); err != nil {
		fmt.Printf("enrich warning: %v\n", err)
	}
	if opts.Lang != "" {
		repos = filterByLang(repos, opts.Lang)
	}
	for i, r := range repos {
		r.Rank = i + 1
	}
	return repos, nil
}

func (s *Service) enrichRepos(ctx context.Context, repos []*TrendingRepo) error {
	for _, r := range repos {
		parts := strings.SplitN(r.FullName, "/", 2)
		if len(parts) != 2 {
			continue
		}
		details, err := s.gh.GetRepository(ctx, parts[0], parts[1])
		if err != nil {
			continue
		}
		r.Language = details.Language
		r.Topics = details.Topics
		if details.StargazersCount > 0 {
			r.Stars = details.StargazersCount
		}
		if details.ForksCount > 0 {
			r.Forks = details.ForksCount
		}
		if details.Description != "" {
			r.Description = details.Description
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(80 * time.Millisecond):
		}
	}
	return nil
}

func filterByLang(repos []*TrendingRepo, lang string) []*TrendingRepo {
	var filtered []*TrendingRepo
	for _, r := range repos {
		if strings.EqualFold(r.Language, lang) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func (s *Service) trendingViaSearch(ctx context.Context, opts TrendingOpts) ([]*TrendingRepo, error) {
	days := 7
	switch opts.Since {
	case "daily":
		days = 7
	case "weekly":
		days = 30
	case "monthly":
		days = 90
	}
	since := time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02")
	query := fmt.Sprintf("stars:>1000 created:>%s", since)
	ghRepos, err := s.gh.SearchRepositories(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("search repositories: %w", err)
	}
	top := opts.Top
	if top <= 0 {
		top = 20
	}
	limit := top
	if limit > len(ghRepos) {
		limit = len(ghRepos)
	}
	var repos []*TrendingRepo
	for i := 0; i < limit; i++ {
		r := ghRepos[i]
		repos = append(repos, &TrendingRepo{
			Rank:        i + 1,
			FullName:    r.FullName,
			Description: r.Description,
			URL:         r.URL,
			Stars:       r.StargazersCount,
			Forks:       r.ForksCount,
			Language:    r.Language,
			Topics:      r.Topics,
		})
	}
	if opts.Lang != "" {
		repos = filterByLang(repos, opts.Lang)
		for i, r := range repos {
			r.Rank = i + 1
		}
	}
	return repos, nil
}
```

- [ ] **Step 4: 运行测试验证通过**

Run: `go test ./internal/discovery/ -v`
Expected: 全部 PASS

- [ ] **Step 5: 验证构建**

Run: `go build ./internal/discovery/`
Expected: 无错误

- [ ] **Step 6: 提交**

```bash
git add internal/discovery/
git commit -m "feat: add discovery package with trending RSS and search fallback"
```

---

### Task 9: trending 趋势发现命令

**Files:**
- Create: `internal/cli/trending.go`
- Modify: `internal/cli/root.go` (注册命令)

**Interfaces:**
- Consumes: `discovery.Service.Trending`（Task 8）、`github.Client.Star`/`GetRepository`（现有 + Task 4）
- Produces: `newTrendingCmd() *cobra.Command`

- [ ] **Step 1: 编写 trending 命令**

`internal/cli/trending.go`:
```go
package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/morehao/starman/internal/discovery"
	"github.com/morehao/starman/internal/github"
	"github.com/spf13/cobra"
)

func newTrendingCmd() *cobra.Command {
	var since string
	var lang string
	var top int
	var source string
	var star bool
	cmd := &cobra.Command{
		Use:   "trending",
		Short: "Browse GitHub trending repositories",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, err := loadConfig(cmd)
			if err != nil {
				return err
			}
			token := resolveGitHubToken(cmd, cfg)
			if token == "" {
				return fmt.Errorf("GitHub token required")
			}
			gh := github.New(token)
			svc := discovery.NewService(gh)
			repos, err := svc.Trending(context.Background(), discovery.TrendingOpts{
				Since:  since,
				Lang:   lang,
				Top:    top,
				Source: source,
			})
			if err != nil {
				return err
			}
			if len(repos) == 0 {
				fmt.Fprintln(os.Stdout, "No trending repositories found.")
				return nil
			}
			printTrendingTable(repos)
			if star {
				return interactiveStar(gh, repos)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&since, "since", "weekly", "time range: daily|weekly|monthly")
	cmd.Flags().StringVar(&lang, "lang", "", "filter by language")
	cmd.Flags().IntVar(&top, "top", 20, "show top N repositories")
	cmd.Flags().StringVar(&source, "source", "rss", "data source: rss|search")
	cmd.Flags().BoolVar(&star, "star", false, "interactively star selected repositories")
	return cmd
}

func printTrendingTable(repos []*discovery.TrendingRepo) {
	fmt.Fprintf(os.Stdout, "%-5s %-40s %-8s %-12s %s\n", "RANK", "REPO", "STARS", "LANGUAGE", "DESCRIPTION")
	for _, r := range repos {
		desc := r.Description
		if len(desc) > 40 {
			desc = desc[:40] + "..."
		}
		lang := r.Language
		if lang == "" {
			lang = "N/A"
		}
		fmt.Fprintf(os.Stdout, "%-5d %-40s %-8d %-12s %s\n", r.Rank, r.FullName, r.Stars, lang, desc)
	}
}

func interactiveStar(gh *github.Client, repos []*discovery.TrendingRepo) error {
	fmt.Fprintln(os.Stdout, "\nEnter rank numbers to star (comma-separated, e.g. 1,3,5):")
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return nil
	}
	input := strings.TrimSpace(scanner.Text())
	if input == "" {
		return nil
	}
	parts := strings.Split(input, ",")
	starred := 0
	for _, p := range parts {
		p = strings.TrimSpace(p)
		rank, err := strconv.Atoi(p)
		if err != nil || rank < 1 || rank > len(repos) {
			fmt.Fprintf(os.Stderr, "Invalid rank: %s\n", p)
			continue
		}
		repo := repos[rank-1]
		parts := strings.SplitN(repo.FullName, "/", 2)
		if len(parts) != 2 {
			continue
		}
		if err := gh.Star(context.Background(), parts[0], parts[1]); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to star %s: %v\n", repo.FullName, err)
			continue
		}
		if r, err := gh.GetRepository(context.Background(), parts[0], parts[1]); err == nil {
			if s, err := openStore(); err == nil {
				_ = s.UpsertRepository(context.Background(), r)
				s.Close()
			}
		}
		fmt.Printf("Starred %s\n", repo.FullName)
		starred++
	}
	fmt.Fprintf(os.Stdout, "Starred %d repositories\n", starred)
	return nil
}
```

- [ ] **Step 2: 在 root.go 中注册命令**

Modify `internal/cli/root.go`, 添加：
```go
	root.AddCommand(newTrendingCmd())
```

- [ ] **Step 3: 验证构建**

Run: `go build -o /tmp/starman ./cmd/starman && /tmp/starman trending --help`
Expected: 输出 trending 命令帮助，显示 `--since`、`--lang`、`--top`、`--source`、`--star` flags

- [ ] **Step 4: 提交**

```bash
git add internal/cli/trending.go internal/cli/root.go
git commit -m "feat: add trending command for GitHub trending discovery"
```

---

### Task 10: sync --watch 自动同步

**Files:**
- Modify: `internal/cli/sync.go`

**Interfaces:**
- Consumes: 现有 sync 逻辑
- Produces: `--watch` 和 `--interval` flags

- [ ] **Step 1: 修改 sync.go，新增 watch 模式**

Replace the full content of `internal/cli/sync.go`:
```go
package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/morehao/starman/internal/github"
	"github.com/spf13/cobra"
)

func newSyncCmd() *cobra.Command {
	var fullSync bool
	var watch bool
	var interval time.Duration
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync starred repositories from GitHub to local DB",
		RunE: func(cmd *cobra.Command, args []string) error {
			if watch && interval < 5*time.Minute {
				return fmt.Errorf("--interval must be at least 5m, got %v", interval)
			}
			if err := runSync(cmd, fullSync); err != nil {
				return err
			}
			if !watch {
				return nil
			}
			return runWatch(cmd, fullSync, interval)
		},
	}
	cmd.Flags().BoolVar(&fullSync, "full", false, "full sync: delete repos no longer starred on GitHub")
	cmd.Flags().BoolVar(&watch, "watch", false, "enable watch mode: periodically sync")
	cmd.Flags().DurationVar(&interval, "interval", 30*time.Minute, "sync interval in watch mode (min 5m)")
	return cmd
}

func runSync(cmd *cobra.Command, fullSync bool) error {
	cfg, _, err := loadConfig(cmd)
	if err != nil {
		return err
	}
	token := resolveGitHubToken(cmd, cfg)
	if token == "" {
		return fmt.Errorf("GitHub token required (set via --token, config, or STARMAN_GITHUB_TOKEN)")
	}
	if cfg.GitHub.Username == "" {
		return fmt.Errorf("github.username not set in config, run 'starman config init'")
	}
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	ctx := context.Background()
	gh := github.New(token)
	repos, err := gh.ListStarred(ctx, cfg.GitHub.Username)
	if err != nil {
		return fmt.Errorf("list starred: %w", err)
	}
	if err := s.UpsertReposOnSync(ctx, repos, fullSync); err != nil {
		return fmt.Errorf("sync to db: %w", err)
	}
	s.SetSyncState(ctx, "last_sync", time.Now().UTC().Format(time.RFC3339))
	fmt.Fprintf(os.Stderr, "Synced %d repositories (fullSync=%v)\n", len(repos), fullSync)
	return nil
}

func runWatch(cmd *cobra.Command, fullSync bool, interval time.Duration) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	fmt.Fprintf(os.Stderr, "Watching with interval %v (Ctrl+C to stop)\n", interval)

	for {
		select {
		case <-ctx.Done():
			fmt.Fprintf(os.Stderr, "Stopping watch...\n")
			return nil
		case <-ticker.C:
			now := time.Now().UTC().Format(time.RFC3339)
			if err := runSync(cmd, fullSync); err != nil {
				fmt.Fprintf(os.Stderr, "[%s] Sync failed: %v\n", now, err)
			}
		}
	}
}
```

- [ ] **Step 2: 验证构建**

Run: `go build -o /tmp/starman ./cmd/starman && /tmp/starman sync --help`
Expected: 帮助中显示 `--full`、`--watch`、`--interval` flags

- [ ] **Step 3: 验证 interval 下限校验**

Run: `/tmp/starman sync --watch --interval 1m 2>&1`
Expected: 报错 `--interval must be at least 5m, got 1m0s`

- [ ] **Step 4: 运行全部测试确保不破坏**

Run: `go test ./internal/cli/ -v`
Expected: 全部 PASS（现有测试 + 新增测试）

- [ ] **Step 5: 运行全项目测试**

Run: `go test ./... -v`
Expected: 全部 PASS

- [ ] **Step 6: 提交**

```bash
git add internal/cli/sync.go
git commit -m "feat: add sync --watch mode for periodic auto-sync"
```
