# Research & Design Decisions

## Summary
- **Feature**: `list-task-visibility`
- **Discovery Scope**: Extension (existing `sava list` command)
- **Key Findings**:
  - `internal/log.Split` already partitions `model.Item` into tasks/memos, sorting each bucket independently with the package-private `byCreatedAt` helper via `sort.SliceStable`. A three-way variant (closed / open+started / memos) can reuse the identical pattern with zero new sorting logic.
  - Existing precedent for combining a filter flag with `--all` is **not uniform**: `--tag`+`--all` returns an explicit error in `command.List`, while `--tag` is silently unconsulted in single-day `--stat` mode (the stat branch returns before the tag-filter code runs). Approved Requirement 5.2/5.3 already specifies the literal outcome for this feature ("retain existing output, do not apply the new behavior"), so the design does not need to invent new validation.
  - No new external dependency is required; the feature is fully contained within the existing `internal/{log,command,view}` layering and the standard library (`sort`).

## Research Log

### `--all` の組み合わせ時の挙動 precedent
- **Context**: `--full`/`--task` を `-a`（`--all`）と併用したときにどう振る舞うべきか、既存コードの先例を確認する必要があった。
- **Sources Consulted**: `internal/command/command.go` の `List`/`listOneDay`（既存実装）。
- **Findings**: `--tag`+`--all` は `List()` 内で `errors.New("tag filter cannot be combined with --all")` を返す。一方、単日 `-s`（stat）モードは `opts.Stat` 分岐が早期 return するため、`opts.Tags` は評価すらされない（サイレントに無視される）。同じ「フィルタ系オプション」でもモードによって挙動が異なっており、単一の先例として踏襲できるものではない。
- **Implications**: 承認済みの Requirement 5.2 と 5.3 は、両方とも「既存の出力をそのまま維持し、新フラグの挙動を適用しない」と明記している。これはエラーを返す設計とは相容れない。したがって本機能では `--full`/`--task` を `-a` 指定時（`-s` の有無を問わず）に一切参照しない設計とし、新しいバリデーションエラーは追加しない（Design Decision 参照）。

### `SplitByStatus` の実装方針
- **Context**: 「完了済みタスク → 未完了タスク（未着手＋着手中） → メモ」の3グループ順序をどう生成するか。
- **Sources Consulted**: `internal/log/log.go` の既存 `Split`・`byCreatedAt`・`model.Item.IsClosed`/`IsTask`。
- **Findings**: `Split` は1パスで `tasks`/`memos` に振り分けた後、各スライスを個別に `sort.SliceStable(_, byCreatedAt(_))` でソートしている。`model.Item.IsClosed()` は `Closed != nil && *Closed` を返す既存メソッドで、タスクの完了判定にそのまま使える。
- **Implications**: `Split` と同じ1パス振り分け＋バケットごとの安定ソートというパターンをそのまま3バケット（closed / open+started / memos）に拡張すれば、新しいソートアルゴリズムを書く必要がない。`byCreatedAt` は同一パッケージ内の非公開ヘルパーなのでそのまま再利用できる。

## Architecture Pattern Evaluation

| Option | Description | Strengths | Risks / Limitations | Notes |
|--------|-------------|-----------|---------------------|-------|
| 3-way split in `internal/log`（採用） | `Split` と同じパターンで closed/open/memos の3バケットに振り分ける専用関数を追加 | 既存パターンを完全踏襲、新規ソートロジック不要、`command` 層は薄いままでよい | バケット数が増えるとタプル戻り値が読みにくくなる（現状3つなら許容範囲） | Simplification原則に合致。将来4バケット目が必要になったら構造体に切り替える |
| 汎用フィルタ述語パイプライン | `--tag` と `--task`/`--full` を統一的な predicate として扱う共通フィルタ層を新設 | 将来のフィルタ軸追加に強い | 現在の要求に対して過剰設計（YAGNI）、`internal/log` の変更範囲が広がる | discoveryセッションで既に不採用と判断済み（アプローチ2） |

## Design Decisions

### Decision: `--all` 指定時の `--full`/`--task` の扱い
- **Context**: `--full`/`--task` を `--all`（`-a`、`-s`の有無を問わず）と併用したときの挙動を決める必要がある。
- **Alternatives Considered**:
  1. `--tag`+`--all` の既存エラー先例を踏襲し、`--full`/`--task`+`--all` も明示的エラーにする
  2. 承認済み要件（5.2, 5.3）の文言通り、`--all` 指定時はフラグをサイレントに無視し既存出力を維持する
- **Selected Approach**: 案2。`command.List` の `opts.All` 分岐、および `listOneDay` の `opts.Stat` 分岐では `opts.Full`/`opts.TasksOnly` を一切参照しない。
- **Rationale**: 承認済みRequirement 5.2/5.3は「既存出力をそのまま維持」と明記しており、エラーを返す案1はこれに反する。案1は独自にUX一貫性を追求した結果の提案だったが、要件文言を優先し、要求されていない新規エラーメッセージを増やさないシンプルな設計を採用する。
- **Trade-offs**: `--full`/`--task` を `-a` と併用してもエラーにならず「静かに無視される」ため、ユーザーがフラグの無効化に気づきにくい可能性がある。将来的にUXの声が上がれば別途検討する。
- **Follow-up**: 実装時、`listOneDay` のstat分岐・`List`のall分岐のどちらにも `opts.Full`/`opts.TasksOnly` の参照を追加しないことをレビューで確認する。

### Decision: `SplitByStatus` の関数シグネチャ
- **Context**: 表示グルーピング（closed/open+started/memos）をどこでどう計算するか。
- **Alternatives Considered**:
  1. `internal/log` に `SplitByStatus(l model.Log) (closedTasks, openTasks, memos []model.Item)` を新設し、`Split` と同じ1パス＋安定ソートパターンを踏襲
  2. `internal/command` 側で既存 `Split` の戻り値をさらにフィルタして closed/open に分ける
- **Selected Approach**: 案1。`internal/log` はドメインロジック（アイテムの分類・整列）を担う既存レイヤーであり、`Split` と対になる関数として配置するのが既存の構造パターン（Boundary/Layering原則）に合致する。
- **Rationale**: `command` 層に分類ロジックを漏らすと、将来 `internal/log` の別の呼び出し元（例: 将来のstat拡張）が同じ分類ロジックを重複実装するリスクがある。
- **Trade-offs**: なし（既存パターンの直接的な拡張のため）。
- **Follow-up**: `internal/log/log_test.go` に `TestSplitByStatus` を追加し、closed/open+started/memosの振り分けとグループ内ソート順序を検証する。

## Risks & Mitigations
- 破壊的なデフォルト挙動変更（完了済みタスクが突然見えなくなる） — READMEのUsageセクションを更新し、`--full` の存在を明記する（ドキュメント更新はタスク化する）。
- `--full`/`--task` を `--all` と併用したときにサイレントに無視される点に気づきにくい — 将来のフォローアップ候補として記録（今回のスコープ外）。

## References
- なし（外部ライブラリ・API調査は不要、既存コードベースのみを参照）
