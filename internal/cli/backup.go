package cli

import (
	"fmt"
	"path/filepath"

	"github.com/morehao/starman/internal/backup"
	"github.com/morehao/starman/internal/config"
	"github.com/morehao/starman/internal/github"
	"github.com/spf13/cobra"
)

func newBackupCmd() *cobra.Command {
	var repoName, msg string
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Backup starman.db to a GitHub repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			if repoName == "" {
				return fmt.Errorf("--repo is required")
			}
			cfg, _, err := loadConfig(cmd)
			if err != nil {
				return err
			}
			token := resolveGitHubToken(cmd, cfg)
			if token == "" {
				return fmt.Errorf("GitHub token required for --repo")
			}
			if cfg.GitHub.Username == "" {
				return fmt.Errorf("GitHub username not configured, run 'starman config init'")
			}

			dir, err := config.DefaultDir()
			if err != nil {
				return err
			}
			dbPath := filepath.Join(dir, "starman.db")
			data, err := backup.ReadDB(dbPath)
			if err != nil {
				return fmt.Errorf("read db: %w", err)
			}

			filePath := "starman-backup/starman.db"
			if msg == "" {
				msg = "backup starman data"
			}

			gh := github.New(token)
			if err := gh.CommitFile(cmd.Context(), cfg.GitHub.Username, repoName, filePath, data, msg); err != nil {
				return fmt.Errorf("push backup to %s/%s: %w", cfg.GitHub.Username, repoName, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Pushed backup to %s/%s/%s\n", cfg.GitHub.Username, repoName, filePath)
			return nil
		},
	}
	cmd.Flags().StringVar(&repoName, "repo", "", "GitHub repo to push backup to (e.g. awesome-stars)")
	cmd.Flags().StringVarP(&msg, "message", "m", "", "commit message (default: auto-generated)")
	return cmd
}
