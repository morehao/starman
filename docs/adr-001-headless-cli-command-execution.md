# ADR-001：TUI 命令执行架构 —— headless 复用 CLI 命令树

- 状态：**Accepted**（替代《starman TUI 开发规范》原 4.3 节"命令模式规范"）
- 关联文档：`starman-tui-standards.md` §4.1、§4.3、§4.4、§8
- 决策范围：TUI 如何执行"对应 CLI 子命令"的操作（sync/analyze/generate/backup/tag/categorize/release/search/config 等），不涉及纯读取型的表格分页/导航逻辑

---

## 1. 背景

《starman TUI 模式设计》落地后暴露的核心问题不是"某几个命令没做"，而是**架构性地无法追上 CLI 的迭代**：

- 原方案：`commands.go` 里手写 `parseCommand`，对每个命令单独解析 flag，再手工映射到业务包函数。
- 结果：CLI 已有 `sync`/`star`/`unstar`/`help`/`q` 五个命令的解析，其余 `analyze`/`generate`/`backup`/`tag`/`categorize`/`release ...`/`config show` 全部缺失（见《starman TUI 模式设计》§12 差异清单）。
- 根因：这不是开发资源问题，是架构在持续制造维护债——CLI 每加一个 flag，TUI 侧就要同步改一遍解析代码，两边永远存在结构性滞后。
- 用户明确诉求：**CLI 模式下的命令要尽可能在 TUI 模式下可用**。手写映射的方式无法满足"尽可能"三个字，因为覆盖率取决于人力投入而不是架构上限。

同时，原设计里 `:config show` 这类结构化输出、`analyze --all` 这类批量长任务的执行过程，在 TUI 里只能看到一个 Task 完成后的摘要，看不到 CLI 裸跑时那种逐行进度，属于体验倒退。

## 2. 决策

### 2.1 核心决策：cobra 命令树是唯一的命令定义来源，TUI 命令模式复用而非重新解析

TUI 的 `:` 命令输入，不再走手写的 `parseCommand`，而是：

```
用户输入 :analyze --all --limit 50
        │
        ▼
按 shell 语义切分参数：["analyze", "--all", "--limit", "50"]
        │
        ▼
headless runner（新增，internal/tui/cmdrunner 包）：
  cmd := cli.NewRootCmd()            // 每次调用新建实例，避免命令间状态污染
  cmd.SetArgs(args)
  cmd.SetOut(drawerWriter)           // 输出实时写入 TUI 输出抽屉，见 2.3
  cmd.SetErr(drawerWriter)
  err := cmd.ExecuteContext(ctx)     // 复用 cobra 自身的 flag 解析、校验、RunE
        │
        ▼
执行完毕 → 触发 TaskFinishedMsg → 对应视图刷新 + footer 状态更新
```

**收益**：任何新增的 CLI 子命令或 flag，TUI 里自动可用，`commands.go` 不需要跟着每次 CLI 迭代同步修改。这是"尽可能在 TUI 模式下可用"这个目标唯一能长期成立的实现方式。

### 2.2 前置改造要求

要让 cobra 命令能被安全地 headless 调用，`internal/cli` 需要满足：

| 要求 | 说明 |
|---|---|
| `RunE` 不得调用 `os.Exit` | 所有错误必须 `return err`，退出逻辑收敛到 `main.go` 一处；TUI 场景下调用方绝不能被意外杀掉进程 |
| 每次执行新建 `*cobra.Command` 实例 | 禁止复用全局单例 Command 对象，避免并发执行（比如用户在等待上一条命令时又敲了新命令）互相污染 flag 状态 |
| 输出全部走 `cmd.OutOrStdout()`/`cmd.ErrOrStderr()` | 不允许命令内部直接 `fmt.Println`/写死 `os.Stdout`，否则 headless 场景下抓不到输出 |
| 幂等或可重入 | 命令执行期间用户可能再次触发同一命令（如连续两次 `:sync`），需要有防重入保护（见 2.5） |

这几条本身也是良好的 CLI 工程实践，不是为了 TUI 而牺牲 CLI 的设计。

### 2.3 输出抽屉（新增组件）

