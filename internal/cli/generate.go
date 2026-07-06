package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/morehao/starman/internal/generate"
	"github.com/morehao/starman/internal/github"
	"github.com/spf13/cobra"
)

func newGenerateCmd() *cobra.Command {
	var sortMode, outPath, repoName, message, templatePath string
	cmd := &cobra.Command{
		Use:   "generate [output]",
		Short: "Generate Markdown awesome list from local DB",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, err := loadConfig(cmd)
			if err != nil {
				return err
			}
			s, err := openStore()
			if err != nil {
				return err
			}
			defer func() { _ = s.Close() }()

			username := cfg.GitHub.Username
			if sortMode == "" {
				sortMode = cfg.Generate.Sort
			}
			g := generate.NewGenerator(s)
			out, err := g.Generate(context.Background(), generate.Options{
				Username: username,
				Sort:     generate.SortMode(sortMode),
				Template: templatePath,
			})
			if err != nil {
				return err
			}
			if len(out) == 0 {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "No repositories found. Run 'starman sync' first.")
				return nil
			}
			if repoName != "" {
				token := resolveGitHubToken(cmd, cfg)
				if token == "" {
					return fmt.Errorf("--repo requires GitHub token")
				}
				gh := github.New(token)
				if err := gh.UpdateReadmeFile(context.Background(), username, repoName, string(out), message); err != nil {
					return fmt.Errorf("update readme: %w", err)
				}
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Updated %s/%s README.md\n", username, repoName)
				return nil
			}
			if outPath != "" {
				return os.WriteFile(outPath, out, 0o644)
			}
			_, _ = fmt.Fprint(cmd.OutOrStdout(), string(out))
			return nil
		},
	}
	cmd.Flags().StringVarP(&sortMode, "sort", "s", "", "sort mode: language|category|flat (default from config)")
	cmd.Flags().StringVarP(&outPath, "output", "o", "", "output file path (default stdout)")
	cmd.Flags().StringVar(&repoName, "repo", "", "push to GitHub repo README (e.g. awesome-stars)")
	cmd.Flags().StringVarP(&message, "message", "m", "update stars", "commit message for --repo")
	cmd.Flags().StringVarP(&templatePath, "template", "T", "", "custom template file path")
	return cmd
}
