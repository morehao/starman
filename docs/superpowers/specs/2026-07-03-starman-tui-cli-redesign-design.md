# starman TUI/CLI 重构设计文档

- **日期**: 2026-07-03
- **状态**: 已评审（设计阶段）
- **分支**: 当前工作分支
- **目标**: 修复当前 TUI 设计问题，完成 CLI 能力合理下沉、布局交互优化、并确保 TUI 功能真实可用

## 1. 背景与问题陈述

当前项目已形成 TUI 默认入口（`starman`）+ 一组 CLI 子命令并存的形态，但存在以下问题：

1. **能力分布不一致**: 部分能力在 CLI 已较完整，但 TUI 页面仍为模拟或简化逻辑，造成“入口在 TUI，能力在 CLI”的割裂。
2. **布局与交互不统一**: 页面间键位语义、返回路径、状态反馈不一致，用户心智负担高。
3. **可验证性不足**: 若干 TUI 页面未接真实业务链路，无法满足“功能已实现”的工程验证标准。

本次重构聚焦于“单一产品体验”：TUI 为主、CLI 为精简高价值补充。

## 2. 目标与范围

### 2.1 目标

1. **CLI 下沉清晰**: 保留最小 CLI 集合，其余业务能力进入 TUI。
2. **布局交互合理**: 形成一致的信息架构、焦点流、键位协议。
3. **TUI 真实可用**: 所有目标页面接入真实业务实现，不保留模拟流程。

### 2.2 范围内

- 命令面重组（root 命令树调整）
- TUI 信息架构与布局体系重构
- TUI 页面对业务能力的接线重构
- 统一任务中心与错误反馈机制
- 对应测试与文档更新

### 2.3 范围外

- 新增与本次目标无关的业务功能
- 更换 TUI 技术栈（继续使用 bubbletea/lipgloss）
- 改动数据库模型（仅必要字段复用）

## 3. 约束与设计决策

### 3.1 已确认约束

- 迁移策略允许破坏性调整（项目仍在开发阶段）。
- CLI 保留集合为:
  - `config`
  - `completion`
  - `sync`
  - `search`
  - `--help` / `--version`
- TUI 交互采用**混合执行模型**:
  - 默认异步后台执行
  - 关键高风险操作前台确认
- 一级导航按**任务流分组**。
- 布局最小设计基线为 **100x30**。

### 3.2 方案对比与选型

#### 方案 A（采用）: TUI Orchestrator + 领域 Action 复用

- 建立统一 Action 层，CLI 与 TUI 共同调用。
- 优点: 行为一致、测试收敛、长期维护成本低。
- 代价: 需要一次性重构调用边界。

#### 方案 B（未采用）: 逐页补齐，不引入统一 Action

- 优点: 局部改动快。
- 缺点: 重复逻辑持续增长，CLI/TUI 语义漂移风险高。

#### 方案 C（未采用）: 先 UI 后能力

- 优点: 短期视觉反馈快。
- 缺点: 与“功能真实可用”目标冲突。

## 4. 目标态架构

### 4.1 分层

```
Entry Layer
  - CLI (config/completion/sync/search)
  - TUI (default)

Application Layer
  - Action/Usecase (统一参数校验、执行、结果模型)

Domain/Infra Layer
  - store / github / ai / generate / release / backup / discovery
```

### 4.2 关键原则

1. **页面不直接拼业务**: TUI Page 只管状态与渲染。
2. **命令与页面同源执行**: `sync/search` CLI 与 TUI 共用 Action。
3. **标准化结果**: 输出 `Result` 统一结构，便于状态栏、任务面板、CLI 渲染复用。

## 5. CLI 下沉映射

### 5.1 保留命令

- `starman config ...`
- `starman completion ...`
- `starman sync ...`
- `starman search ...`
- `starman --help` / `starman --version`

### 5.2 下沉到 TUI 的命令能力

从 root 子命令移除以下业务入口，并迁移到 TUI 页面:

- `analyze`
- `tag`
- `categorize`
- `stats`
- `release`
- `generate`
- `backup`
- `info`
- `trending`

### 5.3 兼容策略

采用**直接收敛**（无兼容壳）：

- 被移除命令不再保留软跳转。
- 文档明确“CLI 精简 + TUI 主入口”。

## 6. TUI 信息架构与布局

### 6.1 一级导航（任务流）

- **发现**: Search / Trending
- **整理**: Repo List / Repo Detail / Tag / Categorize / Stats
- **处理**: Sync / Analyze / Generate / Release / Backup
- **系统**: Config / Help

### 6.2 全局布局（100x30）

- 左栏 `22` 列固定: 分组导航与当前上下文。
- 中央主区: 页面主内容（列表、详情、表单、任务）。
- 底栏 `1` 行固定: 全局状态与任务摘要。

### 6.3 页面布局规则

1. 列表页（Search/Repo List/Trending）: `输入区 + 结果区`。
2. 详情页（Repo Detail/Config）: 宽屏双栏，窄屏单栏降级。
3. 任务页（Sync/Analyze/Generate/Backup/Release）: `参数面板 + 任务面板`。

