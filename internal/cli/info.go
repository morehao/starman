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
