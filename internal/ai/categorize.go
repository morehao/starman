package ai

import (
	"github.com/morehao/starman/internal/store"
)

func ResolveCategory(repo *store.Repository, aiTags []string, cats []*store.Category) string {
	if repo.CategoryLocked && repo.CustomCategory != "" {
		return repo.CustomCategory
	}
	if matched := matchTags(aiTags, cats, true); matched != "" {
		return matched
	}
	if matched := matchTags(aiTags, cats, false); matched != "" {
		return matched
	}
	if repo.CategoryLocked {
		if repo.AICategory != "" {
			return repo.AICategory
		}
		if repo.CustomCategory != "" {
			return repo.CustomCategory
		}
	}
	return "others"
}

func matchTags(tags []string, cats []*store.Category, customOnly bool) string {
	for _, c := range cats {
		if customOnly && !c.IsCustom {
			continue
		}
		if !customOnly && c.IsCustom {
			continue
		}
		for _, tag := range tags {
			for _, kw := range c.Keywords {
				if contains(tag, kw) || contains(kw, tag) {
					return c.Name
				}
			}
		}
	}
	return ""
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || containsFold(s, substr))
}

func containsFold(s, substr string) bool {
	sLower := toLower(s)
	subLower := toLower(substr)
	return len(sLower) >= len(subLower) && indexOf(sLower, subLower) >= 0
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		result[i] = c
	}
	return string(result)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
