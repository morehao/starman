# Starman 拉取同步方案

## 概述

本文档描述 starman（Go CLI）的拉取同步优化完整技术方案。

starman 当前已有良好的基础：并发分页拉取（90 并发）、Upsert 增量写入、本地字段保护、`--full` 全量清理、`--watch` 定时同步。以下优化在此基础上做精细化增强。

---

## 一、Rate Limit 预判等待

### 现状问题

首请求后检查 `resp.Rate.Remaining < 10` 则直接报错退出，后续并发页面不做 rate limit 检查，仅在返回 403 时被动等待。90 并发高负载时大概率触发 secondary rate limit。

### 优化方案

采用持续追踪 + 主动预判策略，在并发拉取场景下用 `atomic.Int64` 维护全局 rate limit 计数器。

```go
// file: internal/github/client.go（改造 ListStarred）

import "sync/atomic"

func (c *Client) ListStarred(ctx context.Context, username string) ([]*Repository, error) {
    opts := &gh.ActivityListStarredOptions{
        ListOptions: gh.ListOptions{PerPage: perPage},
    }
    firstPage, resp, err := c.client.Activity.ListStarred(ctx, username, opts)
    if err != nil {
        return nil, fmt.Errorf("list starred first page: %w", err)
    }

    // 提取首请求的 rate limit 信息，初始化全局计数器
    var rateRemaining atomic.Int64
    rateRemaining.Store(int64(resp.Rate.Remaining))
    rateReset := resp.Rate.Reset.Time
    rateThreshold := int64(20) // 低于此阈值主动等待

    // 首请求后主动检查
    if rateRemaining.Load() < rateThreshold {
        if err := waitForRateReset(ctx, rateReset); err != nil {
            return nil, err
        }
        rateRemaining.Store(5000) // reset 后恢复
    }
    rateRemaining.Add(-1) // 扣减首请求

    totalPages := resp.LastPage
    if totalPages < 1 {
        totalPages = 1
    }

    pages := make([][]*gh.StarredRepository, totalPages)
    pages[0] = firstPage

    if totalPages > 1 {
        eg, ctx := errgroup.WithContext(ctx)
        eg.SetLimit(concurrentMax)
        for page := 2; page <= totalPages; page++ {
            page := page
            eg.Go(func() error {
                // 请求前检查：剩余不足预判等待
                if err := checkAndWaitRateLimit(ctx, &rateRemaining, rateReset, rateThreshold); err != nil {
                    return err
                }
                rateRemaining.Add(-1) // 扣减即将发送的请求

                repos, err := c.fetchStarredPage(ctx, username, page)
                if err != nil {
                    return err
                }
                pages[page-1] = repos
                return nil
            })
        }
        if err := eg.Wait(); err != nil {
            return nil, err
        }
    }

    // ... 合并逻辑不变
}

// checkAndWaitRateLimit 检查 rate limit，不足时主动等待
func checkAndWaitRateLimit(ctx context.Context, remaining *atomic.Int64, resetTime time.Time, threshold int64) error {
    if remaining.Load() < threshold {
        return waitForRateReset(ctx, resetTime)
    }
    return nil
}

// waitForRateReset 等待到 rate limit reset 时间，确保至少等 1 秒
func waitForRateReset(ctx context.Context, resetTime time.Time) error {
    waitDur := time.Until(resetTime) + time.Second
    if waitDur <= 0 {
        return nil
    }
    log.Printf("[starman] Rate limit low, waiting %v until reset...", waitDur.Round(time.Second))
    return sleep(ctx, waitDur)
}
```

**要点**：
- `atomic.Int64` 保证并发场景下计数准确
- 阈值设为 20（适配 CLI 无 UI 阻塞问题）
- 等待期间响应 `ctx.Done()`，支持 Ctrl+C 取消
- 首请求的 rate limit 信息直接作为全局基准，后续不再读响应头（减少复杂性）

---

## 二、指数退避重试

### 现状问题

