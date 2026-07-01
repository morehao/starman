# Starman Sync / Analyze / Search 实现分析

## 项目概述

Starman 是一个用 AI 管理 GitHub 星标仓库的 CLI 工具。三个核心功能形成一条数据管道：**Sync（同步）→ Analyze（分析）→ Search（搜索）**。Sync 从 GitHub 拉取数据到本地 SQLite，Analyze 用 AI 对仓库进行智能分析并将结果写回 SQLite，Search 基于 SQLite 中的分析结果进行语义搜索。

---

## 一、Sync（同步）

### 入口点

| 入口类型 | 文件路径 | 关键行 |
|---------|---------|-------|
| CLI 命令定义 | `internal/cli/sync.go` | 第 13-37 行 `newSyncCmd()` |
| CLI 命令注册 | `internal/cli/root.go` | 第 22 行 `root.AddCommand(newSyncCmd())` |
| 主执行逻辑 | `internal/cli/sync.go` | 第 39-69 行 `runSync()` |
| Watch 模式 | `internal/cli/sync.go` | 第 71-93 行 `runWatch()` |

**CLI 用法：** `starman sync [--full] [--watch] [--interval 30m]`

### 核心实现逻辑

```
CLI 参数解析
    ↓
loadConfig(cmd)  — 加载 ~/.starman/config.yaml
    ↓
resolveGitHubToken(cmd, cfg)  — 按优先级：CLI flag > 环境变量 > 配置文件
    ↓
openStore()  — 打开 ~/.starman/starman.db (SQLite)
    ↓
github.New(token).ListStarred(ctx, username)  — 并发分页拉取 GitHub 星标仓库
    ↓
s.UpsertReposOnSync(ctx, repos, fullSync)  — 增量写入 DB
    ↓
s.SetSyncState(ctx, "last_sync", time)  — 记录最后同步时间
```

### 关键函数

- **`runSync()`** (`sync.go:39-69`)：单次同步的核心逻辑，组合了配置加载、GitHub API 调用、数据库写入三个步骤。

- **`github.New(token).ListStarred()`** (`github/client.go:39-85`)：
  - 使用 **go-github v71** 库，每页 100 条
  - 用 `errgroup` 并发拉取多页（最大并发 90），大幅提升同步速度
  - 内置 **httpcache** (`bartventer/httpcache`) 做 HTTP 缓存，减少重复请求
  - 第 47-48 行：首页请求后检查 rate limit，剩余不足 10 次直接报错

- **`s.UpsertReposOnSync()`** (`store/repository.go:248-311`)：
  - 在事务中逐条处理仓库
  - 对每个仓库，先通过 `full_name` 查询是否已存在（第 261 行）
  - **新仓库**：INSERT 完整数据，保留空白的 AI 字段
  - **已存在仓库**：UPDATE 元数据但**不覆盖** AI 分析结果和自定义字段 —— 这是增量同步保留 AI 分析的关键设计
  - 如果 `fullSync=true`：对比本地全部仓库，删除 GitHub 上已不再星标的仓库（第 287-309 行）

- **`runWatch()`** (`sync.go:71-93`)：使用 `time.Ticker` 定时触发 `runSync()`，默认间隔 30 分钟，最小 5 分钟。

### 依赖的外部服务/模块

| 外部依赖 | 用途 |
|---------|------|
| `github.com/google/go-github/v71` | GitHub API SDK |
| `github.com/bartventer/httpcache` | HTTP 缓存层 |
| `golang.org/x/sync/errgroup` | 并发控制 |
| `modernc.org/sqlite` | 纯 Go SQLite 驱动 |
| GitHub REST API (`Activity.ListStarred`) | 获取星标仓库列表 |

### 输入输出

| 类别 | 内容 |
|------|------|
| **输入（CLI flags）** | `--full` (bool), `--watch` (bool), `--interval` (Duration, 默认30m) |
| **输入（全局 flags）** | `--config`, `--token`, `--verbose` |
| **输入（配置）** | `config.yaml` 中的 `github.token` 和 `github.username` |
| **输入（环境变量）** | `STARMAN_GITHUB_TOKEN`, `GITHUB_TOKEN` |
| **输出** | 本地 SQLite 数据库 `~/.starman/starman.db`（`repositories` 表 + `sync_state` 表） |
| **stdout 输出** | `"Synced %d repositories (fullSync=%v)\n"` |

