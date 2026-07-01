# starman 功能增强设计文档

- **日期**：2026-07-01
- **状态**：待评审
- **项目形态**：Go CLI 工具
- **关联文档**：`docs/superpowers/specs/2026-06-30-starman-design.md`（starman 初始设计）

## 1. 概述

### 1.1 背景

starman 核心功能（sync / generate / analyze / search / release / backup）已实现。参照 GithubStarsManager（GSM）的 GUI 功能集，从中提炼适合 CLI 形态的能力，完善 starman 的使用体验。

### 1.2 目标

本设计新增 7 个功能，分三档：

| 档 | 功能 | 命令 | 价值 |
|----|------|------|------|
| 完善 | 统计 | `stats` | 让本地数据可量化洞察 |
| 完善 | 仓库详情 | `info` | 快速查看单个仓库全貌 |
| 完善 | 搜索增强 | `search`（扩展 flags） | 过滤、排序、JSON 输出 |
| 新增 | 趋势发现 | `trending` | 发现新仓库并收藏 |
| 新增 | 批量标签/分类 | `tag` / `categorize` | 批量整理本地仓库 |
| 增强 | Shell 补全 | `completion` | 命令自动补全 |
| 增强 | 自动同步 | `sync --watch` | 定时增量同步 |

### 1.3 范围

**纳入**：上述 7 个功能。

**明确排除**：
- 向量语义搜索（经评估，维持初始设计文档决定，不引入）
- Gist 管理、Fork 管理（外围功能，控制范围）
- 本地代码仓库相关处理（fork merge-upstream、Actions 触发等）
- GUI/Web 界面

### 1.4 设计原则

- **复用现有架构**：所有新功能遵循 `cli → 业务包 → store` 分层，不引入新的架构模式
- **store 接口最小变更**：能用现有 `ListRepositories` + 内存计算解决的，不新增 SQL 聚合方法
- **零新依赖**：trending 的 RSS 解析用标准库 `encoding/xml`；completion 用 cobra 内置能力
- **渐进式**：每个功能可独立实现和测试，无跨功能依赖

## 2. stats 统计命令

### 2.1 命令形态

```
starman stats [--by language|category|tag] [--top N] [--json]
```

| Flag | 类型 | 默认 | 说明 |
|------|------|------|------|
| `--by` | string | `language` | 统计维度：`language` / `category` / `tag` |
| `--top` | int | `10` | 只显示前 N 项（0=全部） |
| `--json` | bool | `false` | JSON 格式输出 |

### 2.2 数据流

```
1. openStore()
2. store.ListRepositories(ctx) → 全部仓库
3. 内存聚合：按 --by 维度分组计数
4. 排序：按数量降序
5. 截断：--top N
6. 输出：表格或 JSON
```

仅读本地 DB，不调 GitHub API，不需要 token。

### 2.3 聚合逻辑

```go
type StatItem struct {
    Name  string `json:"name"`
    Count int    `json:"count"`
}
```

- `--by language`：按 `Repository.Language` 分组（空值归 `"Others"`），计数
- `--by category`：按有效分类分组，分类取值优先级同 `ListByCategory`（`custom_category` → `ai_category` → `"其他"`），计数
- `--by tag`：展开 `AITags` + `CustomTags`（去重），按 tag 分组计数

### 2.4 输出格式

表格模式（默认）：
```
LANGUAGE             COUNT
Go                   45
TypeScript           32
Others               18
Python               12
```

JSON 模式（`--json`）：
```json
[
  {"name":"Go","count":45},
  {"name":"TypeScript","count":32}
]
```

### 2.5 涉及文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/cli/stats.go` | 新增 | 命令定义与聚合逻辑 |
| `internal/cli/root.go` | 修改 | 注册 `stats` 命令 |

## 3. info 仓库详情命令

### 3.1 命令形态

```
starman info <fullName> [--readme] [--readme-variant <filename>]
```

| Flag | 类型 | 默认 | 说明 |
|------|------|------|------|
| `--readme` | bool | `false` | 拉取并显示 README |
| `--readme-variant` | string | `""` | 指定 README 变体文件名（如 `README_zh.md`），不指定则显示默认 README |

