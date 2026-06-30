# starman 设计文档

- **日期**：2026-06-30
- **状态**：已通过设计评审，待实现
- **项目形态**：Go CLI 工具

## 1. 概述

### 1.1 目标

starman 是一个命令行工具，用于管理 GitHub 星标仓库并提供 AI 增强的整理能力。它融合两个参考项目的功能：

- **starred-go**（`github.com/juev/starred`）：拉取星标 → 按语言生成 Markdown Awesome List → 可自动提交到 GitHub 仓库 README。单功能、无状态、无 AI。
- **GithubStarsManager**（GSM）：功能丰富的星标管理应用，含 AI 分析/分类、Release 追踪、Gist/Fork 管理、语义搜索、备份等。业务逻辑与 UI 解耦较干净。

starman 取两者之长：复刻 starred-go 的全部功能，纳入 GSM 中适合 CLI 模式的功能子集，以 AI 能力作为核心增值。

### 1.2 范围

**纳入的功能模块**：

1. 星标同步 + Markdown 生成（starred-go 全部功能 + AI 分类增强）
2. AI 仓库分析（摘要、标签、平台识别）
3. AI 辅助分类（双向关键词匹配 + 锁定保护）
4. AI 关键词搜索（意图翻译 + 本地多字段打分）
5. Release 追踪（订阅、增量拉取、已读管理）
6. Star/Unstar 管理
7. 数据备份（JSON 导出/导入 + WebDAV 上传/下载）

**明确排除**：

- 向量语义搜索（依赖外部 Cloudflare Worker + Vectorize，CLI 复杂度过高）
- 多 AI 提供商原生适配（Claude/Gemini 独立适配器）——仅用 OpenAI 兼容 API
- Gist 管理、Fork 管理、Trending/Discovery（外围功能，控制范围）
- GUI/Web/Electron 界面
- 前后端同步、IndexedDB 等 Web 持久化机制

### 1.3 技术选型

| 领域 | 选型 | 理由 |
|------|------|------|
| 语言 | Go | 与 starred-go 一致，单二进制分发 |
| CLI 框架 | `spf13/cobra` + `spf13/pflag` | 子命令树、flags，Go 社区标准 |
| 配置 | `gopkg.in/yaml.v3` | 可读性强，`config init` 生成 |
| GitHub API | `google/go-github/v71` + `gregjones/httpcache` | 沿用 starred-go 验证过的方案 |
| 并发 | `sourcegraph/conc` | starred-go 验证过的并发池 |
| SQLite | `modernc.org/sqlite` | 纯 Go 驱动，无 CGO，跨平台构建简单 |
| 文本规范化 | `golang.org/x/text` | 语言名 Title Case |
| AI | OpenAI 兼容 `/v1/chat/completions` | 原生 HTTP，无 SDK，覆盖主流服务商 |

无 CGO 依赖，确保 `goreleaser` 跨平台交叉编译顺利。

## 2. 架构

### 2.1 架构方案

采用**模块化分层架构**（方案 A）。每模块单一职责、可独立测试、通过接口通信。相对的，starred-go 的扁平单包结构无法承载当前范围；GSM 的服务化架构对纯 CLI 过度工程。

### 2.2 目录结构

```
starman/
├── cmd/starman/main.go              # 入口，仅 os.Exit
├── internal/
│   ├── cli/                         # cobra 命令定义与编排
│   │   ├── root.go                  # 根命令、全局 flag、版本
│   │   ├── sync.go                  # 同步星标到本地
│   │   ├── generate.go              # 生成 Markdown
│   │   ├── analyze.go               # AI 分析（单/批量）
│   │   ├── search.go                # AI 关键词搜索
│   │   ├── release.go               # Release 追踪子命令
│   │   ├── star.go                  # star/unstar 管理
│   │   ├── backup.go                # 备份/恢复
│   │   └── config.go                # 配置初始化/查看
│   ├── github/                      # GitHub API 客户端（go-github + httpcache）
│   ├── ai/                          # OpenAI 兼容客户端 + prompt + 分类
│   ├── store/                       # SQLite 数据层（modernc.org/sqlite）
│   ├── generate/                    # Markdown 模板渲染
│   ├── release/                     # Release 追踪逻辑
│   ├── backup/                      # JSON + WebDAV 备份
│   ├── config/                      # YAML 配置加载与校验
│   └── version/                     # 版本信息（ldflags 注入）
├── templates/                       # //go:embed 内嵌模板
│   ├── by_language.tmpl             # 按语言分类（starred-go 兼容）
│   ├── by_category.tmpl             # 按 AI 分类（starman 新增）
│   └── flat.tmpl                    # 平铺列表
├── go.mod
├── README.md
└── .goreleaser.yaml
```

