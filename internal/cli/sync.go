package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/morehao/starman/internal/github"
	"github.com/spf13/cobra"
)

func newSyncCmd() *cobra.Command {
	var fullSync bool
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync starred repositories from GitHub to local DB",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, err := loadConfig(cmd)
			if err != nil {
				return err
			}
			token := resolveGitHubToken(cmd, cfg)
			if token == "" {
				return fmt.Errorf("GitHub token required (set via --token, config, or STARMAN_GITHUB_TOKEN)")
			}
			if cfg.GitHub.Username == "" {
				return fmt.Errorf("github.username not set in config, run 'starman config init'")
			}
			s, err := openStore()
			if err != nil {
				return err
			}
			defer s.Close()

			ctx := context.Background()
			fmt.Fprintf(os.Stderr, "Fetching starred repos for %s...\n", cfg.GitHub.Username)
			gh := github.New(token)
			repos, err := gh.ListStarred(ctx, cfg.GitHub.Username)
			if err != nil {
				return fmt.Errorf("list starred: %w", err)
			}
			if err := s.UpsertReposOnSync(ctx, repos, fullSync); err != nil {
				return fmt.Errorf("sync to db: %w", err)
			}
			s.SetSyncState(ctx, "last_sync", time.Now().UTC().Format(time.RFC3339))
			fmt.Fprintf(os.Stderr, "Synced %d repositories (fullSync=%v)\n", len(repos), fullSync)
			return nil
		},
	}
	cmd.Flags().BoolVar(&fullSync, "full", false, "full sync: delete repos no longer starred on GitHub")
	return cmd
}
