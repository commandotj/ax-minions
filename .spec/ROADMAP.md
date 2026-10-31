# RFC (Request for Comments) 流程管理

---

作者: albert.li/AI
创建时间: 2024-12-19
修改历史:

- 2024-12-19: 初稿 by AI
- 2024-12-19: 合并为统一文档 by AI

---

## 概述

RFC (Request for Comments) 是 commando-react 项目中用于记录和跟踪重要设计决策、架构变更和功能提案的标准化流程。本流程确保所有重大变更都经过充分的讨论、评审和文档化。

## RFC 生命周期

### 1. 提案阶段 (Proposed)

- 创建 RFC 文档，使用标准模板
- 在 RFC 索引中注册
- 状态标记为 `Proposed`

### 2. 讨论阶段 (Under Discussion)

- 团队内部讨论和反馈
- 收集相关利益相关者意见
- 根据反馈修改 RFC 内容

### 3. 评审阶段 (Under Review)

- 技术评审和可行性分析
- 影响评估和风险评估
- 实现计划制定

### 4. 批准阶段 (Approved)

- 获得必要的批准
- 制定实施时间表
- 分配实施资源

### 5. 实施阶段 (In Progress)

- 开始实施 RFC 提案
- 定期更新实施进度
- 跟踪实施过程中的问题

### 6. 完成阶段 (Completed)

- RFC 实施完成
- 更新文档状态
- 归档相关文档

### 7. 废弃阶段 (Deprecated)

- RFC 不再适用
- 标记为废弃状态
- 说明废弃原因

## RFC 文档结构

每个 RFC 文档应包含以下部分：

1. **元信息** - 作者、创建时间、修改历史
2. **背景** - 问题描述和现状分析
3. **目标** - 要达成的目标和预期效果
4. **提案** - 具体的解决方案和实现方案
5. **影响分析** - 对现有系统的影响
6. **风险评估** - 潜在风险和缓解措施
7. **实施计划** - 详细的实施步骤和时间表
8. **测试策略** - 测试计划和验收标准
9. **后续工作** - 相关的后续 RFC 或改进

## RFC 编号规则

- 文件名格式：`NNN-topic-name.md`
- NNN：三位数字序号（如 001, 002, 003...）
- topic-name：主题的英文描述，使用连字符分隔
- 示例：`001-log-module-design.md`、`002-batch-copy-unique-key.md`
- RFC 编号格式：`RFC-YYYY-NNN`（文档内部使用）

## RFC 状态定义

| 状态               | 描述             | 颜色标记 |
| ------------------ | ---------------- | -------- |
| `Proposed`         | 新提案，等待讨论 | 🔵       |
| `Under Discussion` | 正在讨论中       | 🟡       |
| `Under Review`     | 技术评审中       | 🟠       |
| `Approved`         | 已批准，准备实施 | 🟢       |
| `In Progress`      | 正在实施中       | 🔵       |
| `Completed`        | 已完成           | ✅       |
| `Deprecated`       | 已废弃           | ❌       |

## RFC 状态概览

| 状态                | 数量   | 百分比   |
| ------------------- | ------ | -------- |
| 🔵 Proposed         | 10     | 32.3%    |
| 🟡 Under Discussion | 0      | 0%       |
| 🟠 Under Review     | 0      | 0%       |
| 🟢 Approved         | 6      | 19.4%    |
| 🔵 In Progress      | 0      | 0%       |
| ✅ Completed        | 10     | 32.3%    |
| ❌ Deprecated       | 5      | 16.1%    |
| **总计**            | **31** | **100%** |

## 交付优先级

### P1 — CLI sync engine first

P1 只包含完成本地路径 CLI `compare → plan → safe execute → terminal outcome` 闭环所需 RFC。