**包间依赖方向**：`cli` → 各业务包 → `store`；`github`/`ai` 包接收/返回 `store` 的本地类型，不向外暴露 go-github 类型。

## 3. CLI 命令树

```
starman
├── sync                         # 拉取 starred → 入库（增量合并，保留 AI 与自定义字段）
│   flags: --full                # 全量同步（删除已在 GitHub unstar 的仓库）
├── generate [output]            # 生成 Markdown；默认 stdout，-o 写文件
│   flags: -u/--username, --sort(默认 language), --repo, --message, -T/--template
├── analyze                      # AI 分析
│   flags: --all | --repo <fullName>..., --force, --limit <n>
├── search <query>               # AI 关键词搜索，输出匹配仓库
├── release
│   ├── list                     # 列出未读 release（--all 全部）
│   ├── pull                     # 拉取订阅仓库 release（全量+增量水位）
│   ├── subscribe <fullName>     # 订阅 release（写 DB 标记）
│   └── unsubscribe <fullName>   # 取消订阅
├── star <fullName>              # star 仓库
├── unstar <fullName>            # unstar 仓库
├── backup
│   ├── json [--export|--import] # JSON 文件导出/导入
│   └── webdav [--push|--pull]   # WebDAV 上传/下载
└── config
    ├── init                     # 交互式生成配置文件
    └── show                     # 打印当前配置（敏感字段脱敏）

全局 flags (root):
  --config <path>      指定配置文件（默认 ~/.starman/config.yaml）
  --token <token>      GitHub token（覆盖配置/环境变量）
  --verbose            详细日志
```

## 4. 配置管理

### 4.1 配置文件

路径：`~/.starman/config.yaml`（`config init` 交互式生成）。
数据目录：`~/.starman/`（含 `config.yaml` 与 `starman.db`）。

```yaml
github:
  token: ""           # 也可从 STARMAN_GITHUB_TOKEN / GITHUB_TOKEN 环境变量读
  username: ""        # 默认用户名（用于 generate/sync）

ai:
  base_url: "https://api.openai.com/v1"
  api_key: ""         # 也可从 STARMAN_AI_API_KEY 环境变量读
  model: "gpt-4o-mini"
  concurrency: 3      # 批量分析并发数
  custom_prompt: ""   # 可选自定义 prompt 前缀

webdav:
  url: ""
  username: ""
  password: ""        # 也可从 STARMAN_WEBDAV_PASSWORD 环境变量读
  path: "/starman"

generate:
  sort: "language"    # language | category | flat
```

### 4.2 配置优先级

CLI flag > 环境变量 > 配置文件 > 默认值

敏感字段（token/api_key/password）优先从对应环境变量读取，配置文件中可留空避免泄漏。

### 4.3 环境变量

| 变量 | 用途 |
|------|------|
| `STARMAN_GITHUB_TOKEN` | GitHub token |
| `GITHUB_TOKEN` | GitHub token（兼容 starred-go 习惯，优先级低于前者） |
| `STARMAN_AI_API_KEY` | AI API key |
| `STARMAN_WEBDAV_PASSWORD` | WebDAV 密码 |

## 5. 数据模型与 SQLite Schema

### 5.1 表结构

设计原则：复刻 GSM 核心字段，去掉前端交互字段；AI 分析结果与仓库元数据分离，支持增量同步不丢失分析结果。

