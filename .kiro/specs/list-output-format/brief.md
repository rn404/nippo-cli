# Brief: list-output-format

## Problem
`sava list` の出力をMarkdownノート（Obsidian等）へそのまま貼り付けたい場面、またpipe経由で他のコマンド（grep/awk等）からhashだけを取り出したい場面の両方で、現在の固定フォーマットは最適化されていない。また、複数行のメモはそのまま全文出力されるため「1行=1アイテム」という前提が崩れやすく、pipe処理の信頼性を下げている。

## Current State
`internal/view.Timeline` がアイテムを `- <time> <marker> <content> (<hash>)<tags>` という固定フォーマットで出力する。`marker` はメモ(・)/未着手([ ])/着手中([>])/完了([x])の4状態を文字で表現している。`content` は改行を含んでいてもそのまま出力される（先頭行のみに絞る仕組みは無い）。

## Desired Outcome
- デフォルト表示自体が、GFM（GitHub Flavored Markdown）のチェックリスト構文として解釈可能な形になり、Markdownノートに貼ってそのまま意味を持つ
- hashは常に行末の `` (`hash`) ``（括弧の中をバックティックで囲む）という一貫した位置・形式になり、pipe処理から正規表現で確実に抜き取れる
- 複数行のcontentはデフォルトでは先頭行のみを表示する（`git log --oneline` 相当の要約）
- `--full-text` フラグを指定したときだけ、contentの全文（複数行含む）を表示する

## Approach
行フォーマットを以下に統一する（`--full-text` 未指定時は各アイテムのcontentの先頭行のみを使用）:

```
- [ ] <time> <content> (`<hash>`)                      # 未着手タスク
- [ ] `in-progress` <time> <content> (`<hash>`)         # 着手中タスク（標準GFMの2状態[ ]/[x]は保ち、状態はバックティック付きの先頭トークンで表現）
- [x] <time> <content> (`<hash>`)                      # 完了タスク
- <time> <content> (`<hash>`)                          # メモ（チェックボックスなし）
```

- hashは常に行末の `` (`[a-f0-9]{8}`) `` パターンで一貫して抜き取れる（括弧は見た目の親しみやすさを保つための装飾、抜き取り自体はバックティック内の16進8桁にマッチさせればよい）
- `--full-text` 指定時は、content全文（複数行含む）を出す。これはopt-inの詳細表示モードであり、「1行=1アイテム」の保証はこのモードでは求めない

## Scope
- **In**:
  - `sava list [date]`（stat無しの日次表示）のデフォルト行フォーマットの再設計
  - `--full-text` フラグの新設（content全文表示）
- **Out**:
  - `-s/--stat` の出力フォーマット
  - `-a/--all`（stat無し、ファイル名一覧）の出力フォーマット
  - 既存 `--full`/`--task`（list-task-visibility）の可視性ロジック自体（順序・フィルタ条件は変更しない。今回は行の描画のみを変える）
  - タグのフィルタ・表示ロジック自体（表示位置の調整はあり得るが、フィルタ条件の変更はしない）

## Boundary Candidates
- `internal/view.Timeline`（行フォーマットの描画ロジック本体）
- `internal/model.Item`（content先頭行を取り出すヘルパーが必要な場合）
- `internal/command.listOneDay`（`--full-text` オプションの橋渡し）
- `cmd/sava/commands.go`（`--full-text` フラグ登録）

## Out of Boundary
- `-s/--stat`、`-a/--all`（stat無し）の出力
- 既存 `--full`/`--task` の可視性ロジック（list-task-visibilityの境界）
- タグのフィルタロジック自体

## Upstream / Downstream
- **Upstream**: `list-task-visibility`（closed/open/started/memoのグルーピング順序、`--full`/`--task`による可視性ルールはそのまま活用し、その出力順序の上に新フォーマットを適用する）
- **Downstream**: 特になし

## Existing Spec Touchpoints
- **Extends**: なし（新規境界。list-task-visibilityの可視性ロジックには手を入れない）
- **Adjacent**: `list-task-visibility`（同じ`list`コマンドの出力だが、関心が異なる：可視性/順序 vs 行の描画フォーマット）

## Constraints
- デフォルト出力フォーマットの破壊的変更（行全体の構文はGFMチェックリスト寄りになる。hashの括弧表記自体は `(hash)` → `` (`hash`) `` とバックティックが増える程度の差分。README・既存の出力に依存する外部ツールがあれば影響する）
- **将来の検討事項（今回のスコープ外）**: 既存の `--full` フラグ名が曖昧（「fullって何に対してのfullか分かりにくい」）という指摘があり、`--full-list` へのリネーム案が出ている。今回は対応せず、別途検討する。