---

## 二、Analyze（分析）

### 入口点

| 入口类型 | 文件路径 | 关键行 |
|---------|---------|-------|
| CLI 命令定义 | `internal/cli/analyze.go` | 第 15-92 行 `newAnalyzeCmd()` |
| CLI 命令注册 | `internal/cli/root.go` | 第 24 行 |
| AI Service 层 | `internal/ai/analyze.go` | 第 25-43 行 |
| 批量分析器 | `internal/ai/batch.go` | 第 18-128 行 |
| AI 客户端 | `internal/ai/client.go` | 第 13-110 行 |
| 分类匹配逻辑 | `internal/ai/categorize.go` | 第 7-76 行 |

**CLI 用法：** `starman analyze [--all] [--repo X] [--force] [--limit N]`

### 核心实现逻辑

```
CLI 参数解析
    ↓
loadConfig(cmd)   + resolveGitHubToken + resolveAIKey
    ↓
openStore()  — 打开 SQLite
    ↓
确定分析范围：
  ├── --repo X       → s.GetRepository(ctx, name)    逐仓库查询
  ├── --force        → s.ListRepositories(ctx)        全部仓库（含已分析）
  └── 默认           → s.ListUnanalyzed(ctx, limit)   仅未分析的
    ↓
ai.NewClient(cfg.AI.BaseURL, aiKey, cfg.AI.Model)  — 创建 AI HTTP 客户端
github.New(token)  — 创建 GitHub 客户端
ai.NewService(aiClient, gh)  — 创建 AI 服务
ai.NewBatchAnalyzer(svc, s, gh, concurrency)  — 创建批量分析器
    ↓
batch.Run(ctx, repos, BatchOpts{...})  — 并发分析
    ↓  (对每个 repo 并发执行 analyzeOne)
    ├── gh.GetReadme(ctx, owner, repo)  — 从 GitHub 获取 README.md
    ├── svc.AnalyzeRepository(ctx, repo, readme, cats)  — 调用 AI 分析
    │       ↓
    │   AI 返回 JSON: {"summary":"摘要", "tags":[...], "platforms":[...], "search_text":"..."}
    │       ↓
    │   ResolveCategory(repo, tags, cats)  — 双向关键词匹配确定分类
    │       ↓
    └── s.UpdateAIResult(ctx, repo.ID, aiResult)  — 写入 DB
    ↓
s.RebuildFTSIndex(ctx)  — 分析完成后重建 FTS5 全文索引
```

### 关键函数

- **`ai.Client.Complete()`** (`ai/client.go:49-78`)：
  - 纯 HTTP 调用 OpenAI 兼容 API（POST `<baseURL>/chat/completions`）
  - 自动重试 3 次（带指数退避 1s/2s/4s）
  - 处理 429（限流）和 5xx（服务端错误）
  - 请求参数：`Temperature=0.3`, `ResponseFormat=json_object`

- **`buildAnalyzeMessages()`** (`ai/analyze.go:52-81`)：
  - System prompt 指定输出 JSON 格式
  - 传入可见分类名作为标签参考
  - User prompt 包含仓库名、语言、描述、Topics、README（截断至 8000 字）

- **`ResolveCategory()`** (`ai/categorize.go:7-26`)：
  - 如果仓库设置了 `CategoryLocked` 且有自定义分类，直接使用自定义分类
  - 先尝试匹配**用户自定义类别**的关键词
  - 再尝试匹配**系统默认类别**的关键词
  - 都匹配不上则返回 `"others"`
  - 关键词匹配算法：双向包含检查（tag 包含 keyword 或 keyword 包含 tag）

