## Overview
`sava add`/`sava todo`の直接指定モードは現在、contentの空文字列チェックを行っておらず、`sava add ""`のように空文字列（または空白のみの文字列）を渡すと空のメモ・タスクが記録されてしまう。本スペックは`internal/log.Add`にcontent検証を追加し、直接指定モードで空文字列を拒否することで、意図しない空アイテムの記録を防ぐ。

**Users**: `sava add`/`sava todo`をコマンドライン引数で直接使う全ユーザー。誤って空文字列を渡した場合、即座にエラーとしてフィードバックを得られる。

**Impact**: `internal/log.Add`に検証ロジックを追加する。`internal/command.Add`/`command.Todo`は既存のエラー伝播構造（`if err != nil { return err }`）をそのまま利用するため変更不要。非空content時の既存動作は変えない。

### Goals
- `add`/`todo`の直接指定content（空文字列・空白のみの文字列）を拒否する
- 検証失敗時に、ログファイルへの書き込み・タグ付け・タスク開始などの副作用を一切発生させない
- 既存の非空content時の挙動（保存されるcontentの内容含む）を変えない

### Non-Goals
- `sava edit`の直接指定モードの挙動変更（`edit-command`スペックで空文字列許容の方針が既に確定しており対象外）
- content省略時に`$EDITOR`を起動する新規フロー（別スペックの対象）
- 保存されるcontent自体のトリム（検証のみ行い、非空content自体は変更しない）

## Boundary Commitments

### This Spec Owns
- `internal/log.Add`におけるcontentの空文字列（トリム後）検証ロジックと、その検証結果を表すエラー定義
- `internal/command.Add`/`command.Todo`が、その検証エラーをそのまま呼び出し元（CLI層）に伝播すること

### Out of Boundary
- `sava edit`の直接指定モードのバリデーション方針（既存のまま変更しない）
- `$EDITOR`起動によるcontent入力フロー（別スペックの対象）
- タグ検証（`normalizeTags`/`ErrEmptyTag`）自体のロジック変更

### Allowed Dependencies
- 既存の`internal/log`パッケージのエラー定義パターン（`ErrEmptyTag`と同型のsentinel error）
- `internal/command`の既存の呼び出し順序（`log.Add` → `AddTags`/`Start` → `persistItem`）

### Revalidation Triggers
- `internal/log.Add`のシグネチャ変更
- `command.Add`/`command.Todo`における副作用実行順序の変更（`log.Add`呼び出し後にearly-returnしない構造への変更）
- `sava edit`の空文字列許容方針が変わった場合（本スペックとの整合性の再検討が必要）

## Architecture

### Existing Architecture Analysis
既存のレイヤー構成は`cmd/sava` → `internal/command` → `internal/log` → `internal/model`。空文字列検証の前例は`internal/log.normalizeTags`（タグ用）に既にあり、トリム後空文字列を`ErrEmptyTag`として拒否する。本スペックは同じ層（`internal/log`）に同じパターンでcontent検証を追加する。単一コンポーネントの小規模変更のため、アーキテクチャ図は省略する。

**Architecture Integration**:
- 選定パターン: 既存の`normalizeTags`と同じ「呼び出し直後にトリムして空ならsentinel errorを返す」パターンをcontentに適用
- 既存パターン踏襲: sentinel error（`errors.New`、`errors.Is`で判定可能）、`internal/log`層での検証
- 新規コンポーネント: なし（既存の`log.Add`関数を拡張するのみ）

## File Structure Plan

### Modified Files
- `internal/log/log.go` — `ErrEmptyContent`（`ErrEmptyTag`と同型のsentinel error）を追加。`Add`関数の先頭でcontentを`strings.TrimSpace`し、空文字列なら`model.Item{}, ErrEmptyContent`を返し、以降のItem生成処理を行わない。
- `internal/log/log_test.go` — `Add`に対して、空文字列・空白のみcontent（isTask true/falseそれぞれ）で`ErrEmptyContent`が返り`l.Items`が変化しないこと、非空content（前後空白を含む）では従来通りItemが作成され、contentがトリムされずそのまま保存されることを検証するテストを追加。
- `internal/command/command_test.go` — `command.Add`/`command.Todo`に空文字列contentを渡した際、エラーが伝播し、`persistItem`（ログファイル書き込み・index再構築）およびタグ付け・タスク開始（`--start`との組み合わせ含む）が一切実行されないことを検証する統合テストを追加。

