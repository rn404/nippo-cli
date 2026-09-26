# Technical Design

## Overview
本機能は、`sava add`/`sava todo`/`sava edit` がエディタモードでエディタプロセスを起動する処理を修正し、`$EDITOR`/`$VISUAL` にコマンド名+引数からなる複数トークンの値（例: `code --wait`）が設定されている環境でも正しく起動できるようにする。

**Purpose**: `$EDITOR`/`$VISUAL` が複数トークンの値である環境で、エディタ起動が `exec: "<value>": executable file not found in $PATH` エラーで失敗し、`add`/`todo`/`edit` のエディタモードが一切使えなくなる不具合を修正する。
**Users**: `$EDITOR`/`$VISUAL` を複数トークンの値（`code --wait` 等）に設定している `sava` ユーザー。
**Impact**: `cmd/sava/commands.go` の `runEditor` のみを修正する。`internal/editor` パッケージの公開インターフェース（`Name`, `Resolve`）や、単一トークンの `$EDITOR`/`$VISUAL` を使う既存ユーザーの挙動は変更しない。

### Goals
- `$EDITOR`/`$VISUAL` の値が空白区切りの複数トークン（コマンド + 引数）であっても、正しくエディタプロセスを起動できるようにする
- 単一トークンの値（`vim` 等）や、値未設定時の `vi` フォールバックなど、既存の挙動を一切変更しない
- `add`/`todo`/`edit` の3コマンドすべてに、1箇所の修正で一貫して適用する

### Non-Goals
- クオート・エスケープ・環境変数展開・パイプなど、空白区切りを超えるシェル文法のフルパース（`edit-command` design.md のNon-Goalsを踏襲）
- エディタ選択の優先順位ロジック自体の変更（`$EDITOR` → `$VISUAL` → `vi`）
- content を直接引数で渡す直接指定モードへの変更（エディタを経由しないため対象外）

## Boundary Commitments

### This Spec Owns
- `cmd/sava/commands.go` の `runEditor`（本番用エディタ起動関数）における、`$EDITOR`/`$VISUAL` の値をコマンド+引数として解釈するロジック

### Out of Boundary
- `internal/editor.Name`/`internal/editor.Resolve`（エディタ名の優先順位解決、一時ファイル管理、中断判定）— 変更しない
- `cmd/sava` の `add`/`todo`/`edit` コマンド自体の引数パース・フラグ定義 — 変更しない
- 直接指定モード（content引数あり）の挙動 — 変更しない

### Allowed Dependencies
- Go標準ライブラリ `os/exec`（既存の依存、`exec.Command` の呼び出し方法のみ変更）
- Go標準ライブラリ `strings`（トークン分割に使用、新規import）
- `internal/editor.Resolve` が `runEditor` を呼び出す既存の契約（`func(editorName, path string) error`）は変更しない

### Revalidation Triggers
- `internal/editor.Resolve`/`runEditor` 間の呼び出し契約（シグネチャ）が変わる場合
- 将来、クオート等のフルシェル互換パースが必要になった場合（Non-Goalsの再検討が必要）

## Architecture

### Existing Architecture Analysis
- `internal/editor.Resolve` は launch 関数（シグネチャ `func(editorName, path string) error`）を注入される設計で、本番用の実装は `cmd/sava/commands.go` の `runEditor` のみ。テストは `editor_test.go` のフェイク launch 関数、および `cmd/sava/root_test.go` の実プロセス起動（フェイクシェルスクリプト）の2系統で行われている。
- `runEditor` は現在 `exec.Command(name, path)` として `name`（`$EDITOR`/`$VISUAL`の生の値、または `"vi"`）をそのまま実行ファイル名として渡しており、複数トークンの値を分割していない。
- `edit-command` design.md は「空白区切りの簡易分割のみサポートする」ことを既にNon-Goalsとして明記していたが、`runEditor` の実装には反映されていなかった（`research.md` 参照）。本designはこのギャップを埋める。

