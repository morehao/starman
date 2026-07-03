package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/morehao/starman/internal/github"
	"github.com/spf13/cobra"
)

func newStarCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "star <fullName>",
		Short: "Star a GitHub repository",
		Args:  cobra.ExactArgs(1),
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
			parts := strings.SplitN(args[0], "/", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid full name: %s (expected owner/repo)", args[0])
			}
			ctx := context.Background()
			if err := gh.Star(ctx, parts[0], parts[1]); err != nil {
				return err
			}
			fmt.Printf("Starred %s\n", args[0])
			if repo, err := gh.GetRepository(ctx, parts[0], parts[1]); err == nil {
				if s, err := openStore(); err == nil {
					_ = s.UpsertRepository(ctx, repo)
					s.Close()
				}
			}
			return nil
		},
	}
	return cmd
}

func newUnstarCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unstar <fullName>",
		Short: "Unstar a GitHub repository",
		Args:  cobra.ExactArgs(1),
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
			parts := strings.SplitN(args[0], "/", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid full name: %s (expected owner/repo)", args[0])
			}
			ctx := context.Background()
			if err := gh.Unstar(ctx, parts[0], parts[1]); err != nil {
				return err
			}
			fmt.Printf("Unstarred %s\n", args[0])
			s, err := openStore()
			if err != nil {
				return nil
			}
			defer s.Close()
			existing, err := s.GetRepository(ctx, args[0])
			if err != nil {
				return nil
			}
			existing.StarredAt = ""
			_ = s.UpsertRepository(ctx, existing)
			return nil
		},
	}
	return cmd
}