| 顺序 | RFC                                                                       | P1 责任                   | Gate                                                       |
| ---- | ------------------------------------------------------------------------- | ------------------------- | ---------------------------------------------------------- |
| 1    | [RFC-2026-016](./rfc/completed/016-compare-engine-extensions.md)          | Compare engine            | engine 原语+单测；**不含** planner 接线                    |
| 2    | [RFC-2026-019](./rfc/completed/019-filter-system.md)                      | Filter CLI-first ✅       | matcher 经统一 engine 进入 CLI compare；非法 glob 报错     |
| 3    | [RFC-2026-030](./rfc/completed/030-local-smb-filesystem-compatibility.md) | Local filesystem baseline | 本地路径错误与文件系统差异可观察                           |
| 4    | [RFC-2026-017](./rfc/completed/017-sync-variants-changes-custom.md)       | Plan semantics            | Mirror/Update/Two-way 不用名称掩盖错误语义                 |
| 5    | [RFC-2026-031](./rfc/completed/031-compare-plan-e2e-verify.md)            | Compare↔Plan E2E verify   | **017 完成后**；BuildPlan 接 engine；CLI T1–T7 全绿        |
| 6    | [RFC-2026-018](./rfc/completed/018-sync-database.md)                      | Changes database          | Two-way/move detection 有持久状态与恢复                    |
| 7    | [RFC-2026-027](./rfc/completed/027-delete-operations.md)                  | Delete semantics          | Permanent/Trash/Versioning 行为显式                        |
| 8    | [RFC-2026-028](./rfc/completed/028-execution-options.md)                  | Safe execution            | core-owned plan；truthful Outcome；context；atomic replace |
| 9    | [RFC-2026-021](./rfc/completed/021-cli-config.md)                         | CLI contract              | 严格输入、NDJSON stdout、stderr 诊断、稳定退出码           |

**伞 RFC：** [RFC-2026-012](./rfc/completed/012-sync-module-compare-report.md) 为能力目录/Owner 章程。领域 RFC（014–031）均已打开后 **012 = Completed**，不占 P1 实现序号、不挡 016。

**卫生项（非伞阻塞）：** `CORE-01` module path rename、`CORE-02` 单 core、`CORE-03` P1 联调证据 — 见 TASK_TRACKING；可与 P1 并行，不要求“先做完 012 再 016”。

**依赖：** RFC-031 在 RFC-017 Completed 之后实施/验收，不阻塞 016 Done；不替代 017 语义工作。

P1 完成前，UI（**RFC-034** 取代 015/020/022/023）、report 增强（RFC-024）、platform/distribution（RFC-029）为 P2；Realtime/remote（RFC-014/025/026）为 P3。P2/P3 不得反向要求 P1 core 引入 Wails、React 或 provider-specific 类型。

### P1.5 — Compare/Sync E2E（UI 接线 + 收尾闭环）

P1 交付的是 core CLI 闭环；P1.5 接着把它接到 UI，并补上 verify/restore/replay 让 8 步工作流完整闭环。序号接续 P1（10 起）。

| 顺序 | RFC                                                          | P1.5 责任                               | Gate                                                                  |
| ---- | ------------------------------------------------------------ | --------------------------------------- | --------------------------------------------------------------------- |
| 10   | [RFC-2026-033](./rfc/033-sync-execution-progress.md)         | Execute 进度/取消/恢复（CLI 层）        | CLI `run --progress`/`--resume` 已实现；RFC 状态待同步                |
| 11   | [RFC-2026-034](./rfc/034-ui-consolidation.md)                | UI 整合 + **Wails CLI bridge（Gap 4）** | 🔶 In Progress；**Gap 4 挡 038/039/041 UI 验收**                      |
| 12   | [RFC-2026-037](./rfc/037-compare-cli-diff-view.md)           | Compare spawn + diff view（步骤 2–3）   | 依赖 034 Gap 4 + Gap 2；037 §2.1 CLI 部分已实现                       |
| 13   | [RFC-2026-039](./rfc/039-execute-verify-parity.md)           | Verify 覆盖 delete（步骤6 一部分）      | 与 14 可并行，互不依赖                                                |
| 14   | [RFC-2026-041](./rfc/041-plan-persistence-replay-history.md) | Plan/ExecuteResult 持久化、replay、历史 | 与 13 可并行；是 15 的地基；**2026-08-08 批准**                       |
| 15   | [RFC-2026-040](./rfc/040-restorable-cleanup-restore.md)      | Restorable 清理/撤销（步骤7）           | **依赖 13（Restorable 来源）+ 14（按 Plan.ID 持久化查询）**，最后实施 |