```sql
-- 仓库主表（同步自 GitHub，增量更新元数据，保留本地 AI/自定义字段）
CREATE TABLE repositories (
    id                  INTEGER PRIMARY KEY,        -- GitHub repo id
    full_name           TEXT NOT NULL UNIQUE,       -- "owner/repo"
    name                TEXT NOT NULL,
    description         TEXT,
    url                 TEXT NOT NULL,              -- html_url
    language            TEXT,                       -- 主要语言（空则归 Others）
    homepage            TEXT,
    stargazers_count    INTEGER DEFAULT 0,
    forks_count         INTEGER DEFAULT 0,
    topics              TEXT,                       -- JSON array
    owner_login         TEXT,
    owner_avatar        TEXT,
    starred_at          TEXT,                       -- 星标时间（star+json 媒体类型）
    -- AI 分析字段（analyze 命令写入）
    ai_summary          TEXT,
    ai_tags             TEXT,                       -- JSON array
    ai_platforms        TEXT,                       -- JSON array
    ai_category         TEXT,                       -- AI 推断分类
    analyzed_at         TEXT,
    analysis_failed     INTEGER DEFAULT 0,
    -- 自定义字段（用户可覆盖 AI 结果）
    custom_description  TEXT,
    custom_tags         TEXT,                       -- JSON array
    custom_category     TEXT,
    category_locked     INTEGER DEFAULT 0,          -- 锁定后 analyze 不覆盖
    -- Release 订阅
    subscribed_releases INTEGER DEFAULT 0,
    last_release_fetch  TEXT,                       -- 增量水位时间戳
    -- 元数据
    created_at          TEXT DEFAULT (datetime('now')),
    updated_at          TEXT DEFAULT (datetime('now'))
);

-- Release 表（release pull 命令写入）
CREATE TABLE releases (
    id              INTEGER PRIMARY KEY,            -- GitHub release id
    repo_id         INTEGER NOT NULL,
    repo_full_name  TEXT NOT NULL,
    tag_name        TEXT NOT NULL,
    name            TEXT,
    body            TEXT,
    html_url        TEXT,
    published_at    TEXT,
    is_prerelease   INTEGER DEFAULT 0,
    is_draft        INTEGER DEFAULT 0,
    is_read         INTEGER DEFAULT 0,
    assets          TEXT,                           -- JSON array of {name, url, size, content_type}
    fetched_at      TEXT DEFAULT (datetime('now')),
    FOREIGN KEY (repo_id) REFERENCES repositories(id) ON DELETE CASCADE
);

-- 分类表（AI 分类匹配的目标分类，含默认与自定义）
CREATE TABLE categories (
    id          TEXT PRIMARY KEY,                   -- slug
    name        TEXT NOT NULL,
    keywords    TEXT,                               -- JSON array，AI 匹配用
    sort_order  INTEGER DEFAULT 0,
    is_custom   INTEGER DEFAULT 0,
    is_hidden   INTEGER DEFAULT 0
);

-- 同步水位（记录上次同步时间等元信息）
CREATE TABLE sync_state (
    key   TEXT PRIMARY KEY,
    value TEXT
);
```

### 5.2 默认分类（种子数据）

首次初始化时插入（INSERT OR IGNORE）。中英双语 keywords 用于 AI 匹配：

| id (slug) | name | keywords |
|---|---|---|
| web-app | Web 应用 | web, frontend, html, css, react, vue, frontend-framework |
| mobile-app | 移动应用 | mobile, ios, android, react-native, flutter, swift, kotlin |
| desktop-app | 桌面应用 | desktop, electron, tauri, qt, gtk |
| database | 数据库 | database, sql, nosql, redis, postgresql, mysql |
| ai-ml | AI 机器学习 | ai, machine-learning, deep-learning, llm, nlp, pytorch, tensorflow |
| dev-tools | 开发工具 | cli, build-tool, linter, debugger, ide, devtools |
| security | 安全工具 | security, crypto, vulnerability, pentest |
| game | 游戏 | game, engine, graphics, shader |
| design | 设计工具 | design, ui, color, font, icon |
| productivity | 效率工具 | productivity, automation, workflow |
| education | 教育学习 | education, tutorial, documentation, learning |
| social | 社交网络 | social, chat, forum, community |
| data-analysis | 数据分析 | analytics, visualization, dashboard, etl |
| others | 其他 | （兜底分类，无 keywords） |

### 5.3 Go 数据模型

