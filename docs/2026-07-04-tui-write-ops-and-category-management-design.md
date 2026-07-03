# TUI 写操作与分类管理 设计文档

> 2026-07-04

## 概述

为 starman 添加三类功能：
1. **分类 CRUD 管理**（CLI + Store 层接口）
2. **TUI 写操作**：在 TUI 中修改仓库分类、标签，以及 star/unstar
3. **Store 层重构**：抽取单仓操作接口供 CLI 和 TUI 复用

## 架构

```
┌──────────────────────────────────┐
│           CLI 层                  │
│  category list/add/edit/delete   │  ← 新增
│  star/unstar (已有)              │
│  tag (已有)                      │
│  categorize (已有)               │
└──────────────┬───────────────────┘
               │ 调用 store 接口
┌──────────────▼───────────────────┐
│         Store 层                  │
│  ListCategories()           ← 新增
│  AddCategory()              ← 新增
│  UpdateCategory()           ← 新增
│  DeleteCategory()           ← 新增
│  SetRepoCategory()          ← 抽取
│  AddRepoTags()              ← 抽取
│  RemoveRepoTags()           ← 抽取
│  SetRepoTags()              ← 抽取
│  StarRepo()                 ← 已有 (github/client)
│  UnstarRepo()               ← 已有 (github/client)
└──────────────┬───────────────────┘
               │
┌──────────────▼───────────────────┐
│           TUI 层                  │
│  Stars section: x → toggle star  │
│  Stars/Sidebar: c → 改分类       │
│  Stars/Sidebar: t → 改标签       │
└──────────────────────────────────┘
```

- CLI 和 TUI 共享 store 接口，TUI 不直接操作 SQL
- 内置分类不可删除，只能修改名称/关键词/排序
- 删除分类时级联清空相关仓库的 CustomCategory

## CLI 命令设计

### `starman category`

```
starman category list                # 列出所有分类
starman category add <id> [name]     # 新增自定义分类
starman category edit <id> [...]     # 修改分类属性
starman category delete <id>         # 删除自定义分类
```

#### `category list`

表格输出所有分类：ID、名称、关键词、排序、类型（内置/自定义）。支持 `--output json`。

#### `category add`

```
starman category add <id> [name] [--keywords k1,k2] [--sort-order N]
```

- `id` 必填，自动 slug 化（小写 + 连字符）
- `name` 可选，不指定则用 id
- `--keywords` 逗号分隔
- `--sort-order` 默认追加到末尾

#### `category edit`

```
starman category edit <id> [--name NAME] [--keywords k1,k2] [--sort-order N]
```

- 所有 flag 可选，只更新指定字段
- 内置分类也可编辑

#### `category delete`

```
starman category delete <id> [--force]
```

- 内置分类拒绝删除
- 显示受影响仓库数量
- `--force` 跳过确认
- 级联清空关联仓库的 `custom_category`

## Store 层新增接口

文件：`internal/store/category.go`（新增）

```go
func (s *Store) ListCategories() ([]*Category, error)
func (s *Store) GetCategory(id string) (*Category, error)
func (s *Store) AddCategory(cat *Category) error
func (s *Store) UpdateCategory(id string, updates map[string]any) error
func (s *Store) DeleteCategory(id string) (affectedRepoCount int, err error)
```

文件：`internal/store/repository.go`（新增方法）

```go
func (s *Store) SetRepoCategory(repoID int64, category string) error
func (s *Store) AddRepoTags(repoID int64, tags ...string) error
func (s *Store) RemoveRepoTags(repoID int64, tags ...string) error
func (s *Store) SetRepoTags(repoID int64, tags []string) error
```

- `DeleteCategory` 在单个事务中执行：删除分类记录 + 批量清空关联仓库的 `custom_category`
- 批量操作（`categorize --lang` 等）复用现有批量查询 + 循环调用单仓方法

## TUI 交互设计

### 快捷键

| 按键 | 位置 | 功能 |
|------|------|------|
| `x` | Stars 列表 | toggle star / unstar |
| `c` | Stars 列表 / Sidebar | 修改仓库分类 |
| `t` | Stars 列表 / Sidebar | 修改仓库标签 |

### 交互流程

**star/unstar (`x`)**：按下后直接调 GitHub API，footer 显示异步任务结果提示。

**改分类 (`c`)**：弹出分类选择器（可滚动列表），显示所有分类，高亮当前项。Enter 确认，Esc 取消。最终效果等同调用 `SetRepoCategory`。

**改标签 (`t`)**：弹出标签编辑器，预填当前标签，以 `+tag1,-tag2` 格式编辑。Enter 提交，Esc 取消。最终效果等同调用 `AddRepoTags` / `RemoveRepoTags`。

### 实现方式

在 TUI 的 `stars section` Model 中新增：

- `editingMode` 状态字段（`none` / `category` / `tag` / `confirmStar`）
- 分类选择器用 Bubble Tea `List` 组件
- 标签编辑器用 Bubble Tea `Textarea` 组件
- 确认弹窗（star/unstar / 删除确认等）复用通用确认组件

不新增独立组件目录，轻量嵌入现有 `starssection` 的 `Update` / `View` 循环中。

## 错误处理

- GitHub API 调用失败：footer 显示错误提示
- 分类 ID 重复：拒绝创建并报错
- 内置分类删除：拒绝并提示
- 无效分类 ID（分类不存在）：`categorize` 和 `category edit` 报错

## 测试策略

- Store 层分类 CRUD 方法添加单元测试
- `DeleteCategory` 级联清空行为有独立测试用例
- TUI 操作的关键路径（star/unstar、改分类、改标签）通过集成测试覆盖
