package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()
	if cfg.AI.BaseURL != "https://api.openai.com/v1" {
		t.Fatalf("unexpected base_url: %s", cfg.AI.BaseURL)
	}
	if cfg.AI.Model != "gpt-4o-mini" {
		t.Fatalf("unexpected model: %s", cfg.AI.Model)
	}
	if cfg.AI.Concurrency != 3 {
		t.Fatalf("unexpected concurrency: %d", cfg.AI.Concurrency)
	}
	if cfg.Generate.Sort != "language" {
		t.Fatalf("unexpected sort: %s", cfg.Generate.Sort)
	}
}

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte("github:\n  token: abc\n  username: testuser\nai:\n  api_key: sk-test\n  model: gpt-4o\n")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.GitHub.Token != "abc" {
		t.Fatalf("unexpected token: %s", cfg.GitHub.Token)
	}
	if cfg.GitHub.Username != "testuser" {
		t.Fatalf("unexpected username: %s", cfg.GitHub.Username)
	}
	if cfg.AI.APIKey != "sk-test" {
		t.Fatalf("unexpected api_key: %s", cfg.AI.APIKey)
	}
	if cfg.AI.Model != "gpt-4o" {
		t.Fatalf("unexpected model: %s", cfg.AI.Model)
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestResolveToken(t *testing.T) {
	cfg := Default()
	cfg.GitHub.Token = "config-token"
	if got := ResolveToken(cfg, ""); got != "config-token" {
		t.Fatalf("expected config-token, got %s", got)
	}
	if got := ResolveToken(cfg, "flag-token"); got != "flag-token" {
		t.Fatalf("expected flag-token, got %s", got)
	}
	t.Setenv("STARMAN_GITHUB_TOKEN", "env-token")
	if got := ResolveToken(cfg, ""); got != "env-token" {
		t.Fatalf("expected env-token, got %s", got)
	}
}

func TestResolveAIKey(t *testing.T) {
	cfg := Default()
	cfg.AI.APIKey = "config-key"
	if got := ResolveAIKey(cfg, ""); got != "config-key" {
		t.Fatalf("expected config-key, got %s", got)
	}
	t.Setenv("STARMAN_AI_API_KEY", "env-key")
	if got := ResolveAIKey(cfg, ""); got != "env-key" {
		t.Fatalf("expected env-key, got %s", got)
	}
}
