package starssection

import (
	"sort"
	"strings"

	"github.com/morehao/starman/internal/store"
)

const (
	Uncategorized = "Uncategorized"
	Untagged      = "Untagged"
)

type GroupBucket struct {
	Count int
	Repos []*store.Repository
}

type GroupedRepos struct {
	groups map[string]GroupBucket
}

func newGroupedRepos() GroupedRepos {
	return GroupedRepos{groups: make(map[string]GroupBucket)}
}

func (g *GroupedRepos) add(key string, r *store.Repository) {
	bucket := g.groups[key]
	bucket.Repos = append(bucket.Repos, r)
	bucket.Count++
	g.groups[key] = bucket
}

func (g GroupedRepos) Get(key string) []*store.Repository {
	return g.groups[key].Repos
}

func (g GroupedRepos) GroupCount() int { return len(g.groups) }

func (g GroupedRepos) BucketCount(key string) int {
	return g.groups[key].Count
}

func (g GroupedRepos) KeysSorted() []string {
	keys := make([]string, 0, len(g.groups))
	for k := range g.groups {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i] == "" {
			return false
		}
		if keys[j] == "" {
			return true
		}
		return keys[i] < keys[j]
	})
	return keys
}

func seenAny(m map[string]bool) bool {
	for _, v := range m {
		if v {
			return true
		}
	}
	return false
}

func (g GroupedRepos) sortBucketsByRepoName() {
	for k, bucket := range g.groups {
		sort.Slice(bucket.Repos, func(i, j int) bool {
			return bucket.Repos[i].FullName < bucket.Repos[j].FullName
		})
		g.groups[k] = bucket
	}
}

func GroupByLanguage(repos []*store.Repository) GroupedRepos {
	g := newGroupedRepos()
	for _, r := range repos {
		lang := strings.TrimSpace(r.Language)
		g.add(lang, r)
	}
	g.sortBucketsByRepoName()
	return g
}

func GroupByCategory(repos []*store.Repository) GroupedRepos {
	g := newGroupedRepos()
	for _, r := range repos {
		if r.CustomCategory != "" {
			g.add(r.CustomCategory, r)
			continue
		}
		if r.AICategory != "" {
			g.add(r.AICategory, r)
			continue
		}
		g.add(Uncategorized, r)
	}
	return g
}

func GroupByTag(repos []*store.Repository) GroupedRepos {
	g := newGroupedRepos()
	for _, r := range repos {
		tags := append([]string{}, r.AITags...)
		tags = append(tags, r.CustomTags...)
		if len(tags) == 0 {
			g.add(Untagged, r)
			continue
		}
		seen := make(map[string]bool)
		for _, t := range tags {
			t = strings.TrimSpace(t)
			if t == "" || seen[t] {
				continue
			}
			seen[t] = true
			g.add(t, r)
		}
		if !seenAny(seen) {
			g.add(Untagged, r)
		}
	}
	return g
}