- **`BatchAnalyzer.Run()`** (`ai/batch.go:38-94`)：
  - 并发度由 `cfg.AI.Concurrency`（默认 3）控制
  - 使用 semaphore channel 做并发限流
  - 支持进度回调 `OnProgress(done, total, name)`
  - 单个仓库分析失败不中断批处理，标记 `analysis_failed`
  - 分析成功的仓库更新 `analyzed_at` 时间戳

### 依赖的外部服务/模块

| 外部依赖 | 用途 |
|---------|------|
| OpenAI 兼容 API (`/v1/chat/completions`) | AI 分析生成摘要/标签/搜索文本 |
| GitHub API (`Repositories.GetReadme`) | 获取仓库 README.md |
| `modernc.org/sqlite` (FTS5) | 全文索引 |
| 系统默认分类（14个内置分类） | 关键词匹配 |

### 输入输出

| 类别 | 内容 |
|------|------|
| **输入（CLI flags）** | `--all` (bool), `--repo` (stringSlice), `--force` (bool), `--limit` (int, 默认20) |
| **输入（配置）** | `ai.base_url`, `ai.api_key`, `ai.model`, `ai.concurrency` |
| **输入（环境变量）** | `STARMAN_AI_API_KEY` |
| **输入（DB）** | 读取 `repositories` 表 + `categories` 表 |
| **输入（GitHub）** | 获取每个仓库的 README.md |
| **输出（DB）** | `ai_summary`, `ai_tags`, `ai_platforms`, `ai_category`, `ai_search_text`, `analyzed_at`, `analysis_failed` |
| **输出（FTS）** | 重建 `repositories_fts` 虚拟表索引 |
| **stdout** | `"[%d/%d] %s"` 进度 + `"Done: %d success, %d failed, %d total\n"` |

---

## 三、Search（搜索）

### 入口点

| 入口类型 | 文件路径 | 关键行 |
|---------|---------|-------|
| CLI 命令定义 | `internal/cli/search.go` | 第 24-83 行 `newSearchCmd()` |
| CLI 命令注册 | `internal/cli/root.go` | 第 25 行 |
| 核心搜索服务 | `internal/ai/search.go` | 第 37-84 行 `Service.Search()` |
| 查询意图理解 | `internal/ai/search.go` | 第 86-112 行 `understandQuery()` |
| 加权评分 | `internal/ai/search.go` | 第 114-135 行 |
| LLM 精排 | `internal/ai/search.go` | 第 151-185 行 `rerank()` |
| FTS5 SQL 查询 | `internal/store/repository.go` | 第 313-415 行 `SearchFTS()` |
| CLI 过滤与排序 | `internal/cli/search.go` | 第 85-120 行 `filterByCLIOpts()` |

**CLI 用法：** `starman search <query> [--json] [--limit N] [--lang L] [--category C] [--sort score|stars|name] [--rerank]`

### 核心实现逻辑

```
用户输入自然语言查询（如 "终端工具"）
    ↓
阶段1：查询意图理解（LLM）
  svc.understandQuery(ctx, query)
  调用 AI 将自然语言转换为结构化搜索参数：
    {
      "fts_query": "terminal cli",
      "keywords": ["terminal", "cli"],
      "language": "",
      "category": "",
      "min_stars": 0, "max_stars": 0
    }
    ↓
阶段2：FTS5 全文检索 + BM25 加权打分
  st.SearchFTS(ctx, intent.FTSQuery, filters)
    ↓
  SQL: SELECT ... FROM repositories_fts
       JOIN repositories r ON repositories_fts.rowid = r.id
       WHERE repositories_fts MATCH ? AND ...
       ORDER BY rank LIMIT ?
    ↓
  calcWeightedScore(bm25, stars, repo, keywords)
    公式：BM25×0.6 + log(1+stars)/log(1+100000)×0.2 + keywordMatchScore×0.2
    ↓
阶段3（可选）：LLM 精排
  svc.rerank(ctx, query, hits[:topK])
  取 Top 15 候选，调用 AI 对相关性重新评分(0-10)
    ↓
CLI 后处理
  filterByCLIOpts(hits, opts)  再按 lang/category 过滤
  sortHits(hits, opts.Sort)   按 score|stars|name 排序
  截取 limit 条返回
    ↓
输出：表格 或 JSON 格式
```

