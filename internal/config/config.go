package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	GitHub   GitHubConfig   `yaml:"github"`
	AI       AIConfig       `yaml:"ai"`
	WebDAV   WebDAVConfig   `yaml:"webdav"`
	Generate GenerateConfig `yaml:"generate"`
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
		Generate: GenerateConfig{Sort: "language"},
		WebDAV:   WebDAVConfig{Path: "/starman"},
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
