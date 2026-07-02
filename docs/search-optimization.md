# Starman 检索方案（复刻 GithubStarsManager 三层降级）

## 概述

本文档描述将 GithubStarsManager 的三层检索降级方案复刻到 starman（Go CLI）的完整技术方案。

**核心设计**：三层降级搜索 = 向量语义搜索（接口保留，暂不实现）→ LLM 语义搜索 → FTS5/纯文本搜索。

---

## 一、三层降级搜索总流程

```
用户输入: "好看的终端工具"
     │
     ├─ 第一层：向量语义搜索（暂保留接口，不实现）
     │   条件：配置了 embedding API + 向量存储
     │   失败/未配置 → 降级到第二层
     │
     ├─ 第二层：LLM 语义搜索
     │   条件：配置了 AI API
     │   流程：HyDE 查询增强 → FTS5 全文检索 → LLM 语义重排序
     │   失败/未配置 → 降级到第三层
     │
     └─ 第三层：纯文本/FTS5 搜索
         条件：无条件（兜底）
         流程：FTS5 全文检索 + 用户自定义 filter 过滤 + 默认排序

内部流程：
  HyDE（可选，LLM 生成理想仓库描述）→ FTS5 候选召回（top 50）→ 
  加权评分（BM25 + 关键词 + 星标归一化）→ LLM 语义重排序（top 30）→ 
  filter 叠加（语言/标签/平台/状态/Star范围/排序）
```

**关键设计**：
- 每层失败自动降级，不阻断搜索流程
- 无 AI API 时搜索依然可用（FTS5 兜底）
- HyDE 和 LLM 重排序有超时保护（5 秒），失败回退

---

## 二、全新搜索入口：`Search` 方法

### 2.1 入口函数签名

```go
// file: internal/ai/search.go

// SearchResult 三层降级搜索的完整结果
type SearchResult struct {
    Hits      []*SearchHit       // 最终排序结果
    Mode      SearchMode         // 实际使用的搜索模式
    VectorHit bool               // 第一层向量搜索是否命中
}

type SearchMode string
const (
    SearchModeVector     SearchMode = "vector"     // 第一层：向量语义搜索
    SearchModeAI         SearchMode = "ai"         // 第二层：LLM 语义搜索
    SearchModeBasicText  SearchMode = "basic_text" // 第三层：纯文本/FTS5
)

// Search 三层降级搜索入口（替代当前的单路径 Search）
// query: 用户自然语言查询
// st: Store 实例
// searchOpts: 搜索参数（CLI flag 传入）
// aiOpts: AI 搜索可选增强（HyDE、重排序开关）
func (s *Service) Search(
    ctx context.Context,
    query string,
    st store.Store,
    searchOpts SearchOpts,
    aiOpts *AISearchOpts,
) (*SearchResult, error)
```

### 2.2 搜索参数结构

```go
// SearchOpts CLI 层面传入的过滤和排序参数
type SearchOpts struct {
    Language  string   // --lang，编程语言过滤
    Category  string   // --category，分类过滤
    Platform  string   // --platform，平台过滤（web/desktop/mobile/cli/library/service）
    Tags      []string // --tag，标签过滤（匹配 ai_tags + topics + custom_tags）
    MinStars  int      // --min-stars
    MaxStars  int      // --max-stars
    Sort      string   // --sort，score/stars/name/updated/starred
    Limit     int      // --limit，结果数量限制
    Analyzed  *bool    // --analyzed / --no-analyzed，AI 分析状态（与 analysis-failed 互斥）
    AnalysisFailed *bool // --analysis-failed，分析是否失败
}

// AISearchOpts AI 增强搜索的可选开关
type AISearchOpts struct {
    EnableView  bool // --vector-search，是否尝试向量搜索
    EnableHyDE  bool // --hyde，是否启用 HyDE 查询增强（默认 true）
    EnableRerank bool // --no-rerank，是否启用 LLM 语义重排序（默认 true）
    RerankTopK  int  // 重排序时取前多少个候选（默认 30）
}
```

### 2.3 主流程伪代码

