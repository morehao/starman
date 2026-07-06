package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/morehao/starman/internal/discovery"
	"github.com/morehao/starman/internal/github"
	"github.com/spf13/cobra"
)

func newTrendingCmd() *cobra.Command {
	var since string
	var lang string
	var top int
	var source string
	var star bool
	cmd := &cobra.Command{
		Use:   "trending",
		Short: "Browse GitHub trending repositories",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, err := loadConfig(cmd)
			if err != nil {
				return err
			}
			token := resolveGitHubToken(cmd, cfg)
			if token == "" {
				return fmt.Errorf("GitHub token required")
			}
			gh := github.New(token)
			svc := discovery.NewService(gh)
			repos, err := svc.Trending(context.Background(), discovery.TrendingOpts{
				Since:  since,
				Lang:   lang,
				Top:    top,
				Source: source,
			})
			if err != nil {
				return err
			}
			if len(repos) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No trending repositories found.")
				return nil
			}
			printTrendingTable(cmd, repos)
			if star {
				return interactiveStar(cmd, gh, repos)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&since, "since", "weekly", "time range: daily|weekly|monthly")
	cmd.Flags().StringVar(&lang, "lang", "", "filter by language")
	cmd.Flags().IntVar(&top, "top", 20, "show top N repositories")
	cmd.Flags().StringVar(&source, "source", "rss", "data source: rss|search")
	cmd.Flags().BoolVar(&star, "star", false, "interactively star selected repositories")
	return cmd
}

func printTrendingTable(cmd *cobra.Command, repos []*discovery.TrendingRepo) {
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-5s %-40s %-8s %-12s %s\n", "RANK", "REPO", "STARS", "LANGUAGE", "DESCRIPTION")
	for _, r := range repos {
		desc := r.Description
		if len(desc) > 40 {
			desc = desc[:40] + "..."
		}
		lang := r.Language
		if lang == "" {
			lang = "N/A"
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-5d %-40s %-8d %-12s %s\n", r.Rank, r.FullName, r.Stars, lang, desc)
	}
}

func interactiveStar(cmd *cobra.Command, gh *github.Client, repos []*discovery.TrendingRepo) error {
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\nEnter rank numbers to star (comma-separated, e.g. 1,3,5):")
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return nil
	}
	input := strings.TrimSpace(scanner.Text())
	if input == "" {
		return nil
	}
	parts := strings.Split(input, ",")
	starred := 0
	for _, p := range parts {
		p = strings.TrimSpace(p)
		rank, err := strconv.Atoi(p)
		if err != nil || rank < 1 || rank > len(repos) {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Invalid rank: %s\n", p)
			continue
		}
		repo := repos[rank-1]
		parts := strings.SplitN(repo.FullName, "/", 2)
		if len(parts) != 2 {
			continue
		}
		if err := gh.Star(context.Background(), parts[0], parts[1]); err != nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Failed to star %s: %v\n", repo.FullName, err)
			continue
		}
		if r, err := gh.GetRepository(context.Background(), parts[0], parts[1]); err == nil {
			if s, sErr := openStore(); sErr == nil {
				_ = s.UpsertRepository(context.Background(), r)
				_ = s.Close()
			}
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Starred %s\n", repo.FullName)
		starred++
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Starred %d repositories\n", starred)
	return nil
}
