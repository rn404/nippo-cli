# Technical Design

## Overview
本機能は `sava list [date]`（stat無しの日次表示）に、完了済みタスクのデフォルト非表示、`--full`/`--task` フラグ、およびステータス別の固定表示順序を追加する。

**Purpose**: 日々のログを確認する最も頻度の高いユースケース（「今日やることを確認する」）において、完了済みタスクに埋もれず未完了タスクとメモをすばやく把握できるようにする。
**Users**: `sava` を日報補助・作業メモとして使う全ユーザーが、`sava list` 実行時に本機能の恩恵を受ける。
**Impact**: `sava list [date]`（`-s`/`-a` を指定しない場合）のデフォルト出力が変わる破壊的変更。`-s/--stat` と `-a/--all`（stat無し）の出力は変更されない。

### Goals
- フラグなしの `sava list [date]` で完了済みタスクを非表示にする
- `--full` で完了済みタスクを含む全件を表示できるようにする
- `--task` でメモを除いたタスクのみを表示できるようにする
- 表示アイテムを「完了済みタスク → 未完了タスク（未着手＋着手中） → メモ」の順に整列する

### Non-Goals
- `-s/--stat` の集計内容・出力フォーマットの変更
- `-a/--all`（stat無し、ファイル名一覧）の出力変更
- グループごとの見出し表示（今回は並び順のみで表現する）
- `--tag`/`--or` フィルタ自体のロジック変更
- 完了済み/未完了以外の新しいステータス軸（例: 着手中のみ表示）の追加

## Boundary Commitments

### This Spec Owns
- `sava list [date]`（stat無し）における表示対象アイテムの可視性ルール（デフォルト非表示・`--full`・`--task`）
- 表示アイテムのステータス別グルーピング順序（closed → open/started → memo）
- `internal/log.SplitByStatus` という新しいドメイン関数
- `ListOptions` への `Full`/`TasksOnly` フィールド追加と、対応する `--full`/`--task` フラグ

### Out of Boundary
- `-s/--stat` のロジック・出力（`writeFileStat`, `view.FileStat`）— 変更しない
- `-a/--all`（stat無し）のファイル名一覧出力（`view.ListItem`）— 変更しない
- `--tag`/`--or` フィルタの判定ロジック（`log.FilterByTags`）— 既存のまま再利用するのみ
- `start`/`end`/`tag`/`del`/`diff` など他コマンドの仕様
- タスクの状態遷移（`Closed`/`StartedAt` の更新ロジック）自体

### Allowed Dependencies
- `internal/model.Item`（`IsTask`, `IsClosed`, `IsStarted`, `Status`）— 既存の読み取り専用メソッドとして利用
- `internal/log.byCreatedAt` — パッケージ内の既存ソートヘルパーを再利用
- `internal/log.FilterByTags` — 既存関数をそのまま呼び出し、可視性フィルタ後の結果に適用

### Revalidation Triggers
- `model.Item.Status()` の判定優先順位（Closed > Started > Open）が変わった場合
- `log.Split`/`byCreatedAt` のソート保証（作成日時昇順）が変わった場合
- `ListOptions` の構造、または `list` コマンドのフラグ体系が変わった場合
- Item に新しい種別・ステータス（例: 現在の memo/open/started/closed 以外）が追加された場合

## Architecture

### Existing Architecture Analysis
- `cmd/sava/commands.go` の `newListCommand` がフラグをパースし `command.List(...)` に委譲する既存パターンを維持する。
- `internal/command.listOneDay` が表示対象アイテムの選別・整列を行い、`internal/view.Timeline` に渡す既存の責務分担を維持する（`view.Timeline` は「渡された順に描画するだけ」の責務を変えない）。
- `internal/log` は既に `Split`（tasks/memos の2分割＋各バケット安定ソート）を持っており、本機能はこれと対になる3分割版を追加する形で既存レイヤリングに従う。

### Architecture Integration
- **Selected pattern**: 既存のレイヤードCLI構成（`cmd` → `command` → `log`/`view`）をそのまま維持し、新規レイヤーは追加しない。
- **Domain/feature boundaries**: アイテムの「分類・整列」は `internal/log`（ドメイン層）、「どのグループを含めるかの判断」は `internal/command`（ユースケース層）、「描画」は `internal/view`（表示層）に分離する。
- **Existing patterns preserved**: `ListOptions` へのフィールド追加でフラグを表現するパターン（`Tags`, `Or`, `Stat`, `All` と同様）、`internal/log` に分類関数を置くパターン（`Split` と同様）。
- **New components rationale**: `log.SplitByStatus` のみが新規関数。3グループの可視性判断はロジックが小さいため `command.listOneDay` 内にインラインで実装し、新しい抽象を増やさない（Simplification原則）。
- **Steering compliance**: `structure.md` の「依存方向は一方向: cmd → command → {logfile, log, index} → model」を維持。`view` は `command` から呼ばれる横断レイヤーという位置づけも変更しない。

