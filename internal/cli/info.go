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
			defer func() { _ = s.Close() }()
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
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\n--- README (%s) ---\n%s\n", readmeVariant, content)
				} else {
					content, err := gh.GetReadme(ctx, parts[0], parts[1])
					if err != nil {
						return fmt.Errorf("fetch README: %w", err)
					}
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\n--- README ---\n%s\n", content)
					variants, _ := gh.ListReadmeVariants(ctx, parts[0], parts[1])
					if len(variants) > 1 {
						_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nAvailable README variants: %s\n", strings.Join(variants, ", "))
						_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Use --readme-variant <filename> to view a specific variant.\n")
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
	out := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(out, "Repository:  %s\n", r.FullName)
	_, _ = fmt.Fprintf(out, "URL:         %s\n", r.URL)
	_, _ = fmt.Fprintf(out, "Language:    %s\n", r.Language)
	_, _ = fmt.Fprintf(out, "Stars:       %d    Forks: %d\n", r.StargazersCount, r.ForksCount)
	if len(r.Topics) > 0 {
		_, _ = fmt.Fprintf(out, "Topics:      %s\n", strings.Join(r.Topics, ", "))
	}
	if r.StarredAt != "" {
		_, _ = fmt.Fprintf(out, "Starred At:  %s\n", r.StarredAt[:10])
	}
	_, _ = fmt.Fprintf(out, "\n")
	if r.AISummary != "" {
		_, _ = fmt.Fprintf(out, "AI Summary:  %s\n", r.AISummary)
		_, _ = fmt.Fprintf(out, "AI Tags:     %s\n", strings.Join(r.AITags, ", "))
		_, _ = fmt.Fprintf(out, "AI Category: %s\n", r.AICategory)
		if r.AnalyzedAt != nil {
			_, _ = fmt.Fprintf(out, "Analyzed At: %s\n", r.AnalyzedAt.Format("2006-01-02"))
		}
	} else {
		_, _ = fmt.Fprintf(out, "AI Summary:  (not analyzed, run 'starman analyze --repo %s')\n", r.FullName)
	}
	_, _ = fmt.Fprintf(out, "\n")
	if r.CustomCategory != "" {
		_, _ = fmt.Fprintf(out, "Custom Category: %s\n", r.CustomCategory)
	} else {
		_, _ = fmt.Fprintf(out, "Custom Category: (none)\n")
	}
	if len(r.CustomTags) > 0 {
		_, _ = fmt.Fprintf(out, "Custom Tags:     %s\n", strings.Join(r.CustomTags, ", "))
	} else {
		_, _ = fmt.Fprintf(out, "Custom Tags:     (none)\n")
	}
	_, _ = fmt.Fprintf(out, "Category Locked: %v\n", r.CategoryLocked)
}
