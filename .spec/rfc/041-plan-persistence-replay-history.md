# RFC-2026-041: Plan & ExecuteResult Persistence — Replay & History Viewer

---

作者: albert.li/AI
创建时间: 2026-07-31
状态: Approved
父 RFC: [RFC-2026-038](./038-sync-execution-verify-restore-replan.md)（umbrella）

修改历史:

- 2026-07-31: 初稿
- 2026-07-31: `ReplayPlan` 参数顺序；`SyncPlanModal` 只读模式；033↔replay 关系；风险表补全
- 2026-08-08: 批准（albert.li）；交付标准见 ROADMAP 提案中条目与 TASK_TRACKING RFC-041 区段

---

承接: RFC-2026-039/040 均依赖本 RFC 定义的持久化机制（`ExecuteResult` 落盘 + `Plan.ID` 查询）

**前置 gate：** RFC-034 Gap 4（Wails spawn `sync run`/`replay`）。持久化 hook 接在 CLI 路径；Wails 直调 `internal/sync` 时本 RFC 对 UI 无效。

## 1. 摘要

`Plan`/`ExecuteResult` 目前都是纯内存对象，一次 `BuildPlan`/`Execute` 调用后即消失——无法在执行完成后重放同一个 plan，也无法列出/查看历史记录。本 RFC 是 RFC-038 三个子 RFC 里的地基：RFC-039（verify）产出的 `RestoreEntry` 清单、RFC-040（cleanup/restore）按 `Plan.ID` 查询的能力，都依赖本 RFC 定义的持久化层。

自动落盘挂在 **CLI** `sync run` / `sync replay`（及 library `Execute` 被 CLI 调用时），不挂在 Wails 直调路径。

## 2. 现状核实

- `Plan`/`PlanItem`（`types.go:29-47`）：无 ID、无时间戳，`BuildPlan` 调用一次用一次，不落盘
- `ExecuteResult`（`types.go:50-55`）：同样纯内存，`Execute` 返回后即弃
- RFC-012 §3.7 已定 `.commando/` 子目录存放约定（`sync.commando_db`/`SyncProfile` 同级），本 RFC 的持久化路径沿用这个既有治理，不新开一套规则

## 3. 数据结构

```go
// types.go 扩展
type Plan struct {
    ID         string     `json:"id"`         // 新增：uuid，BuildPlan 时生成
    CreatedAt  time.Time  `json:"createdAt"`  // 新增
    LeftRoot   string     `json:"leftRoot"`
    RightRoot  string     `json:"rightRoot"`
    Direction  Direction  `json:"direction"`
    Items      []PlanItem `json:"items"`
    Conflicts  int        `json:"conflicts"`
    ToCopy     int        `json:"toCopy"`
    ToDelete   int        `json:"toDelete"`
    ToSkip     int        `json:"toSkip"`
}

// PlanItem 补充（2026-08-01，空目录同步修复同批加入）：
//   IsDir bool `json:"isDir,omitempty"` — 标记该项对应目录而非文件。
//   ActionCopy 且 IsDir=true 时 executor 走 os.MkdirAll，不走 copyFileAtomic。
//   影响：本 RFC 的持久化/重放对 IsDir 字段透传即可，无需特殊处理
//   （replay 遇到 IsDir 目录项仍按 ActionCopy 走 mkdir 路径）。

type ExecuteResult struct {
    PlanID     string         `json:"planId"`
    Copied     int            `json:"copied"`
    Skipped    int            `json:"skipped"`
    Deleted    int            `json:"deleted"`
    Errors     []string       `json:"errors"`
    Restorable []RestoreEntry `json:"restorable"` // RFC-039 产出
    DoneItems  []string       `json:"doneItems"`
    // 每项成功 copy/delete 时记录，供 Replay stale-source 检测（size/mtime 或 content hash）
    ItemSnapshots map[string]ItemSnapshot `json:"itemSnapshots,omitempty"`
}

type ItemSnapshot struct {
    SourcePath string `json:"sourcePath"`
    Size       int64  `json:"size"`
    ModTimeUnix int64 `json:"modTimeUnix"`
    ContentHash string `json:"contentHash,omitempty"` // Content/Verify 模式
}
```

