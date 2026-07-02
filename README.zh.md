# starman

[English](README.md) | [简体中文](README.zh.md)

一个用 AI 管理 GitHub 星标仓库的终端工具 —— 在交互式 TUI 中同步、分析、分类、搜索，并生成 Awesome List。

## 功能

- **交互式 TUI** — Dashboard 风格的终端界面，左侧导航栏 + 右侧内容面板。可浏览仓库、搜索、查看详情、管理标签分类 — 全部键盘操作。
- **同步** — 并发分页拉取 GitHub 星标仓库到本地 SQLite（再次同步时保留 AI 分析结果）。支持 `--watch` 模式定时自动同步。
- **分析** — 批量 AI 分析（OpenAI 兼容）：生成摘要、标签、分类（双向关键词匹配），同时生成 `search_text` 用于 FTS5 全文索引
- **搜索** — LLM 查询理解 + FTS5 全文检索，BM25 加权打分，支持结构化过滤和 LLM 精排
- **生成** — Markdown Awesome List，三种模式：按语言、按 AI 分类、平铺列表（可自动提交到 GitHub 仓库）
- **Release 追踪** — 订阅仓库并拉取新版本，支持增量水位
- **备份** — JSON 导出/导入 + WebDAV 上传/下载
- **标签与分类** — 批量管理仓库的自定义标签和分类，支持分类锁定
- **统计** — 按语言、分类、标签查看已同步仓库的分布
- **趋势发现** — 浏览 GitHub Trending 仓库（RSS 或 Search API），支持交互式收藏
- **配置** — 交互式配置，敏感字段支持环境变量
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

### 2. 启动 starman

```bash
starman
```

打开交互式终端界面。在里面可以同步星标、AI 分析、搜索、生成 Awesome List、管理标签分类等 — 全部键盘操作。

**TUI 页面：** 仪表盘 · 搜索 · 仓库列表 · 仓库详情 · 趋势 · 同步 · 分析 · 标签 · 分类 · 统计 · Release · 生成 · 备份 · 配置

**快捷键：** `q` 退出，`/` 搜索，`?` 帮助，`↑↓` 导航，`Enter` 确认，`Esc` 返回

## 命令用法

```
starman syncs your GitHub stars, analyzes them with AI, and generates awesome lists.

Usage:
  starman [command]

Available Commands:
  completion   Generate shell completion script
  config       Configuration management

Global Flags:
      --config string   config file path (default ~/.starman/config.yaml)
      --token string    GitHub token (overrides config/env)
      --verbose         verbose output
```

### config

```bash
starman config init      # 交互式创建配置文件
starman config show      # 显示当前配置（敏感字段脱敏）
```

### completion

```bash
starman completion <bash|zsh|fish|powershell>
```

生成 Shell 自动补全脚本。通过管道加载启用（如 `source <(starman completion zsh)`）。

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
- **Trending 双数据源** — Trending 默认使用 RSS（通过 GitHubTrendingRSS），Search API 作为备选。

## 技术栈

| 组件 | 库 |
|------|------|
| CLI 框架 | [cobra](https://github.com/spf13/cobra) + [pflag](https://github.com/spf13/pflag) |
| TUI 框架 | [bubbletea](https://github.com/charmbracelet/bubbletea) + [lipgloss](https://github.com/charmbracelet/lipgloss) |
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
  tui/                       # 终端 UI（bubbletea）
  version/                   # 版本信息（ldflags 注入）
internal/generate/templates/ # 内嵌 Markdown 模板
```

包间依赖单向流动：`cli` → 业务包 → `store`。`github` 包在内部将 go-github 类型转换为本地 `store` 类型，第三方类型不会跨包泄漏。

## License

[Apache License 2.0](LICENSE)
