# starman

[English](README.md) | [简体中文](README.zh.md)

A terminal-based tool to manage your GitHub stars with AI — sync, analyze, categorize, search, and generate awesome lists, all from an interactive TUI.

## Features

- **Interactive TUI** — A dashboard-style terminal UI with sidebar navigation. Browse repos, search, view details, manage tags and categories, check stats — all keyboard-driven.
- **Sync** — Concurrent paginated pull of GitHub starred repos into local SQLite (preserves AI analysis on re-sync). Supports `--watch` mode for periodic auto-sync.
- **Analyze** — Batch AI analysis (OpenAI-compatible): summaries, tags, categories with bidirectional keyword matching, plus `search_text` generation for FTS5 full-text index
- **Search** — LLM query understanding + FTS5 full-text retrieval with BM25 scoring, structured filtering, optional LLM precision reorder
- **Generate** — Markdown Awesome List in 3 modes: by language, by AI category, or flat (auto-push to GitHub repo)
- **Release Tracking** — Subscribe to repos and pull new releases with incremental watermark
- **Backup** — JSON export/import + WebDAV push/pull
- **Tag & Categorize** — Batch manage custom tags and categories on local repos, with category locking
- **Stats** — View distribution of synced repos by language, category, or tag
- **Trending** — Browse GitHub trending repositories (RSS or search API) with interactive starring
- **Config** — Interactive config with env var resolution for secrets
- **Completion** — Shell auto-completion for bash, zsh, fish, and PowerShell

## Installation

### From source

```bash
go install github.com/morehao/starman/cmd/starman@latest
```

### Build from repository

```bash
git clone https://github.com/morehao/starman.git
cd starman
go build -o starman ./cmd/starman
```

No CGO required — uses [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) (pure Go SQLite driver) for cross-platform compilation.

## Quick Start

### 1. Create config

```bash
starman config init
```

Interactive prompt will ask for GitHub username, token, and AI settings. Config is saved to `~/.starman/config.yaml`.

You can also set sensitive fields via environment variables instead of the config file:

| Environment variable | Purpose |
|----------------------|---------|
| `STARMAN_GITHUB_TOKEN` | GitHub token (falls back to `GITHUB_TOKEN`) |
| `STARMAN_AI_API_KEY` | AI API key |
| `STARMAN_WEBDAV_PASSWORD` | WebDAV password |

### 2. Launch starman

```bash
starman
```

This opens the interactive terminal UI. From here you can sync your stars, analyze repos with AI, search, generate awesome lists, manage tags/categories, and more — all from the keyboard.

**TUI pages:** Dashboard · Search · Repo List · Repo Detail · Trending · Sync · Analyze · Tag · Categorize · Stats · Releases · Generate · Backup · Config

**Key shortcuts:** `q` quit, `/` search, `?` help, `↑↓` navigate, `Enter` select, `Esc` back

## Usage

```
starman                    Launch the interactive TUI (default)

starman config init        Create config file interactively
starman config show        Show current config (sensitive fields masked)
starman completion <shell> Generate shell completion (bash|zsh|fish|powershell)
starman --help             Show help
starman --version          Show version
```

### config

```bash
starman config init      # Create config file interactively
starman config show      # Show current configuration (sensitive fields masked)
```

### completion

```bash
starman completion <bash|zsh|fish|powershell>
```

Generates shell auto-completion scripts. Pipe to source to enable (e.g. `source <(starman completion zsh)`).

## Configuration

Config file: `~/.starman/config.yaml` (created by `starman config init`).

```yaml
github:
  token: ""           # or use STARMAN_GITHUB_TOKEN / GITHUB_TOKEN
  username: ""

ai:
  base_url: "https://api.openai.com/v1"
  api_key: ""         # or use STARMAN_AI_API_KEY
  model: "gpt-4o-mini"
  concurrency: 3
  custom_prompt: ""

webdav:
  url: ""
  username: ""
  password: ""        # or use STARMAN_WEBDAV_PASSWORD
  path: "/starman"

generate:
  sort: "language"    # language | category | flat
```

**Priority:** CLI flag > environment variable > config file > default value.

`starman config show` prints the current configuration with sensitive fields masked.

## AI Configuration

starman works with any OpenAI-compatible API endpoint (`/v1/chat/completions`). Compatible providers include:

- OpenAI
- DeepSeek
- Moonshot (MiMo)
- OpenRouter
- Local models via Ollama / vLLM / LM Studio

Set `ai.base_url` and `ai.model` to match your provider. The `ai.concurrency` setting controls batch analysis parallelism.

## Key Design

- **Incremental sync preserves analysis** — Re-syncing from GitHub never overwrites AI summaries, tags, categories, or custom fields you've set.
- **Category locking** — Lock a repo's category with `category_locked` to prevent AI from overwriting your manual assignment.
- **FTS5 full-text index** — Search uses SQLite FTS5 with BM25 scoring. An `ai_search_text` field generated by LLM during analysis enriches the search index for better recall.
- **Analyze failure isolation** — If AI analysis fails for one repo, the batch continues. Failed repos are marked with `analysis_failed` for retry.
- **Release watermark** — Subscribed repos track the latest fetched release timestamp, so `release pull` only retrieves new releases.
- **Trending dual source** — Trending defaults to RSS via GitHubTrendingRSS, with GitHub Search API as fallback.

## Tech Stack

| Component | Library |
|-----------|---------|
| CLI framework | [cobra](https://github.com/spf13/cobra) + [pflag](https://github.com/spf13/pflag) |
| TUI framework | [bubbletea](https://github.com/charmbracelet/bubbletea) + [lipgloss](https://github.com/charmbracelet/lipgloss) |
| GitHub API | [go-github v71](https://github.com/google/go-github) + [httpcache](https://github.com/gregjones/httpcache) |
| Concurrency | [conc](https://github.com/sourcegraph/conc) |
| SQLite | [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) (pure Go, no CGO) |
| Config | [yaml.v3](https://github.com/go-yaml/yaml) |
| AI | OpenAI-compatible HTTP API (no SDK) |

## Development

```bash
# Build
go build -o starman ./cmd/starman

# Test
go test ./...

# Vet
go vet ./...
```

### Project Structure

```
cmd/starman/main.go          # Entry point
internal/
  cli/                       # Cobra command definitions
  config/                    # YAML config loading + env var resolution
  store/                     # SQLite data layer (Store interface)
  github/                    # GitHub API client (go-github wrapper)
  ai/                        # OpenAI-compatible client + analysis/categorization/search
  discovery/                 # Trending repos (RSS + search API fallback)
  generate/                  # Markdown template rendering
  release/                   # Release tracker with watermark
  backup/                    # JSON + WebDAV backup
  tui/                       # Terminal UI (bubbletea)
  version/                   # Version info (ldflags injection)
internal/generate/templates/ # Embedded Markdown templates
```

Package dependencies flow in one direction: `cli` → business packages → `store`. The `github` package converts go-github types to local `store` types internally, so no third-party types leak across boundaries.

## License

[Apache License 2.0](LICENSE)
