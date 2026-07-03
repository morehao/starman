# starman

[English](README.md) | [简体中文](README.zh.md)

A CLI tool to manage your GitHub stars with AI — sync, analyze, categorize, search, and generate awesome lists.

starman syncs your GitHub stars, analyzes them with AI, generates awesome lists, tracks releases, and backs up your data — all from the CLI.

## Features

- **Sync** — Concurrent paginated pull of GitHub starred repos into local SQLite (preserves AI analysis on re-sync). Supports `--watch` mode for periodic auto-sync.
- **Analyze** — Batch AI analysis (OpenAI-compatible): summaries, tags, categories with bidirectional keyword matching, plus embedding vector generation for semantic search and `search_text` for full-text index
- **Search** — Three-tier hybrid search (vector semantic matching > AI query understanding + text retrieval > basic text search), structured filtering (`--lang`/`--category`/`--platform`/`--tag`), `--sort` options, and `--json` output
- **Generate** — Markdown Awesome List in 3 modes: by language, by AI category, or flat (auto-push to GitHub repo)
- **Release Tracking** — Subscribe to repos and pull new releases with incremental watermark
- **Star/Unstar** — Star management with local DB sync
- **Backup** — JSON export/import + WebDAV push/pull + GitHub repo push
- **Config** — Interactive config with env var resolution for secrets
- **Stats** — View distribution of synced repos by language, category, or tag
- **Info** — Inspect repo details including AI summary, README with multi-language variant support
- **Trending** — Browse GitHub trending repositories (RSS or search API) with interactive starring
- **Tag & Categorize** — Batch manage custom tags and categories on local repos, with category locking
- **Completion** — Shell auto-completion for bash, zsh, fish, and PowerShell

## Installation

### macOS / Linux (Homebrew)

```bash
brew install morehao/tap/starman
```

### Linux (Shell)

```bash
curl -fsSL https://raw.githubusercontent.com/morehao/starman/main/scripts/install.sh | sh
```

### Linux (.deb)

