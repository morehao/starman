package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/morehao/starman/internal/config"
	"github.com/morehao/starman/internal/tui"
	"github.com/spf13/cobra"
)

func NewRootCmd(ver string) *cobra.Command {
	root := &cobra.Command{
		Use:     "starman",
		Short:   "Manage your GitHub stars with AI",
		Long:    "starman syncs your GitHub stars, analyzes them with AI, and generates awesome lists.",
		Version: ver,
	}
	root.SetVersionTemplate("{{.Version}}\n")
	root.PersistentFlags().String("config", "", "config file path (default ~/.starman/config.yaml)")
	root.PersistentFlags().String("token", "", "GitHub token (overrides config/env)")
	root.PersistentFlags().Bool("verbose", false, "verbose output")

	root.AddCommand(newSyncCmd())
	root.AddCommand(newGenerateCmd())
	root.AddCommand(newAnalyzeCmd())
	root.AddCommand(newSearchCmd())
	root.AddCommand(newReleaseCmd())
	root.AddCommand(newStarCmd())
	root.AddCommand(newUnstarCmd())
	root.AddCommand(newBackupCmd())
	root.AddCommand(newConfigCmd())
	root.AddCommand(newCompletionCmd(root))
	root.AddCommand(newStatsCmd())
	root.AddCommand(newInfoCmd())
	root.AddCommand(newTagCmd())
	root.AddCommand(newCategorizeCmd())
	root.AddCommand(newCategoryCmd())
	root.AddCommand(newTrendingCmd())
	return root
}

func Run(ver string) {
	if !hasSubcommandArgs(os.Args[1:]) {
		configPath, err := configPathForArgs(os.Args[1:])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		cfg, err := config.Load(configPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		if err := tui.Run(cfg, ver); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	root := NewRootCmd(ver)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func configPathForArgs(args []string) (string, error) {
	root := NewRootCmd("test")
	if err := root.ParseFlags(args); err != nil {
		return "", err
	}

	v, err := root.PersistentFlags().GetString("config")
	if err != nil {
		return "", err
	}
	if v != "" {
		return v, nil
	}

	return config.DefaultPath()
}

func hasSubcommandArgs(args []string) bool {
	root := NewRootCmd("test")
	flags := root.PersistentFlags()
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			return i+1 < len(args)
		}
		if !strings.HasPrefix(arg, "-") {
			return true
		}
		if arg == "-h" || arg == "--help" || arg == "--version" {
			return true
		}
		if !strings.HasPrefix(arg, "--") {
			return true
		}

		name, _, hasValue := strings.Cut(arg[2:], "=")
		f := flags.Lookup(name)
		if f == nil {
			return true
		}
		if hasValue || f.NoOptDefVal != "" {
			continue
		}
		if i+1 >= len(args) {
			return true
		}
		i++
	}
	return false
}