仅在 rate limit 403 时等待重试，无网络错误或 5xx 的重试机制。

### 优化方案

提取通用重试函数，应用到所有 GitHub API 调用。

```go
// file: internal/github/client.go（新增）

// retryWithBackoff 执行带指数退避的重试操作
// maxRetries: 最大重试次数（不含首次调用）
// baseDelay: 初始等待时间
func retryWithBackoff(ctx context.Context, maxRetries int, baseDelay time.Duration, fn func() error) error {
    for i := 0; i <= maxRetries; i++ {
        err := fn()
        if err == nil {
            return nil
        }

        // 最后一次尝试失败，直接返回
        if i == maxRetries {
            return fmt.Errorf("retry exhausted (%d attempts): %w", maxRetries+1, err)
        }

        // 判断是否可重试
        if !isRetryable(err) {
            return err
        }

        delay := baseDelay * time.Duration(1<<i) // 1s, 2s, 4s
        log.Printf("[starman] Request failed, retrying in %v (attempt %d/%d): %v",
            delay, i+1, maxRetries, err)
        if sleepErr := sleep(ctx, delay); sleepErr != nil {
            return sleepErr
        }
    }
    return nil
}

// isRetryable 判断错误是否可重试
func isRetryable(err error) bool {
    var ghErr *gh.ErrorResponse
    if errors.As(err, &ghErr) {
        code := ghErr.Response.StatusCode
        // 5xx 服务端错误
        if code >= 500 {
            return true
        }
        // 403 secondary rate limit（非认证错误）
        if code == 403 {
            return true
        }
        // 4xx 客户端错误不可重试（401/404/422 等）
        return false
    }
    // 网络错误（DNS、连接拒绝、超时）可重试
    return true
}

// sleep 带 context 取消的 sleep
func sleep(ctx context.Context, d time.Duration) error {
    if d <= 0 {
        return nil
    }
    select {
    case <-ctx.Done():
        return ctx.Err()
    case <-time.After(d):
        return nil
    }
}
```

**应用范围**：

```go
// fetchStarredPage 改造
func (c *Client) fetchStarredPage(ctx context.Context, username string, page int) ([]*gh.StarredRepository, error) {
    opts := &gh.ActivityListStarredOptions{
        ListOptions: gh.ListOptions{PerPage: perPage, Page: page},
    }

    var repos []*gh.StarredRepository
    err := retryWithBackoff(ctx, 3, time.Second, func() error {
        result, resp, err := c.client.Activity.ListStarred(ctx, username, opts)
        if err != nil {
            // rate limit 特殊处理：更新全局计数器等待后重试
            if resp != nil && resp.Rate.Remaining == 0 {
                return fmt.Errorf("rate limit exhausted: %w", err)
            }
            return err
        }
        repos = result
        return nil
    })
    return repos, err
}
```

**参数**：`maxRetries = 3`，`baseDelay = 1s`（退避：1s → 2s → 4s）

---

## 三、Release 增量水位

### 现状问题

`pullOne()` 每次同步都**全量拉取**仓库所有 release 分页，再本地用 `LastReleaseFetch` 过滤。对于有 100+ release 的仓库（如 `kubernetes/kubernetes`），每次同步浪费大量 API 配额。

### 优化方案

采用时间水位增量策略，用 `LastReleaseFetch` 做拉取终止条件。

```go
// file: internal/github/operations.go（新方法）

// ListReleasesIncremental 增量拉取 release，直到命中时间水位
// watermark: 上次同步时间，比此时间更早的 release 不再拉取
func (c *Client) ListReleasesIncremental(ctx context.Context, owner, repo string, watermark *time.Time) ([]*store.Release, error) {
    opts := &gh.ListOptions{PerPage: perPage}
    var allReleases []*store.Release

    for {
        ghReleases, resp, err := c.client.Repositories.ListReleases(ctx, owner, repo, &gh.ListOptions{
            Page:    opts.Page,
            PerPage: opts.PerPage,
        })
        if err != nil {
            return nil, fmt.Errorf("list releases page %d: %w", opts.Page, err)
        }

        for _, ghRel := range ghReleases {
            rel := convertRelease(ghRel)
            // 水位检查：如果这条 release 比上次同步时间更早，说明后续都是老数据
            if watermark != nil && !rel.PublishedAt.After(*watermark) {
                return allReleases, nil // 命中水位，停止拉取
            }
            allReleases = append(allReleases, rel)
        }

        if resp.NextPage == 0 {
            break
        }
        opts.Page = resp.NextPage
    }

    return allReleases, nil
}
```