**伞 RFC：** [RFC-2026-038](./rfc/038-sync-execution-verify-restore-replan.md) 为 039/040/041 的能力目录，不占 P1.5 实现序号，完成标准是三份子 RFC 全部 Approved（非物理代码完工）。

**依赖顺序：** 034 **Gap 4** 先于 037/038 步骤 4–5 UI 验收。039、041 可并行；040 最后。037 §2.1：`plan --progress` 已在 CLI 实现，037 剩余 = Wails spawn + diff 视图。

P1.5 完成 = 8 步工作流全部经 **Wails spawn CLI** 端到端可用（038 §4；034 Gap 4）。

## RFC 列表

### 已完成的 RFC

| RFC 编号                                                                               | 标题                                                        | 作者                            | 创建时间   | 完成时间   | 状态         |
| -------------------------------------------------------------------------------------- | ----------------------------------------------------------- | ------------------------------- | ---------- | ---------- | ------------ |
| [RFC-2024-002](./rfc/completed/002-batch-copy-unique-key.md)                           | 批量复制唯一性与路径准确性设计                              | AI+albert.li                    | 2024-06-15 | 2024-06-15 | ✅ Completed |
| [RFC-2024-003](./rfc/completed/003-filepane-key-consistency.md)                        | FilePane 多选 key 类型一致性问题                            | AI (albert.li)                  | 2024-06-09 | 2024-06-09 | ✅ Completed |
| [RFC-2024-004](./rfc/completed/004-log-module-api.md)                                  | 日志模块 API 设计                                           | albert.li/AI                    | 2024-06-09 | 2024-06-09 | ✅ Completed |
| [RFC-2024-005](./rfc/completed/005-file-management-key-functions.md)                   | File Management Key Functions v1                            | Albert Lee/AI                   | 2025-01-14 | 2025-01-14 | ✅ Completed |
| [RFC-2026-011](./rfc/completed/011-commando-app-icon.md)                               | Commando 卡通应用图标                                       | Codex/albert.li                 | 2026-07-27 | 2026-07-27 | ✅ Completed |
| [RFC-2026-013](./rfc/completed/013-dev-prod-app-identity.md)                           | Wails Dev/Prod 应用身份分离                                 | Codex/albert.li                 | 2026-07-27 | 2026-07-27 | ✅ Completed |
| [RFC-2025-006](./rfc/completed/006-real-file-copy-operations.md)                       | Complete File Operations with Total Commander Functionality | Claude Code Assistant/albert.li | 2025-09-25 | 2026-07-29 | ✅ Completed |
| [RFC-2026-012](./rfc/completed/012-sync-module-compare-report.md)                      | FreeFileSync 能力目录与 Commando 决策追踪（伞）             | albert.li/AI                    | 2026-07-27 | 2026-07-29 | ✅ Completed |
| [RFC-2026-042](./rfc/completed/042-sync-empty-dir-toCopy-and-executor-double-count.md) | 空目录 toCopy 缺失 + Executor 计数翻倍：根因与修复          | albert.li/AI                    | 2026-08-06 | 2026-08-06 | ✅ Completed |

### 已批准的 RFC