```go
// Repository 对应 repositories 表
type Repository struct {
    ID               int64
    FullName         string
    Name             string
    Description      string
    URL              string
    Language         string
    Homepage         string
    StargazersCount  int
    ForksCount       int
    Topics           []string
    OwnerLogin       string
    OwnerAvatar      string
    StarredAt        string
    // AI 字段
    AISummary        string
    AITags           []string
    AIPlatforms       []string
    AICategory       string
    AnalyzedAt       *time.Time
    AnalysisFailed   bool
    // 自定义字段
    CustomDescription string
    CustomTags        []string
    CustomCategory    string
    CategoryLocked    bool
    // Release
    SubscribedReleases bool
    LastReleaseFetch   *time.Time
}

// Release 对应 releases 表
type Release struct {
    ID           int64
    RepoID       int64
    RepoFullName string
    TagName      string
    Name         string
    Body         string
    HTMLURL      string
    PublishedAt  string
    IsPrerelease bool
    IsDraft      bool
    IsRead       bool
    Assets       []ReleaseAsset
}

type ReleaseAsset struct {
    Name        string `json:"name"`
    URL         string `json:"url"`
    Size        int64  `json:"size"`
    ContentType string `json:"content_type"`
}

// Category 对应 categories 表
type Category struct {
    ID        string
    Name      string
    Keywords  []string
    SortOrder int
    IsCustom  bool
    IsHidden  bool
}
```

### 5.4 Store 接口

数据层用接口定义，便于测试时 mock：

```go
type Store interface {
    Close() error

    // 仓库
    UpsertRepository(ctx context.Context, r *Repository) error
    UpsertRepositories(ctx context.Context, rs []*Repository) error
    UpsertReposOnSync(ctx context.Context, rs []*Repository, fullSync bool) error
    GetRepository(ctx context.Context, fullName string) (*Repository, error)
    ListRepositories(ctx context.Context) ([]*Repository, error)
    ListUnanalyzed(ctx context.Context, limit int) ([]*Repository, error)
    ListByCategory(ctx context.Context, category string) ([]*Repository, error)
    UpdateAIResult(ctx context.Context, repoID int64, res *AIResult) error
    UpdateCustomFields(ctx context.Context, repoID int64, f *CustomFields) error

    // Release
    UpsertRelease(ctx context.Context, r *Release) error
    ListUnreadReleases(ctx context.Context) ([]*Release, error)
    ListReleasesByRepo(ctx context.Context, repoFullName string) ([]*Release, error)
    MarkReleaseRead(ctx context.Context, releaseID int64) error
    MarkAllReleasesRead(ctx context.Context) error
    SetReleaseSubscription(ctx context.Context, repoFullName string, subscribed bool) error
    UpdateReleaseWatermark(ctx context.Context, repoID int64, t time.Time) error

    // 分类
    ListCategories(ctx context.Context, visibleOnly bool) ([]*Category, error)
    UpsertCategory(ctx context.Context, c *Category) error
    DeleteCategory(ctx context.Context, id string) error

    // 同步状态
    GetSyncState(ctx context.Context, key string) (string, error)
    SetSyncState(ctx context.Context, key, value string) error
}
```

`UpsertReposOnSync` 是同步核心——它只更新元数据字段，**绝不覆盖** AI 字段和 custom 字段。

### 5.5 Schema 迁移

`store` 初始化时执行建表（IF NOT EXISTS）+ 插入默认分类（INSERT OR IGNORE）。用 `schema_version` 表追踪版本，`addColumnIfNotFound` 做兼容性 ALTER，为未来字段演进预留。

## 6. GitHub 客户端与 sync 流程

### 6.1 GitHub 客户端

`internal/github/` 封装 `Client` 结构体，沿用 starred-go 的成熟做法：

