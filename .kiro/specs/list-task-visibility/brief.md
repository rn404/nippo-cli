# Brief: list-task-visibility

## Problem
日々 `sava list` で当日のタイムラインを見るとき、完了済みタスクとメモが未完了タスクと時系列でフラットに混在表示される。完了済みタスクが増えるほど「今なにをやるべきか」が埋もれ、最も頻度の高い「今日やることを確認する」というユースケースでの見やすさが損なわれている。

## Current State
- `internal/command.listOneDay` は `log.Timeline(file.Body)` で全アイテム（Task/Memo、完了未完了問わず）を作成日時昇順のフラットな一覧として表示する。
- `-t/--tag` によるタグ絞り込みは存在するが、アイテム種別（Task/Memo）や完了状態による絞り込み・グルーピングは存在しない。
- `-s/--stat` 表示は `log.Split` と `log.CountUnfinished` で tasks/memos/未完了数をすでに個別集計しており、この点は今回の変更対象外。
- `-a/--all`（stat無し）はファイル名一覧のみを表示し、アイテム自体を表示しない。

## Desired Outcome
- フラグなしの `sava list [date]` では、完了済みタスクは非表示になり、未完了タスクとメモだけが見える（最頻ユースケースの視認性を優先）。
- `--full` フラグを指定したときだけ、完了済みタスクを含めた全件が見える。
- 一覧は「完了済みタスク → 未完了タスク → メモ」の順にグルーピングされ、各グループ内は既存通り作成日時昇順で並ぶ（`--full` 未指定時は完了済みグループが単に存在しない2グルーピングになる）。
- `--task` フラグを指定すると、メモを除いたタスクのみが表示される。`--full` と組み合わせ可能（例: `--task --full` で完了済み+未完了タスクのみ、メモなし）。

## Approach
Discoveryで合意した方針（アプローチ1）:
- `ListOptions` に `TasksOnly bool`（`--task`）と `Full bool`（`--full`、短縮形候補 `-f`）を追加する。
- 表示対象の選別は既存の `log.Split` を再利用し、`Full` の有無で完了済みタスクを含めるか、`TasksOnly` の有無でメモを含めるかを決める。
- 表示順は「完了済み→未完了→メモ」の3グループ固定。各グループ内のソートは既存の作成日時昇順ロジックを流用する。
- `-s/--stat` および `-a`（stat無し、ファイル名一覧）は表示対象アイテムそのものを出さないモードのため変更不要。

## Scope
- **In**:
  - `sava list [date]` のデフォルト挙動変更（完了済みタスクを非表示に）
  - `--full` フラグの新設（完了済みタスクも表示）
  - `--task` フラグの新設（タスクのみ表示、メモ除外）
  - 一覧表示のグルーピング順序変更（完了済み→未完了→メモ）
- **Out**:
  - `-s/--stat` の集計内容・表示フォーマットの変更
  - `-a/--all`（stat無し、ファイル名一覧）の変更
  - `--tag` フィルタとの組み合わせ仕様の再設計（既存のAND/ORロジックをそのまま踏襲するのみ）
  - `diff` / `del` / `start` / `end` など他コマンドへの波及

## Boundary Candidates
- フラグ定義・パース（`cmd/sava/commands.go` の `newListCommand`）
- 表示対象アイテムの選別ロジック（`internal/command.listOneDay`、または `internal/log` への切り出し）
- グルーピング済み出力のview層実装（`internal/view.Timeline` の拡張、またはグルーピング対応の新関数）

## Out of Boundary
- `-s/--stat` のロジック・出力フォーマット
- タスクの状態遷移（`start`/`end`）自体の仕様変更
- 過去ログの `carriedFrom` やタグ管理など、本機能に無関係な既存仕様

## Upstream / Downstream
- **Upstream**: `internal/model.Item`（`Closed` フィールドによるTask/Memo判定・完了判定）、`internal/log.Split`（既存関数をそのまま活用）
- **Downstream**: 特になし。将来的に日付範囲などの別軸フィルタが追加される場合の土台にはなり得る。

## Existing Spec Touchpoints
- **Extends**: なし（プロジェクト初のspec）
- **Adjacent**: なし

## Constraints
- 既存の `-t/--tag`, `-y/--yes`, `-a/--all` の既存挙動（ファイル一覧・全体stat）は変更しない
- フラグの短縮形は未使用の文字のみ使用可能（`-a`, `-s`, `-y`, `-t` は使用済み）
- 本機能はデフォルト出力の破壊的変更を含むため、`/kiro-spec-requirements` で受け入れ基準として明示する