Download the `.deb` file from [GitHub Releases](https://github.com/morehao/starman/releases/latest):

```bash
dpkg -i starman_*.deb
```

### Linux (.rpm)

Download the `.rpm` file from [GitHub Releases](https://github.com/morehao/starman/releases/latest):

```bash
rpm -i starman_*.rpm
```

### Windows (Scoop)

```powershell
scoop bucket add morehao https://github.com/morehao/scoop-bucket
scoop install starman
```

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
| `STARMAN_EMBEDDING_API_KEY` | Embedding API key (for vector search) |
| `STARMAN_WEBDAV_PASSWORD` | WebDAV password |

### 2. Sync starred repos

```bash
starman sync
```

Pulls all your starred repos from GitHub into the local SQLite database at `~/.starman/starman.db`.

### 3. Analyze with AI

```bash
starman analyze
```

Analyzes up to 20 unanalyzed repos by default: fetches README, calls AI for summary/tags/platforms/search_text, resolves a category via keyword matching, and generates embedding vectors for semantic search. Results are cached in the DB — re-running only processes new repos. Use `--all` to analyze all unanalyzed repos, or `--force` to re-analyze existing ones. After analysis, the FTS5 index is automatically rebuilt for search.

### 4. Generate awesome list

```bash
# By programming language
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
# Basic semantic search
starman search "terminal tools"

# Filter by language and sort by stars
starman search "framework" --lang Go --sort stars --limit 10

# Filter by platform and tag
starman search "database" --platform cli --tag go

# JSON output
starman search "machine learning" --json
```

Three-tier hybrid search: attempts vector semantic matching first (requires embedding config), degrades to AI query understanding + full-text retrieval, then falls back to basic text search. Matches against full_name, description, AI summary, search_text, tags, and topics with weighted scoring.

### 6. Discover trending repos

```bash
# Browse weekly trending repos
starman trending

# Daily trending, filtered by language
starman trending --since daily --lang Rust

# Interactively star repos from trending
starman trending --star
```

## Usage

```
starman syncs your GitHub stars, analyzes them with AI, and generates awesome lists.

Usage:
  starman [command]

Available Commands:
  analyze      Analyze repos with AI to generate summaries, tags, and categories
  backup       Backup and restore data
  categorize   Manage custom category on repositories
  completion   Generate shell completion script
  config       Configuration management
  generate     Generate Markdown awesome list from local DB
  info         Show details of a repository
  release      Track repository releases
  search       Search repos by AI-translated keywords
  star         Star a GitHub repository
  stats        Show statistics of synced repositories
  sync         Sync starred repositories from GitHub to local DB
  tag          Manage custom tags on repositories
  trending     Browse GitHub trending repositories
  unstar       Unstar a GitHub repository

Global Flags:
      --config string   config file path (default ~/.starman/config.yaml)
      --token string    GitHub token (overrides config/env)
      --verbose         verbose output
```

### sync

```bash
starman sync [--full] [--watch] [--interval 30m]
```

Pulls starred repos from GitHub and stores them locally. AI analysis results and custom fields are preserved across syncs. Use `--full` to remove repos that are no longer starred on GitHub. `--watch` enables periodic auto-sync (requires `--interval`, minimum 5m).

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
| `--all` | Analyze all unanalyzed repos |
| `--repo` | Specific repo full names to analyze (repeatable) |
| `--force` | Force re-analyze even if already analyzed |
| `--limit` | Max repos to analyze (default: 20, 0 = no limit) |

### release

```bash
starman release list [--all]              # List unread (or all) releases
starman release pull                      # Pull new releases for subscribed repos
starman release subscribe <owner/repo>    # Subscribe and pull initial releases
starman release unsubscribe <owner/repo>  # Unsubscribe
```

### search

```bash
starman search <query> [flags]
```

| Flag | Description |
|------|-------------|
| `--json` | Output as JSON |
| `--limit` | Limit number of results (0 = no limit) |
| `--lang` | Filter by language |
| `--category` | Filter by category |
| `--platform` | Filter by platform: `web` \| `desktop` \| `mobile` \| `cli` \| `library` \| `service` |
| `--tag` | Filter by tag (OR logic, repeatable) |
| `--min-stars` | Minimum star count |
| `--max-stars` | Maximum star count (0 = no limit) |
| `--analyzed` | Only show analyzed repos |
| `--no-analyzed` | Only show unanalyzed repos |
| `--analysis-failed` | Only show analysis-failed repos |
| `--no-vector` | Disable vector search, use text search only |
| `--sort` | Sort by: `score` \| `stars` \| `name` (default: score) |

### stats

```bash
starman stats [--by language|category|tag] [--top N] [--json]
```

Show distribution of synced repos by dimension. Outputs a sorted table or JSON.

### info

```bash
starman info <owner/repo> [--readme] [--readme-variant <file>]
```

Display repo metadata, AI summary, tags, and custom fields. `--readme` fetches the README and lists available multi-language variants.

### tag

```bash
# Single repo mode
starman tag <owner/repo> +awesome,-old

# Batch mode — add tags to all Go repos
starman tag --lang Go --add awesome,cli

# Batch mode — filter by category and remove tags
starman tag --cat-filter "开发工具" --remove deprecated
```

Manages `custom_tags` on repositories. Tags are stored locally and never overwritten by `analyze`.

### categorize

```bash
# Single repo mode
starman categorize <owner/repo> "AI 机器学习" --lock

# Batch mode — set category for all Python repos
starman categorize --lang Python "数据分析"

# Batch mode — filter by existing category
starman categorize --cat-filter "web-app" "其他"
```

Manages `custom_category` on repositories. `--lock` prevents AI analysis from overwriting. `--unlock` releases the lock.

### trending

```bash
starman trending [--since daily|weekly|monthly] [--lang L] [--top N] [--source rss|search] [--star]
```

Browse GitHub trending repositories. Default source is RSS (via GitHubTrendingRSS); use `--source search` for the GitHub Search API fallback. `--star` interactively stars selected repos.

### completion

```bash
starman completion <bash|zsh|fish|powershell>
```

Generates shell auto-completion scripts. Pipe to source to enable (e.g. `source <(starman completion zsh)`).

### backup

```bash
starman backup json --export [-o file]            # Export to JSON
starman backup json --import <file> [--mode merge|replace]  # Import from JSON
starman backup webdav --push                      # Push backup to WebDAV
starman backup webdav --pull                      # Pull latest from WebDAV
starman backup webdav --test                      # Test WebDAV connection
starman backup --repo awesome-stars               # Push backup to GitHub repo
starman backup --repo awesome-stars -m "msg"      # With custom commit message
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

embedding:
  base_url: "https://api.openai.com/v1"
  api_key: ""         # or use STARMAN_EMBEDDING_API_KEY
  model: "text-embedding-3-small"

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

Embedding also works with any OpenAI-compatible embedding API endpoint (`/v1/embeddings`). Configure the `embedding` section to enable vector semantic search; when unconfigured, search gracefully degrades to text retrieval.

## Key Design

- **Incremental sync preserves analysis** — Re-syncing from GitHub never overwrites AI summaries, tags, categories, or custom fields you've set.
- **Category locking** — Lock a repo's category with `category_locked` to prevent AI from overwriting your manual assignment.
- **Three-tier hybrid search** — Search first attempts vector semantic matching (sqlite-vec), degrades to AI query understanding + full-text retrieval, then falls back to basic text search. Transparent degradation when vector config is absent — zero-cost operation. The `ai_search_text` field generated by LLM during analysis enriches the search index for better recall.
- **Analyze failure isolation** — If AI analysis fails for one repo, the batch continues. Failed repos are marked with `analysis_failed` for retry.
- **Release watermark** — Subscribed repos track the latest fetched release timestamp, so `release pull` only retrieves new releases.
- **Generate reads from local DB** — `generate` never calls the GitHub API for data; it reads from SQLite. Run `sync` first, then `analyze` for AI categories.
- **Stats are free** — `stats`, `info`, and `search` (without AI) only read from the local SQLite database. No API calls, no token needed.
- **Trending dual source** — `trending` defaults to RSS via GitHubTrendingRSS. `--source search` falls back to the official GitHub Search API.

## Tech Stack

| Component | Library |
|-----------|---------|
| CLI framework | [cobra](https://github.com/spf13/cobra) + [pflag](https://github.com/spf13/pflag) |
| GitHub API | [go-github v71](https://github.com/google/go-github) + [httpcache](https://github.com/gregjones/httpcache) |
| Concurrency | [conc](https://github.com/sourcegraph/conc) |
| SQLite | [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) + [modernc.org/sqlite/vec](https://pkg.go.dev/modernc.org/sqlite/vec) (pure Go, no CGO) |
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
  backup/                    # JSON + WebDAV + GitHub backup
  version/                   # Version info (ldflags injection)
internal/generate/templates/ # Embedded Markdown templates
```

Package dependencies flow in one direction: `cli` → business packages → `store`. The `github` package converts go-github types to local `store` types internally, so no third-party types leak across boundaries.

## License

[Apache License 2.0](LICENSE)