| RFC 编号                                                            | 标题                                                      | 作者      | 创建时间   | 批准时间   | 状态         |
| ------------------------------------------------------------------- | --------------------------------------------------------- | --------- | ---------- | ---------- | ------------ |
| [RFC-2026-016](./rfc/completed/016-compare-engine-extensions.md)    | Compare Engine — Content, Symlinks, Tolerance & Parallel  | albert.li | 2026-07-29 | 2026-07-30 | ✅ Completed |
| [RFC-2026-017](./rfc/completed/017-sync-variants-changes-custom.md) | Sync Variants — Changes Mode, Two-way, Custom & Swap      | albert.li | 2026-07-29 | 2026-07-30 | ✅ Completed |
| [RFC-2026-018](./rfc/completed/018-sync-database.md)                | Sync Database — `sync.commando_db` & Moved File Detection | albert.li | 2026-07-29 | 2026-07-30 | ✅ Completed |
| [RFC-2026-019](./rfc/completed/019-filter-system.md)                | Filter System — Include/Exclude Glob                      | albert.li | 2026-07-29 | 2026-07-30 | ✅ Completed |
| [RFC-2026-021](./rfc/021-cli-config.md)                             | CLI & Config — Return Codes, Profiles, Schedules          | albert.li | 2026-07-29 | 2026-07-29 | 🟢 Approved  |
| [RFC-2026-024](./rfc/completed/024-reports.md)                      | Reports — Session Log, LastSyncs & Email                  | albert.li | 2026-07-29 | 2026-07-30 | ✅ Completed |
| [RFC-2026-027](./rfc/completed/027-delete-operations.md)            | Delete Operations — Recycle Bin & Versioning              | albert.li | 2026-07-29 | 2026-07-30 | ✅ Completed |
| [RFC-2026-028](./rfc/completed/028-execution-options.md)            | Execution Options — ctx, verify, atomic copy              | albert.li | 2026-07-29 | 2026-07-30 | ✅ Completed |
| [RFC-2026-041](./rfc/041-plan-persistence-replay-history.md)        | Plan & ExecuteResult Persistence — Replay & History       | albert.li | 2026-07-31 | 2026-08-08 | 🟢 Approved  |

### 架构设计 RFC

_暂无_

### 进行中的 RFC

| RFC 编号 | 标题 | 作者 | 创建时间 | 当前阶段 | 架构关联 |
| -------- | ---- | ---- | -------- | -------- | -------- |

### 提案中的 RFC

| RFC 编号                                                                  | 标题                                                                    | 作者         | 创建时间   | 状态        |
| ------------------------------------------------------------------------- | ----------------------------------------------------------------------- | ------------ | ---------- | ----------- |
| [RFC-2026-014](./rfc/014-remote-provider-gdrive-mtp.md)                   | 远程存储扩展 — Google Drive / MTP                                       | albert.li/AI | 2026-07-28 | 🔵 Proposed |
| [RFC-2026-025](./rfc/025-realtimesync.md)                                 | RealtimeSync — File Watcher, Idle Debounce & Service                    | albert.li    | 2026-07-29 | 🔵 Proposed |
| [RFC-2026-026](./rfc/026-remote-sftp-ftp.md)                              | Remote — SFTP & FTP Transfers                                           | albert.li    | 2026-07-29 | 🔵 Proposed |
| [RFC-2026-029](./rfc/029-platform-distribution-and-i18n.md)               | Platform, Distribution, Localization & Scale                            | albert.li/AI | 2026-07-29 | 🔵 Proposed |
| [RFC-2026-030](./rfc/completed/030-local-smb-filesystem-compatibility.md) | Local & SMB Filesystem Compatibility                                    | albert.li/AI | 2026-07-29 | 2026-07-30  | ✅ Completed |
| [RFC-2026-031](./rfc/completed/031-compare-plan-e2e-verify.md)            | Compare ↔ Plan E2E Verify（engine+planner+CLI）                         | albert.li/AI | 2026-07-29 | 2026-07-30  | ✅ Completed |
| [RFC-2026-037](./rfc/037-compare-cli-diff-view.md)                        | Compare CLI 子进程 + Diff View                                          | albert.li/AI | 2026-07-31 | 🔵 Proposed |
| [RFC-2026-038](./rfc/038-sync-execution-verify-restore-replan.md)         | Sync Execution E2E — Verify, Restore & Re-runnable Plans **(Umbrella)** | albert.li/AI | 2026-07-31 | 🔵 Proposed |
| [RFC-2026-039](./rfc/039-execute-verify-parity.md)                        | Execute — Verify Parity for Delete（子 RFC of 038）                     | albert.li/AI | 2026-07-31 | 🔵 Proposed |
| [RFC-2026-040](./rfc/040-restorable-cleanup-restore.md)                   | Restorable — Cleanup & Restore After Sync（子 RFC of 038）              | albert.li/AI | 2026-07-31 | 🔵 Proposed |

