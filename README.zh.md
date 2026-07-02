# starman

[English](README.md) | [简体中文](README.zh.md)

一个用 AI 管理 GitHub 星标仓库的 CLI 工具 —— 同步、分析、分类、搜索，并生成 Awesome List。

starman 支持星标同步、AI 分析、Awesome List 生成、Release 追踪、数据备份——全部在 CLI 中完成。

## 功能

- **同步** — 并发分页拉取 GitHub 星标仓库到本地 SQLite（再次同步时保留 AI 分析结果）。支持 `--watch` 模式定时自动同步。
- **分析** — 批量 AI 分析（OpenAI 兼容）：生成摘要、标签、分类（双向关键词匹配），同时生成 `search_text` 用于 FTS5 全文索引
- **搜索** — LLM 查询理解 + FTS5 全文检索，BM25 加权打分，支持结构化过滤（`--lang`/`--category`）、可选 `--rerank` LLM 精排、`--sort` 排序和 `--json` 输出
- **生成** — Markdown Awesome List，三种模式：按语言、按 AI 分类、平铺列表（可自动提交到 GitHub 仓库）
- **Release 追踪** — 订阅仓库并拉取新版本，支持增量水位
- **Star/Unstar** — 星标管理，同步到本地 DB
- **备份** — JSON 导出/导入 + WebDAV 上传/下载
- **配置** — 交互式配置，敏感字段支持环境变量
- **统计** — 按语言、分类、标签查看已同步仓库的分布
- **详情** — 查看仓库详细信息，包括 AI 摘要、多语言 README 变体支持
- **趋势发现** — 浏览 GitHub Trending 仓库（RSS 或 Search API），支持交互式收藏
- **标签与分类** — 批量管理仓库的自定义标签和分类，支持分类锁定
- **补全** — bash、zsh、fish、PowerShell Shell 自动补全

## 安装

### 从源码安装

```bash
go install github.com/morehao/starman/cmd/starman@latest
```

### 从仓库构建

```bash
git clone https://github.com/morehao/starman.git
cd starman
go build -o starman ./cmd/starman
```

无需 CGO —— 使用 [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite)（纯 Go SQLite 驱动），支持跨平台编译。

## 快速开始

### 1. 创建配置

```bash
starman config init
```

交互式提示会询问 GitHub 用户名、token 和 AI 配置。配置保存到 `~/.starman/config.yaml`。

也可以通过环境变量设置敏感字段，避免写入配置文件：

| 环境变量 | 用途 |
|---------|------|
| `STARMAN_GITHUB_TOKEN` | GitHub token（回退到 `GITHUB_TOKEN`） |
| `STARMAN_AI_API_KEY` | AI API key |
| `STARMAN_WEBDAV_PASSWORD` | WebDAV 密码 |

### 2. 同步星标仓库

```bash
starman sync
```

将你所有的 GitHub 星标仓库拉取到本地 SQLite 数据库 `~/.starman/starman.db`。

### 3. AI 分析

```bash
starman analyze
```

默认分析最多 20 个未分析的仓库：获取 README，调用 AI 生成摘要/标签/平台/搜索文本，通过关键词匹配解析分类。结果缓存在 DB 中 —— 重复运行只处理新仓库。使用 `--all` 分析所有未分析仓库，`--force` 可重新分析已有仓库。分析完成后自动重建 FTS5 索引以供搜索。

### 4. 生成 Awesome List

```bash
# 按编程语言
starman generate -s language > README.md

# 按 AI 分类（展示 AI 摘要和标签）
starman generate -s category > README.md

# 平铺列表
starman generate -s flat > README.md

# 自动提交到 GitHub 仓库
starman generate -s language --repo awesome-stars
```

### 5. 搜索

```bash
# 基础关键词搜索
starman search "终端工具"

# 按语言过滤，按星标排序
starman search "框架" --lang Go --sort stars --limit 10

# LLM 精排候选仓库
starman search "机器学习" --rerank

# JSON 格式输出
starman search "机器学习" --json
```

FTS5 全文索引 + BM25 打分：LLM 先理解查询意图，然后在仓库全名、描述、AI 摘要、AI 搜索文本、标签和 topics 中全文检索。使用 `--rerank` 可让 LLM 对候选集重新排序。

