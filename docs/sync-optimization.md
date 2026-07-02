# Starman 拉取同步优化思路

## 背景

Starman 当前同步方案已经有较好的基础：并发分页拉取（90 并发）、Upsert 增量写入、本地字段保护、`--full` 全量清理、`--watch` 定时同步。

GithubStarsManager 在此基础上做得更精细：rate limit 预判等待、指数退避重试、Release 时间水位增量、显式本地字段清单、网络错误分类处理、全链路取消传递。

本文从 CLI 工具场景出发，提出增量优化方案，按优先级排列。

---

## 优化项

### 1. Rate Limit 预判等待（高优先级）

**现状**：首请求后检查 `resp.Rate.Remaining` 是否低于 10，低于则直接返回错误。后续并发页面不做 rate limit 检查，仅在撞墙（403）时被动等待。对于 1000+ star 的用户，高并发（90）大概率触发 secondary rate limit。

**优化**：参考 GithubStarsManager 的持续追踪 + 主动预判策略。

```
ListStarred 改造：

1. 首请求 → 解析 X-RateLimit-Remaining / X-RateLimit-Reset
2. 维护全局 rateLimitRemaining 计数器（atomic.Int64）
3. 每次发送请求前：
   if rateLimitRemaining < threshold(20) {
       waitDuration = time.Until(resetTime) + 1s
       log "rate limit low (%d remaining), waiting %v..."
       time.Sleep(waitDuration)
   }
4. 每次响应后：atomic.Add(&rateLimitRemaining, -1)
5. 响应头重新同步：若 remaining 差异过大（>10），用真实值覆盖
```

**要点**：
- 并发场景用 `atomic.Int64` 保证计数安全
- 阈值设为 20（比 GithubStarsManager 的 100 宽松，适配 CLI 无 UI 阻塞问题）
- 等待期间响应 `ctx.Done()`，支持 Ctrl+C 取消

### 2. 指数退避重试（高优先级）

**现状**：仅在 rate limit 403 时等待重试，无网络错误或 5xx 的重试机制。并发拉取中一个页面的网络波动可能导致整个同步失败。

**优化**：提取通用 `retryWithBackoff` 函数。

```go
func retryWithBackoff(ctx context.Context, maxRetries int, baseDelay time.Duration, fn func() error) error {
    for i := 0; i < maxRetries; i++ {
        err := fn()
        if err == nil {
            return nil
        }
        if i == maxRetries-1 {
            return fmt.Errorf("retry exhausted (%d attempts): %w", maxRetries, err)
        }
        if !isRetryable(err) {
            return err
        }
        delay := baseDelay * time.Duration(1<<i) // 指数退避: 1s, 2s, 4s
        sleep(ctx, delay)
    }
    return nil
}

func isRetryable(err error) bool {
    var ghErr *github.ErrorResponse
    if errors.As(err, &ghErr) {
        if ghErr.Response.StatusCode >= 500 {
            return true
        }
        // 403 secondary rate limit 也重试
        if ghErr.Response.StatusCode == 403 && isSecondaryRateLimit(ghErr) {
            return true
        }
        return false
    }
    // 网络错误（DNS、连接拒绝、超时）也重试
    return true
}
```

**应用范围**：
- 每个分页拉取 `fetchStarredPage` 内的请求
- `ListStarred` 首请求
- Release 拉取请求
- AI 分析请求（已有超时处理，加网络重试更健壮）

**参数**：
- `maxRetries = 3`
- `baseDelay = 1s`（退避：1s → 2s → 4s）

### 3. Release 增量水位优化（高优先级）

**现状**：`pullOne()` 每次同步都**全量拉取**该仓库所有 release 分页，再本地用 `LastReleaseFetch` 时间戳过滤。对于有 100+ release 的仓库（如 `kubernetes/kubernetes`），每次同步消耗大量 API 配额但实际只有 0-2 个新 release。

**优化**：参考 GithubStarsManager 的时间水位增量策略。

```
pullOne 改造：

if repo.LastReleaseFetch.IsZero() || repo.LastReleaseFetch == nil {
    // 首次同步：全量拉取所有 release 分页
    fetchAllReleases(ctx, repo)
} else {
    // 增量同步：逐页拉取，直到命中时间水位
    for page := 1; ; page++ {
        releases, resp := fetchReleasePage(ctx, repo, page)
        for _, r := range releases {
            if r.PublishedAt.Before(repo.LastReleaseFetch) {
                // 命中水位线，后续页面全部是老数据，停止
                return
            }
            saveNewRelease(r)
        }
        if page >= resp.LastPage {
            break
        }
    }
}
```

**效果**：
- 活跃仓库（每周发版）：仍需拉取 1-2 页
- 稳定仓库（很少发版）：只需拉取第 1 页即命中水位，节省 90%+ API 调用
- 首次同步行为不变

### 4. 显式本地字段清单（高优先级）

**现状**：`UpsertReposOnSync` 通过**不写某些列**来隐式保护本地字段，字段列表散落在 INSERT 和 UPDATE SQL 语句中，新增字段时容易遗漏。

**优化**：定义本地字段常量，合并逻辑集中管理。