位置参数 `<fullName>` 必填（`owner/repo` 格式）。

### 3.2 数据流

```
1. openStore()
2. store.GetRepository(ctx, fullName) → 本地仓库数据
   ├─ 不存在 → 报错提示先 sync
3. 输出仓库元数据 + AI 摘要 + 标签 + 分类
4. 若 --readme：
   a. 若 --readme-variant 指定 → gh.GetContentFile(owner, repo, filename)
   b. 否则 → gh.GetReadme(owner, repo) 获取默认 README
   c. 若 --readme 且未指定 variant，额外列出可用变体（gh.ListReadmeVariants）
5. README 内容原样输出到 stdout
```

`--readme` 需要 GitHub token（通过 `--token`/配置/环境变量）。

### 3.3 README 多语言变体识别

参考 GSM 的 `readmeVariants.ts`，识别仓库根目录中 README 的多语言变体。

**变体匹配规则**（大小写不敏感）：
- `README.md` / `README.txt` / `README` — 默认
- `README_zh.md` / `README.zh-CN.md` / `README-zh.md` — 中文
- `README-ja.md` / `README.ko.md` / `README-es.md` — 其他语言
- `docs/README.md` — docs 子目录

匹配模式：`(?i)^readme[._-]?([a-z]{2})?\.(md|txt|markdown|rst)$` 或 `(?i)^readme$`

### 3.4 输出格式

```
Repository:  owner/repo
URL:         https://github.com/owner/repo
Language:    Go
Stars:       1234    Forks: 56
Topics:      cli, go, tool
Starred At:  2026-06-15

AI Summary:  一个用于管理 GitHub 星标的 CLI 工具
AI Tags:     cli, go, github
AI Category: 开发工具
Analyzed At: 2026-06-20

Custom Category: (none)
Custom Tags: (none)
Category Locked: false

--- README (README.md) ---
# Repo Title
...README content...
```

若无 AI 分析：显示 `(not analyzed, run 'starman analyze --repo owner/repo')`。

### 3.5 涉及文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/cli/info.go` | 新增 | 命令定义与输出 |
| `internal/github/operations.go` | 修改 | 新增 `ListReadmeVariants`、`GetContentFile` 方法 |
| `internal/cli/root.go` | 修改 | 注册 `info` 命令 |

### 3.6 GitHub 客户端新增方法

```go
// ListReadmeVariants 列出仓库根目录的 README 变体文件
func (c *Client) ListReadmeVariants(ctx context.Context, owner, repo string) ([]string, error)
// 实现：调用 Repositories.GetContents(owner, repo, "", nil) 列出根目录
// 过滤匹配 README 变体模式的文件名

// GetContentFile 获取指定路径文件内容
func (c *Client) GetContentFile(ctx context.Context, owner, repo, path string) (string, error)
// 实现：调用 Repositories.GetContents(owner, repo, path, nil)，base64 解码
```

## 4. 搜索增强

### 4.1 命令形态

```
starman search <query> [--json] [--limit N] [--lang L] [--category C] [--sort score|stars|updated|name]
```

| Flag | 类型 | 默认 | 说明 |
|------|------|------|------|
| `--json` | bool | `false` | JSON 格式输出 |
| `--limit` | int | `0` | 限制结果数（0=不限） |
| `--lang` | string | `""` | 按语言过滤 |
| `--category` | string | `""` | 按分类过滤（匹配 `custom_category` → `ai_category`） |
| `--sort` | string | `score` | 排序：`score` / `stars` / `updated` / `name` |

### 4.2 改动点

现有 `search` 命令（`internal/cli/search.go:12-49`）的输出仅 tab 分隔文本，无过滤/排序/格式选项。本次扩展：

1. **过滤**：在 AI 搜索打分后，按 `--lang` / `--category` 过滤 hits
2. **排序**：`--sort` 控制最终排序（`score` 默认保持现有逻辑；`stars`/`updated`/`name` 在 hits 上重排）
3. **截断**：`--limit` 截断结果数
4. **格式**：`--json` 输出 JSON 数组，默认保持 tab 分隔（但增加列对齐）

### 4.3 数据流