### 6. 发现趋势仓库

```bash
# 浏览本周 trending
starman trending

# 按语言浏览每日 trending
starman trending --since daily --lang Rust

# 交互式收藏 trending 仓库
starman trending --star
```

## 命令用法

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

从 GitHub 拉取星标仓库并存储到本地。AI 分析结果和自定义字段在同步时保留。使用 `--full` 可删除已在 GitHub 取消星标的仓库。`--watch` 启用定时自动同步（需指定 `--interval`，最小 5 分钟）。

### generate

```bash
starman generate [flags]
```

| Flag | 说明 |
|------|------|
| `-s, --sort` | 排序模式：`language` \| `category` \| `flat`（默认从配置读取） |
| `-o, --output` | 输出文件路径（默认：stdout） |
| `--repo` | 推送到 GitHub 仓库的 README（如 `awesome-stars`） |
| `-m, --message` | `--repo` 的提交信息（默认："update stars"） |
| `-T, --template` | 自定义模板文件路径 |

### analyze

```bash
starman analyze [flags]
```

| Flag | 说明 |
|------|------|
| `--all` | 分析所有未分析的仓库 |
| `--repo` | 指定仓库名分析（可重复） |
| `--force` | 强制重新分析已有仓库 |
| `--limit` | 最多分析仓库数（默认：20，0 = 不限） |

### release

```bash
starman release list [--all]              # 列出未读（或全部）release
starman release pull                      # 拉取订阅仓库的新 release
starman release subscribe <owner/repo>    # 订阅并拉取初始 release
starman release unsubscribe <owner/repo>  # 取消订阅
```

### search

```bash
starman search <query> [flags]
```

| Flag | 说明 |
|------|------|
| `--json` | JSON 格式输出 |
| `--limit` | 限制结果数（0 = 不限） |
| `--lang` | 按语言过滤 |
| `--category` | 按分类过滤 |
| `--sort` | 排序依据：`score` \| `stars` \| `name`（默认：score） |
| `--rerank` | LLM 精排候选仓库 |

### stats

```bash
starman stats [--by language|category|tag] [--top N] [--json]
```

按维度查看已同步仓库的分布情况。输出排序表格或 JSON。

### info

```bash
starman info <owner/repo> [--readme] [--readme-variant <file>]
```

展示仓库元数据、AI 摘要、标签和自定义字段。`--readme` 拉取 README 并列出可用的多语言变体。

### tag

```bash
# 单仓库模式
starman tag <owner/repo> +awesome,-old

# 批量模式 — 为所有 Go 仓库添加标签
starman tag --lang Go --add awesome,cli

# 批量模式 — 按分类过滤并移除标签
starman tag --cat-filter "开发工具" --remove deprecated
```

管理仓库的 `custom_tags`。标签存储在本地，不会被 `analyze` 覆盖。

### categorize

```bash
# 单仓库模式
starman categorize <owner/repo> "AI 机器学习" --lock

# 批量模式 — 为所有 Python 仓库设置分类
starman categorize --lang Python "数据分析"

# 批量模式 — 按已有分类过滤
starman categorize --cat-filter "web-app" "其他"
```

管理仓库的 `custom_category`。`--lock` 阻止 AI 分析覆盖。`--unlock` 解除锁定。

### trending

```bash
starman trending [--since daily|weekly|monthly] [--lang L] [--top N] [--source rss|search] [--star]
```

浏览 GitHub 趋势仓库。默认使用 RSS 数据源（GitHubTrendingRSS）；使用 `--source search` 切换到 GitHub Search API。`--star` 交互式收藏选中的仓库。

### completion

```bash
starman completion <bash|zsh|fish|powershell>
```

生成 Shell 自动补全脚本。通过管道加载启用（如 `source <(starman completion zsh)`）。

### backup

```bash
starman backup json --export [-o file]            # 导出为 JSON
starman backup json --import <file> [--mode merge|replace]  # 从 JSON 导入
starman backup webdav --push                      # 推送备份到 WebDAV
starman backup webdav --pull                      # 从 WebDAV 拉取最新备份
starman backup webdav --test                      # 测试 WebDAV 连接
```