```go
// file: internal/release/tracker.go（改造 pullOne）

func (t *Tracker) pullOne(ctx context.Context, repo *store.Repository) (int, error) {
    parts := splitName(repo.FullName)
    if len(parts) != 2 {
        return 0, fmt.Errorf("invalid full name: %s", repo.FullName)
    }

    // 首次同步：全量拉取；增量同步：用水位停止
    var releases []*store.Release
    var err error
    if repo.LastReleaseFetch == nil {
        releases, err = t.gh.ListReleases(ctx, parts[0], parts[1]) // 全量
    } else {
        releases, err = t.gh.ListReleasesIncremental(ctx, parts[0], parts[1], repo.LastReleaseFetch)
    }
    if err != nil {
        return 0, err
    }

    newCount := 0
    for _, rel := range releases {
        rel.RepoID = repo.ID
        if err := t.store.UpsertRelease(ctx, rel); err != nil {
            return newCount, err
        }
        newCount++
    }

    if len(releases) > 0 {
        latest := releases[0].PublishedAt
        t.store.UpdateReleaseWatermark(ctx, repo.ID, latest)
    }

    return newCount, nil
}
```

**效果**：
- 活跃仓库（每周发版）：仍需拉取 1-2 页
- 稳定仓库（很少发版）：第 1 页即命中水位，节省 90%+ API 调用
- 首次同步行为不变

---

## 四、显式本地字段清单 + 双层数据合并

### 现状问题

`UpsertReposOnSync` 通过在 UPDATE 语句中**不写某些列**来隐式保护本地字段，字段列表散落在 SQL 中，新增字段时容易遗漏。

### 优化方案

定义显式本地字段清单，合并逻辑集中在 `repositoryMerge` 文件。

```go
// file: internal/store/repository_merge.go（新增文件）

// LocalRepoFields 同步时受保护的本地元数据字段
// 这些字段由 AI 分析或用户编辑生成，同步 GitHub 数据时不应覆盖
var LocalRepoFields = []string{
    "ai_summary",
    "ai_tags",
    "ai_platforms",
    "ai_search_text",
    "ai_category",
    "analyzed_at",
    "analysis_failed",
    "custom_description",
    "custom_tags",
    "custom_category",
    "category_locked",
    "last_released",
    "subscribed_releases",
    "last_release_fetch",
}

// GitHubSourceFields 同步时从 GitHub 覆盖的基础字段
var GitHubSourceFields = []string{
    "name",
    "description",
    "url",
    "language",
    "homepage",
    "stargazers_count",
    "forks_count",
    "topics",
    "owner_login",
    "owner_avatar",
    "starred_at",
}

// MergeReposOnSync 将 GitHub 拉取数据与本地数据合并
// 返回合并后的仓库列表
func MergeReposOnSync(incoming []*Repository, existing map[string]*Repository) []*Repository {
    merged := make([]*Repository, 0, len(incoming))

    for _, newRepo := range incoming {
        exist, ok := existing[newRepo.FullName]
        if !ok {
            // 新仓库：直接使用 GitHub 数据
            merged = append(merged, newRepo)
            continue
        }

        // 已有仓库：以 existing（本地数据）为底，覆盖 GitHub 源字段
        result := &Repository{}
        *result = *exist // 浅拷贝，保留所有本地字段

        // 逐字段覆盖 GitHub 源字段
        result.Name = newRepo.Name
        result.Description = newRepo.Description
        result.URL = newRepo.URL
        result.Language = newRepo.Language
        result.Homepage = newRepo.Homepage
        result.StargazersCount = newRepo.StargazersCount
        result.ForksCount = newRepo.ForksCount
        result.Topics = newRepo.Topics
        result.OwnerLogin = newRepo.OwnerLogin
        result.OwnerAvatar = newRepo.OwnerAvatar
        result.StarredAt = newRepo.StarredAt

        // 注意：不覆盖 ai_summary、ai_tags、custom_tags 等本地字段

        merged = append(merged, result)
    }

    return merged
}
```