```
1. 加载配置（需 AI api_key）
2. openStore() → ListRepositories(ctx)
3. ai.Service.Search(ctx, query, repos) → hits（现有逻辑不变）
4. 过滤：--lang / --category
5. 排序：--sort
6. 截断：--limit
7. 输出：表格或 JSON
```

### 4.4 输出格式

表格模式（默认，改进对齐）：
```
SCORE  REPO                DESCRIPTION
   12  owner/repo           A CLI tool for...
    9  owner2/repo2         Another tool...
```

JSON 模式（`--json`）：
```json
[
  {"score":12,"full_name":"owner/repo","description":"A CLI tool for...","language":"Go","stars":1234}
]
```

### 4.5 涉及文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/cli/search.go` | 修改 | 新增 flags、过滤/排序/格式逻辑 |

## 5. trending 趋势发现命令

### 5.1 命令形态

```
starman trending [--since daily|weekly|monthly] [--lang L] [--top N] [--source rss|search] [--star]
```

| Flag | 类型 | 默认 | 说明 |
|------|------|------|------|
| `--since` | string | `weekly` | 时间范围：`daily` / `weekly` / `monthly` |
| `--lang` | string | `""` | 按语言过滤（客户端过滤） |
| `--top` | int | `20` | 显示前 N 个（也限制 API 补充调用量） |
| `--source` | string | `rss` | 数据源：`rss`（GitHubTrendingRSS）/ `search`（GitHub Search API） |
| `--star` | bool | `false` | 交互式收藏结果中选中的仓库 |

### 5.2 数据源

#### 方案 A：RSS（默认，`--source rss`）

参考 GSM（`githubApi.ts:1195-1329`），使用第三方 `mshibanami/GitHubTrendingRSS` 项目：

```
https://mshibanami.github.io/GitHubTrendingRSS/{daily|weekly|monthly}/all.xml
```

RSS feed 返回全部 trending 仓库，一次拉取后在客户端切片和过滤。

**RSS XML 结构**：
```xml
<rss>
  <channel>
    <item>
      <title>owner/repo</title>
      <link>https://github.com/owner/repo</link>
      <description>⭐ 1,234 | 🍴 56 | Description text...</description>
    </item>
  </channel>
</rss>
```

**解析流程**：
```
1. HTTP GET RSS URL → XML 文本
2. encoding/xml 解析 → 提取 <item> 列表
3. 每个 item 提取：
   - title → owner/repo
   - link → 仓库 URL
   - description → 正则提取 ⭐ 星数、🍴 fork 数，剩余文本为描述
4. 截取前 --top 个
5. 字段补充：RSS 缺 language/topics/准确时间
   对每个仓库调 gh.GetRepository 补全（复用现有方法）
   每请求间 sleep 80ms（避免限流，同 GSM 策略）
6. --lang 过滤（补充字段后）
```

#### 方案 B：Search API（兜底，`--source search`）

使用 GitHub 官方 Search API 近似趋势：

```
GET /search/repositories?q=stars:>1000+created:>2026-06-01&sort=stars&order=desc
```

`created` 日期根据 `--since` 计算（daily=近 7 天、weekly=近 30 天、monthly=近 90 天）。

用 go-github 的 `client.Search.Repositories`。

**降级**：`--source rss` 的 RSS 拉取失败时，提示用户可尝试 `--source search`。

### 5.3 数据模型

```go
type TrendingRepo struct {
    Rank        int
    FullName    string
    Description string
    URL         string
    Stars       int
    Forks       int
    Language    string
    Topics      []string
}
```

### 5.4 `--star` 交互式收藏

```
1. trending 结果输出表格
2. 若 --star：
   a. 读 stdin 行号（用户输入序号，支持多选用逗号分隔）
   b. 对选中的仓库调用 gh.Star + store.UpsertRepository（复用现有 star 逻辑）
   c. 输出收藏结果
```

### 5.5 输出格式

```
RANK  REPO                    STARS  LANGUAGE  DESCRIPTION
   1  owner/repo               1234  Go        A CLI tool for...
   2  owner2/repo2              987  Python    Another tool...
```

