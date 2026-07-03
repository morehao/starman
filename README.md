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

## Command Reference

### Global Flags

Available on every command:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config` | string | `~/.starman/config.yaml` | Config file path |
| `--token` | string | `""` | GitHub token (overrides config/env) |
| `--verbose` | bool | `false` | Verbose output |

---

### `starman sync`

Sync starred repos from GitHub to local SQLite.

```bash
starman sync [--full] [--watch] [--interval 30m]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--full` | bool | `false` | Full sync: remove repos that are no longer starred on GitHub |
| `--watch` | bool | `false` | Watch mode: periodic auto-sync |
| `--interval` | duration | `30m` | Watch mode sync interval (minimum 5m) |

AI analysis results and custom fields are preserved across syncs.

---

### `starman analyze`

Batch AI analysis: README → summary/tags/platform/search_text → category → embedding → FTS5 rebuild.

```bash
starman analyze [--all] [--repo <name> ...] [--force] [--limit N]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--all` | bool | `false` | Analyze all unanalyzed repos |
| `--repo` | stringSlice | `[]` | Specific repo full names to analyze (repeatable) |
| `--force` | bool | `false` | Force re-analyze even if already analyzed |
| `--limit` | int | `20` | Max repos to analyze (0 = no limit) |

Failure isolation: single repo failure doesn't stop the batch. Failed repos are marked `analysis_failed` for retry.

---

### `starman search <query>`

Three-tier hybrid search with structured filtering.

```bash
starman search <query> [--json] [--limit N] [--lang L] [--category C] [--platform P]
                      [--tag T] [--min-stars N] [--max-stars N] [--sort score|stars|name]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--json` | bool | `false` | JSON output |
| `--limit` | int | `0` | Limit results (0 = no limit) |
| `--lang` | string | `""` | Filter by language |
| `--category` | string | `""` | Filter by category |
| `--platform` | string | `""` | Filter by platform: `web` / `desktop` / `mobile` / `cli` / `library` / `service` |
| `--tag` | stringSlice | `[]` | Filter by tag (OR logic, repeatable) |
| `--min-stars` | int | `0` | Minimum star count |
| `--max-stars` | int | `0` | Maximum star count (0 = no limit) |
| `--analyzed` | bool | `false` | Only show analyzed repos |
| `--no-analyzed` | bool | `false` | Only show unanalyzed repos |
| `--analysis-failed` | bool | `false` | Only show analysis-failed repos |
| `--no-vector` | bool | `false` | Disable vector search, text search only |
| `--sort` | string | `score` | Sort by: `score` / `stars` / `name` |

**Search tiers:** vector semantic matching (sqlite-vec) → AI query understanding + FTS5 full-text retrieval → basic text search. Transparent degradation when vector config is absent.

---

### `starman generate [output]`

Generate an Awesome List Markdown file from the local database.

```bash
starman generate [output] [-s language|category|flat] [-o file] [--repo <name>] [-m msg] [-T template]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-s, --sort` | string | from config | Sort mode: `language` / `category` / `flat` |
| `-o, --output` | string | `""` | Output file path (default: stdout) |
| `--repo` | string | `""` | Push to a GitHub repo's README (e.g. `awesome-stars`) |
| `-m, --message` | string | `"update stars"` | Commit message for `--repo` |
| `-T, --template` | string | `""` | Custom template file path |

**Templates:**
- `language` — Group by programming language (embedded: `by_language.tmpl`)
- `category` — Group by AI category with summaries and tags (`by_category.tmpl`)
- `flat` — Flat list (`flat.tmpl`)

---

### `starman config`

Configuration management with interactive init and masked display.

```bash
starman config init   # Interactive config creation
starman config show   # Display current config (secrets masked)
```

**`config init`** walks through GitHub username/token, AI BaseURL/API Key/Model, and writes `~/.starman/config.yaml`.

**`config show`** prints full config with token/key/password showing only first and last 2 characters.

---

### `starman release`

Track repository releases with incremental watermark.

```bash
starman release list [--all]                        # List unread (or all) releases
starman release pull                                 # Pull new releases for subscribed repos
starman release subscribe <owner/repo>               # Subscribe + pull initial releases
starman release unsubscribe <owner/repo>             # Unsubscribe
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--all` | bool | `false` | `list` subcommand: show all releases including read |

---

### `starman star <owner/repo>`

Star a repository on GitHub and sync to local DB.

```bash
starman star <owner/repo>
```

---

### `starman unstar <owner/repo>`

Unstar a repository on GitHub and mark locally as unstarred (`StarredAt=""`).

```bash
starman unstar <owner/repo>
```

---

### `starman backup`

Backup and restore data via JSON, WebDAV, or GitHub repo push.

```bash
starman backup json --export [-o file]                # Export to JSON (stdout if -o omitted)
starman backup json --import <file> [--mode merge|replace]  # Import from JSON
starman backup webdav --push                          # Push backup to WebDAV
starman backup webdav --pull                          # Pull latest from WebDAV
starman backup webdav --test                          # Test WebDAV connection
starman backup --repo <name> [-m "msg"]               # Push backup to GitHub repo
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--export` | bool | `false` | `json` subcommand: export mode |
| `--import` | string | `""` | `json` subcommand: import from file path |
| `-o, --output` | string | `""` | `json` export output path (default: stdout) |
| `--mode` | string | `merge` | `json` import mode: `merge` / `replace` |
| `--push` | bool | `false` | `webdav` subcommand: push backup |
| `--pull` | bool | `false` | `webdav` subcommand: pull backup |
| `--test` | bool | `false` | `webdav` subcommand: test connection |
| `--repo` | string | `""` | Backup to GitHub repo (root command) |
| `-m, --message` | string | auto | Commit message for `--repo` backup |

---

### `starman stats`

Show distribution statistics of synced repositories.

```bash
starman stats [--by language|category|tag] [--top N] [--json]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--by` | string | `language` | Dimension: `language` / `category` / `tag` |
| `--top` | int | `10` | Show top N items (0 = all) |
| `--json` | bool | `false` | JSON output |

---

### `starman info <owner/repo>`

Display detailed repository information.

```bash
starman info <owner/repo> [--readme] [--readme-variant <file>]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--readme` | bool | `false` | Fetch and display README |
| `--readme-variant` | string | `""` | Specific README variant (e.g. `README_zh.md`) |

**Shows:** URL, language, stars/forks, topics, starred-at time, AI summary/tags/category, custom fields, lock status.

---

### `starman tag [owner/repo] [tagExpr]`

Manage custom tags on repositories.

```bash
# Single repo: + to add, - to remove
starman tag <owner/repo> +awesome,-old

