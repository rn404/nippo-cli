# Requirements Document

## Project Description (Input)
sava list のデフォルト表示（--full-text 未指定時）における複数行content の要約ロジックを変更する。現在は content の最初の改行が出現するまでを1行として表示しているが、これを「改行に関わらず一定の文字数（デフォルト50文字）まで表示する」方式に変更する。具体的には、default表示時: (1) content内の改行はすべて半角スペース1つに置換してから、(2) 置換後の文字列を先頭から50文字（rune単位）で切り詰め、(3) 切り詰めが発生した場合のみ末尾に省略記号「…」を付与する。50文字以内で改行を含まないcontentは、改行置換のみ行われ切り詰め・省略記号は付かない。

さらに、`sava list --full-text | grep 'word'` で該当アイテムが確実に1件・1行でヒットする（hashを含む行全体がマッチする）ことも要件とする。この要件を満たすため、--full-text 指定時についても、content内の改行はすべて半角スペース1つに置換し、常に1アイテム=1行を保証する（当初案では --full-text の改行はそのまま維持するとしていたが、grep要件と矛盾するため撤回し、こちらに変更する）。ただし --full-text では50文字への切り詰め・省略記号の付与は行わない（content全文を、改行のみスペースに置換した1行として表示する）。

目的は、grep/awk等でのpipe処理のしやすさ（`sava list`・`sava list --full-text` のどちらでも1アイテム=1行が常に保証される)と、Markdownノート貼り付け時の要約の一貫性を高めること。対象は list-output-format spec で導入された internal/view.Timeline の描画ロジック（デフォルト分岐・--full-text分岐の両方）。

## Introduction
`sava list [date]`（stat無しの日次表示）における複数行contentの表示ロジックを、「最初の改行までを1行とする」方式から「改行を除去したうえで一定の文字数（デフォルト表示は50文字、超過時は省略記号付き）または全文（--full-text指定時）を表示する」方式に変更する。これにより、デフォルト表示・`--full-text`表示のどちらでも常に1アイテム=1行が保証され、Markdownノートへの貼り付けと`grep`/`awk`等のpipe処理の両方で安定した挙動が得られる。

## Boundary Context (Optional)
- **In scope**: `sava list [date]`（stat無しの日次表示）のデフォルト表示における content 部分の要約ロジック（改行除去・文字数切り詰め・省略記号付与）、`--full-text` 指定時の content 部分の表示ロジック（改行除去のみ、切り詰めなし）
- **Out of scope**: `-s`/`--stat` の出力フォーマット、`-a`/`--all`（stat無し）の出力フォーマット、`--full-list`/`--task`/`--tag` による可視性・フィルタロジック自体、チェックリスト接頭辞・hashの表記位置・タグの表記位置（list-output-formatで確定済みのまま変更しない）、文字数上限（50文字）をユーザーが変更できるオプションの追加
- **Adjacent expectations**: 本仕様は list-output-format spec で定義された行フォーマット（チェックリスト接頭辞、hashの位置・表記、タグの位置）の上に、content部分の表示ロジックのみを変更する。list-task-visibility・rename-full-flagで定義された可視性ルール・フラグ名はそのまま前提とする。

## Requirements

### Requirement 1: デフォルト表示における複数行contentの文字数ベース要約
**Objective:** As a sava user who reads or pipes `sava list` output, I want multi-line content summarized to a fixed character length regardless of where line breaks occur, so that every displayed item consistently occupies exactly one line and the summary is not truncated arbitrarily early just because a line break happens to appear near the start.

#### Acceptance Criteria
1. When the List Command renders an item's content in default display (`--full-text` not specified), the List Command shall replace each run of one or more consecutive newline characters in the content with a single half-width space before determining what to display.
2. When the newline-replaced content exceeds 50 characters, the List Command shall display only the first 50 characters and append an ellipsis ("…") immediately after them.
3. When the newline-replaced content is 50 characters or fewer, the List Command shall display the newline-replaced content in full without appending an ellipsis.
4. While `--full-text` is not specified, the List Command shall keep the output for each displayed item to exactly one line.

### Requirement 2: `--full-text` 指定時の改行除去による1行化
**Objective:** As a sava user piping `sava list --full-text` into `grep`/`awk`, I want each item's full content rendered on a single line with line breaks removed, so that a keyword search always matches the item's complete line, including its hash, regardless of which part of the multi-line content the keyword appears in.

#### Acceptance Criteria
1. When `--full-text` is specified and the List Command renders an item's content, the List Command shall replace each run of one or more consecutive newline characters in the content with a single half-width space.
2. When `--full-text` is specified, the List Command shall display the newline-replaced content in full, without truncating it by character count and without appending an ellipsis.
3. While `--full-text` is specified, the List Command shall keep the output for each displayed item to exactly one line.

### Requirement 3: 既存の可視性ロジック・他表示モード・行フォーマット要素との非干渉
**Objective:** As a sava user relying on existing `list` behavior, I want this content-rendering change to leave item selection, other display modes, and the rest of the line format unaffected, so that my existing workflows (filters, stats, file listing) keep working the same way.

#### Acceptance Criteria
1. The List Command shall apply this content-rendering change only to the content portion of `sava list [date]` when neither `-s`/`--stat` nor `-a`/`--all` is specified.
2. Where `-s`/`--stat` is specified, the List Command shall keep the existing summary output format unaffected by this change.
3. Where `-a`/`--all` is specified without `-s`/`--stat`, the List Command shall keep the existing file-listing output format unaffected by this change.
4. The List Command shall continue to decide which items are displayed (default exclusion of completed tasks, and filtering by `--full-list`, `--task`, `--tag`) using the existing rules, unaffected by this change.
5. The List Command shall keep the checklist prefix, the hash's notation and position, and the tags' position unchanged by this change.
