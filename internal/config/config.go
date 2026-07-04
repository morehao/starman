package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	GitHub    GitHubConfig    `yaml:"github"`
	AI        AIConfig        `yaml:"ai"`
	Embedding EmbeddingConfig `yaml:"embedding"`
	WebDAV    WebDAVConfig    `yaml:"webdav"`
	Generate  GenerateConfig  `yaml:"generate"`
	TUI       TUIConfig       `yaml:"tui"`
}

type TUIConfig struct {
	Preview     PreviewConfig  `yaml:"preview"`
	DefaultView string         `yaml:"default_view"`
	ConfirmQuit bool           `yaml:"confirm_quit"`
	Theme       string         `yaml:"theme"`
	Keybindings TUIKeybindings `yaml:"keybindings"`
}

type TUIKeybindings struct {
	Universal UniversalKeybindings `yaml:"universal"`
	Stars     StarsKeybindings     `yaml:"stars"`
}

type UniversalKeybindings struct {
	Quit          string `yaml:"quit"`
	Refresh       string `yaml:"refresh"`
	Search        string `yaml:"search"`
	Command       string `yaml:"command"`
	ToggleSidebar string `yaml:"toggle_sidebar"`
	Help          string `yaml:"help"`
	Filter        string `yaml:"filter"`
}

type StarsKeybindings struct {
	Sync       string `yaml:"sync"`
	Analyze    string `yaml:"analyze"`
	ToggleStar string `yaml:"toggle_star"`
	EditCat    string `yaml:"edit_category"`
	EditTag    string `yaml:"edit_tag"`
}

type PreviewConfig struct {
	Open     bool    `yaml:"open"`
	Position string  `yaml:"position"`
	Width    float64 `yaml:"width"`
	Height   float64 `yaml:"height"`
}

type GitHubConfig struct {
	Token    string `yaml:"token"`
	Username string `yaml:"username"`
}

type AIConfig struct {
	BaseURL      string `yaml:"base_url"`
	APIKey       string `yaml:"api_key"`
	Model        string `yaml:"model"`
	Concurrency  int    `yaml:"concurrency"`
	CustomPrompt string `yaml:"custom_prompt"`
}

type WebDAVConfig struct {
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Path     string `yaml:"path"`
}

type EmbeddingConfig struct {
	BaseURL string `yaml:"base_url"`
	APIKey  string `yaml:"api_key"`
	Model   string `yaml:"model"`
}

type GenerateConfig struct {
	Sort string `yaml:"sort"`
}

func Default() *Config {
	return &Config{
		AI: AIConfig{
			BaseURL:     "https://api.openai.com/v1",
			Model:       "gpt-4o-mini",
			Concurrency: 3,
		},
		Embedding: EmbeddingConfig{
			BaseURL: "https://api.openai.com/v1",
			Model:   "text-embedding-3-small",
		},
		Generate: GenerateConfig{Sort: "language"},
		WebDAV:   WebDAVConfig{Path: "/starman"},
		TUI: TUIConfig{
			Preview: PreviewConfig{
				Open:     true,
				Position: "right",
				Width:    0.42,
				Height:   0.4,
			},
			DefaultView: "stars",
			ConfirmQuit: false,
			Theme:       "dark",
		},
	}
}

func DefaultDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".starman"), nil
}

func DefaultPath() (string, error) {
	d, err := DefaultDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "config.yaml"), nil
}

func Load(path string) (*Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}

func ResolveToken(cfg *Config, flagToken string) string {
	if flagToken != "" {
		return flagToken
	}
	if v := os.Getenv("STARMAN_GITHUB_TOKEN"); v != "" {
		return v
	}
	if v := os.Getenv("GITHUB_TOKEN"); v != "" {
		return v
	}
	return cfg.GitHub.Token
}

func ResolveAIKey(cfg *Config, flagKey string) string {
	if flagKey != "" {
		return flagKey
	}
	if v := os.Getenv("STARMAN_AI_API_KEY"); v != "" {
		return v
	}
	return cfg.AI.APIKey
}

func ResolveWebDAVPassword(cfg *Config) string {
	if v := os.Getenv("STARMAN_WEBDAV_PASSWORD"); v != "" {
		return v
	}
	return cfg.WebDAV.Password
}

func ResolveEmbeddingKey(cfg *Config, flagKey string) string {
	if flagKey != "" {
		return flagKey
	}
	if v := os.Getenv("STARMAN_EMBEDDING_API_KEY"); v != "" {
		return v
	}
	return cfg.Embedding.APIKey
}