# Batch: add tags to all Go repos
starman tag --lang Go --add awesome,cli

# Batch: remove tags filtered by category
starman tag --cat-filter "开发工具" --remove deprecated
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--lang` | string | `""` | Batch: filter by language |
| `--cat-filter` | string | `""` | Batch: filter by existing category |
| `--add` | string | `""` | Batch: comma-separated tags to add |
| `--remove` | string | `""` | Batch: comma-separated tags to remove |

Tags are stored in `custom_tags` and never overwritten by AI analysis.

---

### `starman categorize [owner/repo] <category>`

Manage custom category on repositories with lock support.

```bash
# Single repo: set category with lock
starman categorize <owner/repo> "AI 机器学习" --lock

# Batch: set category for all Python repos
starman categorize --lang Python "数据分析"

# Batch: recategorize from one category to another
starman categorize --cat-filter "web-app" "其他"
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--lang` | string | `""` | Batch: filter by language |
| `--cat-filter` | string | `""` | Batch: filter by existing category |
| `--lock` | bool | `false` | Lock category (prevent AI overwrite) |
| `--unlock` | bool | `false` | Unlock category |

---

### `starman trending`

Browse GitHub trending repositories.

```bash
starman trending [--since daily|weekly|monthly] [--lang L] [--top N] [--source rss|search] [--star]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--since` | string | `weekly` | Time range: `daily` / `weekly` / `monthly` |
| `--lang` | string | `""` | Filter by language |
| `--top` | int | `20` | Show top N repos |
| `--source` | string | `rss` | Data source: `rss` / `search` |
| `--star` | bool | `false` | Interactive star selection |

Default source is RSS (GitHubTrendingRSS); `--source search` uses GitHub Search API fallback.

---

### `starman completion <shell>`

Generate shell auto-completion script.

```bash
starman completion bash        # Bash completion
starman completion zsh         # Zsh completion
starman completion fish        # Fish completion
starman completion powershell  # PowerShell completion
```

Usage: `source <(starman completion zsh)` (or the appropriate shell).

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

[MIT License](LICENSE)