```go
func (s *Service) Search(ctx context.Context, query string, st store.Store, opts SearchOpts, aiOpts *AISearchOpts) (*SearchResult, error) {
    if aiOpts == nil {
        aiOpts = &AISearchOpts{EnableHyDE: true, EnableRerank: true, RerankTopK: 30}
    }
    query = strings.TrimSpace(query)
    if query == "" {
        return &SearchResult{Mode: SearchModeBasicText, Hits: nil}, nil
    }

    // ========== 第一层：向量语义搜索 ==========
    if aiOpts.EnableView && s.isVectorConfigured() {
        result, err := s.vectorSearch(ctx, query, st, opts, aiOpts)
        if err == nil {
            result.Mode = SearchModeVector
            result.VectorHit = true
            return result, nil
        }
        // 失败降级
        log.Printf("vector search failed, falling back to AI search: %v", err)
    }

    // ========== 第二层：LLM 语义搜索 ==========
    if s.cfg.AI.APIKey != "" {
        result, err := s.aiSearch(ctx, query, st, opts, aiOpts)
        if err == nil {
            result.Mode = SearchModeAI
            return result, nil
        }
        log.Printf("AI search failed, falling back to basic search: %v", err)
    }

    // ========== 第三层：纯文本/FTS5 兜底 ==========
    result, err := s.basicTextSearch(ctx, query, st, opts)
    if err != nil {
        return nil, fmt.Errorf("basic text search: %w", err)
    }
    result.Mode = SearchModeBasicText
    return result, nil
}
```

---

## 三、第一层：向量语义搜索（接口保留，暂不实现）

向量搜索层保留完整接口定义，后续引入 sqlite-vec 或外部向量服务时直接对接。

### 3.1 接口定义

```go
// file: internal/store/store.go（新增接口）

// VectorStore 向量存储和搜索接口（暂不实现）
type VectorStore interface {
    // UpsertVectors 写入或更新向量
    UpsertVectors(ctx context.Context, vectors []*VectorRecord) error
    // SearchNN k-近邻搜索
    SearchNN(ctx context.Context, queryVector []float64, topK int, threshold float64) ([]*VectorResult, error)
    // DeleteVector 删除指定仓库的向量
    DeleteVector(ctx context.Context, repoID int64) error
    // CleanupStale 清理不在 keepIDs 中的过期向量
    CleanupStale(ctx context.Context, keepIDs []int64) error
    // Status 获取索引状态（向量数量、维度等）
    Status(ctx context.Context) (*VectorStatus, error)
}

type VectorRecord struct {
    RepoID    int64     // 仓库 ID
    Values    []float64 // embedding 向量（768/1024/1536 维）
    Dim       int       // 向量维度
    Model     string    // embedding 模型名
    TextHash  string    // 入参文本 hash（用于增量索引判断）
    IndexedAt time.Time
}

type VectorResult struct {
    RepoID int64   // 仓库 ID
    Score  float64 // 相似度分数（0~1）
}

type VectorStatus struct {
    Count int // 已索引向量数量
    Dim   int // 向量维度
}
```

### 3.2 Embedding 客户端接口

```go
// file: internal/ai/embedding.go（新增文件，暂不实现核心逻辑）

// EmbeddingProvider embedding API 提供者类型
type EmbeddingProvider string
const (
    EmbeddingOpenAI   EmbeddingProvider = "openai"
    EmbeddingOllama   EmbeddingProvider = "ollama"
    EmbeddingSiliconFlow EmbeddingProvider = "siliconflow"
)

// EmbeddingConfig embedding 配置（暂不加入 config.yaml，后续扩展）
type EmbeddingConfig struct {
    Provider EmbeddingProvider `yaml:"provider"`
    BaseURL  string            `yaml:"base_url"`
    APIKey   string            `yaml:"api_key"`
    Model    string            `yaml:"model"`
}

// EmbeddingClient 向量化客户端
type EmbeddingClient struct {
    cfg EmbeddingConfig
}

// Embed 对文本列表生成 embedding 向量
// task: "query"（查询向量）或 "document"（文档向量）
func (ec *EmbeddingClient) Embed(ctx context.Context, texts []string, task string) ([][]float64, error) {
    // TODO: 实现 OpenAI/Ollama/SiliconFlow 兼容的 embedding API 调用
    return nil, fmt.Errorf("embedding not implemented")
}
```

### 3.3 向量搜索流程（接口级保留）

