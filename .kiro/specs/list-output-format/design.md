# Technical Design

## Overview
本機能は `sava list [date]`（stat無しの日次表示）の1行ごとの描画フォーマットを、GFM（GitHub Flavored Markdown）チェックリスト構文＋一貫したhash表記に再設計する。

**Purpose**: `list` の出力をMarkdownノートへそのまま貼り付けられるようにし、同時にpipe処理からhashを確実に抜き取れるようにする。
**Users**: `sava` を日報補助・作業メモとして使う全ユーザー。
**Impact**: `sava list [date]`（`-s`/`-a` を指定しない場合）のデフォルト行フォーマットが変わる破壊的変更。`-s/--stat` と `-a/--all`（stat無し）の出力は変更されない。

### Goals
- 各行をGFMチェックリスト構文（未着手 `[ ]` / 着手中 `[ ] \`in-progress\`` / 完了 `[x]` / メモはチェックボックス無し）で描画する
- hashを常に行末の `` (`hash`) `` に統一し、pipe処理から一貫して抜き取れるようにする
- 複数行contentはデフォルトで先頭行のみ表示し、`--full-text` で全文表示に切り替えられるようにする

### Non-Goals
- `-s/--stat` の集計内容・出力フォーマットの変更
- `-a/--all`（stat無し、ファイル名一覧）の出力変更
- `list-task-visibility` が定めた可視性ルール（デフォルト非表示・`--full`・`--task`）や表示順序の変更
- タグのフィルタ判定ロジックの変更（表示位置は既存通り維持するのみ）

## Boundary Commitments

### This Spec Owns
- `internal/view.Timeline` の行描画フォーマット（チェックリスト構文・hash表記・content要約/全文切り替え）
- `ListOptions.FullText` フィールドと `--full-text` フラグ

### Out of Boundary
- `-s/--stat` のロジック・出力（`writeFileStat`, `view.FileStat`）
- `-a/--all`（stat無し）のファイル名一覧出力（`view.ListItem`）
- `list-task-visibility` が実装した、表示対象アイテムの選別・グルーピング順序（`log.SplitByStatus`, `listOneDay` の `Full`/`TasksOnly`/`Tags` 処理）— 本specはその結果（最終的に表示される順序済みアイテム列）を受け取って描画するだけ
- タグのフィルタ判定ロジック（`log.FilterByTags`）

### Allowed Dependencies
- `internal/model.Item`（`Status()`, `Content`, `Hash`, `Tags`）— 既存の読み取り専用メソッド/フィールドとして利用
- `internal/view` 既存の `formatTime`, `formatTags` — 変更せず再利用
- `list-task-visibility` が確立した `listOneDay` のアイテム組み立て契約（本specはその出力である `[]model.Item` を受け取るだけで、組み立てロジックには関与しない）

### Revalidation Triggers
- `model.Item.Status()` の判定ステータス種別が変わった場合（現在: memo/open/started/closed の4種）
- `view.Timeline` の呼び出し元が増えた場合（現在は `internal/command.listOneDay` の1箇所のみ）
- 将来、別specが再度 `list` の行フォーマットを変更する場合は、本specとの競合がないか確認が必要

## Architecture

### Existing Architecture Analysis
- `internal/view.Timeline` が「渡されたアイテム列を、渡された順にそのまま描画する」という既存の責務を維持する。今回変更するのは「1アイテムをどの文字列に変換するか」という描画ロジックの内部実装のみで、呼び出し元とのインターフェース上の役割分担（`command`層が選別・順序を決め、`view`層が描画する）は変えない。
- `view.Timeline` の呼び出し元は `internal/command.listOneDay`（非stat分岐）の1箇所のみ（`research.md` で確認済み）。

### Architecture Integration
- **Selected pattern**: 既存のレイヤードCLI構成（`cmd` → `command` → `view`）を維持し、新規レイヤーは追加しない。
- **Domain/feature boundaries**: 「何を表示するか」（`list-task-visibility` の既存責務）と「どう描画するか」（本specの責務）の分離を維持する。
- **Existing patterns preserved**: `ListOptions` へのフィールド追加でフラグを表現するパターン（`Full`, `TasksOnly` と同様）。
- **New components rationale**: 新規関数は `view` パッケージ内の `checklistPrefix`（チェックリスト接頭辞の算出）と `firstLine`（先頭行抽出）のみ。いずれも既存の `marker` 相当の小さなヘルパーであり、新しい抽象レイヤーは増やさない。
- **Steering compliance**: `structure.md` の依存方向（`cmd` → `command` → `view`/`log`/...）を維持。

