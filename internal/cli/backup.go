package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/morehao/starman/internal/backup"
	"github.com/morehao/starman/internal/config"
	"github.com/morehao/starman/internal/github"
	"github.com/spf13/cobra"
)

func newBackupCmd() *cobra.Command {
	var repoName, msg string
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Backup and restore data",
		RunE: func(cmd *cobra.Command, args []string) error {
			if repoName == "" {
				return cmd.Help()
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
			s, err := openStore()
			if err != nil {
				return err
			}
			defer s.Close()

			ctx := context.Background()
			data, err := backup.ExportJSON(ctx, s)
			if err != nil {
				return err
			}

			date := time.Now().UTC().Format("2006-01-02")
			filePath := "starman-backup/" + date + ".json"
			if msg == "" {
				msg = "backup starman data " + date
			}

			gh := github.New(token)
			if err := gh.CommitFile(ctx, cfg.GitHub.Username, repoName, filePath, data, msg); err != nil {
				return fmt.Errorf("push backup to %s/%s: %w", cfg.GitHub.Username, repoName, err)
			}
			fmt.Printf("Pushed backup to %s/%s/%s\n", cfg.GitHub.Username, repoName, filePath)
			return nil
		},
	}
	cmd.AddCommand(newBackupJSONCmd())
	cmd.AddCommand(newBackupWebDAVCmd())
	cmd.Flags().StringVar(&repoName, "repo", "", "push backup to GitHub repo (e.g. awesome-stars)")
	cmd.Flags().StringVarP(&msg, "message", "m", "", "commit message (default: auto-generated)")
	return cmd
}

func newBackupJSONCmd() *cobra.Command {
	var exportFlag bool
	var importFlag, outPath string
	var mode string
	cmd := &cobra.Command{
		Use:   "json",
		Short: "Export/import JSON backup",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore()
			if err != nil {
				return err
			}
			defer s.Close()
			ctx := context.Background()
			if exportFlag {
				data, err := backup.ExportJSON(ctx, s)
				if err != nil {
					return err
				}
				if outPath != "" {
					return os.WriteFile(outPath, data, 0o644)
				}
				fmt.Print(string(data))
				return nil
			}
			if importFlag != "" {
				data, err := os.ReadFile(importFlag)
				if err != nil {
					return err
				}
				return backup.ImportJSON(ctx, s, data, backup.ImportMode(mode))
			}
			return fmt.Errorf("use --export or --import <file>")
		},
	}
	cmd.Flags().BoolVar(&exportFlag, "export", false, "export to JSON")
	cmd.Flags().StringVar(&importFlag, "import", "", "import from JSON file")
	cmd.Flags().StringVarP(&outPath, "output", "o", "", "output file (default stdout)")
	cmd.Flags().StringVar(&mode, "mode", "merge", "import mode: merge|replace")
	return cmd
}

func newBackupWebDAVCmd() *cobra.Command {
	var push, pull, test bool
	cmd := &cobra.Command{
		Use:   "webdav",
		Short: "Push/pull backup via WebDAV",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, err := loadConfig(cmd)
			if err != nil {
				return err
			}
			password := config.ResolveWebDAVPassword(cfg)
			wc := backup.NewWebDAVClient(cfg.WebDAV.URL, cfg.WebDAV.Username, password)
			ctx := context.Background()
			if test {
				return wc.Test(ctx)
			}
			s, err := openStore()
			if err != nil {
				return err
			}
			defer s.Close()
			path := cfg.WebDAV.Path + "/starman-backup-" + time.Now().UTC().Format("2006-01-02") + ".json"
			if push {
				data, err := backup.ExportJSON(ctx, s)
				if err != nil {
					return err
				}
				if err := wc.Push(ctx, path, data); err != nil {
					return err
				}
				fmt.Printf("Pushed backup to %s\n", path)
				return nil
			}
			if pull {
				data, err := wc.Pull(ctx, path)
				if err != nil {
					return err
				}
				return backup.ImportJSON(ctx, s, data, backup.ImportMerge)
			}
			return fmt.Errorf("use --push, --pull, or --test")
		},
	}
	cmd.Flags().BoolVar(&push, "push", false, "push backup to WebDAV")
	cmd.Flags().BoolVar(&pull, "pull", false, "pull latest backup from WebDAV")
	cmd.Flags().BoolVar(&test, "test", false, "test WebDAV connection")
	return cmd
}