```go
type Client struct {
    client *github.Client
}

func New(token string) *Client

// 星标仓库
func (c *Client) ListStarred(ctx context.Context, username string) ([]*Repository, error)
// Star/Unstar
func (c *Client) Star(ctx context.Context, owner, repo string) error
func (c *Client) Unstar(ctx context.Context, owner, repo string) error
// Release
func (c *Client) ListReleases(ctx context.Context, owner, repo string) ([]*Release, error)
// README（AI 分析需要）
func (c *Client) GetReadme(ctx context.Context, owner, repo string) (string, error)
// 仓库内容（验证仓库是否存在）
func (c *Client) GetRepository(ctx context.Context, owner, repo string) (*Repository, error)
// README 自动提交（generate --repo 用）
func (c *Client) UpdateReadmeFile(ctx context.Context, owner, repo, content, message string) error
// 速率限制
func (c *Client) RateLimit(ctx context.Context) (*Rate, error)
```

**关键设计点**：

1. **星标拉取双模式**：`ListStarred` 同步第一页拿到 `resp.LastPage`，再用 `conc` pool（限 90 并发）拉取剩余页——复刻 starred-go 验证过的并发策略。
2. **HTTP 缓存**：`httpcache.NewMemoryCacheTransport()` 减少 ETag 重复请求。
3. **速率限制两层防护**：预检（剩余 <10 直接报错）+ 单页重试（sleep 到 reset）。
4. **数据转换**：`github.StarredRepository` → 本地 `store.Repository`，转换在 `github` 包内完成，`store` 包不依赖 go-github。

### 6.2 sync 命令流程

```
starman sync [--full]
```

```
1. 读取配置（token、username）
2. github.New(token).ListStarred(ctx, username) → 并发拉取所有 starred 仓库
3. store.UpsertReposOnSync(ctx, repos, fullSync)
   ├─ 开事务
   ├─ 构建传入 repos 的 fullName 集合
   ├─ 遍历每条 repo：
   │   SELECT existing WHERE full_name=?
   │   ├─ 不存在 → INSERT（仅元数据，AI 字段为空）
   │   └─ 存在 → UPDATE 元数据字段，AI/custom 字段不动
   └─ fullSync=true → DELETE 不在集合中的 repos（及其 releases，靠 FK CASCADE）
4. Store.SetSyncState(ctx, "last_sync", now)
5. 输出统计：新增 N、更新 M、保留 K 条 AI 分析
```

### 6.3 增量合并语义

`UpsertReposOnSync` 的字段处理：

| 字段类别 | 同步时处理 |
|---|---|
| 元数据（name/description/url/language/stars/topics/owner/starred_at） | **覆盖**为 GitHub 最新值 |
| AI 字段（ai_summary/ai_tags/ai_platforms/ai_category/analyzed_at） | **保留本地**，不覆盖 |
| 自定义字段（custom_description/custom_tags/custom_category/category_locked） | **保留本地**，不覆盖 |
| Release 订阅状态、水位 | **保留本地**，不覆盖 |

这保证 `sync` 幂等且安全——反复同步不丢失 `analyze` 和用户手动分类。

`--full` flag：默认增量（不删除）；`--full` 删除本地有但 GitHub 已 unstar 的仓库。会丢失这些仓库的 AI 分析（通过 backup 可恢复）。

### 6.4 Star/Unstar 命令与本地 DB 的同步

`star <fullName>` 和 `unstar <fullName>` 命令先调用 GitHub API 执行星标操作，成功后同步本地 DB：

- `star`：调用 `gh.Star` 成功后，调用 `gh.GetRepository` 拉取仓库元数据并 `store.UpsertRepository` 写入本地（AI 字段为空，待后续 `analyze`）。
- `unstar`：调用 `gh.Unstar` 成功后，`store` 中保留该仓库记录（不删除），仅清除其 `starred_at`。这样不丢失已积累的 AI 分析。下次 `sync --full` 时若该仓库确实不在 GitHub 星标列表中，才会被删除。

### 6.5 退出码

| 场景 | 退出码 | 说明 |
|---|---|---|
| 正常 | 0 | |
| 配置错误（缺 token/username） | 1 | 提示运行 `starman config init` |
| GitHub API 速率限制 | 2 | 打印重置时间 |
| GitHub API 其他错误 | 3 | 含网络错误、认证失败 |
| DB 错误 | 4 | |

## 7. AI 层

### 7.1 AI 客户端

单一 OpenAI 兼容客户端，通过 `base_url + api_key + model` 配置，调用 `/v1/chat/completions`：

