# Starman 检索方案（复刻 GithubStarsManager 三层降级）

## 概述

本文档描述将 GithubStarsManager 的三层检索降级方案复刻到 starman（Go CLI）的完整技术方案。

**核心设计**：三层降级搜索 = 向量语义搜索 → LLM 语义搜索 → FTS5/纯文本搜索。

> **架构说明**：引入向量搜索需要 sqlite-vec。`modernc.org/sqlite` v1.53.0 已将 `sqlite-vec` 扩展内置为纯 Go 实现（`vec/` 子包），无需切换驱动或依赖 CGO。

---

## 一、三层降级搜索总流程

```
用户输入: "好看的终端工具"
     │
     ├─ 第一层：向量语义搜索（完整实现）
     │   条件：配置了 embedding API（base_url + api_key + model）
     │   流程：HyDE 查询增强 → Embedding API 生成查询向量 → sqlite-vec kNN →
     │         关键词加分 → LLM 语义重排序
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
```

**关键设计**：
- 每层失败自动降级，不阻断搜索流程
- 向量搜索是**优选路径**而非唯一路径，无 embedding API 时自动降级
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
}

type SearchMode string
const (
    SearchModeVector    SearchMode = "vector"     // 第一层：向量语义搜索
    SearchModeAI        SearchMode = "ai"         // 第二层：LLM 语义搜索
    SearchModeBasicText SearchMode = "basic_text" // 第三层：纯文本/FTS5
)

// Search 三层降级搜索入口（替代当前的单路径 Search）
func (s *Service) Search(
    ctx context.Context,
    query string,
    st store.Store,
    searchOpts SearchOpts,
) (*SearchResult, error)
```

### 2.2 搜索参数结构

```go
// SearchOpts CLI 层面传入的过滤和排序参数
type SearchOpts struct {
    Language       string   // --lang
    Category       string   // --category
    Platform       string   // --platform（web/desktop/mobile/cli/library/service）
    Tags           []string // --tag（匹配 ai_tags + topics + custom_tags，OR 逻辑）
    MinStars       int      // --min-stars
    MaxStars       int      // --max-stars
    Sort           string   // --sort（score/stars/name/updated/starred）
    Limit          int      // --limit
    Analyzed       *bool    // --analyzed / --no-analyzed（与 analysis-failed 互斥）
    AnalysisFailed *bool    // --analysis-failed
    EnableHyDE     bool     // --hyde / --no-hyde（默认 true）
    EnableRerank   bool     // --rerank / --no-rerank（默认 true）
    RerankTopK     int      // 重排序候选数（默认 30）
}
```

### 2.3 主流程

```go
func (s *Service) Search(ctx context.Context, query string, st store.Store, opts SearchOpts) (*SearchResult, error) {
    query = strings.TrimSpace(query)
    if query == "" {
        return &SearchResult{Mode: SearchModeBasicText, Hits: nil}, nil
    }

    // ========== 第一层：向量语义搜索 ==========
    if s.hasEmbeddingConfig() {
        result, err := s.vectorSearch(ctx, query, st, opts)
        if err == nil && len(result.Hits) > 0 {
            result.Mode = SearchModeVector
            return result, nil
        }
        // 失败或无结果：降级
        log.Printf("vector search: %v（%d hits）, falling back", err, len(result.Hits))
    }

    // ========== 第二层：LLM 语义搜索 ==========
    if s.hasAIConfig() {
        result, err := s.aiSearch(ctx, query, st, opts)
        if err == nil {
            result.Mode = SearchModeAI
            return result, nil
        }
        log.Printf("AI search failed: %v, falling back", err)
    }

    // ========== 第三层：纯文本/FTS5 兜底 ==========
    result, err := s.basicTextSearch(ctx, query, st, opts)
    if err != nil {
        return nil, fmt.Errorf("basic text search: %w", err)
    }
    result.Mode = SearchModeBasicText
    return result, nil
}

func (s *Service) hasEmbeddingConfig() bool {
    return s.embeddingClient != nil
}

func (s *Service) hasAIConfig() bool {
    return s.cfg.AI.APIKey != ""
}
```

---

## 三、配置设计

### 3.1 config.yaml 配置

在 `~/.starman/config.yaml` 中新增 `embedding` 区块，与 `ai` 同级：

```yaml
ai:
  base_url: "https://api.openai.com/v1"
  api_key: "${AI_API_KEY}"
  model: "gpt-4o-mini"