### 已废弃的 RFC

| RFC 编号                                                               | 标题                                                               | 作者                     | 创建时间   | 废弃时间   | 状态          | 原因                                                                             |
| ---------------------------------------------------------------------- | ------------------------------------------------------------------ | ------------------------ | ---------- | ---------- | ------------- | -------------------------------------------------------------------------------- |
| [RFC-2025-001](./rfc/completed/001-worker-architecture-vite.md)        | Worker Architecture for Electron-Vite                              | Albert Lee/AI            | 2025-01-14 | 2026-07-29 | ❌ Deprecated | 项目已迁移至 Wails 3，Electron-Vite Worker 架构不再适用                          |
| [RFC-2025-010](./rfc/completed/010-electron-spacetime-architecture.md) | Electron Spacetime Architecture with Personified Engines           | Linus Torvalds/albert.li | 2025-01-05 | 2026-07-29 | ❌ Deprecated | Electron-era architecture; Go backend replaces personified engine concepts       |
| [RFC-2025-008](./rfc/completed/008-window-service-api.md)              | Window Service API 设计                                            | AI Assistant             | 2025-01-27 | 2026-07-29 | ❌ Deprecated | Electron IPC window service; Wails 3 handles windows natively                    |
| [RFC-2025-009](./rfc/completed/009-shell-service-api.md)               | Shell Service API 设计                                             | AI Assistant             | 2025-01-27 | 2026-07-29 | ❌ Deprecated | Electron-era shell/LubanEngine service; Go backend replaces with direct packages |
| [RFC-2025-007](./rfc/completed/007-constant-management-system.md)      | Constant Management System                                         | Albert Lee/AI            | 2025-01-14 | 2026-07-29 | ❌ Deprecated | Electron-era TS constants; Go backend uses Go constants/iota                     |
| [RFC-2026-015](./rfc/rejected/015-ui-foundation.md)                    | UI Foundation — Folder Selection & Compare/Sync Settings           | albert.li                | 2026-07-29 | 2026-07-31 | ❌ Rejected   | 内容并入 RFC-2026-034（UI Consolidation）                                        |
| [RFC-2026-020](./rfc/rejected/020-ui-progress.md)                      | UI Progress — Compare/Sync Dialogs & Results                       | albert.li                | 2026-07-29 | 2026-07-31 | ❌ Rejected   | 内容并入 RFC-2026-034（UI Consolidation）                                        |
| [RFC-2026-022](./rfc/rejected/022-ui-panels.md)                        | UI Panels — Tree Overview, Category Filter, Direction & Multi-pair | albert.li                | 2026-07-29 | 2026-07-31 | ❌ Rejected   | 内容并入 RFC-2026-034（UI Consolidation）                                        |
| [RFC-2026-023](./rfc/rejected/023-ui-tools.md)                         | UI Tools — Explorer, External Tools, Context Menu & Rename         | albert.li                | 2026-07-29 | 2026-07-31 | ❌ Rejected   | 内容并入 RFC-2026-034（UI Consolidation）                                        |

## RFC 统计

### 按年份统计

| 年份 | 总数 | 已完成 | 已批准 | 进行中 | 提案中 | 已废弃 |
| ---- | ---- | ------ | ------ | ------ | ------ | ------ |
| 2024 | 4    | 4      | 0      | 0      | 0      | 0      |
| 2025 | 6    | 1      | 0      | 0      | 0      | 5      |
| 2026 | 21   | 3      | 9      | 0      | 9      | 0      |

### 按作者统计

作者统计不再手工维护；RFC 可能有多名共同作者，机械统计应从 RFC metadata 生成。

## FFS 能力决策总览

本节维护产品决策汇总；原子 Feature ID 与实施状态以 [TASK_TRACKING.md](./TASK_TRACKING.md) 为准，设计理由以 Owner RFC 为准。

