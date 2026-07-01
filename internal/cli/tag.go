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
	var catFilter string
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
			if lang != "" || catFilter != "" {
				return batchTag(ctx, s, lang, catFilter, addTags, removeTags)
			}
			if len(args) < 1 {
				return fmt.Errorf("fullName required for single-repo mode, or use --lang/--cat-filter for batch mode")
			}
			if len(args) < 2 && addTags == "" && removeTags == "" {
				return fmt.Errorf("tagExpr required for single-repo mode, or use --add/--remove")
			}
			return singleTag(ctx, s, args, addTags, removeTags)
		},
	}
	cmd.Flags().StringVar(&lang, "lang", "", "batch mode: filter by language")
	cmd.Flags().StringVar(&catFilter, "cat-filter", "", "batch mode: filter by existing category")
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

func batchTag(ctx context.Context, s store.Store, lang, catFilter, addStr, removeStr string) error {
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
		if catFilter != "" {
			cat := r.CustomCategory
			if cat == "" {
				cat = r.AICategory
			}
			if !strings.EqualFold(cat, catFilter) {
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