### Item Visibility Decision Flow

```mermaid
flowchart TD
    Start[list date invoked] --> AllCheck{opts.All?}
    AllCheck -->|true| AllPath[List() all-files branch. Full and TasksOnly not read]
    AllCheck -->|false| StatCheck{opts.Stat?}
    StatCheck -->|true| StatPath[writeFileStat. Full and TasksOnly not read]
    StatCheck -->|false| Split[log.SplitByStatus returns closed, open, memos]
    Split --> FullCheck{opts.Full?}
    FullCheck -->|true| AddClosed[append closed]
    FullCheck -->|false| SkipClosed[skip closed]
    AddClosed --> AddOpen[append open]
    SkipClosed --> AddOpen
    AddOpen --> TaskCheck{opts.TasksOnly?}
    TaskCheck -->|true| SkipMemos[skip memos]
    TaskCheck -->|false| AddMemos[append memos]
    SkipMemos --> TagCheck{len opts.Tags greater than 0?}
    AddMemos --> TagCheck
    TagCheck -->|true| ApplyTag[log.FilterByTags]
    TagCheck -->|false| Render[view.Timeline]
    ApplyTag --> Render
```

**Key Decisions**:
- `opts.All`/`opts.Stat` の分岐は `opts.Full`/`opts.TasksOnly` を一切参照しない（Design Decision、`research.md` 参照）。これにより `-a`/`-s` の既存出力は完全に不変となる。
- `--tag` フィルタは可視性判定（Full/TasksOnly によるグループ選別）の**後**に適用する。`FilterByTags` は入力順序を保持したまま絞り込むため、グルーピング順序（closed→open→memo）は絞り込み後も保たれる。

## Technology Stack

| Layer | Choice / Version | Role in Feature | Notes |
|-------|------------------|------------------|-------|
| CLI | Go 1.26 / spf13/cobra v1.10.2（既存） | `--full`/`--task` フラグの登録 | 新規依存なし |
| ドメインロジック | Go標準ライブラリ `sort`（既存） | `SplitByStatus` のバケット内安定ソート | 既存 `byCreatedAt` を再利用 |

新規の外部ライブラリ・インフラ変更は無い。

## File Structure Plan

### Modified Files
- `cmd/sava/commands.go` — `newListCommand` に `-f/--full`（bool）と `--task`（bool、短縮形なし）のフラグ登録を追加。
- `internal/command/command.go` — `ListOptions` に `Full bool` / `TasksOnly bool` を追加。`listOneDay` の非stat分岐で `log.SplitByStatus` を呼び出し、`Full`/`TasksOnly` に応じて表示アイテムを組み立ててから既存の `FilterByTags` を適用するよう変更。`List`/`listOneDay` の `opts.Stat`/`opts.All` 分岐は変更しない（`Full`/`TasksOnly` を参照しないことを維持）。
- `internal/log/log.go` — `Split` の直後に `SplitByStatus` を新設。`byCreatedAt` を再利用し、closed/open+started/memos の3バケットに振り分けて各々安定ソートする。
- `internal/command/command_test.go` — `TestListToday` 系に、デフォルト非表示・`--full`・`--task`・組み合わせ・`--all`との併用（無視されること）を検証するケースを追加。
- `internal/log/log_test.go` — `TestSplitByStatus` を追加し、3分割とグループ内昇順ソートを検証。
- `README.md` — Usageセクションの `sava list` 説明に `--full`/`--task` を追記（ドキュメント更新、動作変更ではない）。

新規ファイルは無い。`internal/view/view.go` は変更しない（`Timeline` は渡された順に描画するだけの既存責務を維持）。

## Requirements Traceability