`DoneItems` 记录"确认成功"的项（正向清单，不是失败清单）——重放时排除它，剩下的（失败+未跑到）全部重跑，语义比维护"失败清单"更简单，不会漏记 crash/取消导致的未执行项。

## 4. 存储

```
{leftRoot}/.commando/plans/{id}.json           # Plan
{leftRoot}/.commando/plans/{id}.result.json    # ExecuteResult（可能不存在——未执行过的 plan）
```

```go
// internal/sync/planstore.go（新增）
func SavePlan(leftRoot string, plan *Plan) error
func LoadPlan(leftRoot, id string) (*Plan, error)
func ListPlans(leftRoot string) ([]PlanSummary, error)

func SaveExecuteResult(leftRoot string, result *ExecuteResult) error
func LoadExecuteResult(leftRoot, planID string) (*ExecuteResult, error)

type PlanSummary struct {
    ID        string    `json:"id"`
    CreatedAt time.Time `json:"createdAt"`
    LeftRoot  string    `json:"leftRoot"`
    RightRoot string    `json:"rightRoot"`
    ToCopy    int       `json:"toCopy"`
    ToDelete  int       `json:"toDelete"`
    Conflicts int       `json:"conflicts"`
    Executed  bool      `json:"executed"` // 对应 .result.json 是否存在
}
```

**自动落盘，调用方不用额外操心：** `BuildPlan` 生成 `Plan` 后自动 `SavePlan`；`Execute` 跑完自动 `SaveExecuteResult`——跟现有 `executor.go:59-63`"每次 copy 后 saveProgress"是同一种既有模式，不新造范式。

## 5. Replay 语义

Plan 保存的是**执行时刻的 diff 快照**。重放同一个 `Plan.ID`：

- 加载上次的 `ExecuteResult`（若存在），用 `DoneItems` 过滤掉已确认成功的项，**只重跑剩下的**——不是整个 `Items[]` 从头再来
- 无持久化 `ExecuteResult`（首次执行）→ 全量执行
- `PlanItem.Source` 在重放时已不存在 → 标记失败计入 `Errors`，不中止整体重放（`ErrorMode` 仍然生效）

```go
// 参数顺序与同包 LoadPlan(leftRoot, planID) 一致：leftRoot 在前
func ReplayPlan(ctx context.Context, leftRoot, planID string, opts Options, progress ProgressFn) (*ExecuteResult, error) {
    plan, err := LoadPlan(leftRoot, planID)
    if err != nil {
        return nil, err
    }

    remaining := plan.Items
    if prev, loadErr := LoadExecuteResult(leftRoot, planID); loadErr == nil {
        done := make(map[string]bool, len(prev.DoneItems))
        for _, rel := range prev.DoneItems {
            done[rel] = true
        }
        remaining = filterUndone(plan.Items, done)
    } else if !os.IsNotExist(loadErr) {
        return nil, loadErr
    }

    replayPlan := *plan
    replayPlan.Items = remaining
    return Execute(ctx, &replayPlan, opts, progress)
}
```

**重放前校验：** 对 `remaining` 每项，将当前 `PlanItem.Source` 与 `ItemSnapshots[rel]` 比较；不一致 → `stale-source` 错误，**不覆盖**目标。`ErrorMode` 决定是否中止。

## 6. CLI（唯一入口）

```
commando sync replay --plan-id <id> --left <leftRoot>
commando sync list-plans --left <leftRoot> [--json]
```

```
$ commando sync list-plans --left /a --json
[{"id":"a1b2c3d4","createdAt":"...","toCopy":12,"toDelete":3,"conflicts":0,"executed":true}, ...]
```

`list-plans` 一次性输出 JSON 数组，非 NDJSON——查询已落盘文件列表，量小且非流式过程，跟 `plan`/`run` 的进度契约不是同一类操作。

## 7. Wails 桥接（CLI-first）