```go
// file: internal/ai/search_vector.go（新增文件）
func (s *Service) vectorSearch(ctx context.Context, query string, st store.Store, opts SearchOpts, aiOpts *AISearchOpts) (*SearchResult, error) {
    // 1. HyDE 查询增强（可选，5 秒超时降级）
    embeddingQuery := query
    if aiOpts.EnableHyDE {
        hydeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
        defer cancel()
        if hydeResult, err := s.generateHyDEQuery(hydeCtx, query); err == nil {
            embeddingQuery = hydeResult
        }
        // 失败不阻断，降级用原始 query
    }

    // 2. 生成查询向量
    // queryVectors, err := s.embeddingClient.Embed(ctx, []string{embeddingQuery}, "query")

    // 3. 向量相似度搜索
    // results, err := s.vectorStore.SearchNN(ctx, queryVectors[0], 30, 0.35)

    // 4. 关键词加分
    // boosted := boostByKeywordMatch(results, query)

    // 5. 取匹配仓库做 LLM 语义重排序
    // if aiOpts.EnableRerank { ... }

    // 6. filter 叠加 + 排序
    // hits := applyFiltersAndSort(matchedRepos, opts)

    return nil, fmt.Errorf("vector search not implemented")
}
```

### 3.4 向量化索引流程

```go
// file: internal/ai/indexer.go（新增文件，暂不实现）
// 分析完成后自动触发的向量化流程
func (s *Service) IndexReposAfterAnalyze(ctx context.Context, st store.Store, repoIDs []int64) error {
    // TODO: 对已分析的仓库调用 embedding API 生成向量，存入 VectorStore
    // 增量索引：只处理 vector_indexed_at 为空或内容变更的仓库
    return nil
}
```

---

## 四、第二层：LLM 语义搜索（核心实现）

### 4.1 流程

```
HyDE 查询增强（可选）→ FTS5 全文检索（top 50）→ 加权评分 → LLM 语义重排序（top 30）
```

### 4.2 HyDE 查询增强（新增）

参考 GithubStarsManager 的 `generateHyDEQuery`，用 LLM 将用户查询转化为"理想仓库描述"，再提取关键词做 FTS5 搜索。

```go
// file: internal/ai/search_hyde.go（新增文件）

// generateHyDEQuery 用 LLM 生成假设文档（理想仓库描述）
// 超时 5 秒，失败返回原始 query
func (s *Service) generateHyDEQuery(ctx context.Context, userQuery string) (string, error) {
    msgs := []Message{
        {Role: "system", Content: `你是一个 GitHub 仓库推荐专家。用户会描述他们想要的仓库类型，请你生成一段假设的仓库说明文档，用来做语义搜索匹配。

规则：
1. 用 50-100 字的中文描述这个仓库的核心功能、适用场景、技术栈
2. 重点描述功能特性和使用场景，不要说"这是一个..."
3. 保持技术词（如 Go、Rust、React、Kubernetes 等）用英文

输出纯文本，不要 JSON。`},
        {Role: "user", Content: userQuery},
    }
    result, err := s.client.Complete(ctx, msgs)
    if err != nil {
        return userQuery, fmt.Errorf("hyde generation failed: %w", err)
    }
    return strings.TrimSpace(result), nil
}
```

### 4.3 AI 语义搜索主逻辑