### 按领域分布

| 领域                               | A (直接采纳) | AD (适配采纳) | R (拒绝) | D (延后) |
| ---------------------------------- | ------------ | ------------- | -------- | -------- |
| **UI** (界面工作流)                | 6            | 29            | 0        | 0        |
| **CMP/LINK** (比较引擎)            | 9            | 10            | 0        | 1        |
| **VAR/DB** (同步变体)              | 2            | 9             | 0        | 0        |
| **SYN/VER/META/EXEC** (执行与安全) | 4            | 23            | 0        | 1        |
| **FLT** (过滤系统)                 | 6            | 2             | 6        | 3        |
| **CLI/CFG/MAC** (命令行与配置)     | 4            | 12            | 1        | 3        |
| **EXT** (外部工具宏)               | 0            | 7             | 0        | 0        |
| **RPT** (报告)                     | 3            | 1             | 0        | 0        |
| **RTS** (实时同步)                 | 0            | 0             | 0        | 9        |
| **REM** (远程存储)                 | 1            | 0             | 0        | 6        |
| **PLT/NFR/I18N/DIST** (平台与分发) | 0            | 0             | 0        | 15       |
| **总计**                           | **35**       | **93**        | **7**    | **38**   |

### 按 Owner RFC 分布

| Owner RFC | 标题              | A   | AD  | R   | D   | 总计能力 |
| --------- | ----------------- | --- | --- | --- | --- | -------- |
| RFC-015   | UI Foundation     | 3   | 5   | 0   | 0   | 8        |
| RFC-016   | Compare Engine    | 9   | 10  | 0   | 1   | 20       |
| RFC-017   | Sync Variants     | 2   | 9   | 0   | 0   | 11       |
| RFC-018   | Sync Database     | 0   | 3   | 0   | 0   | 3        |
| RFC-019   | Filter System     | 6   | 2   | 6   | 3   | 17       |
| RFC-020   | UI Progress       | 1   | 3   | 0   | 0   | 4        |
| RFC-021   | CLI & Config      | 4   | 12  | 1   | 3   | 20       |
| RFC-022   | UI Panels         | 2   | 12  | 0   | 0   | 14       |
| RFC-023   | UI Tools          | 0   | 9   | 0   | 0   | 9        |
| RFC-024   | Reports           | 3   | 2   | 0   | 1   | 6        |
| RFC-025   | RealtimeSync      | 0   | 0   | 0   | 9   | 9        |
| RFC-026   | Remote SFTP/FTP   | 0   | 0   | 0   | 4   | 4        |
| RFC-027   | Delete Operations | 1   | 7   | 0   | 0   | 8        |
| RFC-028   | Execution Options | 3   | 18  | 0   | 0   | 21       |
| RFC-029   | Platform/Dist     | 0   | 0   | 0   | 15  | 15       |
| RFC-030   | Local/SMB         | 1   | 0   | 0   | 0   | 1        |
| RFC-014   | Remote GDrive/MTP | 0   | 0   | 0   | 3   | 3        |

## 最近更新

