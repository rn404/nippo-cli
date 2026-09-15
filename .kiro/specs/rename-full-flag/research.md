## Summary
- **Feature**: `rename-full-flag`
- **Discovery Scope**: Simple Addition（既存CLIフラグの名称変更）
- **Key Findings**:
  - `--full` は `sava list` にのみ登録されており（`cmd/sava/commands.go:252`）、`command.ListOptions.Full` にバインドされる。`sava add`/`sava todo` には存在しない。
  - `--full` を直接消費するのは `internal/command/command.go` の `listOneDay` 内の1箇所（`if opts.Full { ... }`）のみで、他の可視性ロジックとの結合はない。
  - `--full-text`（`edit-command` ではなく `list-output-format` spec由来）は完全に独立したフラグで、rename対象ではない。
  - テストでのフラグ文字列 `"--full"` への依存は `cmd/sava/root_test.go`（`TestListFullAndTaskFlags`）の2箇所のみ。Go構造体フィールド `Full` への依存は `internal/command/command_test.go` に7箇所（6つのテスト関数にまたがる）。
  - README.md に `--full` の使用例が3箇所（97, 99, 101行目）ある。

## Research Log

### 既存の `--full` / `--full-text` 利用箇所の洗い出し
- **Context**: リネーム対象・影響範囲をスコープするため、`--full` と `--full-text` の定義・消費・テスト・ドキュメント上の全参照箇所を特定する必要があった。
- **Sources Consulted**: `cmd/sava/commands.go`, `internal/command/command.go`, `internal/command/command_test.go`, `cmd/sava/root_test.go`, `README.md`（Explore subagentによるコードベース調査）。
- **Findings**:
  - フラグ文字列 `"full"` はコード中1箇所（`cmd/sava/commands.go:252`）のみで定義される。
  - Go構造体フィールド `Full`（`command.ListOptions`）への参照は本体2箇所・テスト7箇所（上記Key Findings参照）。
  - `--full` はショートハンド `-f` を持ち、既存仕様（`list-task-visibility` spec）でも `-f` は明記されている。
- **Implications**: 変更範囲は極めて限定的（5ファイル）で、新規の依存や外部連携は発生しない。フィールド名 `Full` も `FullList` にリネームすることで、`FullText` との命名の紛らわしさをコード上でも解消できる（ユーザー向けフラグ名の意図をそのまま内部表現にも反映）。

## Architecture Pattern Evaluation
新規アーキテクチャパターンの検討は不要（既存のcobraフラグ定義パターンをそのまま踏襲する単純なリネーム）。

## Design Decisions

### Decision: Go構造体フィールド `Full` も `FullList` にリネームする
- **Context**: `--full` フラグ名を `--full-list` に変更するにあたり、内部で束縛する `command.ListOptions.Full` フィールド名をそのままにするか、合わせてリネームするか。
- **Alternatives Considered**:
  1. フィールド名 `Full` は変更せず、CLIフラグ名だけを `--full-list` に変える（`BoolVarP(&opts.Full, "full-list", "f", ...)`）。
  2. フィールド名も `FullList` にリネームし、フラグ名とフィールド名を一致させる。
- **Selected Approach**: 2（フィールド名も `FullList` にリネーム）。
- **Rationale**: 今回のリネームの動機は「`--full` と `--full-text` の名前が紛らわしい」という指摘であり、同じ紛らわしさはコード上の `Full` と `FullText` という2つのフィールド名にも存在する。フラグ名だけ変えてフィールド名を据え置くと、ユーザー向けの名称とコード上の名称が乖離し、将来の実装者が再び同じ曖昧さに遭遇する。
- **Trade-offs**: 変更差分が（フラグ定義1箇所だけでなく）フィールド宣言・消費箇所・テストにも及ぶが、いずれも機械的な置換で済む規模（本体2箇所・テスト7箇所）であり、コストは小さい。
- **Follow-up**: なし。

## Risks & Mitigations
- Risk: `--full` を使い続けているユーザーのスクリプトが壊れる（意図された破壊的変更） — Mitigation: Linear issue・requirements.md で明示済み。README更新でも新フラグ名を周知する。エイリアス提供は本specのスコープ外（要件が明示的に「旧名廃止」を求めているため）。
- Risk: テスト更新漏れ（`Full` を参照する7箇所のいずれかを見落とす） — Mitigation: 設計段階で全参照箇所を洗い出し済み（本ドキュメントKey Findings参照）。task生成時に各ファイルを明示的にタスク境界へ含める。

## References
- `.kiro/specs/list-task-visibility/brief.md` — `--full`/`-f` の元設計。
- `.kiro/specs/list-output-format/` — `--full-text` の由来。