**UpsertReposOnSync 改造为使用合并逻辑**：

```go
// file: internal/store/repository.go（改造 UpsertReposOnSync）

func (s *sqliteStore) UpsertReposOnSync(ctx context.Context, rs []*Repository, fullSync bool) error {
    // 1. 查询已有仓库，构建 Map
    existing, err := s.listRepoNames(ctx)
    if err != nil {
        return fmt.Errorf("list existing repos: %w", err)
    }

    // 2. 双层合并
    merged := MergeReposOnSync(rs, existing)

    // 3. 批量 upsert 合并后的仓库
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("begin tx: %w", err)
    }
    defer tx.Rollback()

    for _, r := range merged {
        if err := upsertRepoTx(ctx, tx, r); err != nil {
            return err
        }
    }

    // 4. 全量同步：删除不再 star 的仓库
    if fullSync {
        incomingNames := make(map[string]bool, len(rs))
        for _, r := range rs {
            incomingNames[r.FullName] = true
        }
        rows, err := tx.QueryContext(ctx, `SELECT full_name FROM repositories`)
        if err != nil {
            return fmt.Errorf("query all repos for full sync: %w", err)
        }
        var toDelete []string
        for rows.Next() {
            var fn string
            if err := rows.Scan(&fn); err != nil {
                rows.Close()
                return err
            }
            if !incomingNames[fn] {
                toDelete = append(toDelete, fn)
            }
        }
        rows.Close()
        for _, fn := range toDelete {
            if _, err := tx.ExecContext(ctx, `DELETE FROM repositories WHERE full_name = ?`, fn); err != nil {
                return fmt.Errorf("delete repo %s: %w", fn, err)
            }
        }
    }

    return tx.Commit()
}
```

---

## 五、同步状态 KV 扩展

### 现状

`sync_state` 表仅存储 `last_sync` 时间戳。

### 优化方案

扩展为结构化监控指标，便于 troubleshooting 和体验展示。

```go
// file: internal/store/sync_state.go（扩展）

type SyncStats struct {
    TotalSyncCount int       `json:"total_sync_count"`
    LastSync       time.Time `json:"last_sync"`
    LastDuration   string    `json:"last_duration"`   // 如 "4.2s"
    LastRepoCount  int       `json:"last_repo_count"` // 同步的仓库数
    LastNewCount   int       `json:"last_new_count"`  // 新增仓库数
    LastErrorCount int       `json:"last_error_count"`
}

func (s *sqliteStore) SaveSyncStats(ctx context.Context, stats *SyncStats) error {
    data, _ := json.Marshal(stats)
    _, err := s.db.ExecContext(ctx,
        `INSERT INTO sync_state (key, value) VALUES ('sync_stats', ?)
         ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
        string(data))
    return err
}

func (s *sqliteStore) GetSyncStats(ctx context.Context) (*SyncStats, error) {
    var value sql.NullString
    err := s.db.QueryRowContext(ctx,
        `SELECT value FROM sync_state WHERE key = 'sync_stats'`).Scan(&value)
    if err == sql.ErrNoRows {
        return &SyncStats{}, nil
    }
    if err != nil {
        return nil, err
    }
    var stats SyncStats
    if err := json.Unmarshal([]byte(value.String), &stats); err != nil {
        return &SyncStats{}, nil
    }
    return &stats, nil
}
```

**CLI 输出增强**：

```
$ starman sync
Fetching starred repos...
  Rate limit: OK (4987 remaining)
  847 repos fetched (3 new) in 4.2s
  Sync complete.