| Requirement | Summary | Components | Interfaces | Flows |
|-------------|---------|------------|------------|-------|
| 1.1, 1.2, 1.3 | デフォルトで完了済みタスクを非表示 | `command.listOneDay`, `log.SplitByStatus` | `SplitByStatus(l) (closed, open, memos []model.Item)` | Item Visibility Decision Flow |
| 2.1, 2.2 | `--full` で完了済みタスクを表示 | `cmd.newListCommand`, `command.listOneDay` | `ListOptions.Full` | Item Visibility Decision Flow |
| 3.1, 3.2, 3.3 | `--task` でメモを除外 | `cmd.newListCommand`, `command.listOneDay` | `ListOptions.TasksOnly` | Item Visibility Decision Flow |
| 4.1, 4.2, 4.3 | ステータス別の固定順序・グループ内昇順・見出しなし | `log.SplitByStatus`, `command.listOneDay`, `view.Timeline`（変更なし） | `SplitByStatus` の戻り値順序 | Item Visibility Decision Flow |
| 5.1 | `--tag` との併用はAND条件 | `command.listOneDay`, `log.FilterByTags`（既存） | 既存 `FilterByTags(items, tags, or)` | Item Visibility Decision Flow |
| 5.2 | `-s/--stat` は不変 | `command.listOneDay`（stat分岐、変更なし） | — | Item Visibility Decision Flow（StatPath） |
| 5.3 | `-a/--all`（stat無し）は不変 | `command.List`（all分岐、変更なし） | — | Item Visibility Decision Flow（AllPath） |

## Components and Interfaces

| Component | Domain/Layer | Intent | Req Coverage | Key Dependencies (P0/P1) | Contracts |
|-----------|---------------|--------|---------------|---------------------------|-----------|
| `log.SplitByStatus` | ドメイン (`internal/log`) | アイテムをclosed/open+started/memosの3グループに分類・整列 | 1.1, 3.2, 4.1, 4.2 | `model.Item.IsTask`/`IsClosed`（P0） | Service |
| `command.listOneDay`（拡張） | ユースケース (`internal/command`) | Full/TasksOnly/Tagsに応じて表示アイテムを組み立てる | 1.1-1.3, 2.1-2.2, 3.1-3.3, 5.1 | `log.SplitByStatus`（P0）, `log.FilterByTags`（P0） | Service |
| `cmd.newListCommand`（拡張） | CLI (`cmd/sava`) | `-f/--full`, `--task` フラグを登録し `ListOptions` に橋渡し | 2.1, 3.1 | `command.ListOptions`（P0） | Service |

### ドメイン層 (`internal/log`)

#### SplitByStatus

| Field | Detail |
|-------|--------|
| Intent | ログのアイテムを完了済みタスク・未完了タスク（未着手＋着手中）・メモの3グループに分類し、各グループ内を作成日時昇順に整列する |
| Requirements | 1.1, 3.2, 4.1, 4.2 |

**Responsibilities & Constraints**
- 入力 `model.Log` の `Items` を1パスで3バケットに振り分ける（`model.Item.IsTask()`/`IsClosed()` を使用）。
- 各バケットを既存の `byCreatedAt` で `sort.SliceStable` する（`Split` と同一パターン）。
- 元の `l.Items` を変更しない（`Split`/`Timeline` と同様、コピー・新規スライスに対して操作する）。

**Dependencies**
- Inbound: `internal/command.listOneDay`（P0）
- Outbound: なし
- External: なし

**Contracts**: Service [x] / API [ ] / Event [ ] / Batch [ ] / State [ ]

##### Service Interface
```go
// SplitByStatus separates the log items into three status groups —
// closed tasks, non-closed tasks (open or started), and memos — each
// sorted by creation time in ascending order.
func SplitByStatus(l model.Log) (closedTasks, openTasks, memos []model.Item)
```
- Preconditions: `l.Items` は `model.Item` のゼロ値以上（空スライス可）
- Postconditions: 戻り値3つの合計要素数は `len(l.Items)` と一致する。各戻り値内は `CreatedAt` 昇順（同時刻は安定ソートにより元の順序を維持）
- Invariants: `closedTasks` の全要素は `IsClosed() == true`、`openTasks` の全要素は `IsTask() == true && IsClosed() == false`、`memos` の全要素は `IsTask() == false`

**Implementation Notes**
- Integration: `command.listOneDay` が `Full`/`TasksOnly` に応じて3つのスライスを条件付きで `append` し、最後に既存の `log.FilterByTags` を通す。
- Validation: 追加のバリデーションは不要（既存 `Split` と同様、入力は常に有効な `model.Log`）。
- Risks: なし（既存パターンの直接的な拡張）。

### ユースケース層 (`internal/command`)

#### listOneDay（拡張）

| Field | Detail |
|-------|--------|
| Intent | 日次表示の対象アイテムを、可視性フラグとタグフィルタに基づいて組み立てる |
| Requirements | 1.1, 1.2, 1.3, 2.1, 2.2, 3.1, 3.2, 3.3, 5.1 |

**Responsibilities & Constraints**
- 非stat分岐でのみ `log.SplitByStatus` を呼び出す（stat分岐は変更しない = 5.2充足）。
- `opts.Full` が false のとき `closedTasks` を含めない。
- `opts.TasksOnly` が true のとき `memos` を含めない。
- 組み立てた順序（closed→open→memo、それぞれ内部昇順）を変えずに `log.FilterByTags` を適用する（5.1充足、順序保持は `FilterByTags` の既存実装が入力順を保つことに依存）。
- 全アイテムが除外された場合、既存の「There is no body...」相当のメッセージ経路（`view.Timeline` の空リスト処理）をそのまま使う（1.3充足）。

