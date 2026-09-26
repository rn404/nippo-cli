# Research & Design Decisions

## Summary
- **Feature**: `list-content-truncation`
- **Discovery Scope**: Extension（既存の`list-output-format` spec が導入した `internal/view.Timeline` への変更）
- **Key Findings**:
  - `Timeline`の公開シグネチャ（`Timeline(w io.Writer, items []model.Item, fullText bool)`）は変更不要。呼び出し元（`internal/command`, `cmd/sava`）への波及がゼロで済む、非常に閉じた変更である。
  - 既存の`--full-text`関連テスト（`cmd/sava/root_test.go`）は、改行がそのまま出力されることを前提に書かれており、今回の変更でそれらのアサーションは意図的に変更が必要（回帰ではなく仕様変更の反映）。
  - contentの切り詰めはバイト数ではなくrune数で行う必要がある（日本語content等のマルチバイト文字を壊さないため）。Go標準ライブラリの`[]rune`変換のみで実現でき、新規依存は不要。

## Research Log

### 既存実装の調査（`internal/view/view.go`, `internal/view/view_test.go`）
- **Context**: `list-output-format` specで導入された現行の`firstLine`実装と、その呼び出し箇所・テストを確認する必要があった。
- **Sources Consulted**: `internal/view/view.go`（`Timeline`, `firstLine`）、`internal/view/view_test.go`、`internal/command/command.go`（`listOneDay`）、`internal/command/command_test.go`、`cmd/sava/root_test.go`
- **Findings**:
  - `firstLine`は`strings.Cut(content, "\n")`で最初の改行までを切り出すだけの単純な実装。
  - `fullText`が`true`のときはcontentを一切加工せず、複数行のまま出力していた（意図的な仕様。`list-output-format`のPostconditionにも明記）。
  - `--full-text`の統合テスト（`TestFullTextFlag`, `TestFullListAndFullTextFlagsAreIndependent`）は、改行を含む文字列がそのまま出力されることをアサートしている。
- **Implications**: 今回の変更は`internal/view/view.go`内の非公開ヘルパーの置き換えのみで完結する。ただし上記の既存テストは、新しい仕様（`--full-text`でも改行をスペースに置換する）に合わせてアサーション内容を更新する必要がある（design.mdのTesting Strategy参照）。

### マルチバイト文字の切り詰め方法
- **Context**: `sava`のcontentは日本語（マルチバイトUTF-8）を含みうる（`product.md`が日本語で書かれている通り、日本語ユーザーを想定したツールであるため）。50文字への切り詰めをバイト単位で行うと、マルチバイト文字の途中で切れて不正なUTF-8列になる恐れがある。
- **Sources Consulted**: Go標準ライブラリドキュメント（`strings`, `unicode/utf8`の一般的な知識）
- **Findings**: `[]rune(s)`への変換後にスライスすれば、文字（コードポイント）単位で安全に切り詰められる。追加ライブラリは不要。
- **Implications**: `truncate`ヘルパーは`[]rune`変換を使う。結合文字（絵文字の合字等）まで考慮したグラフェムクラスタ単位の切り詰めは行わない（既存コードベース全体がその水準の処理をしておらず、要件上も要求されていないため、過剰実装と判断）。

## Architecture Pattern Evaluation
今回はViewレイヤー内の純粋関数の置き換えのみで、新たなアーキテクチャパターンの検討は不要と判断した（Extension規模の変更のため、パターン比較表は省略）。

## Design Decisions

### Decision: 改行置換とcontentの整形を1つの関数（`summarizeContent`）に統合する
- **Context**: Requirement 1（デフォルト表示）とRequirement 2（`--full-text`表示）は、どちらも「まず改行をスペースに置換する」という共通ステップを持ち、その後の分岐（切り詰めの有無）だけが異なる。
- **Alternatives Considered**:
  1. `fullText`ごとに別々の関数（`summarizeDefault`/`summarizeFullText`）を用意する
  2. 改行置換と切り詰めを1つの`summarizeContent(content string, fullText bool) string`関数にまとめ、内部で分岐する
- **Selected Approach**: 2を採用。
- **Rationale**: 改行置換という共通処理の重複を避けられ、`Timeline`側の呼び出しも「常に`summarizeContent`を呼ぶ」という1行に単純化できる（design-synthesisの一般化の観点）。
- **Trade-offs**: 関数内に条件分岐が1つ増えるが、行数・複雑度ともに小さく、可読性への影響は軽微。
- **Follow-up**: 実装時、`truncate`は`summarizeContent`から呼ばれる純粋なヘルパーとして分離し、境界値（ちょうど50文字）のテストを独立して書けるようにする。

### Decision: 文字数上限（50）と省略記号（`…`）は定数として`view.go`に定義する
- **Context**: 要件上は固定値だが、テストからも参照できる形にしておきたい。
- **Alternatives Considered**:
  1. マジックナンバー・マジック文字列としてハードコードする
  2. `view.go`内にパッケージ定数（例: `defaultSummaryLength`, `ellipsis`）として定義する
- **Selected Approach**: 2を採用。
- **Rationale**: 既存の`const bullet = "-"`と同じ慣習に揃えられ、テストコードや将来の変更（もしあれば）で参照しやすい。
- **Trade-offs**: なし（デメリットが実質ない単純な整理）。

### Decision: 連続する改行はまとめて1つの半角スペースに圧縮する（改行1つ=スペース1つの単純な1対1置換ではない）
- **Context**: `/kiro-validate-design`のレビューで、空行を段落区切りに使うメモ（例: `"段落A\n\n段落B"`）に対して改行を1つずつ独立にスペース置換すると、スペースが連続して表示され（`"段落A  段落B"`）、特にデフォルト表示の50文字予算を空白で無駄に消費するという指摘が出た。
- **Alternatives Considered**:
  1. 仕様通り、改行1つを半角スペース1つに単純に置換する（連続改行はスペース連続のまま）
  2. 連続する改行（1つ以上のラン）をまとめて単一の半角スペース1つに圧縮する
- **Selected Approach**: 2を採用。`requirements.md`のRequirement 1.1 / 2.1を「改行の連続するランを単一の半角スペースに置換する」という表現に更新した。
- **Rationale**: デフォルト表示の50文字という限られた予算を、意味のないスペースの連続で消費してしまうのは、要件が本来意図した「文字数ベースで安定した要約を見せる」という目的に反する。
- **Trade-offs**: 単純な1対1置換より実装が数行増えるが（連続する改行をまとめて処理する必要がある）、複雑度への影響は軽微。
- **Follow-up**: `internal/view/view_test.go`に、連続改行（空行）を含むcontentに対する専用テストケースを追加する（design.mdのTesting Strategy参照）。

## Risks & Mitigations
- **Risk**: 既存の`--full-text`統合テストが、新しい仕様と矛盾するアサーション（改行がそのまま残ることを期待）を持っている — **Mitigation**: design.mdの「Existing Tests Requiring Updates」に明記済み。タスク化の際に、これらのテスト更新を独立したタスクではなく、実装タスクの一部として扱う。
- **Risk**: `\r\n`（CRLF）を含むcontentがあった場合、`\r`が表示に残る — **Mitigation**: 現状のログ入力経路でCRLFが混入する既知のケースはなく、スコープ外として明示（design.mdのRevalidation Triggers参照）。将来問題が顕在化した場合に対応する。

## References
- `.kiro/specs/list-output-format/design.md` — `Timeline`/`firstLine`の元の設計（今回置き換える対象）
- `.kiro/specs/list-output-format/requirements.md` — 現行の「最初の改行までを1行とする」仕様の由来