$ starman stats
Last sync:    2026-07-02 10:30:00
Duration:     4.2s
Repos synced: 847 (3 new)
Total syncs:  42
Avg duration: 3.8s
```

---

## 六、搜索缓存复用（与检索方案联动）

### 设计

与三层降级检索方案联动——检索文档中描述的 `search_cache` 机制在同步完成后需要失效处理。

```go
// file: internal/store/repository.go（在 UpsertReposOnSync 末尾）

// 同步完成后清空搜索缓存（数据变更，缓存失效）
if err := tx.ExecContext(ctx, `DELETE FROM sync_state WHERE key LIKE 'search_cache:%'`); err != nil {
    return fmt.Errorf("clear search cache: %w", err)
}
```

这样当用户同步后，之前缓存的搜索结果自动失效，下次搜索会重新查询最新数据。

---

## 七、全链路 Context 取消

### 优化

所有等待点（rate limit 等待、重试 sleep、watch 间隔）全部改用带 context 的 sleep。

```go
// file: internal/github/client.go

// sleep 带 context 取消的 sleep，用于所有等待点
func sleep(ctx context.Context, d time.Duration) error {
    if d <= 0 {
        return nil
    }
    select {
    case <-ctx.Done():
        return ctx.Err()
    case <-time.After(d):
        return nil
    }
}
```

**应用点**：
- `waitForRateReset` → 内部用 `sleep`
- `retryWithBackoff` → 内部用 `sleep`
- watch 模式间歇 → 改用 `sleep(ctx, interval)`

---

## 八、并发控制优化

### 现状问题

`concurrentMax = 90` 并发过高，容易触发 GitHub 的 secondary rate limit（官方建议约 30 并发）。

### 优化

```go
// file: internal/github/client.go

const (
    perPage          = 100
    concurrentMax    = 20 // 降低并发（原 90，GitHub 官方建议不超 30）
    rateSafetyMargin = 20 // 主动等待阈值
)
```

配合 rate limit 预判等待，即使并发降为 20，整体速度不会明显下降（因为不再被 rate limit 拦截）。

---

## 九、文件变更清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/github/client.go` | 改造 | `ListStarred` 增加 rate limit 预判；提取 `retryWithBackoff`/`sleep` 通用工具函数；降低并发上限 |
| `internal/github/operations.go` | 改造 | 新增 `ListReleasesIncremental` 增量拉取方法 |
| `internal/store/repository_merge.go` | 新增 | 显式本地字段清单 + `MergeReposOnSync` 合并逻辑 |
| `internal/store/repository.go` | 改造 | `UpsertReposOnSync` 使用合并逻辑；清除搜索缓存 |
| `internal/store/sync_state.go` | 改造 | 扩展 `SyncStats` 结构和存取方法 |
| `internal/release/tracker.go` | 改造 | `pullOne` 使用 `ListReleasesIncremental` |
| `internal/cli/sync.go` | 改造 | 显示同步进度和统计信息 |

---

## 十、测试策略

1. **Rate limit 预判测试**：
   - Mock rate limit 响应，验证阈值触发等待和 counter 递减
   - 并发场景下 `atomic.Int64` 原子性验证

2. **指数退避测试**：
   - Mock 网络错误 → 验证 3 次重试
   - Mock 5xx 错误 → 验证退避间隔（1s/2s/4s）
   - Mock 4xx 错误 → 验证不重试直接返回
   - Context 取消 → 验证重试立即终止

3. **Release 增量水位测试**：
   - 首次同步走全量路径
   - 已同步仓库走增量路径，验证命中水位时提前终止
   - 验证水位命中后不再读取后续分页

4. **合并逻辑测试**：
   - 验证 AI 分析字段不被覆盖
   - 验证自定义编辑字段不被覆盖
   - 验证 GitHub 源字段正确更新
   - 验证全量同步时移除未 star 仓库
