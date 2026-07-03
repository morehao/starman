package generate

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/morehao/starman/internal/store"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

type SortMode string

const (
	SortLanguage SortMode = "language"
	SortCategory SortMode = "category"
	SortFlat     SortMode = "flat"
)

type Options struct {
	Username string
	Sort     SortMode
	Template string
}

type Generator struct {
	store store.Store
}

func NewGenerator(s store.Store) *Generator {
	return &Generator{store: s}
}

type renderData struct {
	UserName string
	Groups   map[string][]*store.Repository
}

func (g *Generator) Generate(ctx context.Context, opts Options) ([]byte, error) {
	repos, err := g.store.ListRepositories(ctx)
	if err != nil {
		return nil, fmt.Errorf("list repos: %w", err)
	}
	data := renderData{
		UserName: opts.Username,
		Groups:   groupRepos(repos, opts.Sort),
	}
	content, err := loadTemplate(opts)
	if err != nil {
		return nil, err
	}
	funcMap := template.FuncMap{
		"toLink":        toLink,
		"formatStars":   formatStars,
		"mergeTags":     mergeTags,
		"hasHomepage":   hasHomepage,
		"hasPlatforms":  hasPlatforms,
		"hasTags":       hasTags,
	}
	tmpl, err := template.New("starred").Funcs(funcMap).Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}
	return buf.Bytes(), nil
}

func loadTemplate(opts Options) ([]byte, error) {
	if opts.Template != "" {
		return os.ReadFile(opts.Template)
	}
	var name string
	switch opts.Sort {
	case SortCategory:
		name = "templates/by_category.tmpl"
	case SortFlat:
		name = "templates/flat.tmpl"
	default:
		name = "templates/by_language.tmpl"
	}
	return templateFS.ReadFile(name)
}

func groupRepos(repos []*store.Repository, mode SortMode) map[string][]*store.Repository {
	groups := make(map[string][]*store.Repository)
	for _, r := range repos {
		key := groupKey(r, mode)
		groups[key] = append(groups[key], r)
	}
	if mode == SortFlat {
		sort.Slice(groups["flat"], func(i, j int) bool {
			return groups["flat"][i].FullName < groups["flat"][j].FullName
		})
		return groups
	}
	for _, rs := range groups {
		sort.Slice(rs, func(i, j int) bool {
			return rs[i].FullName < rs[j].FullName
		})
	}
	return groups
}

func groupKey(r *store.Repository, mode SortMode) string {
	switch mode {
	case SortCategory:
		if r.CustomCategory != "" {
			return r.CustomCategory
		}
		if r.AICategory != "" {
			return r.AICategory
		}
		return "其他"
	case SortFlat:
		return "flat"
	default:
		if r.Language == "" {
			return "Others"
		}
		return capitalize(r.Language)
	}
}

var langMap = map[string]string{
	"javascript": "JavaScript", "typescript": "TypeScript", "html": "HTML",
	"css": "CSS", "lua": "Lua", "go": "Go", "rust": "Rust",
	"python": "Python", "java": "Java", "ruby": "Ruby",
}

func capitalize(lang string) string {
	lower := strings.ToLower(lang)
	if v, ok := langMap[lower]; ok {
		return v
	}
	c := cases.Title(language.English)
	return c.String(lower)
}

func toLink(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", "-"))
}

func formatStars(n int) string {
	if n < 1000 {
		return strconv.Itoa(n)
	}
	return fmt.Sprintf("%.1fk", float64(n)/1000)
}

func mergeTags(tagSlices ...[]string) []string {
	seen := make(map[string]struct{})
	var result []string
	for _, tags := range tagSlices {
		for _, t := range tags {
			t = strings.TrimSpace(t)
			if t == "" {
				continue
			}
			if _, ok := seen[t]; ok {
				continue
			}
			seen[t] = struct{}{}
			result = append(result, t)
		}
	}
	return result
}

func hasHomepage(url string) bool {
	return url != ""
}

func hasPlatforms(platforms []string) bool {
	return len(platforms) > 0
}

func hasTags(tags []string) bool {
	return len(tags) > 0
}