本変更はCLI→`ListOptions`フィールド→`view.Timeline`への単純なパラメータ伝播のみで、複数コンポーネントが絡む分岐ロジックも無いため、Mermaid図は省略する（Design Principlesの「Simple features: Basic component diagram or none」に従う）。

## Technology Stack

| Layer | Choice / Version | Role in Feature | Notes |
|-------|------------------|------------------|-------|
| CLI | Go 1.26 / spf13/cobra v1.10.2（既存） | `--full-text` フラグの登録 | 新規依存なし |
| 描画ロジック | Go標準ライブラリ `strings`（既存、`strings.Cut`使用） | 先頭行抽出 | 新規依存なし |

新規の外部ライブラリ・インフラ変更は無い。

## File Structure Plan

### Modified Files
- `internal/view/view.go` — `Timeline` のシグネチャに `fullText bool` を追加し、描画ロジックを新フォーマットに変更。既存の `marker` 関数を削除し、`checklistPrefix`（チェックリスト接頭辞算出）と `firstLine`（content先頭行抽出）を新設。`formatTags`/`formatTime` は変更しない。
- `internal/view/view_test.go` — 既存 `TestTimeline`/`TestTimelineEmpty` を新シグネチャ（3引数）に追従させ、新フォーマット（チェックリスト・hash表記・in-progressトークン・tags位置）を検証するテストケースを追加。先頭行要約・`--full-text`全文表示のテストも追加。
- `internal/command/command.go` — `ListOptions` に `FullText bool` を追加。`listOneDay` の非stat分岐で `view.Timeline(w, items, opts.FullText)` を呼ぶよう変更（第3引数追加）。`opts.Stat`/`List()`の`opts.All`分岐では`FullText`を一切参照しない（`list-task-visibility`の`Full`/`TasksOnly`と同じ非干渉パターン）。
- `internal/command/command_test.go` — 既存の `TestListToday_*` 系テストがcontent文字列の部分一致で検証しているため、新フォーマットでも大半はそのまま通る見込み。`--full-text` のエンドツーエンド動作（複数行contentが全文表示されること）を検証するテストを追加。
- `cmd/sava/commands.go` — `newListCommand` に `--full-text`（bool、短縮形なし）のフラグ登録を追加。
- `cmd/sava/root_test.go` — `--full-text` フラグがエラー無く受理され、`ListOptions.FullText` に反映されることを確認するテストを追加（既存の `TestListFullAndTaskFlags` と同様のパターン）。

新規ファイルは無い。`internal/log/`, `internal/model/`, `internal/logfile/` は変更しない。

## Requirements Traceability

| Requirement | Summary | Components | Interfaces | Flows |
|-------------|---------|------------|------------|-------|
| 1.1, 1.2, 1.3, 1.4 | チェックリスト構文（未着手/着手中/完了/メモ） | `view.checklistPrefix`, `view.Timeline` | `checklistPrefix(status model.Status) string` | CLI → ListOptions → listOneDay → Timeline |
| 2.1, 2.2 | hashの一貫した行末表記 | `view.Timeline` | フォーマット文字列内の `` (`%s`) `` | 同上 |
| 2.3 | タグはhashの後、既存形式 | `view.Timeline`, 既存 `formatTags`（変更なし） | `formatTags(tags []string) string` | 同上 |
| 3.1, 3.2 | 複数行contentの先頭行要約 | `view.firstLine`, `view.Timeline` | `firstLine(content string) string` | 同上 |
| 4.1, 4.2 | `--full-text` で全文表示 | `cmd.newListCommand`, `command.ListOptions.FullText`, `view.Timeline` | `Timeline(w, items, fullText bool)` | CLI → ListOptions → listOneDay → Timeline |
| 5.1, 5.2, 5.3 | `-s`/`-a` は不変 | `command.listOneDay`（stat分岐、変更なし）, `command.List`（all分岐、変更なし） | — | StatPath / AllPath（`FullText`参照なし） |
| 5.4 | 既存の可視性ルールは不変 | `command.listOneDay`（`Full`/`TasksOnly`/`Tags`処理、変更なし） | — | list-task-visibilityの既存フロー |

## Components and Interfaces

