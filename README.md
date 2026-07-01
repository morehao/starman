# starman

[English](README.md) | [简体中文](README.zh.md)

A CLI tool to manage your GitHub stars with AI — sync, analyze, categorize, search, and generate awesome lists.

starman fuses [starred-go](https://github.com/juev/starred) (starred repo sync + Markdown generation) with CLI-friendly features from [GithubStarsManager](https://github.com/oovm/GithubStarsManager) (AI analysis, release tracking, backup), with AI as the core differentiator.

## Features

- **Sync** — Concurrent paginated pull of GitHub starred repos into local SQLite (preserves AI analysis on re-sync)
- **Analyze** — Batch AI analysis (OpenAI-compatible): summaries, tags, categories with bidirectional keyword matching
- **Search** — AI-translated keyword search with local multi-field weighted scoring
- **Generate** — Markdown Awesome List in 3 modes: by language, by AI category, or flat (auto-push to GitHub repo)
- **Release Tracking** — Subscribe to repos and pull new releases with incremental watermark
- **Star/Unstar** — Star management with local DB sync
- **Backup** — JSON export/import + WebDAV push/pull
- **Config** — Interactive config with env var resolution for secrets

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

### 2. Sync starred repos

```bash
starman sync
```

Pulls all your starred repos from GitHub into the local SQLite database at `~/.starman/starman.db`.

### 3. Analyze with AI

```bash
starman analyze --all
```

Analyzes all unanalyzed repos: fetches README, calls AI for summary/tags/platforms, and resolves a category via keyword matching. Results are cached in the DB — re-running only processes new repos (use `--force` to re-analyze existing ones).

### 4. Generate awesome list

```bash
# By programming language (starred-go compatible)
starman generate -s language > README.md

# By AI category (shows AI summaries and tags)
starman generate -s category > README.md

# Flat list
starman generate -s flat > README.md

# Auto-push to a GitHub repo
starman generate -s language --repo awesome-stars
```

### 5. Search

```bash
starman search "terminal tools"
```

AI translates your query into keywords, then scores all local repos against those keywords (full_name, description, AI summary, tags, topics, language) and prints results sorted by relevance.

## Usage

```
starman syncs your GitHub stars, analyzes them with AI, and generates awesome lists.

Usage:
  starman [command]

Available Commands:
  analyze     Analyze repos with AI to generate summaries, tags, and categories
  backup      Backup and restore data
  config      Configuration management
  generate    Generate Markdown awesome list from local DB
  release     Track repository releases
  search      Search repos by AI-translated keywords
  star        Star a GitHub repository
  sync        Sync starred repositories from GitHub to local DB
  unstar      Unstar a GitHub repository

Global Flags:
      --config string   config file path (default ~/.starman/config.yaml)
      --token string    GitHub token (overrides config/env)
      --verbose         verbose output
```

### sync

```bash
starman sync [--full]
```

Pulls starred repos from GitHub and stores them locally. AI analysis results and custom fields are preserved across syncs. Use `--full` to remove repos that are no longer starred on GitHub.

### generate

```bash
starman generate [flags]
```

| Flag | Description |
|------|-------------|
| `-s, --sort` | Sort mode: `language` \| `category` \| `flat` (default from config) |
| `-o, --output` | Output file path (default: stdout) |
| `--repo` | Push to a GitHub repo's README (e.g. `awesome-stars`) |
| `-m, --message` | Commit message for `--repo` (default: "update stars") |
| `-T, --template` | Custom template file path |

### analyze

```bash
starman analyze [flags]
```

| Flag | Description |
|------|-------------|
| `--all` | Analyze all repos, not just unanalyzed ones |
| `--repo` | Specific repo full names to analyze (repeatable) |
| `--force` | Force re-analyze even if already analyzed |
| `--limit` | Max repos to analyze (0 = no limit) |

### release

```bash
starman release list [--all]              # List unread (or all) releases
starman release pull                      # Pull new releases for subscribed repos
starman release subscribe <owner/repo>    # Subscribe and pull initial releases
starman release unsubscribe <owner/repo>  # Unsubscribe
```

### backup

```bash
starman backup json --export [-o file]            # Export to JSON
starman backup json --import <file> [--mode merge|replace]  # Import from JSON
starman backup webdav --push                      # Push backup to WebDAV
starman backup webdav --pull                      # Pull latest from WebDAV
starman backup webdav --test                      # Test WebDAV connection
```

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
- **Analyze failure isolation** — If AI analysis fails for one repo, the batch continues. Failed repos are marked with `analysis_failed` for retry.
- **Release watermark** — Subscribed repos track the latest fetched release timestamp, so `release pull` only retrieves new releases.
- **Generate reads from local DB** — `generate` never calls the GitHub API for data; it reads from SQLite. Run `sync` first, then `analyze` for AI categories.

## Tech Stack

| Component | Library |
|-----------|---------|
| CLI framework | [cobra](https://github.com/spf13/cobra) + [pflag](https://github.com/spf13/pflag) |
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
  generate/                  # Markdown template rendering
  release/                   # Release tracker with watermark
  backup/                    # JSON + WebDAV backup
  version/                   # Version info (ldflags injection)
internal/generate/templates/ # Embedded Markdown templates
```

Package dependencies flow in one direction: `cli` → business packages → `store`. The `github` package converts go-github types to local `store` types internally, so no third-party types leak across boundaries.

## License

[MIT](LICENSE)