embedding:
  base_url: "https://api.openai.com/v1"   # OpenAI 兼容 API 地址
  api_key: "${EMBEDDING_API_KEY}"          # 为空则不启用向量搜索
  model: "text-embedding-3-small"          # 默认 1536 维
```

### 3.2 Config 结构体

```go
// file: internal/config/config.go（改造）

type Config struct {
    GitHub    GitHubConfig    `yaml:"github"`
    AI        AIConfig        `yaml:"ai"`
    Embedding EmbeddingConfig `yaml:"embedding"`
    WebDAV    WebDAVConfig    `yaml:"webdav"`
    Generate  GenerateConfig  `yaml:"generate"`
}

type EmbeddingConfig struct {
    BaseURL string `yaml:"base_url"`
    APIKey  string `yaml:"api_key"`
    Model   string `yaml:"model"`
}
```

### 3.3 配置解析优先级

```go
// file: internal/config/config.go（新增）

func ResolveEmbeddingKey(cfg *Config, flagKey string) string {
    if flagKey != "" {
        return flagKey
    }
    if v := os.Getenv("STARMAN_EMBEDDING_API_KEY"); v != "" {
        return v
    }
    return cfg.Embedding.APIKey
}

func Default() *Config {
    return &Config{
        AI: AIConfig{
            BaseURL:     "https://api.openai.com/v1",
            Model:       "gpt-4o-mini",
            Concurrency: 3,
        },
        Embedding: EmbeddingConfig{
            BaseURL: "https://api.openai.com/v1",
            Model:   "text-embedding-3-small",
        },
        Generate: GenerateConfig{Sort: "language"},
        WebDAV:   WebDAVConfig{Path: "/starman"},
    }
}
```

---

## 四、Embedding 客户端（完整实现）

```go
// file: internal/ai/embedding.go（新增文件）

package ai

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "strings"
    "time"
)

// EmbeddingClient OpenAI 协议 embedding API 客户端
type EmbeddingClient struct {
    baseURL  string
    apiKey   string
    model    string
    client   *http.Client
}

// NewEmbeddingClient 创建 embedding 客户端
// 若 apiKey 为空则返回 nil（向量功能不可用，表示用户未配置）
func NewEmbeddingClient(baseURL, apiKey, model string) *EmbeddingClient {
    if apiKey == "" || baseURL == "" {
        return nil
    }
    return &EmbeddingClient{
        baseURL: strings.TrimRight(baseURL, "/"),
        apiKey:  apiKey,
        model:   model,
        client:  &http.Client{Timeout: 30 * time.Second},
    }
}

// embeddingRequest OpenAI embedding API 请求体
type embeddingRequest struct {
    Input          interface{} `json:"input"`          // string 或 []string
    Model          string      `json:"model"`
    EncodingFormat string      `json:"encoding_format,omitempty"`
}

// embeddingResponse OpenAI embedding API 响应
type embeddingResponse struct {
    Data []struct {
        Embedding []float64 `json:"embedding"`
        Index     int       `json:"index"`
    } `json:"data"`
    Usage struct {
        TotalTokens int `json:"total_tokens"`
    } `json:"usage"`
}

// Embed 对文本列表生成 embedding 向量
// 单次最多 32 条，超过自动分批
func (ec *EmbeddingClient) Embed(ctx context.Context, texts []string) ([][]float64, error) {
    if ec == nil {
        return nil, fmt.Errorf("embedding client not configured")
    }
    return ec.embedBatch(ctx, texts)
}

