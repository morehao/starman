package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/morehao/starman/internal/ai"
	"github.com/morehao/starman/internal/config"
	"github.com/morehao/starman/internal/github"
	"github.com/morehao/starman/internal/store"
	"github.com/spf13/cobra"
)

func newAnalyzeCmd() *cobra.Command {
	var all, force bool
	var repoNames []string
	var limit int
	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Analyze repos with AI to generate summaries, tags, and categories",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, err := loadConfig(cmd)
			if err != nil {
				return err
			}
			token := resolveGitHubToken(cmd, cfg)
			aiKey := config.ResolveAIKey(cfg, "")
			if aiKey == "" {
				return fmt.Errorf("AI api_key required (set via config or STARMAN_AI_API_KEY)")
			}
			s, err := openStore()
			if err != nil {
				return err
			}
			defer s.Close()
			ctx := context.Background()
			effectiveLimit := limit
			if all {
				effectiveLimit = 0
			}
			var repos []*store.Repository
			if len(repoNames) > 0 {
				for _, name := range repoNames {
					r, err := s.GetRepository(ctx, name)
					if err == nil {
						repos = append(repos, r)
					}
				}
			} else if force {
				repos, err = s.ListRepositories(ctx)
			} else {
				repos, err = s.ListUnanalyzed(ctx, effectiveLimit)
			}
			if err != nil {
				return err
			}
			if len(repos) == 0 {
				fmt.Println("No repos to analyze.")
				return nil
			}
			aiClient := ai.NewClient(cfg.AI.BaseURL, aiKey, cfg.AI.Model)
			gh := github.New(token)
			svc := ai.NewService(aiClient, gh)
			batch := ai.NewBatchAnalyzer(svc, s, gh, cfg.AI.Concurrency)
			fmt.Fprintf(os.Stderr, "Analyzing %d repos...\n", len(repos))
			result, err := batch.Run(ctx, repos, ai.BatchOpts{
				Force: force,
				Limit: effectiveLimit,
				OnProgress: func(done, total int, name string) {
					fmt.Fprintf(os.Stderr, "\r[%d/%d] %s", done, total, name)
				},
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "\nDone: %d success, %d failed, %d total\n", result.Success, result.Failed, result.Total)
			return nil
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "analyze all unanalyzed repos")
	cmd.Flags().StringSliceVar(&repoNames, "repo", nil, "specific repo full names to analyze")
	cmd.Flags().BoolVar(&force, "force", false, "force re-analyze even if already analyzed")
	cmd.Flags().IntVar(&limit, "limit", 20, "max repos to analyze (0 = no limit)")
	return cmd
}