| CLI 命令                                | Wails 方法                                 |
| --------------------------------------- | ------------------------------------------ |
| `sync list-plans --left <L> --json`     | `SyncService.ListPlans(leftRoot)`          |
| `sync replay --plan-id <id> --left <L>` | `SyncService.ReplayPlan(leftRoot, planID)` |

均为 spawn 对应 CLI 命令解析 stdout，不直调 `internal/sync`。

## 8. UI

| 文件                                                   | 变更                                         |
| ------------------------------------------------------ | -------------------------------------------- |
| `packages/ui/src/components/sync/PlanHistoryPanel.tsx` | 新建：列出 `listPlans`；点击行加载 plan      |
| `packages/ui/src/components/sync/SyncPlanModal.tsx`    | 新增 `mode: "confirm" \| "view"`（**必做**） |
| `packages/ui/src/services/syncApiService.ts`           | `listPlans`/`loadPlan`/`replayPlan`          |
| `apps/desktop/services/sync.go`                        | spawn CLI（RFC-034 Gap 4）                   |

**`SyncPlanModal` 复用规则（挡 #3 误触 sync）：**

| `mode`      | 行为                                                                                  |
| ----------- | ------------------------------------------------------------------------------------- |
| `"confirm"` | 现有行为：`onConfirm` → `runSync()`；主同步流程                                       |
| `"view"`    | **只读**：无 Confirm 按钮；仅 `关闭`；可选 `从此 Plan 重放` 二次确认后调 `replayPlan` |

**禁止：** 历史查看传 `onConfirm={runSync}` 或 noop——`actionable.length > 0` 时 Confirm 仍可点，会真实执行同步。

**入口：** SyncToolbar「历史 Plan」按钮。

## 9. 测试计划

| 测试名                                                    | 验证什么                                                  |
| --------------------------------------------------------- | --------------------------------------------------------- |
| `TestSavePlan_LoadPlan_RoundTrip`                         | 存取后字段（含 ID/CreatedAt）完全一致                     |
| `TestBuildPlan_AutoSavesPlan`                             | `BuildPlan` 调用后 `.commando/plans/{id}.json` 自动落盘   |
| `TestExecute_AutoSavesExecuteResult`                      | `Execute` 跑完后 `.result.json` 自动落盘，含 `DoneItems`  |
| `TestReplayPlan_FirstRun_ExecutesAllItems`                | 无持久化结果时全量执行                                    |
| `TestReplayPlan_SecondRun_SkipsDoneItems`                 | 有持久化结果时 `DoneItems` 里的项被过滤                   |
| `TestReplayPlan_SkipsAlreadyGoneSource`                   | 重放时 source 已不存在的项标记失败但不中止                |
| `TestReplayPlan_UnknownPlanID_ReturnsError`               | plan 不存在时返回错误                                     |
| `TestReplayPlan_TargetModifiedSinceLastRun_SkipsOrErrors` | 源文件在两次执行间被外部修改 → `stale-source`，不盲目覆盖 |
| `TestListPlans_OrderedByCreatedAtDesc`                    | 展示顺序：最新的在前                                      |
| `TestListPlans_ExecutedFlagReflectsResultFile`            | `Executed` 准确反映 `.result.json` 是否存在               |
| `TestListPlans_EmptyDir_ReturnsEmptySlice`                | 没有任何 plan 时返回空切片不报错                          |
| `TestPlanHistoryPanel_RendersListAndOpensViewMode`        | 点击行后以 `mode=view` 打开 `SyncPlanModal`               |
| `TestSyncPlanModal_ViewMode_NoConfirmButton`              | `mode=view` 时无 Confirm，历史 plan 不可误触 sync         |

## 10. 风险

