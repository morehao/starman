# Starman 检索优化思路

## 背景

Starman 当前检索方案为单一路径：**LLM 查询理解 → SQLite FTS5 全文检索（BM25）→ 加权评分 → 可选 LLM 精排**。核心设计是用 LLM 将自然语言转化为 FTS5 查询表达式，再用全文索引做关键词匹配。

GithubStarsManager 采用三层检索降级：**向量语义搜索 → LLM 语义搜索 → 纯文本搜索**，在召回率、排序质量和用户体验上表现更好。

本文从 starman 的 CLI 场景出发，提出两条优化路径。

---

## 方案 A：不引入向量化，增强 FTS5 方案

适用场景：不想增加新依赖，希望通过工程优化提升现有 FTS5 检索的召回和排序质量。

### 1. HyDE 查询增强

**现状**：`understandQuery()` 将自然语言直接转为 FTS5 查询词，如 `"好看的终端工具"` → `ftsQuery: "terminal OR cli"`，转换过程丢失了语义信息。

**优化**：搜索前先调用 LLM 生成"理想仓库描述"（HyDE：Hypothetical Document Embedding），然后用该描述做 FTS5 全文匹配。

```
用户输入: "好看的终端工具"
     ↓
LLM 生成 HyDE: "一个基于 Rust 的现代化终端模拟器，支持 GPU 加速渲染、
           分屏、标签页管理、主题定制，提供流畅的渲染性能和丰富的插件生态"
     ↓
提取关键词 → FTS5 MATCH: "terminal OR emulator OR cli OR tui"
```

**关键设计**：
- HyDE 生成独立于 `understandQuery`，作为一个可选预处理步骤
- 用 `--hyde` flag 控制开关，默认开启
- LLM 调用 5 秒超时降级：失败则退回原始 query
- HyDE 结果可缓存（同一 query 不重复生成）

### 2. LLM 语义重排序增强

**现状**：`--rerank` 可选，取 FTS5 前 15 条调 LLM 打分 0-10。

**优化**：
- **扩大候选集**：FTS5 取 top 30 而非 15 条送重排序，提升召回
- **默认可开**：`--rerank` 改为默认开启，`--no-rerank` 关闭
- **并行重排序**：30 条候选按 5 条一批并发调 LLM，减少延迟
- **分值融合**：`final = bm25 * 0.3 + rerankScore * 0.5 + starNorm * 0.2`（增加 LLM 精排权重）

### 3. 多维度过滤增强

**现状**：仅支持 `--lang`、`--category`、`--stars` 过滤。

**优化**：参考 GithubStarsManager 的过滤体系，增加：

| 过滤维度 | flag | 说明 |
|----------|------|------|
| 平台过滤 | `--platform` | 按 AI 分析的 `ai_platforms` 过滤（CLI/GUI/Web/Library 等） |
| 状态过滤 | `--analyzed` / `--no-analyzed` | 只显示已分析/未分析的仓库 |
| 标签过滤 | `--tag` | 按 AI 标签 + 自定义标签过滤，支持多值 |
| 排序方式 | `--sort` | 扩展 `relevance/stars/name/updated` 排序选项 |

### 4. 搜索历史与缓存

**优化**：
- 搜索历史持久化到 SQLite（`search_history` 表），支持 `starman search --history`
- FTS5 查询结果缓存：同一 query + filters 组合不重复查 DB
- LLM 查询理解结果缓存：相同自然语言查询不重复调 LLM

### 5. 实时搜索增强（CLI 场景适配）

CLI 天然不适合实时搜索，但可通过以下方式提升交互体验：

- `starman search --live`：进入交互模式，输入即搜（用 bubbletea 做 TUI）
- `starman search` 无参数时显示搜索历史 + 热门标签快速导航

---

## 方案 B：引入向量化，增加语义搜索

适用场景：追求最佳召回率和语义理解能力，愿意引入向量存储依赖。

### 整体架构

```
用户输入: "好看的终端工具"
     ↓
HyDE 查询增强（LLM 生成理想仓库描述）
     ↓
Embedding API（生成查询向量 768/1536 维）
     ↓
本地向量搜索（sqlite-vec kNN，topK=30）
     ↓
FTS5 关键词加分（名称/描述/标签匹配加权）
     ↓
LLM 语义重排序（精排 top 15）
     ↓
返回结果
```

### 1. Embedding 方案选型

| 方案 | 优点 | 缺点 | 推荐 |
|------|------|------|------|
| **sqlite-vec**（`asg017/sqlite-vec`） | 纯本地、零外部依赖、与现有 SQLite 集成 | 需要 CGO 或 WASM（Go 绑定不成熟，需 CGO） | **推荐** |
| 本地文件（gob/bolt） | 无额外依赖 | 暴力搜索 O(n)、无索引加速 | 小规模可接受 |
| 外部 Vectorize 服务 | 免维护 | CLI 工具需联网、增加延迟 | 不推荐 CLI 场景 |