func (ec *EmbeddingClient) embedBatch(ctx context.Context, texts []string) ([][]float64, error) {
    body := embeddingRequest{
        Input:          texts,
        Model:          ec.model,
        EncodingFormat: "float",
    }
    bodyBytes, err := json.Marshal(body)
    if err != nil {
        return nil, fmt.Errorf("marshal request: %w", err)
    }

    req, err := http.NewRequestWithContext(ctx, "POST", ec.baseURL+"/embeddings", bytes.NewReader(bodyBytes))
    if err != nil {
        return nil, fmt.Errorf("create request: %w", err)
    }
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+ec.apiKey)

    resp, err := ec.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("do request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        var errBody bytes.Buffer
        errBody.ReadFrom(resp.Body)
        return nil, fmt.Errorf("embedding API error %d: %s", resp.StatusCode, errBody.String())
    }

    var result embeddingResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("decode response: %w", err)
    }

    vectors := make([][]float64, len(result.Data))
    for _, d := range result.Data {
        vectors[d.Index] = d.Embedding
    }
    return vectors, nil
}
```

**错误处理策略**（参考 GithubStarsManager）：
- 网络错误 → 重试（由调用方的 `retryWithBackoff` 处理）
- 返回长度超限错误（常见于 `text-embedding-3-small` 的 8191 token 限制）→ 自动截断文本后重试
- API key 无效（401）→ 日志警告，降级到第二层

---

## 五、sqlite-vec 向量存储（完整实现）

### 5.1 数据库迁移

```go
// file: internal/store/sqlite.go（改造）

// 数据库驱动不变，继续使用 modernc.org/sqlite（纯 Go，无 CGO）
// sqlite-vec 已内置在 modernc.org/sqlite v1.53.0 的 vec/ 子包中
// vec0 虚拟表可以直接创建，无需手动加载扩展
func Open(path string) (Store, error) {
    db, err := sql.Open("sqlite", path+"?_journal_mode=WAL&_foreign_keys=on&_busy_timeout=5000")
    if err != nil {
        return nil, fmt.Errorf("open db: %w", err)
    }

    // ... 原有 WAL、迁移逻辑（在 migrate 中新增 vec0 虚拟表创建）
}
```

### 5.2 vec0 虚拟表 & Repository 模型扩展

```sql
-- file: internal/store/sqlite.go（migrate 中新增步骤）

-- 创建向量虚拟表（用于存储仓库 README/描述的 embedding）
CREATE VIRTUAL TABLE IF NOT EXISTS repo_vectors USING vec0(
    embedding float[1536]
);

-- repositories 表增加向量索引时间戳列
ALTER TABLE repositories ADD COLUMN vector_indexed_at TEXT;
```

```go
// file: internal/store/models.go（改造 Repository）

type Repository struct {
    // ... 现有字段
    VectorIndexedAt *time.Time `json:"vector_indexed_at,omitempty"` // 向量索引时间
}
```

### 5.3 VectorStore 接口实现

```go
// file: internal/store/vector.go（新增文件）

package store

import (
    "context"
    "encoding/json"
    "fmt"
    "math"
    "strings"
)

// InsertVector 写入或更新仓库向量
// embedding 长度必须匹配 vec0 定义的维度（如 1536）
func (s *sqliteStore) InsertVector(ctx context.Context, repoID int64, embedding []float64) error {
    // 将向量序列化为 JSON 数组
    vecJSON, _ := json.Marshal(embedding)

    // 先删除旧向量，再插入新向量
    _, err := s.db.ExecContext(ctx, `DELETE FROM repo_vectors WHERE rowid = ?`, repoID)
    if err != nil {
        return fmt.Errorf("delete old vector: %w", err)
    }

    _, err = s.db.ExecContext(ctx,
        `INSERT INTO repo_vectors (rowid, embedding) VALUES (?, ?)`,
        repoID, string(vecJSON))
    if err != nil {
        return fmt.Errorf("insert vector: %w", err)
    }

    return nil
}

