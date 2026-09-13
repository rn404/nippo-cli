## Summary
- **Feature**: `empty-content-validation`
- **Discovery Scope**: Simple Addition（既存関数への検証追加のみ、新規コンポーネントなし）
- **Key Findings**:
  - `internal/log.AddTags`（`normalizeTags`）に既に「トリム後空文字列は`ErrEmptyTag`で拒否する」という確立済みパターンがあり、content検証もこれに倣うのが自然。
  - `internal/command.Add`/`command.Todo`は`log.Add`のエラーをそのまま早期returnしているため、`log.Add`側でエラーを返すだけでタグ付け・タスク開始・ログ書き込みが自動的にスキップされる。command層のコード変更は不要。
  - `sava edit`の直接指定モードは既に空文字列を許容する方針が確定済み（`edit-command`スペックのdiscoveryで判明した既存ギャップ）であり、本スペックは`add`/`todo`のみを対象として`edit`には触れない。

## Research Log

### 既存の空文字列検証パターン
- **Context**: content検証をどこにどう実装するかを決めるため、既存コードに類似のバリデーションがないか調査した。
- **Sources Consulted**: `internal/log/log.go`（`normalizeTags`, `ErrEmptyTag`）, `internal/command/command.go`（`Add`, `Todo`）, `cmd/sava/commands.go`（`newAddCommand`, `newTodoCommand`）
- **Findings**:
  - `normalizeTags`は`strings.TrimSpace`後に空なら`ErrEmptyTag`（`errors.New`のsentinel）を返す。
  - `command.Add`/`command.Todo`は`log.Add`のエラーを`if err != nil { return err }`でそのまま伝播し、以降のタグ付け・開始処理・`persistItem`（ファイル書き込み・index再構築）を実行しない。
  - CLIエントリポイント（`cobra.ExactArgs(1)`）は空文字列も1個の引数として受理するため、`sava add ""`が素通りしている。
- **Implications**: content検証は`internal/log.Add`に集約し、既存の`ErrEmptyTag`と同じ判定基準（トリム後空文字列）を用いる`ErrEmptyContent`を新設するのが最小差分。command層・CLI層の変更は不要。

## Architecture Pattern Evaluation

| Option | Description | Strengths | Risks / Limitations | Notes |
|--------|-------------|-----------|---------------------|-------|
| `internal/log.Add`で検証 | 既存の`normalizeTags`と同じ層でsentinel errorを返す | 既存パターンと一貫、command/CLI層の変更不要 | なし | 採用 |
| `internal/command.Add`/`Todo`で検証 | ユースケース層でcontentをチェックしてから`log.Add`を呼ぶ | — | タグ検証との層不一致、`log`パッケージ単体でのテストがしにくくなる | 不採用 |
| `cmd/sava`のCLI層で検証 | フラグパース直後にチェック | — | `internal/log`パッケージ単体テストで空文字列拒否を検証できなくなり、ライブラリとしての`log.Add`が空文字列を無条件に受理してしまう | 不採用 |

## Design Decisions

### Decision: content検証を`internal/log.Add`に置く
- **Context**: `add`/`todo`直接指定時の空文字列（トリム後）contentを拒否する必要がある。
- **Alternatives Considered**:
  1. `internal/command`層（`Add`/`Todo`）でチェック
  2. `cmd/sava`のCLI層でチェック
- **Selected Approach**: `internal/log.Add`の先頭でcontentを`strings.TrimSpace`し、空なら新設の`ErrEmptyContent`を返す。既存itemsは変更せずreturnする。
- **Rationale**: 既存のタグ空文字列検証（`normalizeTags`/`ErrEmptyTag`）と同じ層・同じ判定基準にすることで一貫性を保ち、`command.Add`/`command.Todo`の既存のearly-return構造がそのままタグ付け・開始処理・ファイル書き込みのスキップを保証する。
- **Trade-offs**: なし（既存の呼び出し元コードの変更が不要な最小差分）。
- **Follow-up**: なし。

### Decision: 保存されるcontent自体はトリムしない
- **Context**: contentが空でない場合、前後の空白を保存前にトリムするかどうか。
- **Alternatives Considered**:
  1. 検証のみ行い、保存するcontentは元の文字列のまま
  2. 検証と同時にトリム済みcontentを保存する
- **Selected Approach**: 検証（空判定）にのみトリム結果を使い、Itemに保存するcontentは呼び出し時の元の文字列のままとする。
- **Rationale**: 本チケットのスコープは「空文字列の拒否」のみであり、非空content時の既存動作（前後空白を保持したまま保存する現状の挙動）を変えないため（Requirement 2）。
- **Trade-offs**: 前後に空白を含むcontentがそのまま保存され続けるが、これは本スペックのスコープ外の別課題。
- **Follow-up**: なし。

## Risks & Mitigations
- 特になし（既存パターンへの追従のみで、新規リスクは生じない）。

## References
- なし（社内既存コードパターンの踏襲のみ、外部参照不要）。
