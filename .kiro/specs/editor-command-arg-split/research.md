# Research & Design Decisions

## Summary
- **Feature**: `editor-command-arg-split`
- **Discovery Scope**: Extension（既存の `internal/editor` / `cmd/sava` エディタ起動経路の修正）
- **Key Findings**:
  - `cmd/sava/commands.go` の `runEditor` は `$EDITOR`/`$VISUAL` の値をそのまま `exec.Command` の第一引数（実行ファイル名）として渡しており、複数トークンの値（`code --wait` 等）で `exec: "code --wait": executable file not found in $PATH` エラーになる。
  - **この「空白区切りで簡易分割する」対応は、実は `edit-command` spec の `design.md` に既に明記されていた意図だった**が、実装（`runEditor`）には反映されていなかった。今回は新しい設計判断ではなく、既存specの意図と実装のギャップを埋める修正という位置づけになる。
  - `internal/editor` パッケージ（`Name`/`Resolve`）は起動対象の文字列を決めるだけで、実際に `exec.Command` を呼ぶのは `cmd/sava/commands.go` の `runEditor`（本番用の launch 関数）のみ。分割ロジックはここに閉じ込められる。
  - `add`/`todo`/`edit` の3コマンドすべてが同じ `runEditor` を launch 関数として注入しているため、1箇所の修正で3コマンドすべてに適用される。

## Research Log

### `$EDITOR`/`$VISUAL` の実行ファイル名解決
- **Context**: 会社のMac環境で `sava edit` 実行時に `sava: exec: "code --wait": executable file not found in $PATH` が発生したとの報告。
- **Sources Consulted**:
  - `cmd/sava/commands.go`（`runEditor`, L155-164）
  - `internal/editor/editor.go`（`Name`, `Resolve`）
  - `.kiro/specs/edit-command/design.md`（Non-Goals: 「`$EDITOR`に指定されたコマンド文字列の完全なシェル互換パース（クオート等）。空白区切りの簡易分割のみサポートする」、Component層でも同旨の記述あり）
  - Go標準ライブラリ `os/exec`: `exec.Command(name, args...)` はシェルを経由せず、`name` をそのまま `$PATH` 上のファイル名としてルックアップする（スペースを含む文字列は1つのファイル名として扱われる）
- **Findings**:
  - `git` の `core.editor` は同様の複数トークン設定（`"code --wait"` 等）を許容する慣習があり、ユーザー環境で広く使われている。
  - Goの `exec.Command` にはシェル的なトークン分割機能がない。分割は呼び出し側の責務。
  - `edit-command` design.md はこの前提をすでに認識しており、「空白区切りの簡易分割」を明示的なNon-Goal境界（＝フルパース非対応）として合意済みだった。実装時に反映漏れがあったとみられる。
- **Implications**: 新規の設計判断ではなく、既存合意の実装漏れの是正として扱う。スコープも `edit-command` design.md のNon-Goalsを踏襲し、空白区切りの単純分割のみをサポートし、クオート・シェル展開等のフルパースはサポートしない。

## Architecture Pattern Evaluation

| Option | Description | Strengths | Risks / Limitations | Notes |
|--------|-------------|-----------|----------------------|-------|
| 空白分割（`strings.Fields`） | `$EDITOR`/`$VISUAL`の値をGo標準ライブラリで空白区切りにトークン化し、`exec.Command(tokens[0], tokens[1:]+path...)`で起動 | 標準ライブラリのみで完結（新規依存なし）、シェルを経由しないため注入リスクがない、`edit-command` design.mdの既存合意と一致 | クオートされた引数内の空白は分割できない（意図的にサポート外） | 採用 |
| シェル経由起動（`sh -c "$EDITOR" path`） | `/bin/sh -c` 経由でエディタ文字列をシェルに解釈させる | クオート・展開等、シェル文法をフルサポートできる | シェルへの新たな依存、パスのエスケープを誤ると壊れやすい、`edit-command`のNon-Goalsで明示的に除外された範囲まで踏み込む＝スコープ逸脱 | 不採用 |

## Design Decisions

### Decision: `$EDITOR`/`$VISUAL` の複数トークン値を空白区切りで分割する
- **Context**: 複数トークンの `$EDITOR`/`$VISUAL` 値でエディタ起動が失敗する不具合を修正する。
- **Alternatives Considered**:
  1. 空白分割（`strings.Fields`) — 標準ライブラリのみ、`edit-command`合意済みスコープと一致
  2. シェル経由起動（`sh -c`) — フルパース可能だが新規依存とスコープ逸脱
- **Selected Approach**: `cmd/sava/commands.go` の `runEditor` で、渡された `name` を `strings.Fields` でトークン化し、先頭トークンを実行ファイル、残りのトークン＋一時ファイルパスを引数として `exec.Command` に渡す。
- **Rationale**: 標準ライブラリのみで完結する（`tech.md`の依存方針に合致）。`edit-command` design.mdで既に合意済みの境界（空白区切りのみ、シェルクオート非対応）をそのまま踏襲する。
- **Trade-offs**: クオートされた引数内の空白（例: `EDITOR='"my editor" --wait'`）は非対応のまま。実利用上、この形式で `$EDITOR` を設定するケースは稀（`code --wait`のような単純な複数トークンが大半）。
- **Follow-up**: `strings.Fields` が空スライスを返すケース（`$EDITOR`が空白のみの値）で `parts[0]` の範囲外アクセスにならないよう、実装時にガードを入れる（既存挙動でもこのケースはエラーになるため、パニックさせず同じ「起動失敗」エラー経路に落とす）。

## Risks & Mitigations
- リスク: 分割ロジックの導入により、単一トークンの既存ケース（`vim`, `vi` 等）で回帰が起きる — 緩和: 既存のエディタモード統合テスト（`add`/`todo`/`edit`）をそのまま流用し、単一トークンのケースも引き続きカバーする。
- リスク: `$EDITOR` が空白のみの値の場合にパニックする — 緩和: トークン化結果が空の場合はエラーとして扱うガードを実装する（Design Decision参照）。

## References
- `.kiro/specs/edit-command/design.md` — 空白区切り分割という境界の初出（Non-Goals / Component記述）