### 5.6 涉及文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/cli/trending.go` | 新增 | 命令定义、编排 |
| `internal/discovery/trending.go` | 新增 | RSS 解析 + search fallback + 字段补充 |
| `internal/cli/root.go` | 修改 | 注册 `trending` 命令 |

### 5.7 discovery 包设计

```go
package discovery

type Service struct {
    gh    *github.Client
    http  *http.Client  // 用于 RSS 拉取
}

func NewService(gh *github.Client) *Service

func (s *Service) Trending(ctx context.Context, opts TrendingOpts) ([]*TrendingRepo, error)
type TrendingOpts struct {
    Since   string  // daily|weekly|monthly
    Lang    string
    Top     int
    Source  string  // rss|search
}

// RSS 解析
func parseRSS(xmlData []byte) ([]*TrendingRepo, error)

// Search fallback
func (s *Service) trendingViaSearch(ctx context.Context, opts TrendingOpts) ([]*TrendingRepo, error)

// 字段补充
func (s *Service) enrichRepos(ctx context.Context, repos []*TrendingRepo) error
```

## 6. 批量标签/分类命令

### 6.1 命令形态

```
# 单仓库操作
starman tag <fullName> [+tag1,+tag2,-tag3]
starman categorize <fullName> <category> [--lock] [--unlock]

# 批量操作（按过滤条件）
starman tag --lang Go --add awesome,cli
starman tag --category "开发工具" --remove old-tag
starman categorize --lang Python "AI 机器学习" --lock
```

### 6.2 tag 命令

```
starman tag <fullName> <tagExpr>          # 单仓库
starman tag --lang L|--category C --add t1,t2 [--remove t3,t4]  # 批量
```

| 参数/Flag | 类型 | 说明 |
|-----------|------|------|
| `fullName` | 位置参数 | 单仓库模式时必填 |
| `tagExpr` | 位置参数 | 标签表达式：`+awesome,-old,+cli` |
| `--add` | string | 批量模式：添加标签（逗号分隔） |
| `--remove` | string | 批量模式：移除标签（逗号分隔） |
| `--lang` | string | 批量过滤：按语言 |
| `--category` | string | 批量过滤：按分类 |

**标签操作语义**：
- 操作 `CustomTags` 字段（不影响 `AITags`）
- `+tag` 添加，`-tag` 移除
- 批量模式下 `--add` / `--remove` 逗号分隔

**单仓库模式**：
```
1. store.GetRepository(ctx, fullName)
2. 解析 tagExpr → add set / remove set
3. 更新 CustomTags：当前 tags + add - remove（去重）
4. store.UpdateCustomFields(ctx, repo.ID, updatedFields)
```

**批量模式**（有 `--lang` 或 `--category` flag 时）：
```
1. store.ListRepositories(ctx) → 全部仓库
2. 过滤：--lang / --category
3. 对每个匹配仓库执行标签增删
4. store.UpdateCustomFields 逐个写入
5. 输出：已更新 N 个仓库
```

### 6.3 categorize 命令

```
starman categorize <fullName> <category> [--lock] [--unlock]
starman categorize --lang L <category> [--lock]
```

| 参数/Flag | 类型 | 说明 |
|-----------|------|------|
| `fullName` | 位置参数 | 单仓库模式时必填 |
| `category` | 位置参数 | 分类名称 |
| `--lock` | bool | 锁定分类（analyze 不覆盖） |
| `--unlock` | bool | 解锁分类 |
| `--lang` | string | 批量过滤：按语言 |
| `--category` | string | 批量过滤：按现有分类 |

**操作语义**：
- 操作 `CustomCategory` 字段和 `CategoryLocked` 字段
- `--lock` 设置 `CategoryLocked=true`，防止 `analyze` 命令覆盖
- `--unlock` 设置 `CategoryLocked=false`

### 6.4 涉及文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/cli/tag.go` | 新增 | tag 命令定义与逻辑 |
| `internal/cli/categorize.go` | 新增 | categorize 命令定义与逻辑 |
| `internal/cli/root.go` | 修改 | 注册两个命令 |

### 6.5 与现有代码的复用

