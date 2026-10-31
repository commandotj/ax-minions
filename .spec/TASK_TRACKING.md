# Task Tracking

本文件是原子实施任务与 Feature ID 的权威追踪。RFC 状态、P1/P2/P3 和跨 RFC gate 见 [ROADMAP.md](./ROADMAP.md)；产品决策与设计理由见对应 Owner RFC。

**状态**: 🔵 To do · 🟡 In Progress · 🟢 Done

`CORE-*` 是 Commando 内部架构 gate，不计入 FFS 能力统计。

---

## P1 — CLI sync engine first

P1 目标：本地路径 CLI `compare → deterministic plan → safe execute → terminal outcome`，不依赖 Wails 或 React。

| 顺序 | RFC     | 原子 Gate                                                           | 状态 |
| ---- | ------- | ------------------------------------------------------------------- | ---- |
| 1    | RFC-016 | engine 原语：link、容差、并行、确定性输出（不含 planner）           | 🟢   |
| 2    | RFC-019 | Include/exclude 经统一 engine 进入 CLI compare                      | 🟢   |
| 3    | RFC-030 | 本地路径错误和文件系统差异可观察                                    | 🟢   |
| 4    | RFC-017 | Mirror/Update/Two-way plan 语义正确                                 | 🟢   |
| 5    | RFC-031 | **017 完成后** Compare↔Plan E2E：BuildPlan 接 engine；CLI T1–T7     | 🟢   |
| 6    | RFC-018 | Two-way/move detection 有持久状态与恢复                             | 🟢   |
| 7    | RFC-027 | Permanent/Trash/Versioning 删除语义                                 | 🟢   |
| 8    | RFC-028 | core-owned plan；context、fail-safe、verify、lock、truthful Outcome | 🟢   |
| 9    | RFC-021 | 严格输入、NDJSON stdout、stderr 诊断、稳定退出码                    | 🟢   |

**RFC-012 伞：** ✅ Completed（领域 RFC 已齐）。不占上表序号。

**卫生 CORE-\***（可与 P1 并行，不挡 016）：

| Task                                         | IDs     | Status |
| -------------------------------------------- | ------- | ------ |
| Go module → `github.com/commandotj/commando` | CORE-01 | 🟢     |
| CLI/Desktop 单一 compare/copy/execute core   | CORE-02 | 🔵     |
| P1 末段 CLI destructive/integration 证据     | CORE-03 | 🟢     |

P1 RFC 内的 UI-only、email/HTML、remote、RealtimeSync、distribution 任务不进入 P1 gate。

---

## 已批准的 RFC

### RFC-012 — 伞（Completed）

章程已归档 `rfc/completed/012-…`。CORE-* 见上表卫生项，不阻塞领域 RFC。

### RFC-016 — Compare Engine（engine only；接线见 031）

