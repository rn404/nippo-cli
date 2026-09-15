# Technical Design Document

## Overview
**Purpose**: `sava list` の `--full` フラグを `--full-list` にリネームし、同じく `list` に存在する `--full-text` フラグとの命名の紛らわしさを解消する。
**Users**: `sava list` で完了済みタスクを含めて表示したい `sava` 利用者。
**Impact**: `--full` フラグは廃止され、以後は未定義フラグとして扱われる（破壊的変更）。新フラグ名 `--full-list` が同じ挙動（ショートハンド `-f` を含む）を引き継ぐ。

### Goals
- `--full` を `--full-list` に置き換え、`-f` ショートハンドは維持する。
- `--full`/`--full-text` の命名衝突感を解消する。
- 既存の可視性ロジック（完了済みタスクを含めるかどうかの判定、`--task`/`--full-text`/`--tag` との組み合わせ）は一切変更しない。

### Non-Goals
- `--full-text` フラグ自体の名称・挙動の変更（対象外）。
- `--full` を指す非推奨エイリアスの提供（要件で旧名の完全廃止が明示されているため行わない）。
- 完了済みタスクの表示判定ロジック自体の変更。

## Boundary Commitments

### This Spec Owns
- `sava list` に登録されるCLIフラグの名称（`--full` → `--full-list`、`-f` は維持）。
- `command.ListOptions` 内の対応するGo構造体フィールド名（`Full` → `FullList`）とその消費箇所。
- `--full` を参照している既存テスト・READMEの該当記述の更新。

### Out of Boundary
- `--full-text` フラグの名称・実装（`internal/view.Timeline` のfullText分岐を含む）。
- 完了済みタスクを含めるかどうかを決める可視性ロジック自体（`listOneDay` 内の分岐条件）。
- `--task`・`--tag`・`--stat`・`--all` の既存挙動。

### Allowed Dependencies
- 既存の `github.com/spf13/cobra`（フラグ定義）以外の新規依存は発生しない。

### Revalidation Triggers
- `--full-text` が将来リネームされる場合、README・ヘルプ文言上での `--full-list` との併記箇所を再確認する必要がある。
- `command.ListOptions` の構造体フィールドを扱う他spec（将来の `list` 拡張）は、フィールド名が `Full` ではなく `FullList` になったことを前提にする。

## Architecture
既存のcobraフラグ定義パターン（`cmd/sava/commands.go` の `newListCommand` 内で `cmd.Flags().BoolVarP(...)` を呼ぶ）をそのまま踏襲する、単一コンポーネント内の名称変更のみ。新規のアーキテクチャパターンや図は不要（変更は1ファイルの1関数、および内部の対応する1フィールドに閉じる）。

### Technology Stack
本feature専用の新規技術選定はない（既存のGo + cobra CLIスタックをそのまま使用）。

## File Structure Plan

### Modified Files
- `cmd/sava/commands.go` — `newListCommand` 内、`cmd.Flags().BoolVarP(&opts.Full, "full", "f", false, "include completed tasks")` を `cmd.Flags().BoolVarP(&opts.FullList, "full-list", "f", false, "include completed tasks")` に変更（フラグ名・バインド先フィールド名のみ変更、ショートハンド `-f` とヘルプ文言は維持）。
- `internal/command/command.go` — `ListOptions` 構造体のフィールド宣言 `Full bool // include closed tasks in the daily (non-stat) view` を `FullList bool // include closed tasks in the daily (non-stat) view` に、`listOneDay` 内の参照 `if opts.Full { ... }` を `if opts.FullList { ... }` に変更。
- `internal/command/command_test.go` — 全 `ListOptions{..., Full: true, ...}` 参照（同ファイル内7箇所、6つのテスト関数にまたがる）を `FullList: true` に、`TestListStatAndAll_UnaffectedByNewFlags` 内の関連するエラーメッセージ文言（`Full/TasksOnly/FullText`）を `FullList/TasksOnly/FullText` に更新。
- `cmd/sava/root_test.go` — `TestListFullAndTaskFlags` のコマンド文字列 `"--full"`（2箇所）を `"--full-list"` に更新。加えて、`sava list --full`（旧フラグ名）が未定義フラグエラーになることを検証する新規テストケースを同関数内（または新規関数として）追加する。テスト名・doc commentの `"--full"`/`-f` 表記もあわせて更新する。
- `README.md` — 97, 99, 101行目の `--full` を `--full-list` に更新（`sava list --full-list` / コメント文中の `--full` / `sava list --task --full-list`）。