```go
// file: internal/ai/search_ai.go（新增文件）

func (s *Service) aiSearch(
    ctx context.Context,
    query string,
    st store.Store,
    opts SearchOpts,
    aiOpts *AISearchOpts,
) (*SearchResult, error) {

    // 1. HyDE 查询增强（可选，失败降级）
    searchText := query
    if aiOpts.EnableHyDE {
        hydeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
        defer cancel()
        if hydeResult, err := s.generateHyDEQuery(hydeCtx, query); err == nil && hydeResult != "" {
            searchText = hydeResult
        }
    }

    // 2. LLM 查询理解：将增强后的描述转为 FTS5 查询 + 提取关键词
    intent, err := s.understandQuery(ctx, searchText)
    if err != nil {
        return nil, fmt.Errorf("understand query: %w", err)
    }

    // 3. FTS5 全文检索（候选集 50 条）
    filters := &store.SearchFilters{
        Language: coalesce(opts.Language, intent.Language),
        Category: coalesce(opts.Category, intent.Category),
        Platform: coalesce(opts.Platform, intent.Platform),
        MinStars: max(opts.MinStars, intent.MinStars),
        MaxStars: maxZero(opts.MaxStars, intent.MaxStars),
        Limit:    50,
    }
    ftsResults, err := st.SearchFTS(ctx, intent.FTSQuery, filters)
    if err != nil {
        return nil, fmt.Errorf("fts search: %w", err)
    }

    // 4. 加权评分
    hits := make([]*SearchHit, 0, len(ftsResults))
    for _, fr := range ftsResults {
        score := calcWeightedScore(fr.BM25Score, fr.Repo.StargazersCount, fr.Repo, intent.Keywords)
        hits = append(hits, &SearchHit{Repo: fr.Repo, Score: score})
    }

    if len(hits) == 0 {
        return &SearchResult{Hits: []*SearchHit{}}, nil
    }

    // 5. LLM 语义重排序（取 topK 个候选）
    if aiOpts.EnableRerank {
        topK := aiOpts.RerankTopK
        if topK <= 0 || topK > len(hits) {
            topK = len(hits)
        }
        reranked, err := s.rerank(ctx, query, hits[:topK])
        if err == nil {
            hits = reranked
        }
        // 失败不阻断，保留加权评分排序
    }

    // 6. 应用 CLI filter 排序
    sortHits(hits, opts.Sort)

    // 7. 截断到 limit
    if opts.Limit > 0 && opts.Limit < len(hits) {
        hits = hits[:opts.Limit]
    }

    return &SearchResult{Hits: hits}, nil
}
```

### 4.4 LLM 语义重排序增强

改造现有 `rerank` 方法：扩大候选集到 30 条，增加并发批量重排序。

```go
// file: internal/ai/search.go（改造 rerank 方法）

// rerank LLM 对候选仓库做语义相关性评分，返回重排序后的结果
// topK: 参与重排序的候选数（最多 50）
// 使用并发批量调用加速：每批 5 个候选仓库
func (s *Service) rerank(ctx context.Context, query string, hits []*SearchHit) ([]*SearchHit, error) {
    topK := len(hits)
    if topK > 50 {
        topK = 50
    }
    candidates := hits[:topK]

    // 并发批量重排序：每批 10 个
    batchSize := 10
    allRankings := make(map[int]float64)
    var mu sync.Mutex
    var g errgroup.Group

    for i := 0; i < len(candidates); i += batchSize {
        start := i
        end := i + batchSize
        if end > len(candidates) {
            end = len(candidates)
        }
        g.Go(func() error {
            batch := candidates[start:end]
            scores, err := s.rerankBatch(ctx, query, batch, start)
            if err != nil {
                return err
            }
            mu.Lock()
            for idx, score := range scores {
                allRankings[idx] = score
            }
            mu.Unlock()
            return nil
        })
    }

    if err := g.Wait(); err != nil {
        return nil, err
    }

    // 新数组，按 LLM 评分排序，缺失的保留原顺序
    reranked := make([]*SearchHit, len(hits))
    copy(reranked, hits)
    sort.SliceStable(reranked[:len(candidates)], func(i, j int) bool {
        si := allRankings[candidates[i].Index()]  // 需要确保 SearchHit 有唯一标识
        sj := allRankings[candidates[j].Index()]
        return si > sj
    })

    return reranked, nil
}

// rerankBatch 对一批候选仓库做 LLM 语义评分
func (s *Service) rerankBatch(ctx context.Context, query string, candidates []*SearchHit, offset int) (map[int]float64, error) {
    // 构建候选信息
    type candidateInfo struct {
        Index      int    `json:"index"`
        FullName   string `json:"full_name"`
        Summary    string `json:"summary"`
        SearchText string `json:"search_text"`
    }
    infos := make([]candidateInfo, len(candidates))
    for i, h := range candidates {
        infos[i] = candidateInfo{
            Index:      offset + i,
            FullName:   h.Repo.FullName,
            Summary:    h.Repo.AISummary,
            SearchText: h.Repo.AISearchText,
        }
    }
    candJSON, _ := json.Marshal(infos)

    msgs := []Message{
        {Role: "system", Content: fmt.Sprintf(
            `对候选仓库按查询相关性评分 (0-10)。