### 6.4 统一交互协议

- `Tab / Shift+Tab`: 页面内焦点循环。
- `↑↓ / j/k`: 当前焦点区域内导航。
- `Enter`: 触发当前焦点主动作。
- `Esc`: 返回上一级上下文（非硬跳 Dashboard）。
- `?`: 当前页上下文帮助。

### 6.5 尺寸降级

- `<100x30`: 侧栏压缩、双栏降单栏、底栏降信息密度。
- `<80x24`: 进入精简模式并提示窗口过小。

## 7. 任务中心与执行模型

### 7.1 状态机

`Queued -> Running -> Success | Failed | Cancelled`

### 7.2 任务类型

- 默认异步后台:
  - `sync`
  - `analyze`
  - `generate`
  - `release pull`
  - `backup export`
- 前台确认/受控执行:
  - `backup import`（覆盖类）
  - 大批量 `tag/categorize`（高影响面）

### 7.3 并发策略

- 默认串行队列，避免资源竞争与状态冲突。
- 同类任务去重（例如重复触发 `sync`）。

## 8. 数据流与错误处理

### 8.1 数据流

`UI 输入 -> Action 参数校验 -> Service 调用 -> 标准 Result -> UI 渲染`

### 8.2 标准结果模型

Result 至少包含:

- `Summary`
- `Data/Rows`
- `Warnings`
- `Err`
- `Metrics`（如 duration、count）

### 8.3 错误分类

- `ValidationError`: 参数/配置错误，可页内修复。
- `DependencyError`: token/key/webdav 等依赖缺失或不可用。
- `RuntimeError`: 网络超时、限流、DB 锁等运行期错误。
- `PartialError`: 批处理部分失败，展示成功/失败统计。

## 9. 模块重构计划（文件级）

### 9.1 CLI

- 修改 `internal/cli/root.go`
  - 仅保留 config/completion/sync/search 及全局能力
  - 移除下沉命令注册

- 保持并改造:
  - `internal/cli/sync.go`
  - `internal/cli/search.go`
  - 两者改为调用统一 Action

### 9.2 新增应用层

- 新增目录 `internal/app`（命名可在实现阶段最终确定）
  - `sync_action.go`
  - `search_action.go`
  - `analyze_action.go`
  - `...`
  - `result.go`

### 9.3 TUI

- 修改 `internal/tui/tui.go`
  - 接入 TaskCenter
  - 统一消息分发与页面路由

- 修改 `internal/tui/components/sidebar.go`
  - 调整为任务流分组导航

- 修改 `internal/tui/pages/*`
  - 移除模拟逻辑
  - 统一接线 Action

## 10. 验收标准（Definition of Done）

### 10.1 命令面

1. `starman` 默认进入 TUI。
2. root 仅保留约定 CLI 集。
3. 被下沉命令不再出现在 `starman --help` 子命令列表。

### 10.2 交互面

1. 一级导航按任务流分组。
2. 关键键位在页面间语义一致。
3. 在 100x30 下布局完整可用。

### 10.3 功能面

1. 目标 TUI 页面全部接入真实执行路径。
2. 不存在“假进度/假结果”模拟页面。
3. 任务中心能展示完整生命周期与失败详情。

## 11. 测试策略

### 11.1 单元测试

- Action 参数校验
- 结果组装
- 错误分类与映射

### 11.2 组件/页面测试

- 统一键位流（Tab/Enter/Esc）
- 页面消息路由与状态变迁

### 11.3 集成测试

- `sync/search` CLI 可执行且行为稳定
- TUI 关键页触发真实 Action（用 mock 依赖）
- root 命令树回归校验

### 11.4 回归命令

```bash
go test ./...
starman --help
starman sync --help
starman search --help
```

## 12. 实施顺序

1. 调整 root 命令树与 README 命令文档
2. 引入 Action 层与 Result 结构
3. 对齐 `sync/search` 的 CLI 与 TUI 调用路径
4. 重构侧栏与全局交互协议
5. 逐页替换模拟逻辑为真实执行
6. 接入任务中心与错误面板
7. 完成测试回归与文档收口

## 13. 风险与缓解

- **风险**: 一次性下沉导致短期命令不兼容
  - **缓解**: 文档同步更新 + 帮助信息明确说明

- **风险**: Action 层重构初期引入回归
  - **缓解**: 先收敛 `sync/search` 两条主链路，建立模板后再扩展

- **风险**: TUI 异步任务带来状态复杂度
  - **缓解**: 状态机与消息协议先落地，再接页面

## 14. 与本次需求的对应关系

1. **“确保 CLI 模式相关命令合理放到 TUI”**
   - 已定义明确保留集与下沉集，且采用直接收敛策略。
2. **“确保合理布局和交互”**
   - 已给出任务流 IA、三段式布局、统一键位协议、尺寸降级规则。
3. **“确保新方案中 TUI 功能正常实现”**
   - 已建立真实执行链路约束、任务中心状态机、可验证验收标准与测试矩阵。