| Component | Domain/Layer | Intent | Req Coverage | Key Dependencies (P0/P1) | Contracts |
|-----------|---------------|--------|---------------|---------------------------|-----------|
| `view.Timeline`（変更） | 表示 (`internal/view`) | アイテム列をGFMチェックリスト構文で描画する | 1.1-1.4, 2.1-2.3, 3.1-3.2, 4.1-4.2 | `model.Item.Status`（P0） | Service |
| `view.checklistPrefix`（新規） | 表示 (`internal/view`) | ステータスに応じたチェックリスト接頭辞を算出する | 1.1-1.4 | `model.Status`（P0） | Service |
| `view.firstLine`（新規） | 表示 (`internal/view`) | contentの先頭行を抽出する | 3.1 | なし | Service |
| `command.listOneDay`（拡張） | ユースケース (`internal/command`) | `FullText` を `view.Timeline` に橋渡しする | 4.1, 4.2, 5.1-5.4 | `view.Timeline`（P0） | Service |
| `cmd.newListCommand`（拡張） | CLI (`cmd/sava`) | `--full-text` フラグを登録する | 4.1 | `command.ListOptions`（P0） | Service |

### 表示層 (`internal/view`)

#### Timeline（変更）

| Field | Detail |
|-------|--------|
| Intent | アイテム列を、GFMチェックリスト構文・一貫したhash表記で1行ずつ描画する |
| Requirements | 1.1, 1.2, 1.3, 1.4, 2.1, 2.2, 2.3, 3.1, 3.2, 4.1, 4.2 |

**Responsibilities & Constraints**
- 各アイテムの `model.Item.Status()` に応じて `checklistPrefix` でチェックリスト接頭辞を決める（未着手/着手中は `[ ]`、着手中のみ `` `in-progress` `` トークンを追加、完了は `[x]`、メモは接頭辞なし）。
- hashは常に `` (`%s`) ``（括弧＋バックティック）で描画する。この表記・位置はアイテム種別にかかわらず同一。
- `fullText` が `false` のとき、`firstLine` でcontentを先頭行のみに要約する。`true` のときはcontentをそのまま使う。
- タグは既存の `formatTags` をそのまま使い、hashトークンの後ろに続ける（変更なし）。
- 空の `items` に対しては既存通り `"There is no body..."` を出力する（変更なし）。

**Dependencies**
- Inbound: `internal/command.listOneDay`（P0、唯一の呼び出し元）
- Outbound: `internal/model.Item`（P0）

**Contracts**: Service [x] / API [ ] / Event [ ] / Batch [ ] / State [ ]

##### Service Interface
```go
// Timeline prints items (tasks and memos mixed) as a single list in
// the order given: a GFM checklist prefix (or a plain bullet for
// memos), the creation time, the content, and the hash/tags for
// reference. The hash is always wrapped as "(`hash`)" so it can be
// extracted from any line with the same pattern. When fullText is
// false, multi-line content is shown as its first line only, so one
// line always maps to one item; when true, content is shown in full.
func Timeline(w io.Writer, items []model.Item, fullText bool)

// checklistPrefix renders the leading bullet and, for tasks, GFM
// checklist syntax for an item's lifecycle status: "[ ]" for both
// open and started (GFM has no third checkbox state), with a
// literal "`in-progress`" token distinguishing started from open;
// "[x]" for closed; no checkbox at all for memos.
func checklistPrefix(status model.Status) string

// firstLine returns content's first line, or content itself if it
// has no newline.
func firstLine(content string) string
```
- Preconditions: `items` は `nil` でも可（空リスト扱い）
- Postconditions: 出力行数は、`fullText == false` の場合 `len(items)` と一致する（1行=1アイテム）。`fullText == true` の場合、content内の改行数に応じて行数が増えることがある
- Invariants: 各行の末尾付近に必ず `` (`[a-f0-9]{8}`) `` 形式のhashが1つだけ現れる

**Implementation Notes**
- Integration: `marker` 関数は本変更で不要になるため削除する（`research.md` で他の利用箇所が無いことを確認済み）。
- Validation: 追加のバリデーションは不要（既存 `Timeline` と同様、入力は常に有効な `[]model.Item`）。
- Risks: なし（既存パターンの直接的な拡張）。

### ユースケース層 (`internal/command`)

#### listOneDay（拡張）