| 时间       | RFC          | 更新内容                                                                           | 作者            |
| ---------- | ------------ | ---------------------------------------------------------------------------------- | --------------- |
| 2026-08-08 | RFC-2026-041 | 批准：Plan/ExecuteResult 持久化、replay、历史 Viewer（子 RFC of 038）              | albert.li/AI    |
| 2026-07-30 | RFC-2026-019 | CLI-first filter 完成：BuildPlan→engine.IndexRoot, Validate(), --include/--exclude | albert.li/AI    |
| 2026-07-29 | RFC-2026-012 | Feature ID 全部绑定现存 Owner RFC；新增 RFC-029/030                                | Codex/albert.li |
| 2026-07-27 | RFC-2026-013 | Dev/Prod 身份、图标、打包入口与运行时验证完成                                      | Codex/albert.li |
| 2026-07-27 | RFC-2026-013 | 用户批准 Wails Dev/Prod 身份分离方案                                               | Codex/albert.li |
| 2026-07-27 | RFC-2026-011 | C1 应用于正式应用与平台资产，验证完成                                              | Codex/albert.li |
| 2026-07-27 | RFC-2026-011 | 生成并验证 C1 候选，进入候选审批                                                   | Codex/albert.li |
| 2026-07-27 | RFC-2026-012 | 创建 Go Sync Module Compare Report RFC（RFC-first 治理）                           | albert.li/AI    |
| 2026-07-27 | RFC-2026-012 | v2：轻量自研 engine；拒绝第三方 sync 库；FFS 够用范围                              | albert.li/AI    |
| 2026-07-28 | RFC-2026-012 | v3：**FFS 14.x 全功能 1:1**；§4 验收表 75+ 项；废弃裁剪                            | albert.li/AI    |
| 2026-07-28 | RFC-2026-012 | v3.9：§12 待决事项 OQ-01…05 全部关闭                                               | albert.li/AI    |
| 2026-07-28 | RFC-2026-014 | 从 RFC-2026-012 OQ-02 拆出，创建远程存储扩展 RFC（Google Drive / MTP）             | albert.li/AI    |
| 2026-07-27 | RFC-2026-011 | 用户批准，进入候选生成阶段                                                         | Codex/albert.li |
| 2026-07-27 | RFC-2026-011 | 创建 Commando 卡通应用图标提案                                                     | Codex/albert.li |
| 2025-01-27 | RFC-2025-009 | 创建 Shell Service API 设计文档                                                    | AI Assistant    |
| 2025-01-27 | RFC-2025-008 | 创建 Window Service API 设计文档                                                   | AI Assistant    |
| 2024-12-19 | RFC-2024-005 | 清理重复文件，更新索引                                                             | albert.li/AI    |
| 2024-12-19 | RFC-2024-004 | 转换为RFC格式并重命名                                                              | albert.li/AI    |
| 2024-12-19 | RFC-2024-003 | 转换为RFC格式并重命名                                                              | albert.li/AI    |
| 2024-12-19 | RFC-2024-002 | 转换为RFC格式并重命名                                                              | albert.li/AI    |
| 2024-12-19 | RFC-2024-001 | 创建 RFC 索引系统                                                                  | albert.li/AI    |
| 2025-09-27 | RFC-2025-006 | LubanEngine 进入实施阶段，记录实现路线                                             | Codex AI        |
| 2024-06-15 | RFC-2024-002 | 批量复制设计完成                                                                   | AI+albert.li    |
| 2024-06-09 | RFC-2024-001 | 日志模块设计完成                                                                   | albert.li/AI    |

## 下一步行动

### 待创建的 RFC

1. **主题系统 RFC** - 统一主题管理机制
2. **文件操作优化 RFC** - 批量操作性能优化
3. **国际化扩展 RFC** - 多语言支持扩展
4. **测试策略 RFC** - 测试覆盖率和质量保证

> **治理规则（2026-07-29）：** [RFC-2026-012](./rfc/completed/012-sync-module-compare-report.md) 管理 FFS 能力库存、Commando 决策、Owner 与跨 RFC 交付 gate。具体架构与验收由领域 RFC-014…041 管理；领域 RFC 获批前不得扩写其范围。

### 需要评审的 RFC

_暂无_

### 需要更新的 RFC

_暂无_

## 创建新 RFC

1. 使用下面的 RFC 模板创建新的 RFC 文档
2. 按照 RFC 编号规则命名文件（格式：`NNN-topic-name.md`）
3. 填写完整的 RFC 内容
4. 在此文档中注册新的 RFC
5. 提交 PR 并请求评审

## RFC 模板

