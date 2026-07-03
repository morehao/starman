package store

import "strings"

func ParseTagExpr(expr string) (add, remove []string) {
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

func ApplyTags(current, add, remove []string) []string {
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
	seen := make(map[string]bool)
	var result []string
	for _, t := range current {
		if set[t] && !seen[t] {
			seen[t] = true
			result = append(result, t)
		}
	}
	for _, t := range add {
		if set[t] && !seen[t] {
			seen[t] = true
			result = append(result, t)
		}
	}
	return result
}

func ParseCSV(s string) []string {
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