| Task                                                   | IDs                                | Status                       |
| ------------------------------------------------------ | ---------------------------------- | ---------------------------- |
| 并行目录遍历 WalkDir（errgroup+semaphore，确定性排序） | CMP-10                             | 🟢                           |
| 并行二进制比较（批量 ParallelCompare，buffer pool）    | CMP-11                             | 🟢                           |
| Content 原语 `IsEqual(Content)`（已有）                | CMP-02                             | 🟢                           |
| 文件时间/大小比较（已有）                              | CMP-01, 03                         | 🟢                           |
| Symbolic link 处理（已有）                             | CMP-04, 05, 06, LINK-01            | 🟢                           |
| 文件时间容差配置（已有）                               | CMP-08                             | 🟢                           |
| Unicode 路径支持（已有 + 新增测试）                    | CMP-13                             | 🟢                           |
| Symlink Follow 循环检测（路径-based + visited set）    | CMP-07 (partial)                   | 🟢                           |
| CaseMode 配置（sensitive/insensitive/auto）            | CMP-14                             | 🟢                           |
| Junction/Mount/WSL 链接种类识别                        | CMP-07 (linkKind), LINK-02, 03, 04 | 🔵 skip — Windows-only，记债 |
| FAT 夏令时处理                                         | CMP-09                             | 🔵 Defer                     |
| Windows 长路径支持（`\\?\` 前缀）                      | CMP-12                             | 🔵 skip — Windows-only，记债 |

### RFC-031 — Compare ↔ Plan E2E Verify

| Task                                                             | IDs       | Status |
| ---------------------------------------------------------------- | --------- | ------ |
| BuildPlan/BuildReport 只经 engine.IndexRoot + Categorize/IsEqual | VERIFY-01 | 🟢     |
| 删除 entriesEqual                                                | VERIFY-02 | 🟢     |
| UseChecksum → CompareMode.Content                                | VERIFY-03 | 🟢     |
| TimeAndSize + ToleranceSec 一致                                  | VERIFY-04 | 🟢     |
| SymlinkMode 传入 IndexRoot（默认 Exclude）                       | VERIFY-05 | 🟢     |
| CLI 与库同一 BuildPlan                                           | VERIFY-06 | 🟢     |
| T1–T7 集成测全绿                                                 | VERIFY-07 | 🟢     |

### RFC-017 — Sync Variants (11 actionable)

| Task                                  | IDs        | Status         |
| ------------------------------------- | ---------- | -------------- |
| Mirror / Update 变体（已有）          | VAR-01, 02 | 🟢             |
| 冲突识别（已有）                      | CMP-15     | 🟢             |
| 安全交换左右路径（Swap）              | VAR-08     | 🟢             |
| Custom 自定义规则（differences 模式） | VAR-05     | 🟢             |
| 跨变体保留过滤器                      | VAR-07     | 🟢             |
| Update (changes) 基于 DB 变更检测     | VAR-03     | 🟢             |
| Two-way (changes) — 双向 DB           | VAR-04     | 🔵 RFC-017-ext |
| Custom (changes) — 变更类型动作映射   | VAR-06     | 🔵 RFC-017-ext |

### RFC-018 — Sync Database (3 actionable)

| Task                             | IDs           | Status |
| -------------------------------- | ------------- | ------ |
| 持久化 sync.commando_db 变更检测 | SYN-01, DB-01 | 🟢     |
| 基于稳定文件 ID 的移动检测       | SYN-02, DB-02 | 🟢     |
| 无稳定 ID 时回退复制+删除        | SYN-02, DB-03 | 🟢     |

### RFC-019 — Filter System

| Task                                                          | IDs                    | Status       |
| ------------------------------------------------------------- | ---------------------- | ------------ |
| Matcher 原语已有；planner/CLI 集成完成（BuildPlan→IndexRoot） | FLT-01, 02, 04, 05, 08 | 🟢           |
| 非法 glob 返回 typed configuration error                      | FLT-03                 | 🟢           |
| 路径分隔符规范化（已有）                                      | FLT-07                 | 🟢           |
| 快速排除 UI 入口                                              | FLT-06                 | 🔵 → RFC-032 |

### RFC-021 — CLI & Config

| Task                                                         | IDs                | Status |
| ------------------------------------------------------------ | ------------------ | ------ |
| sync plan/run 骨架已有；Compare/Sync/Export 端到端未完成     | CLI-01, 02, 03     | 🔵     |
| 单行 NDJSON progress/item_error/terminal contract            | CLI-11             | 🔵     |
| 命令行覆盖左右目录                                           | CLI-04             | 🔵     |
| 未知 direction/strategy 拒绝；destructive run 无隐式 Two-way | CLI-04, 05         | 🔵     |
| Outcome 映射退出码 0/1/2/3                                   | CLI-05             | 🔵     |
| 多配置文件合并                                               | CLI-06             | 🔵     |
| 免 GUI 执行已保存配置                                        | CLI-07, CFG-05     | 🔵     |
| 启动 GUI 后直接对比                                          | CLI-09             | 🔵     |
| 选择全局配置                                                 | CLI-10             | 🔵     |
| 详细终端输出（状态/计时/计数/路径）                          | CLI-12             | 🔵     |
| 全局配置与同步配置分离                                       | CFG-01             | 🔵     |
| 宏展开（日期时间/环境变量/CSIDL）                            | CFG-02, MAC-01..03 | 🔵     |
| 可变盘符按卷标解析                                           | CFG-03             | 🔵     |
| 全局默认与配对级覆盖                                         | CFG-04             | 🔵     |
| 保存/加载同步配置 (UI)                                       | UI-12, 13          | 🔵     |

### RFC-024 — Reports (5 actionable)

| Task                         | IDs    | Status |
| ---------------------------- | ------ | ------ |
| CSV 导出（已有）             | RPT-02 | 🟢     |
| Text 导出（已有）            | RPT-03 | 🟢     |
| 结构化 CompareReport（已有） | RPT-01 | 🟢     |
| 详细 HTML/Text 同步日志      | RPT-04 | 🔵     |
| 最近同步日志 UI              | UI-24  | 🔵     |

### RFC-027 — Delete Operations (8 actionable)

| Task                                 | IDs                | Status |
| ------------------------------------ | ------------------ | ------ |
| 永久删除（已有）                     | SYN-03             | 🟢     |
| 回收站删除（macOS Finder/Linux XDG） | SYN-04             | 🟢     |
| Versioning（时间戳重命名）           | SYN-05, 06         | 🟢     |
| Versioning Replace（覆盖）           | SYN-07             | 🟢     |
| 宏路径（待后续）                     | SYN-08, VER-01, 02 | 🔵     |

### RFC-028 — Execution Options

| Task                                           | IDs                | Status |
| ---------------------------------------------- | ------------------ | ------ |
| 自动创建目标目录（已有）                       | SYN-13             | 🟢     |
| 保留 mtime（copyFileAtomic 已含 Chtimes）      | SYN-14             | 🟢     |
| Truthful typed Outcome + ErrorMode stop/ignore | EXEC-01, SYN-09,10 | 🟢     |
| Context 取消贯穿 BuildPlan/Execute/copy/delete | EXEC-02            | 🟢     |
| 复制后二进制校验                               | SYN-11             | 🟢     |
| Fail-safe copy（temp+rename atomic）           | SYN-15             | 🟢     |
| VSS 卷影复制（Windows）                        | SYN-12             | 🔵     |
| 唯一复制实现；temp+flush+verify+atomic replace | SYN-15             | 🔵     |
| 每设备并行数配置                               | SYN-16             | 🔵     |
| 执行顺序优化降峰值                             | SYN-17             | 🔵     |
| 文件夹锁文件                                   | SYN-18             | 🔵     |
| 降低 IO 优先级                                 | SYN-19             | 🔵     |
| NTFS 元数据保留（压缩/加密/DACL/ADS）          | META-01, 02, 03    | 🔵     |
| HFS+ 元数据保留（扩展属性/ACL）                | META-04, 05        | 🔵     |

### RFC-042 — 空目录 toCopy + Executor 计数（Completed）

| Task                                                          | IDs         | 状态 |
| ------------------------------------------------------------- | ----------- | ---- |
| planner：缺失空目录 → `ActionCopy` + `IsDir`（两方向）        | EXEC-03     | 🟢   |
| executor：`IsDir` 走 `os.MkdirAll`，不再走 copyFileAtomic     | EXEC-04     | 🟢   |
| executor：file copy 只计 `Copied` 一次（删 double `Copied++`) | EXEC-05     | 🟢   |
| UI：normalize + SyncDiffView 保留并渲染空目录行               | UI-36/UI-37 | 🟢   |
| 回归测试集（Go executor + UI）见 RFC-042 §4                   | EXEC-06     | 🟢   |

### RFC-041 — Plan/ExecuteResult 持久化 + Replay + History（Approved 2026-08-08）

| Task                                                                                    | IDs     | 状态 |
| --------------------------------------------------------------------------------------- | ------- | ---- |
| Plan 增加 ID/CreatedAt；planstore.go：Save/Load/List/SaveResult/LoadResult              | PLAN-01 | 🔵   |
| BuildPlan 自动 SavePlan；Execute 自动 SaveExecuteResult（含 ItemSnapshots）             | PLAN-02 | 🔵   |
| ReplayPlan：按 DoneItems 过滤 + stale-source 校验                                       | PLAN-03 | 🔵   |
| CLI：`sync replay` + `sync list-plans --json`（与 --resume 互斥）                       | CLI-13  | 🔵   |
| 桌面服务 ListPlans/ReplayPlan（spawn CLI）                                              | UI-38   | 🔵   |
| UI：PlanHistoryPanel + SyncPlanModal view 模式（只读，禁止误触 sync）                   | UI-39   | 🔵   |
| Go 测试集 11 项 + UI 测试 2 项（RFC-041 §9）                                            | PLAN-04 | 🔵   |
| 评审注记：pinned 输出测试更新 + 缺失快照降级 + DoneItems 断言 + 互斥测试（RFC-041 §13） | PLAN-05 | 🔵   |

---

## 提案中的 RFC

### RFC-015 — UI Foundation (8 actionable)

| Task               | IDs   | Status |
| ------------------ | ----- | ------ |
| 文件夹选择（已有） | UI-01 | 🟢     |
| 开始对比（已有）   | UI-02 | 🟢     |
| 对比后同步（已有） | UI-06 | 🟢     |
| 比较设置面板       | UI-03 | 🔵     |
| 同步设置面板       | UI-04 | 🔵     |
| 免对比直接同步     | UI-28 | 🔵     |
| 拖拽填入文件夹     | UI-29 | 🔵     |
| 清除历史路径       | UI-34 | 🔵     |

### RFC-020 — UI Progress (4 actionable)

| Task                             | IDs   | Status |
| -------------------------------- | ----- | ------ |
| 统计摘要条（已有）               | UI-10 | 🟢     |
| 对比进度与取消                   | UI-20 | 🔵     |
| 同步进度/ETA                     | UI-21 | 🔵     |
| 终端成功/警告/错误汇总           | UI-22 | 🔵     |
| 完成流程（最小化/自动关闭/可见） | UI-23 | 🔵     |

### RFC-022 — UI Panels (14 actionable)

| Task                 | IDs   | Status |
| -------------------- | ----- | ------ |
| 同步计划预览（已有） | UI-07 | 🟢     |
| 分类图例（已有）     | UI-11 | 🟢     |
| 目录树总览           | UI-08 | 🔵     |
| 按分类过滤列表       | UI-09 | 🔵     |
| 多文件夹对支持       | UI-14 | 🔵     |
| 单条动作覆写         | UI-15 | 🔵     |
| 子树动作覆写         | UI-16 | 🔵     |
| 仅同步选中项         | UI-27 | 🔵     |
| 仅对比/仅同步模式    | UI-27 | 🔵     |
| 复制到替代目标       | UI-31 | 🔵     |
| 驱动器空间分布       | UI-26 | 🔵     |
| 图片缩略图           | UI-32 | 🔵     |
| 默认视图过滤持久化   | UI-33 | 🔵     |
| 文件夹对重排序       | UI-35 | 🔵     |

### RFC-023 — UI Tools (9 actionable)

| Task                 | IDs        | Status |
| -------------------- | ---------- | ------ |
| 平台文件管理器中显示 | UI-17      | 🔵     |
| 外部工具配置         | UI-18      | 🔵     |
| 上下文命令执行       | UI-19      | 🔵     |
| 批量重命名           | UI-25      | 🔵     |
| 外部工具宏系统       | EXT-01..07 | 🔵     |

### RFC-030 — Local/SMB (1 actionable)

| Task                        | IDs    | Status |
| --------------------------- | ------ | ------ |
| 本地路径与 SMB 共享（已有） | REM-01 | 🟢     |

---

## 无 Actionable 项的 RFC

以下 RFC 的能力项全部为 **D (Defer)** 或 **R (Reject)**，暂无实现任务：

| RFC                             | 原因                          |
| ------------------------------- | ----------------------------- |
| RFC-014 (Remote GDrive/MTP)     | 全部 D — 远程提供商延后       |
| RFC-025 (RealtimeSync)          | 全部 D — 独立产品功能，延后   |
| RFC-026 (Remote SFTP/FTP)       | 全部 D — 远程传输延后         |
| RFC-029 (Platform/Distribution) | 全部 D — 平台/分发/国际化延后 |

---

## 汇总规则

- 不手工维护库存总数；本文件只统计实际原子任务。
- `Done` 必须附代码、测试或运行证据。
- Reject/Defer 只保留在 Owner RFC，不伪装成实施任务。
- ROADMAP priority、RFC lifecycle 与本文件任务状态必须同批更新。