### 关键函数

- **`understandQuery()`** (`ai/search.go:86-112`)：
  - 将自然语言查询转换为 FTS5 可用查询词（同义词展开、去停用词）、关键词、语言/分类约束
  - AI 调用失败时 fallback 使用原始 query 作为 FTS 查询词

- **`SearchFTS()`** (`store/repository.go:313-415`)：
  - SQLite **FTS5** 虚拟表 `repositories_fts`，内容来自 `repositories` 表
  - 索引字段：`full_name`, `description`, `ai_summary`, `ai_tags`, `ai_search_text`, `language`, `topics`
  - Tokenizer：`unicode61 remove_diacritics 2`
  - 使用 SQLite 内置的 BM25 评分
  - 支持组合过滤器：语言、分类、最低/最高星标数
  - 默认返回 50 条

- **`calcWeightedScore()`** (`ai/search.go:114-135`)：
  - **BM25 (0.6)**：全文匹配相关性
  - **星标归一化 (0.2)**：`log1p(stars)/log1p(100000)`，对数压缩
  - **关键词匹配 (0.2)**：精确关键词在 FullName（权重3）、AISearchText（权重2）、Description（权重1）中的累积加分

- **`rerank()`** (`ai/search.go:151-185`)：
  - 将 Top 15 候选信息发送给 LLM
  - LLM 对每个候选项评分 0-10
  - 按 LLM 评分重新排序

- **`filterByCLIOpts()`** (`cli/search.go:85-120`)：
  - `--lang` 按语言大小写不敏感过滤
  - `--category` 按分类过滤
  - `--sort` 支持 `score`/`stars`/`name` 三种排序
  - `--limit` 截取结果

### 依赖的外部服务/模块

| 外部依赖 | 用途 |
|---------|------|
| OpenAI 兼容 API | `understandQuery` + `rerank` |
| SQLite FTS5 | 全文检索 + BM25 评分 |
| 分析结果数据 | `ai_summary`, `ai_tags`, `ai_search_text`（由 Analyze 生成） |

> Search 的 FTS5 检索阶段不调用外部 API，仅 SQLite 本地查询。LLM 只在 `understandQuery`（必须）和 `rerank`（可选）时调用。

### 输入输出

| 类别 | 内容 |
|------|------|
| **输入（CLI args）** | `<query>` 自然语言搜索词（必选） |
| **输入（CLI flags）** | `--json`, `--limit` (int), `--lang` (string), `--category` (string), `--sort` (string, 默认score), `--rerank` (bool) |
| **输入（配置/环境变量）** | `ai.api_key` 或 `STARMAN_AI_API_KEY` |
| **输出（表格模式）** | `"SCORE  REPO  DESCRIPTION"` 三列表格 |
| **输出（JSON模式）** | `[{"score":..., "full_name":..., "language":..., "stars":..., "category":..., "summary":...}]` |

---

## 四、三者之间的关系

### 架构图

```
                    ┌─────────────────────────────────────────┐
                    │              starman CLI                 │
                    │  (internal/cli/root.go)                  │
                    └──────┬──────────┬──────────┬────────────┘
                           │          │          │
                    ┌──────▼──┐  ┌───▼────┐  ┌──▼──────┐
                    │  sync   │  │analyze │  │ search  │
                    └────┬───┘  └───┬────┘  └────┬─────┘
                         │          │─────────────│────────┐
                         │          │             │        │
                    ┌────▼──────────▼──┐          │        │
                    │   SQLite DB     │◄─────────┘        │
                    │  (store/store)  │    FTS5 检索       │
                    │                 │                    │
                    │  ┌───────────┐  │                    │
                    │  │repositories├──┤───ai_summary      │
                    │  │           ├──┤───ai_tags          │
                    │  │           ├──┤───ai_category      │
                    │  │           ├──┤───ai_search_text   │
                    │  │           ├──┤───ai_platforms     │
                    │  └───────────┘  │                    │
                    │  ┌───────────┐  │                    │
                    │  │repos_fts  │◄─┤  FTS5 虚拟表       │
                    │  └───────────┘  │                    │
                    │  ┌───────────┐  │                    │
                    │  │categories │  │                     │
                    │  └───────────┘  │                    │
                    │  ┌───────────┐  │                    │
                    │  │sync_state │  │                     │
                    │  └───────────┘  │                    │
                    └─────────────────┘                    │
                         ▲          ▲                      │
                         │          │                      │
                    ┌────┴───┐  ┌──┴──────┐               │
                    │GitHub  │  │ OpenAI  │◄──────────────┘
                    │API     │  │API      │   understandQuery
                    │        │  │         │   rerank (可选)
                    └────────┘  └─────────┘
```

