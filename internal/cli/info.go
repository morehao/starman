package cli

import (
	"context"
	"fmt"
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
			printRepoInfo(cmd, repo)
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
					fmt.Fprintf(cmd.OutOrStdout(), "\n--- README (%s) ---\n%s\n", readmeVariant, content)
				} else {
					content, err := gh.GetReadme(ctx, parts[0], parts[1])
					if err != nil {
						return fmt.Errorf("fetch README: %w", err)
					}
					fmt.Fprintf(cmd.OutOrStdout(), "\n--- README ---\n%s\n", content)
					variants, _ := gh.ListReadmeVariants(ctx, parts[0], parts[1])
					if len(variants) > 1 {
						fmt.Fprintf(cmd.OutOrStdout(), "\nAvailable README variants: %s\n", strings.Join(variants, ", "))
						fmt.Fprintf(cmd.OutOrStdout(), "Use --readme-variant <filename> to view a specific variant.\n")
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

func printRepoInfo(cmd *cobra.Command, r *store.Repository) {
	fmt.Fprintf(cmd.OutOrStdout(), "Repository:  %s\n", r.FullName)
	fmt.Fprintf(cmd.OutOrStdout(), "URL:         %s\n", r.URL)
	fmt.Fprintf(cmd.OutOrStdout(), "Language:    %s\n", r.Language)
	fmt.Fprintf(cmd.OutOrStdout(), "Stars:       %d    Forks: %d\n", r.StargazersCount, r.ForksCount)
	if len(r.Topics) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "Topics:      %s\n", strings.Join(r.Topics, ", "))
	}
	if r.StarredAt != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Starred At:  %s\n", r.StarredAt[:10])
	}
	fmt.Fprintf(cmd.OutOrStdout(), "\n")
	if r.AISummary != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "AI Summary:  %s\n", r.AISummary)
		fmt.Fprintf(cmd.OutOrStdout(), "AI Tags:     %s\n", strings.Join(r.AITags, ", "))
		fmt.Fprintf(cmd.OutOrStdout(), "AI Category: %s\n", r.AICategory)
		if r.AnalyzedAt != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "Analyzed At: %s\n", r.AnalyzedAt.Format("2006-01-02"))
		}
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "AI Summary:  (not analyzed, run 'starman analyze --repo %s')\n", r.FullName)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "\n")
	if r.CustomCategory != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Custom Category: %s\n", r.CustomCategory)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Custom Category: (none)\n")
	}
	if len(r.CustomTags) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "Custom Tags:     %s\n", strings.Join(r.CustomTags, ", "))
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Custom Tags:     (none)\n")
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Category Locked: %v\n", r.CategoryLocked)
}