| 风险                                      | 缓解                                                                                   |
| ----------------------------------------- | -------------------------------------------------------------------------------------- |
| `.commando/plans/` 目录无限增长           | YAGNI；用户手动管理                                                                    |
| 重放遇到 sync root 已变化                 | `Plan.LeftRoot`/`RightRoot` 绝对路径；不存在则报错                                     |
| 并发写入 plans 目录                       | 单用户单进程；后续再加锁                                                               |
| `Plan`/`ExecuteResult` 新 JSON 字段       | 更新 `TestPlanCmd_NoProgressFlag_UnchangedOutput` 等 pinned 输出测；或测只断言子集字段 |
| **RFC-033 `--resume` vs 本 RFC `replay`** | 见 §11；文档化互斥与优先级                                                             |

## 11. 与 RFC-033 `--resume` 的关系

两者都是「续跑未完成同步」，**机制不同，不可混为一谈：**

| 机制               | 存储                                               | 键                         | 适用场景                                |
| ------------------ | -------------------------------------------------- | -------------------------- | --------------------------------------- |
| RFC-033 `--resume` | SQLite `{leftRoot}/.commando/sync.db`              | `job_id` + `relative_path` | **同一** `sync run` 会话崩溃/取消后续跑 |
| RFC-041 `replay`   | JSON `{leftRoot}/.commando/plans/{id}.result.json` | `Plan.ID` + `DoneItems`    | **跨会话**重放已保存 plan 的未完成项    |

**规则：**

- `sync run --resume` 与 `sync replay --plan-id` **互斥**（CLI 同时传则报错）。
- 崩溃后：若当时有 `Plan.ID` 且已落盘 → 优先 `replay`；否则 `--resume`（仅当 job_id 匹配且 progress 表有记录）。
- Wails UI 默认走 `replay`（041）；`--resume` 保留给纯 CLI 幂等重跑。

## 12. 待定

- `list-plans`/`PlanHistoryPanel` 是否要支持跨多个 sync root 聚合查看，还是只看当前选中的 leftRoot——本 RFC 默认只看当前 leftRoot，跨 root 聚合是更大的功能，不在此处处理
- Plan 里 `RightRoot` 若在多次执行之间发生变化（用户改了同步目标），`ListPlans` 展示的历史记录如何标注失效——本 RFC 不处理，留待后续迭代

## 13. 评审记录（2026-08-08，Linus/Kent/Martin 三人评审）

**结论**：3× APPROVE（Linus/Kent 带注记），无 blocking 项。

### 注记（已并入实施要求，非阻塞）

1. **§2 行号漂移**：`types.go` Plan/PlanItem 现为 33-54（非 29-47）、`ExecuteResult` 57-64（非 50-55）、executor saveProgress 现为 74-79（非 59-63）。已按现状文本更新——实施时以符号名定位，不以行号。
2. **`ExecuteResult` 缺字段（必改）**：现状 struct（types.go:57-64）无 `DoneItems`/`ItemSnapshots`。§3 新增字段必须落地，且每个成功路径 append——需要 `TestExecute_ResultCarriesDoneItemsAndSnapshots` 断言"部分失败时 DoneItems 长度 == 实际成功项"，防止 042 double-count 类 bug 以静态形式回归。
3. **stale-source 缺失快照降级规则**：`ItemSnapshots[rel]` 缺失（旧 result 文件被删/手改 JSON）→ 降级为"重跑该项，不阻塞"，不得因缺快照而中止。补测试 `TestReplayPlan_MissingSnapshot_RunsItemNotBlocks`。
4. **pinned 输出测试更新（必做）**：Plan/ExecuteResult 新增 JSON 字段会破坏 `TestPlanCmd_NoProgressFlag_UnchangedOutput` 等 pinned 测——从"风险"升级为任务项，见 TASK_TRACKING PLAN-05。
5. **replay↔resume 互斥需测试**：`sync replay --plan-id` + `--resume` 同传必须报错——补 `TestCLI_ReplayResume_MutuallyExclusive`。
6. **view 模式 UI 守则**：`SyncPlanModal` 现始终渲染 Confirm（143-152）——view 模式必须**隐藏**（非 disable）Confirm，且拒绝 `onConfirm={runSync}` 接线；补 UI 负向断言。

---

**状态**: Approved
**最后更新**: 2026-08-08 (三人评审注记归档 §13)