### 数据依赖关系

```
sync (外部依赖: GitHub API)
    ↓ 写入
repositories 表 (元数据: full_name, language, description, topics, stargazers_count...)
    ↓ 读取
analyze (外部依赖: GitHub API + OpenAI API)
    ↓ 写入
repositories 表 (AI 字段: ai_summary, ai_tags, ai_category, ai_search_text...)
    ↓ 自动触发
FTS5 索引重建 (repositories_fts 虚拟表)
    ↓ 读取
search (外部依赖: OpenAI API 用于 query intent + 可选 rerank)
    ↓ 核心依赖
repositories_fts (FTS5 全文检索 + BM25)
```

### 核心要点

1. **Sync 是数据源头**：所有仓库元数据来自 GitHub API，增量更新时**不覆盖已有的 AI 分析结果**。

2. **Analyze 是 Search 的前提**：Search 的 FTS5 索引字段全部由 Analyze 阶段生成，未分析的仓库搜索召回率和精度会大幅降低。

3. **Search 是 Analyze 的消费者**：读取 AI 数据通过 FTS5 全文检索，配合 BM25 + 加权打分 + LLM 精排的多层排序策略。

4. **标准使用流程**：`sync → analyze → search`。Search 可脱离 Analyze 独立运行，但结果质量取决于数据完整度。

---

## 五、关键文件清单

| 功能 | 文件路径 | 说明 |
|------|---------|------|
| **全局入口** | `internal/cli/root.go` | CLI 根命令，注册所有子命令 |
| **Sync - CLI** | `internal/cli/sync.go` | sync 命令定义与执行 |
| **Sync - GitHub** | `internal/github/client.go` | ListStarred 并发分页拉取 |
| **Sync - Store** | `internal/store/repository.go` | UpsertReposOnSync 增量写入 |
| **Analyze - CLI** | `internal/cli/analyze.go` | analyze 命令定义与执行 |
| **Analyze - AI Client** | `internal/ai/client.go` | OpenAI 兼容 HTTP 客户端 |
| **Analyze - Service** | `internal/ai/analyze.go` | AnalyzeRepository 分析单个仓库 |
| **Analyze - Batch** | `internal/ai/batch.go` | BatchAnalyzer 并发批量分析 |
| **Analyze - Category** | `internal/ai/categorize.go` | ResolveCategory 关键词匹配分类 |
| **Analyze - GitHub Ops** | `internal/github/operations.go` | GetReadme 获取 README |
| **Search - CLI** | `internal/cli/search.go` | search 命令定义、过滤、输出 |
| **Search - AI Service** | `internal/ai/search.go` | Search/understandQuery/rerank 全流程 |
| **Search - FTS Store** | `internal/store/repository.go` | SearchFTS + RebuildFTSIndex |
| **数据模型** | `internal/store/models.go` | Repository/Category/AIResult/FTSResult 等 |
| **Store 接口** | `internal/store/store.go` | Store interface 定义 |
| **SQLite 实现** | `internal/store/sqlite.go` | 表结构、迁移、FTS 索引创建 |
| **配置** | `internal/config/config.go` | 配置加载、Token/APIKey 解析 |
| **公共辅助** | `internal/cli/helpers.go` | loadConfig/openStore/resolveGitHubToken |
