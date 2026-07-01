# starman

[English](README.md) | [简体中文](README.zh.md)

一个用 AI 管理 GitHub 星标仓库的 CLI 工具 —— 同步、分析、分类、搜索，并生成 Awesome List。

starman 融合了 [starred-go](https://github.com/juev/starred)（星标同步 + Markdown 生成）与 [GithubStarsManager](https://github.com/oovm/GithubStarsManager) 中适合 CLI 的功能（AI 分析、Release 追踪、备份），以 AI 能力作为核心增值。

## 功能

- **同步** — 并发分页拉取 GitHub 星标仓库到本地 SQLite（再次同步时保留 AI 分析结果）
- **分析** — 批量 AI 分析（OpenAI 兼容）：生成摘要、标签、分类，基于双向关键词匹配
- **搜索** — AI 翻译关键词 + 本地多字段加权打分
- **生成** — Markdown Awesome List，三种模式：按语言、按 AI 分类、平铺列表（可自动提交到 GitHub 仓库）
- **Release 追踪** — 订阅仓库并拉取新版本，支持增量水位
- **Star/Unstar** — 星标管理，同步到本地 DB
- **备份** — JSON 导出/导入 + WebDAV 上传/下载
- **配置** — 交互式配置，敏感字段支持环境变量

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
starman analyze --all
```

分析所有未分析的仓库：获取 README，调用 AI 生成摘要/标签/平台，通过关键词匹配解析分类。结果缓存在 DB 中 —— 重复运行只处理新仓库（使用 `--force` 可重新分析已有仓库）。

### 4. 生成 Awesome List

```bash
# 按编程语言（兼容 starred-go）
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
starman search "终端工具"
```

AI 将你的查询翻译为关键词，然后对所有本地仓库进行多字段打分匹配（仓库名、描述、AI 摘要、标签、topics、语言），按相关度排序输出。

## 命令用法

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

从 GitHub 拉取星标仓库并存储到本地。AI 分析结果和自定义字段在同步时保留。使用 `--full` 可删除已在 GitHub 取消星标的仓库。

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
| `--all` | 分析所有仓库，不只是未分析的 |
| `--repo` | 指定仓库名分析（可重复） |
| `--force` | 强制重新分析已有仓库 |
| `--limit` | 最多分析仓库数（0 = 不限） |

### release

```bash
starman release list [--all]              # 列出未读（或全部）release
starman release pull                      # 拉取订阅仓库的新 release
starman release subscribe <owner/repo>    # 订阅并拉取初始 release
starman release unsubscribe <owner/repo>  # 取消订阅
```

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
- **分析失败隔离** — 某个仓库 AI 分析失败时，批量继续执行。失败的仓库标记 `analysis_failed` 以供重试。
- **Release 水位** — 订阅的仓库记录最后拉取的 release 时间戳，`release pull` 只获取新版本。
- **生成器读取本地 DB** — `generate` 不调用 GitHub API 获取数据，而是从 SQLite 读取。请先运行 `sync`，再运行 `analyze` 获取 AI 分类。

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
  generate/                  # Markdown 模板渲染
  release/                   # Release 追踪器（水位）
  backup/                    # JSON + WebDAV 备份
  version/                   # 版本信息（ldflags 注入）
internal/generate/templates/ # 内嵌 Markdown 模板
```

包间依赖单向流动：`cli` → 业务包 → `store`。`github` 包在内部将 go-github 类型转换为本地 `store` 类型，第三方类型不会跨包泄漏。

## License

[Apache License 2.0](LICENSE)