### Architecture Integration
- **Selected pattern**: 既存のレイヤー構成・依存方向は変更しない。`internal/editor` は launch 関数のシグネチャを知るのみで、実際の `exec.Command` 呼び出し方法には関与しない設計を維持する。
- **Domain/feature boundaries**: 「`$EDITOR`/`$VISUAL` の値をどう解釈してプロセスを起動するか」は `runEditor`（`cmd/sava`層）の責務のまま。「どのエディタ名を使うべきか」（優先順位解決）は引き続き `internal/editor.Name` の責務。両者の境界は変更しない。
- **Existing patterns preserved**: launch関数の注入パターン（本番は `runEditor`、テストはフェイク関数）。
- **New components rationale**: 新規コンポーネントは無い。既存の `runEditor` 内部ロジックのみの変更。
- **Steering compliance**: `tech.md` の「標準ライブラリのみで完結させる方針」を維持（`strings.Fields` のみ追加、新規外部依存なし）。

## File Structure Plan

### Modified Files
- `cmd/sava/commands.go` — `runEditor(name, path string) error` を変更し、`name` を空白区切りでトークン化（`strings.Fields`）してから、先頭トークンを実行ファイル、残りのトークン + `path` を引数として `exec.Command` を呼ぶ。トークン化結果が空（`$EDITOR`/`$VISUAL` が空白のみの値）の場合は、`exec.Command` を呼ばずにエラーを返す（パニック防止）。
- `cmd/sava/root_test.go` — 複数トークンの `$EDITOR` 値（例: フェイクスクリプトのパス + ダミー引数）でも `add`/`todo`/`edit` のエディタモードが正しく起動し、引数がエディタプロセスに渡ることを検証する統合テストを追加する。既存の単一トークンのエディタモードテストは変更しない（回帰確認としてそのまま維持）。

他ファイル（`internal/editor/`, `internal/log/`, `internal/command/`, `internal/view/`）は変更しない。

## Requirements Traceability

| Requirement | Summary | Components | Interfaces | Flows |
|-------------|---------|------------|------------|-------|
| 1.1, 1.2 | 複数トークンの値をコマンド+引数として解釈し起動 | `runEditor` | `runEditor(name, path string) error` | — |
| 1.3, 1.4 | 単一トークン・`vi`フォールバックの既存挙動維持 | `runEditor` | 同上 | — |
| 2.1, 2.2 | 分割後もコマンドが見つからない場合はエラーで中断、自動フォールバックしない | `runEditor` | 同上 | — |
| 3.1 | `add`/`todo`/`edit` 全経路への一貫適用 | `runEditor`（共通の launch 関数として3コマンドから注入される） | 同上 | — |

## Components and Interfaces

| Component | Domain/Layer | Intent | Req Coverage | Key Dependencies (P0/P1) | Contracts |
|-----------|---------------|--------|---------------|---------------------------|-----------|
| `runEditor` | CLI (`cmd/sava`) | `$EDITOR`/`$VISUAL`解決済みの値をコマンド+引数に分割し、エディタプロセスを起動する | 1.1-1.4, 2.1-2.2, 3.1 | `os/exec`（P0）, `internal/editor.Resolve`（P0、呼び出し元） | Service |

### CLI層 (`cmd/sava`)

#### runEditor

| Field | Detail |
|-------|--------|
| Intent | `internal/editor.Resolve` から注入される本番用 launch 関数。解決済みのエディタ値を分割し、実プロセスを起動する |
| Requirements | 1.1-1.4, 2.1-2.2, 3.1 |

**Responsibilities & Constraints**
- `name`（`internal/editor.Name()` が返す `$EDITOR`/`$VISUAL`、または `"vi"`）を `strings.Fields` で空白区切りにトークン化する。
- トークンが1つ以上ある場合、先頭トークンを実行ファイル名、残りのトークンに編集対象の一時ファイルパス `path` を追加した列を引数として `exec.Command` に渡す。
- トークンが0個（`name` が空白のみ）の場合、`exec.Command` を呼ばずにエラーを返す（範囲外アクセスによるパニックを防止し、Requirement 2.1と同じ「起動失敗」のエラー経路に統一する）。
- 分割後の実行ファイルが `$PATH` 上に見つからない場合や、起動自体がエラーになった場合は、そのエラーをそのまま呼び出し元（`internal/editor.Resolve`）に返す。設定と異なる別のエディタへの自動フォールバックは行わない。
- stdin/stdout/stderrを実ターミナルに接続する既存の挙動は変更しない。