查询："%s"
候选仓库列表：
%s
输出 JSON：{"rankings":[{"index":%d,"score":8.5}]}`, query, string(candJSON), offset)},
    }

    resp, err := s.client.Complete(ctx, msgs)
    if err != nil {
        return nil, err
    }

    var result struct {
        Rankings []struct {
            Index int     `json:"index"`
            Score float64 `json:"score"`
        } `json:"rankings"`
    }
    if err := json.Unmarshal([]byte(resp), &result); err != nil {
        return nil, fmt.Errorf("parse rerank: %w", err)
    }

    scores := make(map[int]float64, len(result.Rankings))
    for _, r := range result.Rankings {
        scores[r.Index] = r.Score
    }
    return scores, nil
}
```

---

## 五、第三层：纯文本/FTS5 兜底搜索

当用户未配置 AI API 时，使用 FTS5 全文搜索兜底。

```go
// file: internal/ai/search_basic.go（新增文件）

func (s *Service) basicTextSearch(
    ctx context.Context,
    query string,
    st store.Store,
    opts SearchOpts,
) (*SearchResult, error) {

    // 直接使用原始 query 做 FTS5 匹配
    filters := &store.SearchFilters{
        Language: opts.Language,
        Category: opts.Category,
        Platform: opts.Platform,
        MinStars: opts.MinStars,
        MaxStars: opts.MaxStars,
        Limit:    50,
    }

    ftsResults, err := st.SearchFTS(ctx, query, filters)
    if err != nil {
        return nil, fmt.Errorf("fts search: %w", err)
    }

    hits := make([]*SearchHit, 0, len(ftsResults))
    for _, fr := range ftsResults {
        score := fr.BM25Score
        hits = append(hits, &SearchHit{Repo: fr.Repo, Score: score})
    }

    // 按指定方式排序
    sortHits(hits, opts.Sort)

    if opts.Limit > 0 && opts.Limit < len(hits) {
        hits = hits[:opts.Limit]
    }

    return &SearchResult{Hits: hits}, nil
}
```

---

## 六、FTS5 索引和 Store 改造

### 6.1 FTS5 索引字段增强

当前 FTS5 索引 7 个字段，需要增加 `ai_platforms` 字段：

```sql
-- file: internal/store/sqlite.go（改造 createFTSIndex）
CREATE VIRTUAL TABLE IF NOT EXISTS repositories_fts USING fts5(
    full_name, description, ai_summary, ai_tags, ai_platforms, ai_search_text, language, topics,
    content='repositories', content_rowid='id',
    tokenize='unicode61 remove_diacritics 2'
)
```

需要在 `migrate` 中增加一步：检测 FTS5 索引是否已包含 `ai_platforms` 列，若不包含则重建。

### 6.2 SearchFilters 扩展

```go
// file: internal/store/models.go（改造 SearchFilters）

type SearchFilters struct {
    Language string   // 编程语言精确匹配
    Category string   // 分类精确匹配
    Platform string   // 平台类型（web/desktop/mobile/cli/library/service）
    Tags     []string // 标签（匹配 ai_tags + topics + custom_tags，OR 逻辑）
    MinStars int
    MaxStars int
    Limit    int
    Analyzed        *bool // nil=不限, true=已分析, false=未分析
    AnalysisFailed *bool // nil=不限，与 Analyzed 互斥
}

// SearchFTS 改造：增加 platform、tags、analyzed、analysis_failed 条件
```

### 6.3 SearchFTS 改造

在现有 `SearchFTS` 方法中增加 filter 支持：