// SearchVectors k-近邻搜索
// topK: 返回结果数
// threshold: 相似度阈值（0~1），低于此值的丢弃
// 返回按相似度降序排列的 (repoID, distance) 列表
func (s *sqliteStore) SearchVectors(ctx context.Context, queryVec []float64, topK int, threshold float64) ([]VectorMatch, error) {
    vecJSON, _ := json.Marshal(queryVec)

    // sqlite-vec KNN 查询
    rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
        SELECT rowid, distance
        FROM repo_vectors
        WHERE embedding MATCH ?
        ORDER BY distance
        LIMIT %d`, topK),
        string(vecJSON))
    if err != nil {
        return nil, fmt.Errorf("vector search: %w", err)
    }
    defer rows.Close()

    var results []VectorMatch
    for rows.Next() {
        var match VectorMatch
        if err := rows.Scan(&match.RepoID, &match.Distance); err != nil {
            return nil, fmt.Errorf("scan vector result: %w", err)
        }
        // 将 distance 转为 similarity: similarity = 1.0 / (1.0 + distance)
        match.Similarity = 1.0 / (1.0 + match.Distance)
        if match.Similarity < threshold {
            continue
        }
        results = append(results, match)
    }
    return results, rows.Err()
}

// DeleteVector 删除指定仓库的向量
func (s *sqliteStore) DeleteVector(ctx context.Context, repoID int64) error {
    _, err := s.db.ExecContext(ctx, `DELETE FROM repo_vectors WHERE rowid = ?`, repoID)
    return err
}

// SetVectorIndexedAt 标记仓库的向量索引时间
func (s *sqliteStore) SetVectorIndexedAt(ctx context.Context, repoID int64, t time.Time) error {
    _, err := s.db.ExecContext(ctx,
        `UPDATE repositories SET vector_indexed_at = ? WHERE id = ?`,
        t.Format(time.RFC3339), repoID)
    return err
}

// ListVectorUnindexed 列出已分析但未向量化的仓库
func (s *sqliteStore) ListVectorUnindexed(ctx context.Context, limit int) ([]*Repository, error) {
    query := repositoryColumns + ` WHERE analyzed_at IS NOT NULL AND analysis_failed = 0 AND vector_indexed_at IS NULL ORDER BY full_name`
    if limit > 0 {
        query += ` LIMIT ?`
    }
    // ... 执行查询并 parse
}

// VectorMatch 向量匹配结果
type VectorMatch struct {
    RepoID     int64
    Distance   float64 // L2/squared distance
    Similarity float64 // 1.0 / (1.0 + distance)
}
```

---

## 六、第一层：向量语义搜索（完整实现）

### 6.1 buildEmbeddingText

参考 GithubStarsManager，构建用于 embedding 的结构化文本。

```go
// file: internal/ai/search_vector.go（新增）

// buildEmbeddingText 拼接仓库文本用于 embedding
func buildEmbeddingText(repo *store.Repository, readmeContent string, maxReadmeChars int) string {
    if maxReadmeChars <= 0 {
        maxReadmeChars = 6000
    }
    var parts []string

    if repo.FullName != "" {
        parts = append(parts, fmt.Sprintf("Repository: %s", repo.FullName))
    }

    // description 去重：若 ai_summary 已包含 description 核心内容，跳过
    desc := repo.Description
    summary := repo.AISummary
    if desc != "" && !strings.Contains(summary, desc) {
        parts = append(parts, fmt.Sprintf("Description: %s", desc))
    }
    if repo.CustomDescription != "" {
        parts = append(parts, fmt.Sprintf("About: %s", repo.CustomDescription))
    }
    if summary != "" {
        parts = append(parts, fmt.Sprintf("Summary: %s", summary))
    }

    // 合并 topics + ai_tags + custom_tags，去重
    tagSet := make(map[string]struct{})
    for _, t := range repo.Topics {
        tagSet[t] = struct{}{}
    }
    for _, t := range repo.AITags {
        tagSet[t] = struct{}{}
    }
    for _, t := range repo.CustomTags {
        tagSet[t] = struct{}{}
    }
    allTags := make([]string, 0, len(tagSet))
    for t := range tagSet {
        allTags = append(allTags, t)
    }
    if len(allTags) > 0 {
        parts = append(parts, fmt.Sprintf("Topics: %s", strings.Join(allTags, ", ")))
    }

    if repo.Language != "" {
        parts = append(parts, fmt.Sprintf("Language: %s", repo.Language))
    }

    // README 清理：移除图片、徽章、HTML 标签、压缩空行，截断
    if readmeContent != "" {
        cleaned := cleanReadme(readmeContent)
        if len(cleaned) > maxReadmeChars {
            cleaned = cleaned[:maxReadmeChars]
        }
        if cleaned != "" {
            parts = append(parts, fmt.Sprintf("README:\n%s", cleaned))
        }
    }

    return strings.Join(parts, "\n")
}

// cleanReadme 清理 README 中的无关内容
func cleanReadme(content string) string {
    // 移除 Markdown 图片: ![alt](url) 或 [![alt](url)](url)
    content = regexp.MustCompile(`!\[.*?\]\(.*?\)`).ReplaceAllString(content, "")
    // 移除 HTML 标签
    content = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(content, " ")
    // 压缩多个连续空行
    content = regexp.MustCompile(`\n{3,}`).ReplaceAllString(content, "\n\n")
    // 移除徽章 URL（通常在 README 头部）
    content = regexp.MustCompile(`\[!\[.*?\]\(.*?\)\]\(.*?\)`).ReplaceAllString(content, "")
    return strings.TrimSpace(content)
}
```

### 6.2 向量搜索主流程

```go
// file: internal/ai/search_vector.go（新增）

func (s *Service) vectorSearch(
    ctx context.Context,
    query string,
    st store.Store,
    opts SearchOpts,
) (*SearchResult, error) {

    // 1. HyDE 查询增强（可选，5 秒超时降级）
    embeddingQuery := query
    if opts.EnableHyDE && s.hasAIConfig() {
        hydeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
        defer cancel()
        if hydeResult, err := s.generateHyDEQuery(hydeCtx, query); err == nil && hydeResult != "" {
            embeddingQuery = hydeResult
        }
        // 失败不阻断，降级用原始 query
    }

    // 2. 生成查询向量
    queryVectors, err := s.embeddingClient.Embed(ctx, []string{embeddingQuery})
    if err != nil {
        return nil, fmt.Errorf("embed query: %w", err)
    }
    if len(queryVectors) == 0 || len(queryVectors[0]) == 0 {
        return nil, fmt.Errorf("empty query vector")
    }

    // 3. sqlite-vec kNN 搜索（topK=30, threshold=0.35）
    matches, err := st.SearchVectors(ctx, queryVectors[0], 30, 0.35)
    if err != nil {
        return nil, fmt.Errorf("vector search: %w", err)
    }
    if len(matches) == 0 {
        return &SearchResult{Hits: []*SearchHit{}}, nil
    }

    // 4. 关键词加分：精确匹配字段给予分数微调
    queryLower := strings.ToLower(query)
    scoreMap := make(map[int64]float64, len(matches))
    for _, m := range matches {
        bonus := 0.0
        repo, err := st.GetRepositoryByID(ctx, m.RepoID)
        if err != nil {
            continue
        }
        name := strings.ToLower(repo.FullName)
        desc := strings.ToLower(repo.Description)
        tags := make([]string, 0, len(repo.AITags)+len(repo.Topics))
        for _, t := range repo.AITags {
            tags = append(tags, strings.ToLower(t))
        }
        for _, t := range repo.Topics {
            tags = append(tags, strings.ToLower(t))
        }
        if strings.Contains(name, queryLower) {
            bonus += 0.05
        }
        if strings.Contains(desc, queryLower) {
            bonus += 0.03
        }
        for _, tag := range tags {
            if strings.Contains(tag, queryLower) {
                bonus += 0.02
                break // 一个标签匹配即加分，仅加一次
            }
        }
        scoreMap[m.RepoID] = m.Similarity + bonus
    }

    // 5. 取匹配仓库按分数排序
    type scoredRepo struct {
        repo  *store.Repository
        score float64
    }
    var scored []scoredRepo
    for repoID, score := range scoreMap {
        repo, err := st.GetRepositoryByID(ctx, repoID)
        if err != nil {
            continue
        }
        scored = append(scored, scoredRepo{repo, score})
    }
    sort.Slice(scored, func(i, j int) bool { return scored[i].score > scored[j].score })

    // 6. LLM 语义重排序（可选，对 top 30 取交集重排序）
    hits := make([]*SearchHit, len(scored))
    for i, sr := range scored {
        hits[i] = &SearchHit{Repo: sr.repo, Score: sr.score}
    }

    if opts.EnableRerank && s.hasAIConfig() && len(hits) > 0 {
        topK := min(opts.RerankTopK, len(hits))
        reranked, err := s.rerankForVector(ctx, query, hits[:topK])
        if err == nil {
            // LLM 重排序结果 + 保留超出 topK 的原始结果
            hits = append(reranked, hits[topK:]...)
        }
        // 失败不阻断，保留向量分数排序
    }

    // 7. 应用 CLI filter 排序
    sortHits(hits, opts.Sort)
    if opts.Limit > 0 && opts.Limit < len(hits) {
        hits = hits[:opts.Limit]
    }

    return &SearchResult{Hits: hits}, nil
}

// rerankForVector 对向量召回结果做 LLM 语义重排序
// 与 FTS5 版本的 rerank 共用底层实现，但 context 文本包含向量相似度信息
func (s *Service) rerankForVector(ctx context.Context, query string, hits []*SearchHit) ([]*SearchHit, error) {
    return s.rerank(ctx, query, hits)
}
```

---

## 七、第二层：LLM 语义搜索

向量搜索未配置或失败时的降级路径，流程不变：

```
HyDE 查询增强（可选）→ LLM 查询理解 → FTS5 全文检索（top 50）→
加权评分（BM25*0.6 + starNorm*0.2 + kwMatch*0.2）→ LLM 语义重排序（top 30）
```

### 7.1 HyDE 查询增强

```go
// file: internal/ai/search_hyde.go（新增文件）

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

### 7.2 AI 语义搜索主逻辑

```go
// file: internal/ai/search_ai.go（新增文件）

func (s *Service) aiSearch(
    ctx context.Context, query string, st store.Store, opts SearchOpts,
) (*SearchResult, error) {

    // 1. HyDE 查询增强（可选，失败降级）
    searchText := query
    if opts.EnableHyDE {
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
    filters := buildSearchFilters(opts, intent)
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
    if opts.EnableRerank {
        topK := min(opts.RerankTopK, len(hits))
        reranked, err := s.rerank(ctx, query, hits[:topK])
        if err == nil {
            hits = reranked
        }
    }

    // 6. 排序 + 截断
    sortHits(hits, opts.Sort)
    if opts.Limit > 0 && opts.Limit < len(hits) {
        hits = hits[:opts.Limit]
    }

    return &SearchResult{Hits: hits}, nil
}
```

### 7.3 LLM 语义重排序增强

改造现有 `rerank` 方法：扩大候选集到 30 条，增加并发批量重排序（每批 10 个）。

```go
// file: internal/ai/search.go（改造 rerank 方法）

// rerank LLM 对候选仓库做语义相关性评分
// 使用并发批量调用加速：每批 10 个候选仓库
func (s *Service) rerank(ctx context.Context, query string, hits []*SearchHit) ([]*SearchHit, error) {
    if len(hits) == 0 {
        return hits, nil
    }
    topK := min(len(hits), 50)
    candidates := hits[:topK]

    batchSize := 10
    allScores := make(map[int]float64, len(candidates))
    var mu sync.Mutex
    var g errgroup.Group

    for i := 0; i < len(candidates); i += batchSize {
        start := i
        end := min(start+batchSize, len(candidates))
        g.Go(func() error {
            batch := candidates[start:end]
            scores, err := s.rerankBatch(ctx, query, batch, start)
            if err != nil {
                return err
            }
            mu.Lock()
            for idx, score := range scores {
                allScores[idx] = score
            }
            mu.Unlock()
            return nil
        })
    }

    if err := g.Wait(); err != nil {
        return nil, err
    }

    // 按 LLM 评分重新排序
    reranked := make([]*SearchHit, len(hits))
    copy(reranked, hits)
    sort.SliceStable(reranked[:len(candidates)], func(i, j int) bool {
        return allScores[start+i] > allScores[start+j]
    })

    return reranked, nil
}

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
输出 JSON：{"rankings":[{"index":%d,"score":8.5}]}`,
            query, string(candJSON), offset)},
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

## 八、第三层：纯文本/FTS5 兜底搜索

```go
// file: internal/ai/search_basic.go（新增文件）

func (s *Service) basicTextSearch(
    ctx context.Context, query string, st store.Store, opts SearchOpts,
) (*SearchResult, error) {
    filters := buildSearchFilters(opts, nil)
    filters.Limit = 50

    ftsResults, err := st.SearchFTS(ctx, query, filters)
    if err != nil {
        return nil, fmt.Errorf("fts search: %w", err)
    }

    hits := make([]*SearchHit, 0, len(ftsResults))
    for _, fr := range ftsResults {
        hits = append(hits, &SearchHit{Repo: fr.Repo, Score: fr.BM25Score})
    }

    sortHits(hits, opts.Sort)
    if opts.Limit > 0 && opts.Limit < len(hits) {
        hits = hits[:opts.Limit]
    }

    return &SearchResult{Hits: hits}, nil
}

// buildSearchFilters 从 SearchOpts 和 QueryIntent 构建 SearchFilters
func buildSearchFilters(opts SearchOpts, intent *QueryIntent) *store.SearchFilters {
    filters := &store.SearchFilters{
        Language:       opts.Language,
        Category:       opts.Category,
        Platform:       opts.Platform,
        Tags:           opts.Tags,
        MinStars:       opts.MinStars,
        MaxStars:       opts.MaxStars,
        Analyzed:       opts.Analyzed,
        AnalysisFailed: opts.AnalysisFailed,
    }
    if intent != nil {
        if filters.Language == "" {
            filters.Language = intent.Language
        }
        if filters.Category == "" {
            filters.Category = intent.Category
        }
        if filters.Platform == "" {
            filters.Platform = intent.Platform
        }
        if filters.MinStars == 0 {
            filters.MinStars = intent.MinStars
        }
        if filters.MaxStars == 0 {
            filters.MaxStars = intent.MaxStars
        }
    }
    return filters
}
```

---

## 九、Analyze 完成自动向量化

### 9.1 流程

```
分析仓库 → 拉取 README → LLM 生成 ai_summary/ai_tags/ai_platforms/ai_search_text
         → store.UpdateAIResult()
         → 检查配置了 embedding API：
             buildEmbeddingText(repo, readme) → EmbeddingClient.Embed() →
             VectorStore.InsertVector() → store.SetVectorIndexedAt()
```

### 9.2 改造 AI 分析调用

```go
// file: internal/ai/analyze.go（改造 AnalyzeRepository 的调用方）

// 在批量分析器 batch.go 中，每个仓库分析成功后：
func (s *Service) AnalyzeAndIndex(ctx context.Context, repo *Repository, readme string) error {
    // 原有 AI 分析
    result, err := s.AnalyzeRepository(ctx, repo, readme, categories)
    if err != nil {
        return err
    }
    if err := s.store.UpdateAIResult(ctx, repo.ID, result); err != nil {
        return err
    }

    // 新增：分析完成后自动向量化
    if s.embeddingClient != nil {
        text := buildEmbeddingText(repo, readme, 6000)
        vectors, err := s.embeddingClient.Embed(ctx, []string{text})
        if err != nil {
            // 向量化失败不阻断分析流程，仅打日志
            log.Printf("WARN: vectorization failed for %s: %v", repo.FullName, err)
            return nil
        }
        if len(vectors) > 0 {
            if err := s.store.InsertVector(ctx, repo.ID, vectors[0]); err != nil {
                log.Printf("WARN: insert vector failed for %s: %v", repo.FullName, err)
                return nil
            }
            if err := s.store.SetVectorIndexedAt(ctx, repo.ID, time.Now()); err != nil {
                log.Printf("WARN: set vector_indexed_at failed for %s: %v", repo.FullName, err)
            }
        }
    }

    return nil
}
```

### 9.3 手动重建向量索引命令

```go
// file: internal/cli/vectorize.go（新增文件）

// starman vectorize [--full]
// --full: 全量重建，清除所有已索引标记
// 默认：增量索引，只处理 vector_indexed_at 为空或内容变更的仓库
```

---

## 十、FTS5 和 Store 改造

### 10.1 FTS5 索引字段增强

```sql
-- file: internal/store/sqlite.go（改造 createFTSIndex）
CREATE VIRTUAL TABLE IF NOT EXISTS repositories_fts USING fts5(
    full_name, description, ai_summary, ai_tags, ai_platforms, ai_search_text, language, topics,
    content='repositories', content_rowid='id',
    tokenize='unicode61 remove_diacritics 2'
)
```

### 10.2 SearchFilters 扩展

```go
// file: internal/store/models.go（改造 SearchFilters）

type SearchFilters struct {
    Language       string   // 编程语言精确匹配
    Category       string   // 分类精确匹配
    Platform       string   // 平台类型（JSON LIKE 匹配）
    Tags           []string // 标签（ai_tags + topics + custom_tags，OR 逻辑）
    MinStars       int
    MaxStars       int
    Limit          int
    Analyzed       *bool // nil=不限
    AnalysisFailed *bool // 与 Analyzed 互斥
}
```

### 10.3 SearchFTS 改造

```go
// file: internal/store/repository.go（改造 SearchFTS，增加 filter 条件）

func (s *sqliteStore) SearchFTS(ctx context.Context, query string, filters *SearchFilters) ([]*FTSResult, error) {
    where := "repositories_fts MATCH ?"
    args := []interface{}{query}

    if filters != nil {
        // Platform: JSON LIKE 匹配
        if filters.Platform != "" {
            where += " AND r.ai_platforms LIKE ?"
            args = append(args, "%"+filters.Platform+"%")
        }
        // Tags: 三类标签 OR 匹配
        if len(filters.Tags) > 0 {
            parts := make([]string, 0, len(filters.Tags))
            for _, tag := range filters.Tags {
                parts = append(parts, `(r.ai_tags LIKE ? OR r.topics LIKE ? OR r.custom_tags LIKE ?)`)
                args = append(args, "%"+tag+"%", "%"+tag+"%", "%"+tag+"%")
            }
            where += " AND (" + strings.Join(parts, " OR ") + ")"
        }
        // Analyzed / AnalysisFailed 互斥
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
            }
        }
        // ... 其余条件（Language/Category/MinStars/MaxStars）保持现有逻辑
    }

    // ... 原有 FTS5 JOIN 查询逻辑
}
```

---

## 十一、CLI 命令改造

### 11.1 新增 flag

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
```