```go
type Client struct {
    baseURL string
    apiKey  string
    model   string
    http    *http.Client
}

func New(cfg AIConfig) *Client

func (c *Client) Complete(ctx context.Context, messages []Message) (string, error)
    // POST {base_url}/chat/completions
    // Body: {model, messages, temperature, response_format:{type:"json_object"}}
    // 重试：5xx 指数退避 3 次（1s/2s/4s）；429 读取 Retry-After
```

`response_format: json_object` 强制模型返回合法 JSON，便于解析分析结果。

### 7.2 仓库分析

```go
type AnalysisResult struct {
    Summary   string   `json:"summary"`    // 一句话摘要
    Tags      []string `json:"tags"`       // 3-5 个应用类型标签
    Platforms []string `json:"platforms"`  // ["web","desktop","mobile","cli","library","service"]
}

func (s *Service) AnalyzeRepository(ctx context.Context, repo *store.Repository, readme string, cats []*store.Category) (*AnalysisResult, error)
```

**Prompt 设计**（System + User）：

```
SYSTEM:
你是一个 GitHub 仓库分析助手。根据仓库信息和 README，输出 JSON：
{"summary": "一句话摘要(中文,≤80字)", "tags": ["3-5个标签"], "platforms": ["web|desktop|mobile|cli|library|service"]}

可选分类（tags 尽量从中选取，也可补充）：[列出所有可见分类的 name]

USER:
仓库：{full_name}
语言：{language}
描述：{description}
Topics：{topics}
README（截断 8000 字）：{readme}
```

README 超长截断到 8000 字（约 2K token），避免超出上下文窗口。README 获取失败时仅用 description+topics 分析，结果质量降低但流程不中断。

### 7.3 分类解析

分析完成后立即解析分类——复刻 GSM 的双向匹配逻辑：

```go
func ResolveCategory(repo *store.Repository, aiTags []string, cats []*store.Category) string
```

**解析优先级**：

1. **`category_locked` 且 custom_category 有效** → 返回 custom_category（锁定保护，不被 AI 覆盖）
2. **AI tags × 自定义分类 keywords 双向 includes 匹配** → 返回命中的自定义分类 name
3. **AI tags × 默认分类 keywords 双向 includes 匹配** → 返回默认分类 name
4. **无匹配** → 锁定则保持现状，否则返回 `"others"`

`analyze` 命令调用 `AnalyzeRepository` 后紧接着调 `ResolveCategory`，一次事务写入 `ai_summary/ai_tags/ai_platforms/ai_category/analyzed_at`。

### 7.4 批量分析编排器

```go
type BatchAnalyzer struct {
    client    *Client
    store     store.Store
    gh        *github.Client
    concurrency int
}

func (b *BatchAnalyzer) Run(ctx context.Context, repos []*store.Repository, opts BatchOpts) error
    // opts: Force bool, Limit int, OnProgress func(done, total int, cur string)
```

**批量执行流程**：

```
1. 过滤：未分析的 或 (--force 强制重新分析)
2. 限并发：用带缓冲 channel 控制（默认 concurrency=3，可配）
   每个仓库一个 goroutine：
   a. 取 README（gh.GetReadme）
   b. 调 AI 分析（client.Complete）——受 concurrency 限制
   c. 解析分类（ResolveCategory）
   d. 写 DB（store.UpdateAIResult）——逐条提交，失败隔离不中断整批
3. 进度回调：每完成一个打印进度（done/total + 当前仓库名）
4. 支持 Ctrl+C：context cancel，已完成的结果已落库
```

容量评估：1000 仓库、concurrency=3、每次 ~3s → 约 17 分钟。进度输出让长任务可感知，Ctrl+C 安全退出已落库结果。

### 7.5 关键词搜索

复刻 GSM 的 AI 意图翻译 + 本地多字段打分（比 GSM 简化，无向量搜索）：

```go
func (s *Service) Search(ctx context.Context, query string, repos []*store.Repository) ([]*store.SearchHit, error)

type SearchHit struct {
    Repo *store.Repository
    Score float64  // 相关度
}
```

**两步策略**：