- 标签/分类更新复用 `store.UpdateCustomFields`（`store/repository.go:204`），`CustomFields` 结构体（`store/models.go:71-76`）已包含 `Description`/`Tags`/`Category`/`CategoryLocked` 四个字段
- 单仓库查 `GetRepository`，批量查 `ListRepositories`，均无需新增 store 方法

## 7. completion Shell 补全命令

### 7.1 命令形态

```
starman completion <bash|zsh|fish|powershell>
```

### 7.2 实现

cobra 内置 `Cobra.GenBashCompletion` / `GenZshCompletion` / `GenFishCompletion` / `GenPowerShellCompletionWithDesc`。只需注册命令调用对应生成函数。

```go
func newCompletionCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "completion <shell>",
        Short: "Generate shell completion script",
        Args:  cobra.ExactArgs(1),
        ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
        RunE: func(cmd *cobra.Command, args []string) error {
            switch args[0] {
            case "bash":
                return root.GenBashCompletion(os.Stdout)
            case "zsh":
                return root.GenZshCompletion(os.Stdout)
            case "fish":
                return root.GenFishCompletion(os.Stdout)
            case "powershell":
                return root.GenPowerShellCompletionWithDesc(os.Stdout)
            }
            return fmt.Errorf("unsupported shell: %s", args[0])
        },
    }
    return cmd
}
```

### 7.3 涉及文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/cli/completion.go` | 新增 | 补全命令定义 |
| `internal/cli/root.go` | 修改 | 注册 `completion` 命令 |

### 7.4 安装提示

命令执行后输出安装指引：
```
# Bash
source <(starman completion bash)

# Zsh
source <(starman completion zsh)

# Fish
starman completion fish | source
```

## 8. sync --watch 自动同步

### 8.1 命令形态

```
starman sync [--full] [--watch] [--interval 30m]
```

| Flag | 类型 | 默认 | 说明 |
|------|------|------|------|
| `--full` | bool | `false` | 全量同步（现有） |
| `--watch` | bool | `false` | 启用定时同步模式 |
| `--interval` | duration | `30m` | 同步间隔（`--watch` 模式下生效） |

### 8.2 行为

**单次模式**（默认，无 `--watch`）：与现有 `sync` 完全一致，互不影响。

**watch 模式**（`--watch`）：
```
1. 立即执行一次 sync（同现有逻辑）
2. 用 time.Ticker 定时触发 sync
3. 每次 sync 输出统计到 stderr（含时间戳）
4. Ctrl+C 优雅退出（context cancel）
5. --full 在每次定时同步时生效
```

### 8.3 输出示例

```
[2026-07-01T10:00:00Z] Synced 120 repositories (fullSync=false)
[2026-07-01T10:30:00Z] Synced 122 repositories (fullSync=false)
[2026-07-01T11:00:00Z] Synced 122 repositories (fullSync=false)
^C
Stopping watch... last sync at 2026-07-01T11:00:00Z
```

### 8.4 涉及文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/cli/sync.go` | 修改 | 新增 `--watch`/`--interval` flags，watch 循环逻辑 |

### 8.5 设计要点

- **不引入守护进程机制**：watch 模式是前台阻塞循环，用户通过 `nohup` 或终端窗口保持运行。这符合 CLI 工具的无守护进程惯例
- **间隔下限**：`--interval` 最小 5 分钟，防止频繁同步触发 GitHub rate limit
- **错误不中断**：单次 sync 失败输出错误到 stderr，继续等待下次 ticker

## 9. Store 接口变更

**本次设计不新增 store 接口方法**。所有新功能通过现有接口实现：

| 功能 | 使用的现有方法 |
|------|---------------|
| stats | `ListRepositories` |
| info | `GetRepository` |
| 搜索增强 | `ListRepositories` |
| 批量标签/分类 | `ListRepositories` + `GetRepository` + `UpdateCustomFields` |

理由：star 仓库典型量级几百到几千，全量加载内存聚合的性能开销可忽略（毫秒级）。新增 SQL 聚合查询会增加接口复杂度且收益有限。

## 10. GitHub 客户端变更

新增 3 个方法（`internal/github/operations.go`）：