```markdown
# RFC-YYYY-NNN: [RFC 标题]

---

作者: [作者姓名]
创建时间: [YYYY-MM-DD]
修改历史:

- [YYYY-MM-DD]: [修改描述] by [修改者]

---

## 摘要

[简要描述 RFC 的核心内容和目标，1-2 段话]

## 背景

### 问题描述

[详细描述当前存在的问题或需要改进的地方]

### 现状分析

[分析当前系统的现状，包括技术栈、架构、性能等方面]

### 业务驱动

[说明为什么需要这个 RFC，业务价值是什么]

## 目标

### 主要目标

- [ ] 目标 1
- [ ] 目标 2
- [ ] 目标 3

### 成功标准

[定义如何衡量 RFC 的成功，包括具体的指标和标准]

## 提案

### 解决方案概述

[高层次的解决方案描述]

### 技术方案

[详细的技术实现方案，包括架构图、流程图等]

### 实现细节

[具体的实现细节，包括代码结构、API 设计等]

## 影响分析

### 对现有系统的影响

- **正面影响**：
    - [影响 1]
    - [影响 2]
- **负面影响**：
    - [影响 1]
    - [影响 2]

### 兼容性分析

[分析对现有功能的兼容性影响]

### 性能影响

[分析对系统性能的影响，包括正面和负面]

## 风险评估

### 技术风险

| 风险     | 概率       | 影响       | 缓解措施   |
| -------- | ---------- | ---------- | ---------- |
| [风险 1] | [高/中/低] | [高/中/低] | [缓解措施] |
| [风险 2] | [高/中/低] | [高/中/低] | [缓解措施] |

### 业务风险

[分析可能对业务造成的影响]

### 实施风险

[分析实施过程中可能遇到的风险]

## 实施计划

### 阶段划分

1. **准备阶段** ([时间])
    - [ ] 任务 1
    - [ ] 任务 2
2. **开发阶段** ([时间])
    - [ ] 任务 1
    - [ ] 任务 2
3. **测试阶段** ([时间])
    - [ ] 任务 1
    - [ ] 任务 2
4. **部署阶段** ([时间])
    - [ ] 任务 1
    - [ ] 任务 2

### 资源需求

- **开发人员**：[人数] 人，[时间] 周
- **测试人员**：[人数] 人，[时间] 周
- **其他资源**：[具体需求]

### 时间计划

[甘特图或时间线，显示各个阶段的开始和结束时间]

## 测试策略

### 测试范围

[定义测试的范围和边界]

### 测试用例

[列出主要的测试用例]

### 验收标准

[定义验收标准，包括功能、性能、安全等方面]

## 后续工作

### 相关 RFC

[列出与此 RFC 相关的其他 RFC]

### 后续改进

[描述可能的后续改进和优化]

### 监控指标

[定义需要监控的指标，用于评估 RFC 的效果]

## 附录

### 参考资料

[列出相关的技术文档、标准、最佳实践等]

### 相关讨论

[记录相关的讨论和决策过程]

### 变更记录

[记录 RFC 的变更历史]

---

**状态**: [Proposed/Under Discussion/Under Review/Approved/In Progress/Completed/Deprecated]
**最后更新**: [YYYY-MM-DD]
**下次评审**: [YYYY-MM-DD]
```

## RFC 评审流程

### 技术评审

- 技术可行性分析
- 架构影响评估
- 性能影响分析
- 安全性评估

### 业务评审

- 业务价值评估
- 用户体验影响
- 市场竞争力分析
- ROI 分析

### 实施评审

- 资源需求评估
- 时间计划评估
- 风险评估
- 回滚计划

## 最佳实践

1. **明确问题**：RFC 应明确描述要解决的问题
2. **充分调研**：提供多种解决方案的对比分析
3. **详细实施**：包含详细的实施计划和步骤
4. **影响评估**：全面评估对现有系统的影响
5. **持续更新**：及时更新 RFC 状态和内容

## 联系信息

如有 RFC 流程相关问题，请联系：

- 项目维护者：albert.li
- 邮箱：albert_lee@hotmail.com
- 项目主页：https://www.systembug.com/commando

---

**最后更新**: 2024-12-19
**维护者**: albert.li
**联系方式**: albert_lee@hotmail.com
| 2026-07-29 | RFC-2026-012 | v4：改为 FFS 14.10 能力目录；设计所有权移交领域 RFC | Codex/albert.li |
