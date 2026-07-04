package cli

import (
	"context"
	"fmt"
	"text/tabwriter"

	"github.com/morehao/starman/internal/store"
	"github.com/spf13/cobra"
)

func newCategoryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "category",
		Short: "Manage custom categories",
	}
	cmd.AddCommand(newCategoryListCmd())
	cmd.AddCommand(newCategoryAddCmd())
	cmd.AddCommand(newCategoryEditCmd())
	cmd.AddCommand(newCategoryDeleteCmd())
	return cmd
}

func newCategoryListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all categories",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore()
			if err != nil {
				return err
			}
			defer s.Close()
			ctx := context.Background()
			cats, err := s.ListCategories(ctx, false)
			if err != nil {
				return err
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tKEYWORDS\tORDER\tTYPE")
			for _, c := range cats {
				typeStr := "自定义"
				if !c.IsCustom {
					typeStr = "内置"
				}
				kw := ""
				if len(c.Keywords) > 0 {
					kw = fmt.Sprintf("%v", c.Keywords)
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n", c.ID, c.Name, kw, c.SortOrder, typeStr)
			}
			w.Flush()
			return nil
		},
	}
}

func newCategoryAddCmd() *cobra.Command {
	var name string
	var keywords string
	var sortOrder int
	cmd := &cobra.Command{
		Use:   "add <id>",
		Short: "Add a custom category",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := store.Slugify(args[0])
			if id == "" {
				return fmt.Errorf("invalid category id: %s", args[0])
			}
			displayName := name
			if displayName == "" {
				displayName = id
			}
			var kws []string
			if keywords != "" {
				kws = store.ParseCSV(keywords)
			}
			if sortOrder == 0 {
				s, err := openStore()
				if err != nil {
					return err
				}
				cats, err := s.ListCategories(context.Background(), false)
				s.Close()
				if err != nil {
					return err
				}
				maxOrder := 0
				for _, c := range cats {
					if c.SortOrder > maxOrder {
						maxOrder = c.SortOrder
					}
				}
				sortOrder = maxOrder + 1
			}
			s, err := openStore()
			if err != nil {
				return err
			}
			defer s.Close()
			if err := s.UpsertCategory(context.Background(), &store.Category{
				ID:        id,
				Name:      displayName,
				Keywords:  kws,
				SortOrder: sortOrder,
				IsCustom:  true,
			}); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Added category '%s' (%s)\n", displayName, id)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "display name (defaults to id)")
	cmd.Flags().StringVar(&keywords, "keywords", "", "comma-separated keywords")
	cmd.Flags().IntVar(&sortOrder, "sort-order", 0, "sort order (default appends to end)")
	return cmd
}

func newCategoryEditCmd() *cobra.Command {
	var name string
	var keywords string
	var sortOrder int
	cmd := &cobra.Command{
		Use:   "edit <id>",
		Short: "Edit a category",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore()
			if err != nil {
				return err
			}
			defer s.Close()
			ctx := context.Background()
			cats, err := s.ListCategories(ctx, false)
			if err != nil {
				return err
			}
			var target *store.Category
			for _, c := range cats {
				if c.ID == args[0] {
					target = c
					break
				}
			}
			if target == nil {
				return fmt.Errorf("category %s not found", args[0])
			}
			if name != "" {
				target.Name = name
			}
			if cmd.Flags().Changed("keywords") {
				target.Keywords = store.ParseCSV(keywords)
			}
			if cmd.Flags().Changed("sort-order") {
				target.SortOrder = sortOrder
			}
			if err := s.UpsertCategory(ctx, target); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Updated category '%s'\n", target.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "new display name")
	cmd.Flags().StringVar(&keywords, "keywords", "", "new keywords (comma-separated)")
	cmd.Flags().IntVar(&sortOrder, "sort-order", 0, "new sort order")
	return cmd
}

func newCategoryDeleteCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a custom category",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore()
			if err != nil {
				return err
			}
			defer s.Close()
			ctx := context.Background()
			cats, err := s.ListCategories(ctx, false)
			if err != nil {
				return err
			}
			var target *store.Category
			for _, c := range cats {
				if c.ID == args[0] {
					target = c
					break
				}
			}
			if target == nil {
				return fmt.Errorf("category %s not found", args[0])
			}
			if !target.IsCustom {
				return fmt.Errorf("cannot delete built-in category %s", args[0])
			}
			if !force {
				fmt.Fprintf(cmd.ErrOrStderr(), "This will delete category '%s' (%s) and clear category from all associated repos.\nUse --force to confirm.\n", target.Name, target.ID)
				return nil
			}
			affected, err := s.DeleteCategory(ctx, args[0])
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted category '%s', cleared category from %d repos\n", args[0], affected)
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "skip confirmation")
	return cmd
}