```
Step 1 - AI 意图翻译：
  输入：用户 query（如"好用的终端工具"）
  Prompt：将查询意图翻译为 3-5 个英文关键词，输出 JSON {"keywords":["terminal","cli","shell"]}
  目的：中文 query → 英文关键词，匹配英文为主的仓库描述

Step 2 - 本地多字段匹配打分：
  对每个仓库，用关键词匹配以下字段，加权累计 Score：
  - full_name:      ×3
  - description:    ×2
  - ai_summary:     ×2
  - ai_tags:        ×3
  - topics:         ×1
  - language:       ×1
  按 Score 降序，输出 Score > 0 的结果
```

不调 AI 对每个仓库打分（那样太贵太慢）——AI 只做一次 query 翻译，搜索在本地秒级完成。这是无向量搜索下的最佳折中。

### 7.6 成本控制

- API key 只在内存中，不写日志（日志脱敏）。
- `analyze` 默认只处理未分析仓库（`analyzed_at IS NULL`），`--force` 才重新分析——避免意外消耗额度。
- README 获取失败、AI 调用失败：标记 `analysis_failed=1`，批量继续，结束后汇总报告失败数。

## 8. 生成器

### 8.1 数据源

`generate` 从本地 DB 读取数据（而非实时拉取 GitHub），因为 `category` 模式依赖 AI 字段（仅存于 DB）。因此 **`generate` 前需先运行 `sync`**；若需 AI 分类，还需运行 `analyze`。若 DB 为空，`generate` 输出空文档并提示先运行 `sync`。

### 8.2 生成器结构

复刻 starred-go 的模板渲染，扩展 AI 分类模式：

```go
type SortMode string
const (
    SortLanguage SortMode = "language"  // starred-go 兼容：按编程语言分组
    SortCategory SortMode = "category"  // starman 新增：按 AI 分类分组
    SortFlat     SortMode = "flat"      // 平铺列表
)

type Generator struct {
    store store.Store
}

func (g *Generator) Generate(ctx context.Context, opts Options) ([]byte, error)
type Options struct {
    Username string
    Sort     SortMode
    Template string  // 自定义模板文件路径（覆盖内嵌模板）
}
```

### 8.3 统一渲染数据

```go
type renderData struct {
    UserName string
    Groups   map[string][]*store.Repository  // language 或 category → repos
}
```

语言模式下 `Groups` 按 `Repository.Language`（空归 "Others"）分组；分类模式下按 `custom_category`（优先）→ `ai_category` → "Others" 分组。每组内按 `FullName` 字典序排序，组间按名称字典序排序。语言名沿用 starred-go 的 `capitalize` + 映射表规范化。

### 8.4 模板

三种模板通过 `//go:embed` 内嵌，`-T` 可指定自定义模板文件覆盖。

**`by_language.tmpl`**（兼容 starred-go，含目录锚点）：按语言分节，每节列出仓库链接 + 描述。

**`by_category.tmpl`**（starman 新增）：按 AI 分类分节，展示仓库链接 + AI 摘要 + 标签。

**`flat.tmpl`**：平铺列表，无分组。

### 8.5 自定义函数

```go
template.FuncMap{
    "toLink": func(name string) string { /* 小写 + 空格替换为 -，生成锚点 ID */ },
    "join":   func(strs []string, sep string) string { /* 标签数组拼接 */ },
}
```

### 8.6 输出

- 默认 stdout（可用 `>` 重定向）
- `-o README.md` 写文件
- `--repo awesome-stars` 自动提交到 GitHub 仓库 README（通过 Contents API，带 SHA 更新）

## 9. Release 追踪

### 9.1 Tracker 结构

```go
type Tracker struct {
    store store.Store
    gh    *github.Client
}

func (t *Tracker) PullReleases(ctx context.Context) (*PullStats, error)
type PullStats struct {
    Subscribed   int
    NewReleases  int
    Errors       int
}
```

### 9.2 release pull 流程

```
1. 查询 subscribed_releases=1 的仓库列表
2. 逐个仓库（3 并发）：
   a. gh.ListReleases(owner, repo) 拉取全部 release
   b. 对比 last_release_fetch 水位：
      - 增量：只入库 published_at > watermark 的新 release
      - 首次：全量入库
   c. upsert 到 releases 表（ON CONFLICT DO UPDATE）
   d. 更新 last_release_fetch = 最新 release 的 published_at
3. 输出统计：N 个订阅仓库，新增 M 条 release
```