新增一个可收起的底部抽屉（快捷键呼出，如 `` ` ``），承接：
- 所有 headless 命令执行的实时流式输出（替代原 4.4 节"输出面板"里"执行完才展示结果"的设计）；
- 命令历史（`↑`/`↓` 翻阅，回车重新执行）；
- 结构化只读结果（如 `:config show`），复用同一个 viewport 组件。

布局位置：在原三段式骨架的 Footer 之上新增一层，默认收起不占空间，展开时挤压主体区域高度。抽屉本身遵循《starman TUI 开发规范》§4.2 "独立组件"要求，是一个独立 `tea.Model`。

抽屉收起时，Footer 仍按 §4.5 的规则展示单行 Task 状态摘要（进行中 spinner / ✅ / ❌），不依赖抽屉展开与否。

### 2.4 命令面板自动补全

`:` 之后的候选命令列表、参数提示，通过内省 `cmd.Commands()` / `cmd.Flags()` 生成，不手工维护。好处：候选列表和 help 面板文案永远与 CLI 实际能力同步，不会出现文档/面板过时的情况。

### 2.5 只读高频操作的例外边界

**不是所有操作都走 headless runner**。划分标准：

| 类型 | 判断标准 | 处理方式 |
|---|---|---|
| 独立 CLI 子命令 | 在 CLI 里是一个有名字、有自己 flag 的子命令（`sync`/`analyze`/`generate`/`backup`/`tag`/`categorize`/`release *`/`search`/`config show`/`star`/`unstar`） | 走 headless runner（2.1） |
| 高频读取/分页 | 表格滚动、sidebar 详情加载、行选中触发的懒加载 | 直接调用 `internal/store` 等业务包，不经过 cobra，避免每次翻页都拉起一个命令执行的开销 |

`search` 是过渡地带：`/` 搜索框输入回车后，内部调用 `search <query> --json`（走 headless runner，拿到结构化结果渲染成表格行），保证 TUI 搜索结果与 CLI `starman search` 完全一致的排序和召回逻辑，取代原来退化的内存字符串匹配。但搜索结果出来后的**表格内导航**（j/k 翻行）仍是纯本地状态操作，不重新调用命令。

### 2.6 例外命令：无法直接 headless 化的场景

| 命令 | 问题 | 处理方式 |
|---|---|---|
| `config init` | 依赖交互式 stdin 问答（多轮 prompt） | TUI 里不直接跑这个命令，改为原生的多步 Prompt 组件收集同样的字段，收集完直接调用 `internal/config` 的写入函数（不经过 cobra 的交互式 RunE） |
| `sync --watch` | 是一个常驻阻塞轮询进程，语义上不是"执行一次就结束"的命令 | TUI 拦截这个 flag：不允许通过 `:sync --watch` 触发真实的阻塞循环，改为 TUI 自己的定时 Task（复用现有 Task 系统）按 `--interval` 周期性触发普通 `:sync`，效果等价但不阻塞事件循环 |

这两类需要在 headless runner 的分发层做显式拦截和特殊处理，不能假装它们和其他命令一样直接转发。

### 2.7 键位是命令的语法糖

快捷键不再有独立的执行逻辑，而是"预填好参数、可能带确认弹窗的命令模式调用"：

| 按键 | 等价命令 | 备注 |
|---|---|---|
| `s` | `:sync` | 无需确认 |
| `S` | `:sync --full`（弹 y/n 确认） | 有删除语义，走确认 |
| `a` | `:analyze --limit 20` | 默认增量小批量 |
| `A` | `:analyze --all`（弹确认） | 全量 |
| `x`（当前行） | `:star <repo>` 或 `:unstar <repo>` | 按当前状态反转动词 |
| `t`（当前行） | 弹小输入框收集参数 → `:tag <repo> +a,-b` | 输入框是"命令参数收集器"，不是独立业务逻辑 |
| `c`（当前行） | 弹分类选择器 → `:categorize <repo> <cat>` | 同上 |
| `p`/`u`（Releases 视图） | `:release subscribe/unsubscribe <repo>` | |

好处：每个按键操作都能在输出抽屉的历史里追溯到一条具体命令字符串，便于调试，也降低从 CLI 迁移过来的用户的学习成本——TUI 里的每个动作背后都是他们已经熟悉的那条命令。

**破坏性操作的确认规则不受此影响**：`--full`、`--mode replace`、`unsubscribe` 等仍然强制弹 y/n 确认，确认逻辑在"拼出命令字符串"和"实际执行"之间插入一步，不允许绕过。

## 3. 考虑过的备选方案

**方案 B：继续手写 `parseCommand`，逐个补齐缺失命令**
- 优点：改动范围小，不涉及 cobra 改造。
- 拒绝原因：治标不治本，覆盖率永远滞后于 CLI 迭代速度，是当前问题的根源，不是解法。

**方案 C：把 CLI 命令背后的业务逻辑下沉成 `internal/<domain>` 的可导出函数，TUI 直接调用函数（不经过 cobra）**
- 优点：不需要处理 cobra 的 flag 解析、`os.Exit` 改造。
- 拒绝原因：TUI 侧仍然需要自己写一份"命令名 + flag → 函数参数"的映射，等于把手写解析从 `commands.go` 换了个地方，没有解决"两边定义分离、必然漂移"的根本问题；`generate`/`backup` 这类命令参数较多，重复定义的维护成本依然存在。
- 部分采纳：对于确实无法或不应该走完整 cobra 执行路径的场景（如上文 2.5 的高频读取），依然直接调用业务函数，作为 headless runner 之外的补充路径，而不是主路径。

**方案 D：`os/exec` 拉起真正的 `starman` 子进程执行命令**
- 优点：完全隔离，不需要改造 `internal/cli` 的 `os.Exit`/输出写入方式。
- 拒绝原因：进程启动开销大、跨进程 IO 流式传输复杂、无法直接复用 TUI 进程内的 Task 系统和 store 连接（子进程会打开自己的 SQLite 连接，存在锁冲突风险），且失去了类型安全的错误处理。

## 4. 影响与后续工作

**需要改造的既有代码**：
- `internal/cli` 各命令排查并移除裸 `os.Exit`（如有）
- `internal/cli` 的输出改为统一走 `cmd.OutOrStdout()`
- 新增 `internal/tui/cmdrunner` 包，封装 headless 执行逻辑 + 例外命令拦截（2.6）
- 新增输出抽屉组件（`components/drawer/`）
- `commands.go` 原有的手写 `parseCommand`/`executeCommand` 废弃，改为参数切分 + 转发给 `cmdrunner`

**测试影响**（对应《starman TUI 开发规范》§5 分层测试体系的调整）：
- L1：`cmdrunner` 的参数切分函数（字符串 → `[]string`，需处理带引号的参数）需要表驱动测试；例外命令拦截逻辑（`--watch` 改写为 Task、`config init` 路由到 Prompt）需要单独用例。
- L2：抽屉组件的 Update/View 测试（流式追加内容、历史翻阅）。
- L4：新增旅程——"`:analyze --all` 触发 headless 执行 → 抽屉实时显示逐行进度 → 完成后表格刷新"，替代原来只测试命令解析结果的用例。
- 不再需要为每个新命令写单独的 `parseCommand` 解析测试（因为不再手写解析），但需要一条"新增 CLI 命令后，TUI 命令面板自动补全能发现它"的回归测试，防止内省逻辑本身出 bug。

**风险与权衡**：
- headless 执行仍在同一进程内，命令执行期间如果某个 `RunE` 意外阻塞（忘记走 goroutine/context 超时），会卡住 TUI 事件循环。缓解措施：`cmdrunner` 统一在 `tea.Cmd` 里以独立 goroutine 执行，并对 `ExecuteContext` 传入的 `context` 设置合理超时或支持用户 `Ctrl+C` 取消。
- 命令行为一旦在 CLI 侧变更（比如某个 flag 含义调整），TUI 侧无需改代码就会"自动"跟着变，这既是优点也要求 CLI 侧变更时同步考虑 TUI 场景下的体验（比如某个 flag 在交互式场景下是否还合理）。

## 5. 与规范其他章节的关系

- 《starman TUI 开发规范》§4.3 整节由本 ADR 替代。
- §4.4"长文本/结构化输出的展示规范"中的"输出面板"概念，并入本 ADR §2.3 的输出抽屉，不再是弹出式 overlay，而是常驻可收起的抽屉。
- §4.1 键位分层规则保持有效，新增约束：动作类键位的处理函数职责收窄为"拼出命令字符串 + 按需弹确认"，不得在按键处理函数里直接调用业务包（业务逻辑收口到 headless runner 或高频读取的直连路径）。
- §7 架构约束表中"命令解析与执行只能在 `commands.go`，禁止反向调用 cobra Command"一条，由本 ADR 反转：**鼓励**通过 headless runner 调用 cobra 命令，原有"禁止调用 cobra"的表述作废。
