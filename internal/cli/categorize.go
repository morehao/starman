package cli

import (
	"context"
	"fmt"
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
				return batchCategorize(ctx, s, args[0], lang, catFilter, lock, unlock, cmd)
			}
			if len(args) < 2 {
				return fmt.Errorf("fullName and category required for single-repo mode, or use --lang for batch mode")
			}
			return singleCategorize(ctx, s, args[0], args[1], lock, unlock, cmd)
		},
	}
	cmd.Flags().StringVar(&lang, "lang", "", "batch mode: filter by language")
	cmd.Flags().StringVar(&catFilter, "cat-filter", "", "batch mode: filter by existing category")
	cmd.Flags().BoolVar(&lock, "lock", false, "lock category (prevent analyze from overwriting)")
	cmd.Flags().BoolVar(&unlock, "unlock", false, "unlock category")
	return cmd
}

func singleCategorize(ctx context.Context, s store.Store, fullName, cat string, lock, unlock bool, cmd *cobra.Command) error {
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
	fmt.Fprintf(cmd.OutOrStdout(), "Set category '%s' for %s (locked=%v)\n", cat, fullName, locked)
	return nil
}

func batchCategorize(ctx context.Context, s store.Store, cat, lang, catFilter string, lock, unlock bool, cmd *cobra.Command) error {
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
			fmt.Fprintf(cmd.ErrOrStderr(), "Failed to update %s: %v\n", r.FullName, err)
			continue
		}
		count++
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Set category '%s' for %d repositories\n", cat, count)
	return nil
}