**sqlite-vec 说明**：sqlite-vec 是 SQLite 的向量搜索扩展，支持 IVFFlat 索引、kNN 查询。Go 端需用 CGO 编译（`modernc.org/sqlite` 不支持扩展加载），对于 starman 当前纯 Go 构建方式是个破坏性变更。替代方案：
- 用 `mattn/go-sqlite3`（支持 CGO + 扩展加载）替换 `modernc.org/sqlite`
- 或使用纯 Go 的向量搜索库（如 `viterbi/faiss` 的 Go binding）

### 2. 向量存储设计

```
starred_repositories_vector 表
├── id            INTEGER PRIMARY KEY
├── repo_id       INTEGER UNIQUE (FK → starred_repositories.id)
├── embedding     BLOB        (向量，最多 1536 维 × 4 字节 = 6KB)
├── dim           INTEGER     (向量维度)
├── model         TEXT        (使用的 embedding 模型名)
├── text_hash    TEXT        (入参文本 hash，用于增量判断)
├── indexed_at    TEXT        (索引时间)
```

**索引策略**：
- AI 分析完成后**自动触发**向量化（可配置 `auto_vectorize: true/false`）
- `starman vectorize [--full]` 手动触发全量/增量重建
- 增量索引：对比 `analyzed_at` 和 `indexed_at`，只处理新分析或内容变更的仓库

### 3. Embedding API 多后端支持

参考 GithubStarsManager 的 `EmbeddingClient`，支持多种 Embedding API：

| API 类型 | 端点 | 说明 |
|----------|------|------|
| `openai` | `/v1/embeddings` | text-embedding-3-small（1536 维） |
| `siliconflow` | `/v1/embeddings` | 国产廉价 embedding，BGE-M3（1024 维） |
| `ollama` | `/api/embed` | 本地部署，推荐 nomic-embed-text（768 维） |
| `openai-compat` | 自定义 URL | 兼容 OpenAI 格式的任意服务 |

**配置方式**（`config.yaml`）：
```yaml
embedding:
  provider: openai           # openai / siliconflow / ollama / openai-compat
  model: text-embedding-3-small
  api_key: ${EMBEDDING_API_KEY}
  base_url: ""              # 自定义 endpoint（ollama: http://localhost:11434/v1）
```

### 4. 向量搜索流程

```
Search(service):
  1. HyDE 查询增强（可选，--hyde flag）
  2. EmbeddingClient.Embed(query) → queryVector
  3. VecStore.KNN(queryVector, topK=30, threshold=0.35)
     → 返回 [(repoId, similarity)]
  4. 关键词加分：
     - 名称精确匹配 +0.05
     - 描述匹配 +0.03
     - 标签匹配 +0.02
  5. 对 topN 做 LLM 语义重排序（--rerank，默认开启）
  6. 叠加结构化过滤（--lang/--tag/--platform/--stars）
  7. 返回最终排序结果
```

### 5. 搜索缓存

向量搜索结果按 `(query_hash, filters_hash)` 缓存到 SQLite：

```
search_cache 表
├── query_hash    TEXT
├── filters_hash  TEXT
├── results       BLOB    (gob 编码的 [(repoId, score)])
├── created_at    TEXT
```

存续时间 1 小时，超过失效自动清除。用户仅修改过滤条件时直接复用缓存结果重排序，不重复调 LLM 和向量搜索。

---

## 方案对比总结

| 维度 | 方案 A：FTS5 增强 | 方案 B：引入向量化 |
|------|-------------------|-------------------|
| **召回率** | 依赖关键词匹配，复杂语义可能漏召回 | 语义匹配，召回率显著提升 |
| **精度** | 依赖 LLM 精排补偿，上限受限 | 向量初步排序 + LLM 精排，精度更高 |
| **延迟** | 低（FTS5 毫秒级 + LLM 精排 3-5s） | 中（Embedding 0.5-1s + kNN 毫秒级 + LLM 精排 3-5s） |
| **依赖** | 无新依赖，纯工程优化 | 需引入向量存储（sqlite-vec / 纯 Go 库） |
| **复杂度** | 低，现有架构增量改进 | 中，需新增向量化流程和存储层 |
| **成本** | 仅 LLM token 成本 | LLM + Embedding API token 成本 |
| **适用场景** | 仓库量 < 5000，对召回率要求不极端 | 仓库量大（5000+），追求最佳语义匹配 |

**推荐路径**：先实施方案 A（低风险、快收益），验证优化效果。如果 FTS5 增强后召回率仍不满足需求，再考虑引入向量化。两种方案不互斥，方案 B 可视为方案 A 的超集。