| Field | Detail |
|-------|--------|
| Intent | `FullText` オプションを `view.Timeline` に橋渡しする |
| Requirements | 4.1, 4.2, 5.1, 5.2, 5.3, 5.4 |

**Responsibilities & Constraints**
- 非stat分岐でのみ `view.Timeline(w, items, opts.FullText)` を呼ぶ（第3引数を追加するのみ。アイテムの選別・順序ロジックは`list-task-visibility`のまま変更しない）。
- `opts.Stat` 分岐、`List()` の `opts.All` 分岐では `opts.FullText` を一切参照しない（`Full`/`TasksOnly`と同じ非干渉パターンを踏襲。5.1-5.3充足）。

**Dependencies**
- Inbound: `cmd.newListCommand`（P0）
- Outbound: `view.Timeline`（P0、シグネチャ変更に追従）

**Contracts**: Service [x] / API [ ] / Event [ ] / Batch [ ] / State [ ]

**Implementation Notes**
- Integration: `ListOptions` に `FullText bool` を追加するのみ。既存の `Full`/`TasksOnly`/`Tags` によるアイテム選別ロジックには触れない。
- Validation: `--all`/`--stat` との組み合わせに対する新規エラーは追加しない（`list-task-visibility`の前例と同じ方針）。
- Risks: なし。

### CLI層 (`cmd/sava`)

#### newListCommand（拡張）

| Field | Detail |
|-------|--------|
| Intent | `--full-text` フラグを登録し `command.ListOptions` に橋渡しする |
| Requirements | 4.1 |

既存の `cmd.Flags().BoolVar(&opts.TasksOnly, "task", false, ...)` と同じパターン（短縮形なし）で追加する:
```go
cmd.Flags().BoolVar(&opts.FullText, "full-text", false, "show full content instead of the first line only")
```
短縮形を持たない理由: `-f` は既に `--full` に割り当て済みであり、複数語フラグ（`--or`, `--task`）は本リポジトリの既存の慣習として短縮形を持たない。

## Testing Strategy

### Unit Tests（`internal/view/view_test.go`）
- `TestTimeline_OpenTaskChecklistFormat`: 未着手タスクが `- [ ] <time> <content> (\`hash\`)` で描画されることを検証（1.1, 2.1, 2.2）。
- `TestTimeline_StartedTaskInProgressToken`: 着手中タスクが `` - [ ] `in-progress` <time> <content> (`hash`) `` で描画されることを検証（1.2）。
- `TestTimeline_ClosedTaskChecklistFormat`: 完了タスクが `- [x] ...` で描画されることを検証（1.3）。
- `TestTimeline_MemoNoCheckbox`: メモがチェックボックス無しの `- <time> <content> (\`hash\`)` で描画されることを検証（1.4）。
- `TestTimeline_TagsAfterHash`: タグ付きアイテムで、タグがhashトークンの後ろに既存形式のまま現れることを検証（2.3）。
- `TestTimeline_MultilineContentShowsFirstLineByDefault`: 複数行contentを持つアイテムに対し `fullText=false` で呼び出したとき、出力が先頭行のみを含み、そのアイテムの出力が1行であることを検証（3.1, 3.2）。
- `TestTimeline_FullTextShowsAllLines`: 同じ複数行contentに対し `fullText=true` で呼び出したとき、全行が出力に含まれることを検証（4.1, 4.2）。
- `TestTimelineEmpty`（既存、変更なし）: 空リストで `"There is no body..."` が出ることを維持。

### Integration Tests（`internal/command/command_test.go`）
- `TestListToday_FullTextFlagShowsMultilineContent`: `sava list` を `FullText: true` で呼び、複数行メモ/タスクの全文が出力に含まれることを検証（4.1, 4.2のエンドツーエンド確認）。
- 既存の `TestListToday_*` 系テスト: 新フォーマットの下でも、content文字列の部分一致ベースの既存アサーションがそのまま通るか確認する（通らないものがあれば、この変更に合わせて期待値を更新する）。
- `TestListStatAndAll_UnaffectedByNewFlags` の拡張: `FullText: true` を加えても `-s`/`-a`（stat無し）の出力が変化しないことを検証（5.2, 5.3）。

### CLI Tests（`cmd/sava/root_test.go`）
- `--full-text` フラグがエラー無く受理され、複数行content全文表示という実際の挙動につながることを確認するテストを追加（既存 `TestListFullAndTaskFlags` と同様のエンドツーエンドパターン）。