## Components and Interfaces

| Component | Domain/Layer | Intent | Req Coverage | Key Dependencies (P0/P1) | Contracts |
|-----------|--------------|--------|---------------|---------------------------|-----------|
| ListCommandFlags | CLI (`cmd/sava`) | `sava list` のフラグ定義・パースを担当し、`--full-list`（`-f`）を登録する | 1.1, 1.2, 2.1, 2.2, 3.1 | ListOptions (P0) | Service |
| ListOptionsVisibility | ユースケース層 (`internal/command`) | `FullList` フィールドを受け取り、`listOneDay` の完了済みタスク包含判定に反映する | 1.1, 1.3, 1.4 | — | State |

いずれも新しい責務境界を導入しない既存コンポーネントの名称変更のみのため、詳細ブロックは省略し、Implementation Notesに要点をまとめる。

**Implementation Notes**
- Integration: `ListCommandFlags` が `ListOptionsVisibility`（`ListOptions.FullList`）へ値を渡す既存の結線をそのまま維持し、フラグ名とフィールド名のみを変更する。`--task`（`TasksOnly`）・`--full-text`（`FullText`）との組み合わせロジックは変更しない。
- Validation: `--full` を指定した場合の未定義フラグエラーは、cobraの標準フラグパース挙動（未登録フラグ検出時に `unknown flag: --full` 相当のエラーを返す）にそのまま委譲する。カスタムのエラーハンドリングは追加しない。
- Risks: `Full` フィールドを参照する全箇所（本体2・テスト4）を洗い出し済み（`research.md` 参照）。取りこぼすとコンパイルエラーで即座に検出できるため、実装後の `go build`/`go test` 実行で網羅性を機械的に担保できる。

## Requirements Traceability

| Requirement | Summary | Components | Interfaces | Flows |
|-------------|---------|-------------|------------|-------|
| 1.1 | `--full-list` が完了済みタスクを含める | ListCommandFlags, ListOptionsVisibility | BoolVarP登録 → `ListOptions.FullList` | — |
| 1.2 | `-f` ショートハンドが `--full-list` と同じ挙動 | ListCommandFlags | BoolVarP登録（`-f`） | — |
| 1.3 | `--full-list` + `--task` の組み合わせ挙動を維持 | ListOptionsVisibility | `listOneDay` 内の既存分岐 | — |
| 1.4 | `--full-list` + `--full-text` の組み合わせ挙動を維持 | ListOptionsVisibility | `listOneDay`/`view.Timeline` 既存分岐（変更なし） | — |
| 2.1 | `--full` は未定義フラグエラーになる | ListCommandFlags | cobra標準フラグパース | — |
| 2.2 | `--full` という名前のフラグを提供しない | ListCommandFlags | BoolVarP登録（`"full-list"`のみ） | — |
| 3.1 | `sava list --help` が `--full-list` を表示する | ListCommandFlags | cobraヘルプ生成（ヘルプ文言は既存のまま） | — |
| 3.2 | READMEが `--full-list` を記載する | — (ドキュメント) | `README.md` 該当箇所 | — |

## Testing Strategy

### Unit Tests
- `internal/command/command_test.go`: `TestListStatAndAll_UnaffectedByNewFlags` を `FullList` フィールド名で更新し、`--stat`/`--all` 出力が `FullList: true` を渡しても変化しないことを引き続き検証する（Req 1.3, 1.4 の非干渉境界を保証）。

### Integration Tests
- `cmd/sava/root_test.go`: `TestListFullAndTaskFlags` を `--full-list`/`-f` を使う形に更新し、完了済みタスクを含む挙動・`--task` との組み合わせ挙動を引き続き検証する（Req 1.1, 1.2, 1.3）。
- `cmd/sava/root_test.go`: `sava list --full` を実行した際に `root.Execute()` がエラーを返すことを検証する新規テストケースを追加する（Req 2.1）。
- `cmd/sava/root_test.go`: `TestFullTextFlag` は変更不要だが、`--full-list --full-text` の組み合わせで両方の効果が独立して現れることを検証するケースを追加する（Req 1.4）。

### Documentation Verification
- `README.md` の `--full` 記載箇所（97, 99, 101行目）がすべて `--full-list` に置き換わっていることをタスク完了時にレビューで確認する（自動テスト対象外、Req 3.2）。