**Dependencies**
- Inbound: `internal/editor.Resolve`（`add`/`todo`/`edit` の3コマンドいずれも同じ `runEditor` を launch 関数として注入する）
- Outbound: `os/exec`（P0）

**Contracts**: Service [x] / API [ ] / Event [ ] / Batch [ ] / State [ ]

##### Service Interface
```go
// runEditor is the production launch function injected into
// editor.Resolve: it splits name (the resolved $EDITOR/$VISUAL
// value, or "vi") into a command and its arguments on whitespace,
// then execs it against the real terminal's stdin/stdout/stderr.
func runEditor(name, path string) error
```
- Preconditions: `name` は `internal/editor.Name()` の戻り値（空文字列にはならない）。ただし空白のみの値は許容し得る。
- Postconditions: トークン化結果が空でなければ、先頭トークンを実行ファイルとして `exec.Command` を呼び出し、その結果（成功/エラー）をそのまま返す。トークン化結果が空であれば、`exec.Command` を呼ばずにエラーを返す。
- Invariants: 単一トークンの `name`（既存の全テストケースが対象）では、これまでと同一のコマンド・引数（`[path]`）で起動される。

**Implementation Notes**
- Integration: `internal/editor.Resolve` からのみ呼ばれる（呼び出し契約は変更しない）。
- Validation: `name` のトークン化のみ。トークン内容自体の検証（存在確認等）は `exec.Command`/OSに委譲する（既存と同じ）。
- Risks: クオートされた引数内の空白を含む `$EDITOR` 値は、意図通りには分割されない（Non-Goalsで明記済み、許容するリスク）。

## Error Handling

### Error Strategy
既存のエラー方針を維持する: エディタ起動に失敗した場合（分割後の実行ファイルが見つからない、起動時にエラーが発生した、またはトークン化結果が空）、`runEditor` はエラーを返し、`internal/editor.Resolve` を経由してそのエラーがそのまま `add`/`todo`/`edit` コマンドまで伝播し、コマンド全体を中断する。自動的な別エディタへのフォールバックは行わない（Requirement 2.2）。

### Error Categories and Responses
**System Errors**: 分割後の実行ファイルが `$PATH` 上に見つからない、またはOSレベルでプロセス起動に失敗 → 既存通り、`exec.Command`/`cmd.Run()` が返すエラーをそのまま伝播する（メッセージの書き換えは行わない）。
**Invalid Configuration**: `$EDITOR`/`$VISUAL` が空白のみの値でトークン化結果が空 → `exec.Command` を呼ばずに明示的なエラーを返す（パニック防止）。

## Testing Strategy

### Unit Tests
- `runEditor` 相当のロジック（トークン化＋引数組み立て）について、複数トークンの値・単一トークンの値・空白のみの値の3パターンで、`exec.Command` に渡される実行ファイル名と引数列が期待通りになることを検証する。

### Integration Tests
- `cmd/sava/root_test.go`: `$EDITOR` を「フェイクスクリプトのパス + ダミー引数（例: `--flag`)」に設定し、`add`/`edit` のエディタモードでフェイクスクリプトが引数付きで正しく起動され、一時ファイルパスが最後の引数として渡ることを検証する（少なくとも1コマンドで検証すれば、共通の `runEditor` を経由する `todo` にも波及する設計だが、既存テストのパターンに合わせて主要コマンドで確認する）。
- 既存の単一トークン `$EDITOR` の統合テスト（`add`/`todo`/`edit` の各エディタモードテスト）は変更せず、そのまま回帰確認として機能させる。
