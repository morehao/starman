package cli

import (
	"context"
	"fmt"

	"github.com/morehao/starman/internal/github"
	"github.com/morehao/starman/internal/release"
	"github.com/morehao/starman/internal/store"
	"github.com/spf13/cobra"
)

func newReleaseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "release",
		Short: "Track repository releases",
	}
	cmd.AddCommand(newReleaseListCmd())
	cmd.AddCommand(newReleasePullCmd())
	cmd.AddCommand(newReleaseSubscribeCmd())
	cmd.AddCommand(newReleaseUnsubscribeCmd())
	return cmd
}

func newReleaseListCmd() *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List releases",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore()
			if err != nil {
				return err
			}
			defer s.Close()
			ctx := context.Background()
			var rels []*store.Release
			if all {
				rels, err = s.ListAllReleases(ctx)
			} else {
				rels, err = s.ListUnreadReleases(ctx)
			}
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%-30s %-15s %-12s %s\n", "REPO", "TAG", "PUBLISHED", "ASSETS")
			for _, r := range rels {
				fmt.Fprintf(cmd.OutOrStdout(), "%-30s %-15s %-12s %d\n", r.RepoFullName, r.TagName, formatDate(r.PublishedAt), len(r.Assets))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "show all releases including read")
	return cmd
}

func newReleasePullCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull releases for subscribed repos",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, err := loadConfig(cmd)
			if err != nil {
				return err
			}
			token := resolveGitHubToken(cmd, cfg)
			s, err := openStore()
			if err != nil {
				return err
			}
			defer s.Close()
			gh := github.New(token)
			tracker := release.NewTracker(s, gh)
			stats, err := tracker.PullReleases(context.Background())
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Subscribed: %d, New: %d, Errors: %d\n", stats.Subscribed, stats.NewReleases, stats.Errors)
			return nil
		},
	}
	return cmd
}

func newReleaseSubscribeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "subscribe <fullName>",
		Short: "Subscribe to a repo's releases",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore()
			if err != nil {
				return err
			}
			defer s.Close()
			ctx := context.Background()
			if err := s.SetReleaseSubscription(ctx, args[0], true); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Subscribed to %s\n", args[0])
			cfg, _, err := loadConfig(cmd)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Initial pull skipped: %v\n", err)
				return nil
			}
			token := resolveGitHubToken(cmd, cfg)
			gh := github.New(token)
			tracker := release.NewTracker(s, gh)
			stats, err := tracker.PullReleases(ctx)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Initial pull failed: %v\n", err)
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Pulled %d releases\n", stats.NewReleases)
			return nil
		},
	}
	return cmd
}

func formatDate(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	if s == "" {
		return "N/A"
	}
	return s
}

func newReleaseUnsubscribeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unsubscribe <fullName>",
		Short: "Unsubscribe from a repo's releases",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore()
			if err != nil {
				return err
			}
			defer s.Close()
			if err := s.SetReleaseSubscription(context.Background(), args[0], false); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Unsubscribed from %s\n", args[0])
			return nil
		},
	}
	return cmd
}