**Dependencies**
- Inbound: `cmd.newListCommand`（P0）
- Outbound: `log.SplitByStatus`（P0）, `log.FilterByTags`（P0）, `view.Timeline`（P0、既存・変更なし）

**Contracts**: Service [x] / API [ ] / Event [ ] / Batch [ ] / State [ ]

##### Service Interface
```go
// (既存シグネチャは変更しない。ListOptions のフィールド追加のみ)
func List(w io.Writer, r io.Reader, dir string, opts ListOptions) error
```

**Implementation Notes**
- Integration: `List()` の `opts.All` 分岐、`listOneDay` の `opts.Stat` 分岐は `opts.Full`/`opts.TasksOnly` を一切参照しない（`research.md` の Design Decision 参照。5.2, 5.3充足）。
- Validation: `--all`/`--stat` と `--full`/`--task` の組み合わせに対する新規エラーは追加しない（既存の `--tag`+`--all` エラーとは非対称だが、承認済み要件の文言を優先）。
- Risks: ユーザーが `-a --full` のように誤って組み合わせても無反応（サイレントに無視）になる点は、`research.md` のRisksに記録済み。将来のフォローアップ候補。

### CLI層 (`cmd/sava`)

#### newListCommand（拡張）

| Field | Detail |
|-------|--------|
| Intent | `-f/--full` と `--task` フラグを登録し `command.ListOptions` に橋渡しする |
| Requirements | 2.1, 3.1 |

既存の `cmd.Flags().BoolVarP(&opts.All, "all", "a", ...)` と同じパターンで以下を追加する（Contracts: Service のみ、詳細は既存コードと同型のためブロック省略）:
```go
cmd.Flags().BoolVarP(&opts.Full, "full", "f", false, "include completed tasks")
cmd.Flags().BoolVar(&opts.TasksOnly, "task", false, "show only tasks (exclude memos)")
```

## Error Handling

本機能は新しいエラーパスを導入しない。`--all`+`--tag` の既存エラー（`"tag filter cannot be combined with --all"`）は変更しない。`--full`/`--task` を `--all`/`--stat` と併用した場合は、エラーにせずサイレントに無視する（`research.md` Design Decision参照、5.2・5.3充足）。

## Testing Strategy

### Unit Tests（`internal/log/log_test.go`）
- `TestSplitByStatus_PartitionsByStatus`: closed/open(未着手)/started(着手中)/memoが混在するLogを渡し、closedTasksにclosedのみ、openTasksに未着手・着手中の両方、memosにメモのみが入ることを検証（4.1関連）。
- `TestSplitByStatus_OrdersByCreatedAtAscending`: 各グループ内で作成日時が昇順になっていること、同時刻アイテムは元の順序が保たれること（安定ソート）を検証（4.2関連）。
- `TestSplitByStatus_EmptyLog`: 空の`model.Log`を渡したとき3つとも空スライスを返すことを検証。

### Integration Tests（`internal/command/command_test.go`）
- `TestListToday_HidesClosedTasksByDefault`: closedタスクを含む当日ログに対し `--full` なしで `List` を呼び、出力にclosedタスクが含まれないことを検証（1.1, 1.2）。
- `TestListToday_AllClosedShowsEmptyMessage`: 当日ログが全てclosedタスクのとき、`--full`なしでは既存の空メッセージが出ることを検証（1.3）。
- `TestListToday_FullShowsClosedTasks`: `--full` 指定時にclosedタスクを含む全件が「closed→open→memo」の順で出力されることを検証（2.1, 2.2, 4.1）。
- `TestListToday_TaskOnlyExcludesMemos`: `--task` 指定時にメモが出力されないこと、`--full`なしではclosedタスクも出ないことを検証（3.1, 3.2）。
- `TestListToday_TaskAndFullShowsAllTasksNoMemos`: `--task --full` でclosed+open両方のタスクが出て、メモは出ないことを検証（3.3）。
- `TestListToday_TagFilterCombinesWithVisibility`: `--tag` と `--full`/`--task` を組み合わせたとき、AND条件で絞り込まれ、かつグルーピング順序が保たれることを検証（5.1）。
- `TestListStatAndAll_UnaffectedByNewFlags`: 既存の `TestListStatAndAll` に `--full`/`--task` を渡すケースを追加し、`-s`/`-a` の出力が変化しないことを検証（5.2, 5.3）。