```go
// 本地元数据字段清单（同步时保护，不被 GitHub 数据覆盖）
var localRepoFields = []string{
    "ai_summary",
    "ai_tags",
    "ai_platforms",
    "ai_search_text",
    "analyzed_at",
    "analysis_failed",
    "custom_description",
    "custom_tags",
    "custom_category",
    "category_locked",
    "last_edited",
    "vector_indexed_at",    // 向量化方案引入时
}

// UpsertReposOnSync 改造：
// 1. 以 GitHub 数据为基础构建新对象
// 2. 若仓库已存在，逐字段覆盖回 localRepoFields 中的值
// 3. 使用 ON CONFLICT(owner, repo) DO UPDATE 替代先查后插
```

**收益**：
- 新增本地字段只需添加到数组，不会因遗漏导致数据被覆盖
- 代码意图明确，不再依赖"忘记写某列"来保护数据
- 减少一次 SELECT 查询前的存在性检查

### 5. 双层数据合并（中优先级）

**现状**：同步时对已存在仓库做 UPDATE，只更新 GitHub 源字段（name、description、url、language、homepage、stargazers_count、forks_count、topics、owner_login、owner_avatar、starred_at）。

**优化**：参考 GithubStarsManager 的双层合并策略。

```
第 1 层（mergeRepositoriesPreservingLocalMetadata）：
  incomingData = {从 GitHub 拉取的字段}
  for each field in localRepoFields:
      若仓库已存在：
          incomingData[field] = existingData[field]
  → 保护本地编辑字段

第 2 层（handleStarSync 式逐字段赋值）：
  existing.name = incoming.name          // 更新 GitHub 源字段
  existing.starred_at = incoming.starred_at
  // 不动 ai_summary、custom_description 等本地字段
```

这种双层设计的好处是：第一层保证所有本地字段不被覆盖，第二层做精细化的字段级更新，即使 `localRepoFields` 定义有遗漏，第二层也能兜底。

### 6. 同步状态 KV 扩展（中优先级）

**现状**：`sync_state` 表仅存储 `last_sync` 时间戳。

**优化**：扩展 stats 记录，便于监控同步健康度。

```sql
sync_state 扩展字段：
├── last_sync         TEXT     -- 最后一次同步完成时间
├── last_sync_duration TEXT    -- 耗时（如 "3.2s"）
├── last_sync_repos  INTEGER  -- 同步的仓库数
├── last_sync_new    INTEGER  -- 新增仓库数
├── last_sync_errors INTEGER  -- 错误数
└── total_sync_count INTEGER  -- 历史同步总次数
```

CLI 展示优化：

```
$ starman sync
Syncing...
Fetched 847 repos (3 new) in 4.2s
$ starman stats
Last sync: 2026-07-02 10:30:00 (847 repos, 4.2s)
Total syncs: 42, avg duration: 3.8s
```

### 7. HTTP 缓存增强（中优先级）

**现状**：已使用 `httpcache` 做 HTTP 层内存缓存，但对 star 列表拉取场景作用有限（列表分页的 URL 每次相同，但 ETag 能命中更多缓存）。

**优化**：
- 启用 `httpcache` 的 ETag/If-None-Match 支持
- 列表首次拉取后缓存 ETag，第二次 `starman sync` 时若 304 Not Modified 则跳过重复拉取
- 缓存 TTL 设为 5 分钟（超过则即使有 ETag 也重新拉取）

```go
client := httpcache.NewMemoryClient()
// 优先读本地缓存，命中则跳过请求
```

**注意**：GitHub Star API 在 star 列表变化时 ETag 会改变，所以对大多数场景（star 列表变化不频繁）有效。

### 8. 全链路 Context 取消（低优先级）

**现状**：部分关键路径（rate limit 等待、重试 sleep）不响应 `ctx.Done()`，用户按 Ctrl+C 后可能需要等待较长时间才能终止。

**优化**：

```go
func sleep(ctx context.Context, d time.Duration) error {
    select {
    case <-ctx.Done():
        return ctx.Err()
    case <-time.After(d):
        return nil
    }
}
```

应用到所有等待点：rate limit 等待、指数退避 sleep、watch 模式间隔。

---

## 优先级总结

| 优先级 | 优化项 | 预期收益 | 侵入性 |
|--------|--------|----------|--------|
| **高** | Rate limit 预判等待 | 减少 403 错误，提升大仓库量同步稳定性 | 低 |
| **高** | 指数退避重试 | 网络波动容错，减少同步失败率 | 低 |
| **高** | Release 增量水位 | 大幅减少 release 同步 API 消耗 | 中 |
| **高** | 显式本地字段清单 | 提升代码可维护性，防止数据丢失 | 低 |
| **中** | 双层数据合并 | 数据保护更进一步 | 低 |
| **中** | 同步状态 KV 扩展 | 监控同步健康度 | 低 |
| **中** | HTTP 缓存增强 | 减少重复带宽消耗 | 低 |
| **低** | 全链路 Context 取消 | 提升用户体验 | 极低 |

建议按优先级从上到下逐步实施，每个优化项独立可测，互不阻塞。