## 配置

配置文件：`~/.starman/config.yaml`（由 `starman config init` 创建）。

```yaml
github:
  token: ""           # 或使用 STARMAN_GITHUB_TOKEN / GITHUB_TOKEN
  username: ""

ai:
  base_url: "https://api.openai.com/v1"
  api_key: ""         # 或使用 STARMAN_AI_API_KEY
  model: "gpt-4o-mini"
  concurrency: 3
  custom_prompt: ""

webdav:
  url: ""
  username: ""
  password: ""        # 或使用 STARMAN_WEBDAV_PASSWORD
  path: "/starman"

generate:
  sort: "language"    # language | category | flat
```

**优先级：** CLI flag > 环境变量 > 配置文件 > 默认值。

`starman config show` 打印当前配置（敏感字段脱敏）。

## AI 配置

starman 兼容任何 OpenAI 兼容 API 端点（`/v1/chat/completions`），支持的提供商包括：

- OpenAI
- DeepSeek
- Moonshot (MiMo)
- OpenRouter
- 本地模型（通过 Ollama / vLLM / LM Studio）

设置 `ai.base_url` 和 `ai.model` 匹配你的提供商。`ai.concurrency` 控制批量分析的并发数。

## 关键设计

- **增量同步保留分析结果** — 从 GitHub 重新同步时不会覆盖 AI 摘要、标签、分类或你设置的自定义字段。
- **分类锁定** — 通过 `category_locked` 锁定仓库分类，防止 AI 覆盖手动设置的分类。
- **FTS5 全文索引** — 搜索使用 SQLite FTS5 + BM25 评分。分析时 LLM 生成的 `ai_search_text` 字段丰富了搜索索引，提升召回率。
- **分析失败隔离** — 某个仓库 AI 分析失败时，批量继续执行。失败的仓库标记 `analysis_failed` 以供重试。
- **Release 水位** — 订阅的仓库记录最后拉取的 release 时间戳，`release pull` 只获取新版本。
- **生成器读取本地 DB** — `generate` 不调用 GitHub API 获取数据，而是从 SQLite 读取。请先运行 `sync`，再运行 `analyze` 获取 AI 分类。
- **统计无成本** — `stats`、`info` 和（不使用 AI 的）`search` 仅读取本地 SQLite 数据库，不调用 API，无需 token。
- **Trending 双数据源** — `trending` 默认使用 RSS（通过 GitHubTrendingRSS）。`--source search` 切换到官方 GitHub Search API。

## 技术栈

| 组件 | 库 |
|------|------|
| CLI 框架 | [cobra](https://github.com/spf13/cobra) + [pflag](https://github.com/spf13/pflag) |
| GitHub API | [go-github v71](https://github.com/google/go-github) + [httpcache](https://github.com/gregjones/httpcache) |
| 并发 | [conc](https://github.com/sourcegraph/conc) |
| SQLite | [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite)（纯 Go，无 CGO） |
| 配置 | [yaml.v3](https://github.com/go-yaml/yaml) |
| AI | OpenAI 兼容 HTTP API（无 SDK） |

## 开发

```bash
# 构建
go build -o starman ./cmd/starman

# 测试
go test ./...

# 代码检查
go vet ./...
```

### 项目结构

```
cmd/starman/main.go          # 入口
internal/
  cli/                       # Cobra 命令定义
  config/                    # YAML 配置加载 + 环境变量解析
  store/                     # SQLite 数据层（Store 接口）
  github/                    # GitHub API 客户端（go-github 封装）
  ai/                        # OpenAI 兼容客户端 + 分析/分类/搜索
  discovery/                 # 趋势仓库发现（RSS + Search API 兜底）
  generate/                  # Markdown 模板渲染
  release/                   # Release 追踪器（水位）
  backup/                    # JSON + WebDAV 备份
  version/                   # 版本信息（ldflags 注入）
internal/generate/templates/ # 内嵌 Markdown 模板
```

包间依赖单向流动：`cli` → 业务包 → `store`。`github` 包在内部将 go-github 类型转换为本地 `store` 类型，第三方类型不会跨包泄漏。

## License

[Apache License 2.0](LICENSE)