```go
// file: internal/store/repository.go（改造 SearchFTS）

func (s *sqliteStore) SearchFTS(ctx context.Context, query string, filters *SearchFilters) ([]*FTSResult, error) {
    where := "repositories_fts MATCH ?"
    args := []interface{}{query}

    if filters != nil {
        if filters.Language != "" {
            where += " AND r.language = ?"
            args = append(args, filters.Language)
        }
        if filters.Platform != "" {
            // ai_platforms 存的是 JSON 数组，需要用 json_each 或 LIKE 匹配
            where += " AND r.ai_platforms LIKE ?"
            args = append(args, "%"+filters.Platform+"%")
        }
        if filters.Category != "" {
            where += " AND COALESCE(NULLIF(r.custom_category,''), NULLIF(r.ai_category,''), '其他') = ?"
            args = append(args, filters.Category)
        }
        if len(filters.Tags) > 0 {
            // 标签 OR 匹配：ai_tags + topics + custom_tags 中任意一个匹配
            tagConditions := make([]string, 0, len(filters.Tags))
            for _, tag := range filters.Tags {
                tagConditions = append(tagConditions,
                    `(r.ai_tags LIKE ? OR r.topics LIKE ? OR r.custom_tags LIKE ?)`)
                args = append(args, "%"+tag+"%", "%"+tag+"%", "%"+tag+"%")
            }
            where += " AND (" + strings.Join(tagConditions, " OR ") + ")"
        }
        if filters.Analyzed != nil {
            if *filters.Analyzed {
                where += " AND r.analyzed_at IS NOT NULL AND r.analysis_failed = 0"
            } else {
                where += " AND r.analyzed_at IS NULL"
            }
        }
        if filters.AnalysisFailed != nil {
            if *filters.AnalysisFailed {
                where += " AND r.analyzed_at IS NOT NULL AND r.analysis_failed = 1"
            } else {
                where += " AND NOT (r.analyzed_at IS NOT NULL AND r.analysis_failed = 1)"
            }
        }
        if filters.MinStars > 0 { /* existing logic */ }
        if filters.MaxStars > 0 { /* existing logic */ }
    }

    // ... rest of existing logic
}
```

### 6.4 新增 sync_state 搜索缓存 KV

```go
// file: internal/store/sync_state.go（扩展）

// CacheSearchResult 缓存搜索结果（key: search:<hash>）
func (s *sqliteStore) CacheSearchResult(ctx context.Context, cacheKey string, value string, ttl time.Duration) error {
    expiresAt := time.Now().Add(ttl).Format(time.RFC3339)
    _, err := s.db.ExecContext(ctx,
        `INSERT INTO sync_state (key, value) VALUES (?, ?)
         ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
        "search_cache:"+cacheKey, value)
    return err
}

// GetCachedSearchResult 获取缓存的搜索结果
func (s *sqliteStore) GetCachedSearchResult(ctx context.Context, cacheKey string) (string, bool, error) {
    var value sql.NullString
    err := s.db.QueryRowContext(ctx,
        `SELECT value FROM sync_state WHERE key = ?`, "search_cache:"+cacheKey).Scan(&value)
    if err == sql.ErrNoRows {
        return "", false, nil
    }
    if err != nil {
        return "", false, err
    }
    return value.String, value.Valid, nil
}
```

缓存策略：
- cacheKey = `sha256(query + JSON(filters))`
- TTL = 1 小时
- 仅缓存 FTS5 检索结果（不缓存 LLM 排序后的最终结果，因为 LLM 结果非确定性）

---

## 七、CLI 命令改造

### 7.1 新增 flag

```go
// file: internal/cli/search.go（改造 newSearchCmd）

searchCmd.Flags().String("lang", "", "按编程语言过滤")
searchCmd.Flags().String("category", "", "按分类过滤")
searchCmd.Flags().String("platform", "", "按平台过滤 (web/desktop/mobile/cli/library/service)")
searchCmd.Flags().StringSlice("tag", nil, "按标签过滤（可多次使用，OR 逻辑）")
searchCmd.Flags().Int("min-stars", 0, "最低 star 数")
searchCmd.Flags().Int("max-stars", 0, "最高 star 数（0 表示不限）")
searchCmd.Flags().Bool("analyzed", false, "只显示已 AI 分析的仓库")
searchCmd.Flags().Bool("no-analyzed", false, "只显示未 AI 分析的仓库")
searchCmd.Flags().Bool("analysis-failed", false, "只显示 AI 分析失败的仓库")

searchCmd.Flags().String("sort", "score", "排序方式 (score/stars/name/updated/starred)")
searchCmd.Flags().Int("limit", 10, "结果数量限制")
searchCmd.Flags().Bool("rerank", true, "启用 LLM 语义重排序")
searchCmd.Flags().Bool("no-rerank", false, "禁用 LLM 语义重排序")
searchCmd.Flags().Bool("hyde", true, "启用 HyDE 查询增强")
searchCmd.Flags().Bool("no-hyde", false, "禁用 HyDE 查询增强")
searchCmd.Flags().Bool("json", false, "JSON 格式输出")

