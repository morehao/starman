package cli

import (
	"context"
	"fmt"

	"github.com/morehao/starman/internal/ai"
	"github.com/morehao/starman/internal/config"
	"github.com/spf13/cobra"
)

func newSearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search repos by AI-translated keywords",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, err := loadConfig(cmd)
			if err != nil {
				return err
			}
			aiKey := config.ResolveAIKey(cfg, "")
			if aiKey == "" {
				return fmt.Errorf("AI api_key required")
			}
			s, err := openStore()
			if err != nil {
				return err
			}
			defer s.Close()
			ctx := context.Background()
			repos, err := s.ListRepositories(ctx)
			if err != nil {
				return err
			}
			aiClient := ai.NewClient(cfg.AI.BaseURL, aiKey, cfg.AI.Model)
			svc := ai.NewService(aiClient, nil)
			hits, err := svc.Search(ctx, args[0], repos)
			if err != nil {
				return err
			}
			for _, h := range hits {
				fmt.Printf("%.0f\t%s\t%s\n", h.Score, h.Repo.FullName, h.Repo.Description)
			}
			return nil
		},
	}
	return cmd
}
