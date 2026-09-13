# Research & Design Decisions

## Summary
- **Feature**: `list-output-format`
- **Discovery Scope**: Extension（既存の `internal/view.Timeline` の行フォーマット変更）
- **Key Findings**:
  - `view.Timeline` の呼び出し元は `internal/command/command.go:543`（`listOneDay` の非stat分岐）の1箇所のみ。シグネチャ変更の影響範囲は極めて小さい。
  - `view.go` 内の `marker` 関数は `Timeline` からのみ呼ばれている。今回の行フォーマット全面変更により `marker` は不要になり、削除して新しい `checklistPrefix` に置き換えるのが妥当。
  - Go標準ライブラリ `strings.Cut`（Go 1.18+、本リポジトリは1.26）で「最初の改行で分割」が1行で書ける。新規外部依存は不要。

## Research Log

### `view.Timeline` の呼び出し元調査
- **Context**: シグネチャに `fullText bool` を追加して問題ないか確認する必要があった。
- **Sources Consulted**: `grep -rn "view.Timeline\|Timeline(" internal/ cmd/`
- **Findings**: `internal/command/command.go:543` の1箇所のみが `view.Timeline(w, items)` を呼んでいる（`internal/log.Timeline` は同名だが別パッケージの無関係な関数）。
- **Implications**: 既存呼び出し元は1箇所だけなので、シグネチャ変更（第3引数追加）によるリスクは最小限。新しい関数を並立させる必要はない。

### `marker` 関数の再利用可否
- **Context**: 新しいチェックリスト構文（`[ ]`/`[x]`/メモ無しチェックボックス/`in-progress`トークン）は既存の `marker`（`・`/`[ ]`/`[>]`/`[x]`）とは表現が異なる。
- **Sources Consulted**: `internal/view/view.go` 全体、`grep -rn "marker(" internal/`
- **Findings**: `marker` は `Timeline` 内の1箇所でのみ使用。新フォーマットでは「着手中」を `[ ]` + `` `in-progress` `` トークンで表すため、`marker` の4分岐ロジックとは別の新しい関数が必要。
- **Implications**: `marker` を削除し、`checklistPrefix` という新しい関数に置き換える（死んだコードを残さない）。

## Design Decisions

### Decision: `fullText` をパラメータとして `view.Timeline` に追加する
- **Context**: 複数行content のデフォルト要約（先頭行のみ） vs `--full-text` 指定時の全文表示を、どのレイヤーで切り替えるか。
- **Alternatives Considered**:
  1. `internal/command.listOneDay` 側でcontentを事前に要約してから `view.Timeline` に渡す（Timelineは要約済みの文字列を受け取るだけ）
  2. `view.Timeline` に `fullText bool` を追加し、Timeline自身が要約するかどうかを決める
- **Selected Approach**: 案2。
- **Rationale**: 「contentをどう描画するか（1行に要約 or 全文）」は表示フォーマットの責務であり、`internal/view` パッケージが担うべき関心事。`internal/command` 層に文字列整形ロジックを漏らさないほうが、既存の層分離（`structure.md` の「依存方向は一方向」）に合致する。
- **Trade-offs**: `view.Timeline` のシグネチャが変わる（破壊的）が、呼び出し元は1箇所のみなので実害は小さい。
- **Follow-up**: `internal/view/view_test.go` の既存 `TestTimeline`/`TestTimelineEmpty` は新シグネチャ（3引数）に追従させる必要がある。

### Decision: hashの抜き取り用パターンは `` `[a-f0-9]{8}` `` のみに依存する
- **Context**: hashを `` (`hash`) `` で囲む際、pipe処理側が安定して抜き取れることを保証する必要がある。
- **Alternatives Considered**:
  1. 括弧全体 `` \(`[a-f0-9]{8}`\) `` にマッチさせる
  2. バックティック内の16進8桁 `` `[a-f0-9]{8}` `` だけにマッチさせる（外側の括弧は無視）
- **Selected Approach**: 案2。
- **Rationale**: 外側の括弧は見た目の装飾であり、pipe処理にとって本質的ではない。バックティック＋16進8桁という組み合わせは行内の他の要素（content、タグ）と衝突しにくく、一貫して安定している。
- **Trade-offs**: なし。
- **Follow-up**: 設計・実装時にこの前提を明記し、README等の利用例でも同じ抜き取り方法を示す。

## Risks & Mitigations
- デフォルト出力フォーマットの破壊的変更（2回目）— `list-task-visibility` に続き、短期間でまた `sava list` の出力形式が変わる。README更新と合わせてユーザーに周知する（README更新は本specのtasks.md生成方針上コード専用タスクからは除外されるため、必要なら別途対応する）。
- `marker` 削除によるデッドコード化の見落とし — 削除前に `grep` で他の利用箇所が無いことを確認済み（上記Research Log参照）。

## References
- なし（外部ライブラリ・API調査は不要、既存コードベースのみを参照）
