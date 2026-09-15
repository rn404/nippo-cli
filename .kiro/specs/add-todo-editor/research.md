# Research & Design Decisions

## Summary
- **Feature**: `add-todo-editor`
- **Discovery Scope**: Extension（既存の`sava edit`エディタモード・`internal/editor`パッケージの再利用）
- **Key Findings**:
  - `internal/editor.Resolve`は`current`文字列を受け取る汎用実装で、`current=""`を渡すだけで新規作成フローに転用できる（追加のシグネチャ変更は不要）。
  - `editor.Resolve`自身のok判定は「保存内容が`current`と完全一致、または完全な空文字列」でしか中断としない。`current=""`のとき、空白文字のみの保存は中断と判定されない（`" " != ""`）。
  - `internal/log.Add`は`empty-content-validation`スペックにより、トリム後空文字列を`ErrEmptyContent`として拒否する。エディタ経由の空白のみcontentをそのまま`command.Add`/`command.Todo`に渡すと、このエラーがそのままユーザーに表面化し、「エディタ経由は中断（無エラー）」という要件と矛盾する。

## Research Log

### editor.Resolveのok判定とlog.Addの空文字列検証の不整合
- **Context**: 要件2.4（保存内容が空白文字のみの場合も中断とし、エラーにしない）を、既存コンポーネントの変更なしに満たせるかを検討。
- **Sources Consulted**: `internal/editor/editor.go`（`Resolve`実装）、`internal/log/log.go`（`Add`のトリム検証、`empty-content-validation`スペックで追加）。
- **Findings**:
  - `editor.Resolve`のok判定: `candidate == current || candidate == ""`。空白のみの保存はどちらにも該当せず`ok=true`になる。
  - `log.Add`はトリム後空文字列を`ErrEmptyContent`で拒否するが、これは直接指定モード向けの検証であり、「無エラーで中断する」というエディタモードの意味論とは異なる。
- **Implications**: 空白のみのcontentを`command.Add`/`command.Todo`（＝`log.Add`）に到達させると、意図しないエラー扱いになる。CLI層で`editor.Resolve`のok=true後に追加で`strings.TrimSpace(content) == ""`をチェックし、該当すれば中断経路に合流させることで、`internal/editor`・`internal/log`のどちらも変更せずに要件を満たせる。

## Architecture Pattern Evaluation

| Option | Description | Strengths | Risks / Limitations | Notes |
|--------|-------------|-----------|---------------------|-------|
| CLI層での追加TrimSpaceチェック（採用） | `cmd/sava`内で`editor.Resolve`のok=true後にもう一段`TrimSpace`判定を行う | `internal/editor`/`internal/log`いずれも無変更。既存の「空白のみ＝空」基準を１行で踏襲するだけで済む | 判定基準（TrimSpace）が`log.normalizeTags`/`log.Add`と3箇所目の重複になる | 各箇所とも既存実装が独立しており、共通ヘルパー化は現時点の要件規模に対して過剰（Non-Goals参照） |
| `internal/editor.Resolve`のok判定自体をTrimSpaceベースに変更 | `Resolve`内の空文字列比較をトリムベースにする | 判定ロジックの重複が無くなる | `sava edit`のエディタモードの挙動も変わってしまう（末尾以外の空白のみ保存を無変更扱いにする等）。out-of-scopeの「`sava edit`のエディタ挙動自体は変更しない」に抵触する | 却下 |
| `command.Add`/`command.Todo`にエディタ由来かどうかのフラグを渡し、`ErrEmptyContent`をそこで中断扱いに変換 | 呼び出し元の由来を`command`層まで伝搬させる | — | `command.Add`/`command.Todo`のシグネチャ変更が必要になり、直接指定モードのシンプルさを損なう。境界（本Spec Owns）が`internal/command`にまで広がってしまう | 却下 |

## Design Decisions

### Decision: 空白のみcontentの中断判定はCLI層で行う
- **Context**: 要件2.4を満たしつつ、`internal/editor`・`internal/log`・`command.Add`/`command.Todo`のいずれも変更しないという最小差分の設計にしたい。
- **Alternatives Considered**:
  1. `internal/editor.Resolve`のok判定自体をトリムベースに変更する
  2. `command.Add`/`command.Todo`にエディタ由来フラグを渡し、`ErrEmptyContent`を中断に変換する
- **Selected Approach**: `cmd/sava`の`resolveNewContent`が、`editor.Resolve`のok=true後に`strings.TrimSpace(content) == ""`を追加チェックし、該当すれば`ok=false`と同じ中断経路に合流させる。
- **Rationale**: `internal/editor`は`sava edit`と共有されるため変更すると影響範囲が広がる。`command.Add`/`command.Todo`のシグネチャ変更は境界を広げすぎる。CLI層での1行追加が最小差分。
- **Trade-offs**: 「空白のみ＝空」という判定基準が`internal/log`（`normalizeTags`, `Add`）とCLI層の計3箇所に独立して存在することになるが、いずれも1行程度の単純な条件であり、共通化の複雑さに見合わない。
- **Follow-up**: 将来この基準が変わる場合（例: タブ文字の扱いを変える等）は3箇所すべてを確認する必要がある。`design.md`のRevalidation Triggersに記録済み。

## Risks & Mitigations
- 判定基準（TrimSpaceで空とみなす）が複数箇所に分散している — 変更時は`design.md`のRevalidation Triggersを起点に横断確認する運用で対応する。
- `$EDITOR`にスペースを含むパス等、シェルクオートが必要なケースは非対応 — `edit-command`スペックのNon-Goalsで既に明記済みの既存制約であり、本スペックで新たに増える制約ではない。

## References
- `.kiro/specs/edit-command/design.md` — `internal/editor.Resolve`の設計・`newEditCommand`の分岐パターン
- `.kiro/specs/empty-content-validation/design.md` — `internal/log.Add`の`ErrEmptyContent`検証