### 9.3 命令行为

- `release list`：查询 `is_read=0` 的 release，按 `published_at` 降序输出表格（REPO / TAG / PUBLISHED / ASSETS）。
- `release subscribe <fullName>`：`SetReleaseSubscription(true)`，并立即触发一次全量 pull 初始化水位。
- `release unsubscribe <fullName>`：取消订阅（不删除已入库 release）。
- `release list --read` / `--mark-read <id>` / `--mark-all-read`：已读管理。

## 10. 备份

### 10.1 JSON 导出/导入

```go
type Backup struct {
    Version      int                  `json:"version"`
    ExportedAt   string               `json:"exported_at"`
    Repositories []*store.Repository  `json:"repositories"`
    Releases     []*store.Release     `json:"releases"`
    Categories   []*store.Category    `json:"categories"`
}

func ExportJSON(ctx context.Context, s store.Store) ([]byte, error)
func ImportJSON(ctx context.Context, s store.Store, data []byte, mode ImportMode) error
    // mode: merge（按 full_name 合并，本地 AI 字段优先）/ replace（清表后导入）
```

### 10.2 WebDAV 备份

纯 HTTP 调用，与 GSM 实现一致（去掉浏览器专属的 CORS 提示）：

```go
type WebDAVClient struct {
    url, username, password string
    http *http.Client
}

func (w *WebDAVClient) Test(ctx context.Context) error     // PROPFIND 根目录验证连接
func (w *WebDAVClient) Push(ctx context.Context, data []byte) error
    // PUT {url}{path}/starman-backup-YYYY-MM-DD.json
func (w *WebDAVClient) Pull(ctx context.Context) ([]byte, error)
    // PROPFIND 列出文件 → 取最新的 → GET
```

WebDAV 认证用 Basic Auth（`Authorization: Basic base64(user:pass)`）。

### 10.3 命令

```
starman backup json --export [-o file]      # 默认输出 stdout 或指定文件
starman backup json --import <file>         # 从文件导入 [--mode merge|replace]
starman backup webdav --push                # 上传到 WebDAV
starman backup webdav --pull                # 从 WebDAV 拉取最新并导入（merge）
starman backup webdav --test                # 测试连接
```

## 11. 依赖清单

| 依赖 | 用途 |
|---|---|
| `spf13/cobra` + `spf13/pflag` | CLI 命令树 |
| `gopkg.in/yaml.v3` | 配置文件 |
| `google/go-github/v71` | GitHub API |
| `gregjones/httpcache` | HTTP 缓存 |
| `sourcegraph/conc` | 并发池 |
| `modernc.org/sqlite` | SQLite（纯 Go，无 CGO） |
| `golang.org/x/text` | 语言名规范化 |

无 UI 依赖、无向量服务依赖、无 CGO 依赖。

## 12. 测试策略

沿用 starred-go 的测试手法（标准库 `testing` + `httptest.NewServer` mock API），扩展覆盖范围：

- **store**：用临时 SQLite 文件测试 CRUD、`UpsertReposOnSync` 增量合并语义（重点验证 AI 字段不被覆盖）、分类匹配。
- **github**：`httptest.NewServer` mock GitHub API，测试分页合并、速率限制处理、星标/release 拉取。
- **ai**：mock HTTP server 模拟 `/v1/chat/completions`，测试分析结果解析、分类匹配、搜索打分。不打真实 API。
- **generate**：测试三种模式的模板渲染输出。
- **release**：测试增量水位逻辑。
- **backup**：测试 JSON 序列化/反序列化、WebDAV mock server。

关键不变式测试：
- `sync` 后 AI 字段不被覆盖
- `sync --full` 删除已 unstar 的仓库
- `analyze` 失败不中断批量
- `generate` 输出与 starred-go 兼容（by_language 模式）

## 13. 发布

使用 `goreleaser` 构建 linux/windows/darwin 二进制。因无 CGO 依赖，交叉编译无障碍。版本信息通过 ldflags 注入（`version`/`commit`/`date`）。