// 暂不启用向量搜索 flag（向量层未实现）
// searchCmd.Flags().Bool("vector", false, "启用向量语义搜索")
```

### 7.2 改造后的 runSearch

```go
func runSearch(cmd *cobra.Command, args []string) error {
    // 解析参数
    query := strings.Join(args, " ")
    lang, _ := cmd.Flags().GetString("lang")
    category, _ := cmd.Flags().GetString("category")
    platform, _ := cmd.Flags().GetString("platform")
    tags, _ := cmd.Flags().GetStringSlice("tag")
    minStars, _ := cmd.Flags().GetInt("min-stars")
    maxStars, _ := cmd.Flags().GetInt("max-stars")
    sortBy, _ := cmd.Flags().GetString("sort")
    limit, _ := cmd.Flags().GetInt("limit")
    rerank := cmd.Flags().GetBool("rerank") && !cmd.Flags().GetBool("no-rerank")
    hyde := cmd.Flags().GetBool("hyde") && !cmd.Flags().GetBool("no-hyde")
    jsonOut, _ := cmd.Flags().GetBool("json")

    // Analyzed 与 AnalysisFailed 互斥处理
    var analyzed *bool
    var analysisFailed *bool
    analyzerFlag := cmd.Flags().GetBool("analyzed")
    noAnalyzerFlag := cmd.Flags().GetBool("no-analyzer")
    analysisFailedFlag := cmd.Flags().GetBool("analysis-failed")
    if analyzerFlag || noAnalyzerFlag {
        v := !noAnalyzerFlag
        analyzed = &v
    }
    if analysisFailedFlag {
        v := true
        analysisFailed = &v
    }

    // 构建搜索参数
    opts := ai.SearchOpts{
        Language: lang, Category: category, Platform: platform,
        Tags: tags, MinStars: minStars, MaxStars: maxStars,
        Sort: sortBy, Limit: limit,
        Analyzed: analyzed, AnalysisFailed: analysisFailed,
    }
    aiOpts := &ai.AISearchOpts{
        EnableHyDE: hyde, EnableRerank: rerank, RerankTopK: 30,
    }

    // 三层降级搜索
    result, err := svc.Search(ctx, query, st, opts, aiOpts)
    // ... 输出
}
```

### 7.3 输出增强

```go
// 输出时标注搜索模式
if jsonOut {
    outputJSON(result, query)
} else {
    outputTable(result, query)
}

// 输出模式标注
fmt.Fprintf(os.Stderr, "Search mode: %s (共 %d 条结果)\n", result.Mode, len(result.Hits))
```

---

## 八、文件变更清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/ai/search.go` | 改造 | 新增 `Search` 三层降级入口，改造 `rerank` 为批量并发 |
| `internal/ai/search_ai.go` | 新增 | 第二层 LLM 语义搜索实现 |
| `internal/ai/search_basic.go` | 新增 | 第三层纯文本兜底搜索 |
| `internal/ai/search_hyde.go` | 新增 | HyDE 查询增强 |
| `internal/ai/search_vector.go` | 新增 | 第一层向量搜索（接口保留，暂不实现） |
| `internal/ai/embedding.go` | 新增 | Embedding 客户端接口（暂不实现核心逻辑） |
| `internal/store/store.go` | 改造 | 新增 `VectorStore` 接口、`CacheSearchResult`/`GetCachedSearchResult`、扩展 `SearchFilters` |
| `internal/store/repository.go` | 改造 | `SearchFTS` 增加 platform/tags/analyzed/analysis_failed 过滤 |
| `internal/store/sqlite.go` | 改造 | FTS5 索引增加 `ai_platforms` 字段 |
| `internal/store/models.go` | 改造 | `SearchFilters` 结构体扩展 |
| `internal/cli/search.go` | 改造 | 新增 flag，调用三层降级搜索，输出增强 |

---

## 九、测试策略

1. **单元测试**：
   - 各搜索模式的 filter 组合测试（language + platform + tags + analyzed 同时生效）
   - HyDE 超时降级测试（mock 5 秒延迟）
   - LLM 重排序失败降级测试（mock API 错误）
   - FTS5 查询结果缓存命中/过期测试

2. **集成测试**：
   - 无 AI API 配置时搜索正常（pure FTS5 only）
   - 有 AI API 时走 LLM 语义搜索
   - Analyzed 与 AnalysisFailed 互斥逻辑验证
