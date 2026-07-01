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
	var watch bool
	var interval time.Duration
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync starred repositories from GitHub to local DB",
		RunE: func(cmd *cobra.Command, args []string) error {
			if watch && interval < 5*time.Minute {
				return fmt.Errorf("--interval must be at least 5m, got %v", interval)
			}
			if err := runSync(cmd, fullSync); err != nil {
				return err
			}
			if !watch {
				return nil
			}
			return runWatch(cmd, fullSync, interval)
		},
	}
	cmd.Flags().BoolVar(&fullSync, "full", false, "full sync: delete repos no longer starred on GitHub")
	cmd.Flags().BoolVar(&watch, "watch", false, "enable watch mode: periodically sync")
	cmd.Flags().DurationVar(&interval, "interval", 30*time.Minute, "sync interval in watch mode (min 5m)")
	return cmd
}

func runSync(cmd *cobra.Command, fullSync bool) error {
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
}

func runWatch(cmd *cobra.Command, fullSync bool, interval time.Duration) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	fmt.Fprintf(os.Stderr, "Watching with interval %v (Ctrl+C to stop)\n", interval)

	for {
		select {
		case <-ctx.Done():
			fmt.Fprintf(os.Stderr, "Stopping watch...\n")
			return nil
		case <-ticker.C:
			now := time.Now().UTC().Format(time.RFC3339)
			fmt.Fprintf(os.Stderr, "[%s] ", now)
			if err := runSync(cmd, fullSync); err != nil {
				fmt.Fprintf(os.Stderr, "Sync failed: %v\n", err)
			}
		}
	}
}
