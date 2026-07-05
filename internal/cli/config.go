package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"

	"github.com/morehao/starman/internal/config"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v3"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configuration management",
	}
	cmd.AddCommand(newConfigInitCmd())
	cmd.AddCommand(newConfigShowCmd())
	return cmd
}

func newConfigInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create config file interactively",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := config.DefaultDir()
			if err != nil {
				return err
			}
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
			path := filepath.Join(dir, "config.yaml")
			reader := bufio.NewReader(os.Stdin)
			cfg := config.Default()
			fmt.Print("GitHub username: ")
			cfg.GitHub.Username = readLine(reader)
			fmt.Print("GitHub token (leave empty to use env var): ")
			cfg.GitHub.Token = readLine(reader)
			fmt.Print("AI base URL (default " + cfg.AI.BaseURL + "): ")
			if v := readLine(reader); v != "" {
				cfg.AI.BaseURL = v
			}
			fmt.Print("AI API key (leave empty to use env var): ")
			cfg.AI.APIKey = readLine(reader)
			fmt.Print("AI model (default " + cfg.AI.Model + "): ")
			if v := readLine(reader); v != "" {
				cfg.AI.Model = v
			}
			data, err := yaml.Marshal(cfg)
			if err != nil {
				return err
			}
			if err := os.WriteFile(path, data, 0o600); err != nil {
				return err
			}
			fmt.Printf("Config written to %s\n", path)
			return nil
		},
	}
	return cmd
}

func newConfigShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show current configuration (sensitive fields masked)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, path, err := loadConfig(cmd)
			if err != nil {
				return err
			}
			fmt.Printf("Config file: %s\n", path)
			fmt.Printf("github:\n")
			fmt.Printf("  username: %s\n", cfg.GitHub.Username)
			fmt.Printf("  token: %s\n", mask(cfg.GitHub.Token))
			fmt.Printf("ai:\n")
			fmt.Printf("  base_url: %s\n", cfg.AI.BaseURL)
			fmt.Printf("  model: %s\n", cfg.AI.Model)
			fmt.Printf("  api_key: %s\n", mask(cfg.AI.APIKey))
			fmt.Printf("  concurrency: %d\n", cfg.AI.Concurrency)
			if cfg.AI.CustomPrompt != "" {
				fmt.Printf("  custom_prompt: %s\n", cfg.AI.CustomPrompt)
			}
			fmt.Printf("embedding:\n")
			fmt.Printf("  base_url: %s\n", cfg.Embedding.BaseURL)
			fmt.Printf("  api_key: %s\n", mask(cfg.Embedding.APIKey))
			fmt.Printf("  model: %s\n", cfg.Embedding.Model)
			fmt.Printf("generate:\n")
			fmt.Printf("  sort: %s\n", cfg.Generate.Sort)
	return nil
		},
	}
	return cmd
}

func mask(s string) string {
	if s == "" {
		return "(empty, using env var if set)"
	}
	if len(s) <= 4 {
		return "****"
	}
	return s[:2] + "****" + s[len(s)-2:]
}

func readLine(r *bufio.Reader) string {
	line, _ := r.ReadString('\n')
	return trimNewline(line)
}

func trimNewline(s string) string {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] != '\n' && s[i] != '\r' {
			return s[:i+1]
		}
	}
	return ""
}