```go
// ListReadmeVariants 列出仓库根目录的 README 变体文件名
func (c *Client) ListReadmeVariants(ctx context.Context, owner, repo string) ([]string, error)

// GetContentFile 获取指定路径文件内容（base64 解码后返回）
func (c *Client) GetContentFile(ctx context.Context, owner, repo, path string) (string, error)

// SearchRepositories 通过 GitHub Search API 搜索仓库（trending --source search 用）
func (c *Client) SearchRepositories(ctx context.Context, query string, opts *gh.SearchOptions) ([]*Repository, error)
```

实现均基于 go-github 现有方法，无新依赖。

## 11. 配置变更

**无配置项变更**。trending 命令的 RSS URL 和 search API 参数在代码中硬编码（与 GSM 一致）。

## 12. 更新后的 CLI 命令树

```
starman
├── sync                         # 同步星标（现有）
│   flags: --full, --watch, --interval
├── generate [output]            # 生成 Markdown（现有）
├── analyze                      # AI 分析（现有）
├── search <query>               # AI 搜索（现有，增强）
│   flags: --json, --limit, --lang, --category, --sort
├── release                      # Release 追踪（现有）
│   ├── list / pull / subscribe / unsubscribe
├── star <fullName>              # star（现有）
├── unstar <fullName>            # unstar（现有）
├── backup                       # 备份（现有）
│   ├── json / webdav
├── config                       # 配置（现有）
│   ├── init / show
├── stats                        # 统计（新增）
│   flags: --by, --top, --json
├── info <fullName>              # 仓库详情（新增）
│   flags: --readme, --readme-variant
├── trending                     # 趋势发现（新增）
│   flags: --since, --lang, --top, --source, --star
├── tag                          # 标签管理（新增）
│   单仓库: tag <fullName> <tagExpr>
│   批量: tag --lang L --add t1 --remove t2
├── categorize                   # 分类管理（新增）
│   单仓库: categorize <fullName> <category> [--lock]
│   批量: categorize --lang L <category> [--lock]
└── completion <shell>           # Shell 补全（新增）

全局 flags (root): --config, --token, --verbose
```

## 13. 测试策略

沿用现有测试手法（`testing` + `httptest.NewServer`）：

| 功能 | 测试要点 |
|------|---------|
| stats | 内存聚合正确性：language/category/tag 分组计数、排序、`--top` 截断、JSON 输出 |
| info | 本地数据缺失报错；README 变体正则匹配；`--readme` 调用 mock GitHub API |
| 搜索增强 | 过滤（lang/category）、排序（stars/updated/name）、`--limit` 截断、JSON 输出 |
| trending | RSS XML 解析（mock RSS server）；search fallback（mock GitHub Search API）；字段补充；`--lang` 过滤 |
| tag | 单仓库标签增删；批量过滤与更新；tagExpr 解析 |
| categorize | 单仓库分类设置；`--lock`/`--unlock`；批量分类 |
| completion | 生成的脚本非空（集成测试，验证 cobra 生成不报错） |
| sync --watch | ticker 逻辑：验证至少执行 2 次同步后被 context cancel 终止 |

**关键不变式**：
- `tag` 操作不影响 `AITags`
- `categorize --lock` 后 `analyze` 不覆盖 `CustomCategory`
- `trending --source rss` 失败时明确报错并提示 `--source search`
- `sync --watch` 的 `--interval` 下限 5 分钟

## 14. 依赖清单

无新增依赖。全部功能基于现有依赖实现：

| 依赖 | 新增用途 |
|------|---------|
| `encoding/xml`（标准库） | trending RSS 解析 |
| `net/http`（标准库） | trending RSS 拉取 |
| `spf13/cobra`（现有） | completion 命令生成补全脚本 |
| `google/go-github/v71`（现有） | trending search fallback、info README 变体 |

## 15. 实现顺序建议

按依赖关系和价值排序：

1. **completion**（最简单，cobra 内置，独立）
2. **stats**（仅读 DB，独立，快速见效）
3. **搜索增强**（改现有文件，提升日常使用体验）
4. **info**（需 GitHub 客户端新增方法，但独立）
5. **tag / categorize**（一组关联功能，可并行实现）
6. **trending**（最复杂，新增 discovery 包）
7. **sync --watch**（改现有 sync，需注意不破坏单次模式）