`internal/command/command.go`と`cmd/sava/commands.go`は変更しない — 前者は既存の`if err != nil { return err }`がそのままタグ付け/開始処理をスキップし、後者はcobraの`RunE`エラーをそのまま標準エラー出力・非ゼロ終了に変換する既存の仕組みを使う。

## Components and Interfaces

| Component | Domain/Layer | Intent | Req Coverage | Key Dependencies (P0/P1) | Contracts |
|-----------|--------------|--------|---------------|---------------------------|-----------|
| `log.Add` | internal/log | contentを検証し、有効なら新規Item（Task/Memo）を作成する | 1.1, 1.2, 1.3, 1.4, 2.1, 2.2 | `model.NewTaskItem`/`model.NewMemoItem` (P0) | Service |

### internal/log

#### log.Add

| Field | Detail |
|-------|--------|
| Intent | 新規Item作成前にcontentの空文字列検証を行う |
| Requirements | 1.1, 1.2, 1.3, 1.4, 2.1, 2.2 |

**Responsibilities & Constraints**
- contentを`strings.TrimSpace`した結果が空文字列なら、Itemを作成せず`ErrEmptyContent`を返す（Task/Memo共通、Req 1.1〜1.3）
- トリム結果が空でなければ、既存動作通り元の（トリムしていない）contentでItemを作成する（Req 2.1, 2.2）
- 呼び出し元（`command.Add`/`command.Todo`）は本エラーをそのまま返すことで、後続のタグ付け・タスク開始処理を実行しない（Req 1.4）— 既存のearly-return構造がそのまま満たすため、呼び出し元側の追加実装は不要

**Dependencies**
- Inbound: `command.Add`, `command.Todo` — 新規Item作成の呼び出し元 (P0)
- Outbound: `model.NewTaskItem`, `model.NewMemoItem` — Item生成 (P0)

**Contracts**: Service [x] / API [ ] / Event [ ] / Batch [ ] / State [ ]

##### Service Interface
```go
// Add appends a new task or memo to the log and returns the created item.
// content is rejected with ErrEmptyContent when it is empty after
// trimming leading/trailing whitespace.
func Add(l *model.Log, content string, isTask bool) (model.Item, error)
```
- Preconditions: `l`は非nilな`*model.Log`
- Postconditions: 成功時、`l.Items`に一意なhashを持つItemが1件追加され、そのcontentは引数`content`のまま（トリムなし）。`ErrEmptyContent`時、`l.Items`は変更されない。
- Invariants: `strings.TrimSpace(content) == ""`となるcontentは、isTaskの値に関わらず一律で拒否される

**Implementation Notes**
- Integration: `command.Add`/`command.Todo`は既に`log.Add`のエラーを無条件で早期returnしているため、コード変更不要。
- Validation: 判定基準は`strings.TrimSpace(content) == ""`。既存のタグ検証（`normalizeTags`）と同じ基準に揃える。
- Risks: なし。既存呼び出し元の変更が不要で、ログファイルフォーマット（`model.Item`のフィールド）にも影響しない。

## Error Handling

### Error Strategy
`ErrEmptyContent`を`ErrEmptyTag`と同様のsentinel error（`errors.New`、`errors.Is`で判定可能）として`internal/log`に定義する。`internal/command`・`cmd/sava`はラップせずそのまま伝播し、cobraの既存の`RunE`エラーハンドリング（標準エラー出力・非ゼロ終了）をそのまま利用する。

### Error Categories and Responses
**User Errors**: 空文字列・空白のみのcontent → `ErrEmptyContent`を返し、Item未作成・ファイル未書き込みのまま呼び出し元にエラーを伝播する。

## Testing Strategy

### Unit Tests（`internal/log/log_test.go`）
1. `Add(&l, "", false)` / `Add(&l, "", true)` → `ErrEmptyContent`、`l.Items`は変化しない
2. `Add(&l, "   ", false)` → 空白のみのcontentも`ErrEmptyContent`として拒否される
3. `Add(&l, " 出社 ", false)` → 成功し、作成されたItemのcontentは`" 出社 "`のまま（トリムされない、既存動作維持）

### Integration Tests（`internal/command/command_test.go`）
4. `command.Add(w, dir, "", AddOptions{})` → エラーを返し、ログファイルが新規/更新書き込みされない
5. `command.Todo(w, dir, "", TodoOptions{Start: true, Tags: []string{"x"}})` → エラーを返し、タスク開始（`Start`）・タグ付け（`AddTags`）・index再構築が一切実行されない