### 11.2 新增 vectorize 命令

```go
// file: internal/cli/vectorize.go（新增文件）
// starman vectorize [--full]
// 手动触发全量或增量向量索引重建
```

### 11.3 输出增强

```go
// 结果输出时标注搜索模式
fmt.Fprintf(os.Stderr, "Search mode: %s (%d results)\n", result.Mode, len(result.Hits))
```

---

## 十二、文件变更清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/ai/search.go` | 改造 | 新增 `Search` 三层降级入口，改造 `rerank` 为批量并发 |
| `internal/ai/search_vector.go` | 新增（完整实现） | 向量搜索主流程 + `buildEmbeddingText` + `cleanReadme` |
| `internal/ai/search_ai.go` | 新增 | 第二层 LLM 语义搜索 |
| `internal/ai/search_basic.go` | 新增 | 第三层 FTS5 兜底搜索 |
| `internal/ai/search_hyde.go` | 新增 | HyDE 查询增强 |
| `internal/ai/embedding.go` | 新增（完整实现） | OpenAI 协议 Embedding 客户端 |
| `internal/ai/analyze.go` | 改造 | 分析完成后自动触发向量化 |
| `internal/ai/batch.go` | 改造 | 批量分析时传入 embeddingClient |
| `internal/store/vector.go` | 新增（完整实现） | sqlite-vec VectorStore 实现 |
| `internal/store/sqlite.go` | 改造 | vec0 向量虚拟表建表、vector_indexed_at 列迁移、FTS5 增加 ai_platforms（驱动不变，sqlite-vec 已内置） |
| `internal/store/models.go` | 改造 | Repository 增加 VectorIndexedAt；SearchFilters 扩展 |
| `internal/store/repository.go` | 改造 | 新增 GetRepositoryByID/SetVectorIndexedAt/ListVectorUnindexed；SearchFTS 增强过滤 |
| `internal/store/store.go` | 改造 | Store 接口扩展（InsertVector/SearchVectors/DeleteVector/SetVectorIndexedAt） |
| `internal/config/config.go` | 改造 | 新增 EmbeddingConfig |
| `internal/cli/search.go` | 改造 | 新增 flag，调用三层降级搜索，输出增强 |
| `internal/cli/vectorize.go` | 新增 | `starman vectorize` 命令 |
| `go.mod` | 无需修改 | sqlite-vec 已内置在 modernc.org/sqlite v1.53.0 中 |

---

## 十三、测试策略

1. **向量搜索完整链路测试**：
   - 配置 embedding API → 分析仓库 → 验证向量写入 → 搜索返回结果
   - HyDE 增强效果对比（with/without HyDE）
   - 关键词加分验证（名称/描述/标签匹配分别加分）

2. **三层降级测试**：
   - 有 embedding + AI → 走向量搜索
   - 有 AI 无 embedding → 走 LLM 语义搜索
   - 无任何 API → 走 FTS5 兜底
   - 向量搜索失败 → 自动降级到第二层
   - AI 搜索失败 → 自动降级到第三层

3. **sqlite-vec 测试**：
   - vec0 虚拟表正常工作
   - InsertVector / SearchVectors 正确性
   - 阈值过滤效果验证
   - 多向量并发写入安全性

4. **Analyze 自动向量化测试**：
   - 分析后自动写入向量
   - vector_indexed_at 时间戳正确
   - 向量化失败不影响分析结果
   - 批量分析并发安全性
