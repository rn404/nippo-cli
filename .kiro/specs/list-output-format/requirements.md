# Requirements Document

## Project Description (Input)
`sava list` の出力をMarkdownノート（Obsidian等）へそのまま貼り付けたい場面、またpipe経由で他のコマンド（grep/awk等）からhashだけを取り出したい場面の両方で、現在の固定フォーマット（`- <time> <marker> <content> (<hash>)<tags>`）は最適化されていない。また、複数行のメモはそのまま全文出力されるため「1行=1アイテム」という前提が崩れやすく、pipe処理の信頼性を下げている。

この課題に対し、以下の変更を行う:
- デフォルト表示自体を、GFM（GitHub Flavored Markdown）のチェックリスト構文として解釈可能な形に再設計する（未着手 `- [ ]` / 着手中 `- [ ] \`in-progress\`` / 完了 `- [x]` / メモ `-` チェックボックスなし）
- hashは常に行末の `` (`hash`) ``（括弧＋バックティック）で一貫して表し、pipe処理から正規表現で確実に抜き取れるようにする
- 複数行のcontentはデフォルトでは先頭行のみを表示する（`git log --oneline` 相当の要約）
- `--full-text` フラグを指定したときだけ、contentの全文（複数行含む）を表示する

対象は `sava list [date]`（stat無しの日次表示）のみ。`-s/--stat`、`-a/--all`（stat無し）の出力、既存の `--full`/`--task`（list-task-visibility）の可視性ロジック自体、タグのフィルタロジック自体は変更しない。

詳細な背景・スコープ・制約は `.kiro/specs/list-output-format/brief.md`（discoveryセッションの成果物）を参照。

## Introduction
`sava list [date]`（stat無しの日次表示）のデフォルト出力を、Markdownノートへの貼り付けとpipe処理の両方に適した行フォーマットへ再設計する。各アイテムはGFMチェックリスト構文で表現され、hashは常に行末の一貫した位置・形式になる。複数行のcontentはデフォルトで先頭行のみを示し、`--full-text` フラグで全文表示に切り替えられる。

## Boundary Context (Optional)
- **In scope**: `sava list [date]`（stat無しの日次表示）のデフォルト行フォーマット、`--full-text` フラグ
- **Out of scope**: `-s/--stat` の出力フォーマット、`-a/--all`（stat無し）の出力フォーマット、既存の `--full`/`--task`（list-task-visibility）の可視性ロジック自体、タグのフィルタロジック自体
- **Adjacent expectations**: 新フォーマットは、list-task-visibilityが決めた「どのアイテムを表示するか・どの順で表示するか」のルールの上にそのまま適用される。タグは既存と同様、各行の末尾（hashの後）に表示され続ける。

## Requirements

### Requirement 1: デフォルト表示のMarkdownチェックリスト構文化
**Objective:** As a sava user who pastes `list` output into Markdown notes, I want each item rendered in valid GFM checklist syntax, so that it renders correctly as a checklist without manual editing.

#### Acceptance Criteria
1. When the List Command が未着手（未着手かつ未完了）のタスクを表示するとき、the List Command shall その行を `- [ ] ` で始まるGFMチェックリスト行として描画する。
2. When the List Command が着手中（着手済みかつ未完了）のタスクを表示するとき、the List Command shall その行を `- [ ] ` の直後にリテラルな `` `in-progress` `` トークンを続けた形で描画する。
3. When the List Command が完了済みのタスクを表示するとき、the List Command shall その行を `- [x] ` で始まる形で描画する。
4. When the List Command がメモを表示するとき、the List Command shall チェックボックス構文を含まない `- ` で始まる単純なMarkdown箇条書き行として描画する。

### Requirement 2: hashの一貫した表記とタグの扱い
**Objective:** As a sava user piping `list` output into other tools, I want the hash to appear in a consistent, parseable position and format on every line, so that I can reliably extract it regardless of item type.

#### Acceptance Criteria
1. When the List Command が表示対象アイテム（タスク・メモのいずれも）を描画するとき、the List Command shall そのアイテムのhashを行末の `` (`hash`) ``（バックティックで囲んだhashを括弧で囲む形式）として配置する。
2. The List Command shall 未着手・着手中・完了・メモのすべての行種別において、hashの表記形式と位置を同一に保つ。
3. Where アイテムがタグを持つ場合、the List Command shall そのタグをhashトークンの後に、既存のタグ表記形式のまま表示する。

### Requirement 3: 複数行contentの先頭行要約
**Objective:** As a sava user, I want multi-line memo/task content summarized to its first line by default, so that each displayed item still occupies exactly one line for reliable pipe processing.

#### Acceptance Criteria
1. When アイテムのcontentが複数行であり、かつ `--full-text` フラグが指定されていないとき、the List Command shall そのcontentの先頭行のみを表示する。
2. While `--full-text` フラグが指定されていない間、the List Command shall 表示対象の各アイテムにつき、出力を必ず1行に保つ。

### Requirement 4: `--full-text` フラグによる全文表示
**Objective:** As a sava user, I want an explicit option to see an item's complete content including all lines, so that I can read multi-line notes in full when needed.

#### Acceptance Criteria
1. When `--full-text` フラグが指定され、かつアイテムのcontentが複数行であるとき、the List Command shall そのcontentの全文（すべての行）を表示する。
2. Where `--full-text` フラグが指定されている場合、the List Command shall 表示対象アイテムのcontentを1行に切り詰めない。

### Requirement 5: 既存の可視性ロジック・他表示モードとの非干渉
**Objective:** As a sava user relying on existing `list` behavior, I want this formatting change to leave visibility rules and other display modes unaffected, so that my existing workflows (filters, stats, file listing) keep working the same way.

#### Acceptance Criteria
1. The List Command shall 今回の新しい行フォーマットを、`-s`/`--stat` も `-a`/`--all` も指定されていない `sava list [date]` にのみ適用する。
2. Where `-s`/`--stat` フラグが指定されている場合、the List Command shall 既存のサマリー出力フォーマットを、本フォーマット変更の影響を受けずに維持する。
3. Where `-a`/`--all` フラグが `-s`/`--stat` なしで指定されている場合、the List Command shall 既存のファイル名一覧の出力フォーマットを、本フォーマット変更の影響を受けずに維持する。
4. The List Command shall どのアイテムを表示するか（完了済みタスクの既定非表示、`--full`、`--task`、`--tag` によるフィルタ）を、list-task-visibilityで定義された既存のルールのまま、本フォーマット変更による影響なく決定し続ける。
