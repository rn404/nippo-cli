# Requirements Document

## Project Description (Input)
`sava list --full`（完了済みタスクを含める）は、フラグ名が曖昧（「fullって何に対してのfullか分かりにくい」）という指摘があった。当時は対応せず、将来の検討事項として保留した（`.kiro/specs/list-task-visibility/brief.md` Constraints参照）。その後`edit-command`specで`--full-text`（全文表示）フラグが追加され、命名の衝突感・曖昧さがさらに増している。

`sava`の利用者（開発者本人を含む）は、`--full`と`--full-text`という似た名前の2つのフラグを区別しづらい状況に置かれている。

`--full`を`--full-list`等、より明確な名前にリネームする（既存ユーザーへの破壊的変更として扱う）。

Linear issue: [RN4-8](https://linear.app/rn404/issue/RN4-8/full-フラグの命名変更)

## Introduction
`sava list` の `--full`（完了済みタスクを含める）フラグは、後から追加された `--full-text`（全文表示）フラグと名前が似ているため、両者を区別しづらいという指摘がある。この課題に対し、`--full` を `--full-list` にリネームし、`--full-text` との名前の衝突感を解消する。既存ユーザーには破壊的変更として扱い、旧フラグ名は廃止する。

## Boundary Context (Optional)
- **In scope**: `sava list` の `--full` フラグの名称変更（`--full-list` への変更）、旧 `--full` フラグ名の廃止、ヘルプ文言・README の該当箇所の更新
- **Out of scope**: `--full-text` フラグ自体の名称・挙動、`--full`（新名称では `--full-list`）が制御する可視性ロジック（完了済みタスクを含めるかどうかの判定自体）の変更、`--task`/`--tag`/`--stat`/`--all` 自体の挙動変更
- **Adjacent expectations**: `--full-list` は `--task`・`--full-text`・`--tag` との組み合わせにおいて、現行の `--full` が持つ既存の組み合わせ挙動（list-task-visibility / list-output-format で定義済み）をそのまま引き継ぐ。ショートハンド `-f` は `--full-list` に対して維持される。

## Requirements

### Requirement 1: `--full-list` フラグの導入
**Objective:** As a sava user, I want an unambiguous flag name for including completed tasks in `list` output, so that I can clearly distinguish it from `--full-text`.

#### Acceptance Criteria
1. When ユーザーが `sava list` に `--full-list` フラグを指定したとき、the List Command shall 完了済みタスクを含めた一覧を表示する（現行の `--full` と同じ挙動）。
2. When ユーザーが `sava list` に `-f` フラグを指定したとき、the List Command shall `--full-list` を指定した場合と同じ挙動をする。
3. Where `--full-list` と `--task` が同時に指定されている場合、the List Command shall 完了済みタスクと未完了タスクのみ（メモを含めず）を表示する（現行の `--full` と `--task` の組み合わせと同じ挙動）。
4. Where `--full-list` と `--full-text` が同時に指定されている場合、the List Command shall 両フラグそれぞれの既存の独立した効果（完了済みタスクを含める、およびcontentを全文表示する）を適用する。

### Requirement 2: 旧 `--full` フラグ名の廃止
**Objective:** As a sava maintainer, I want the ambiguous `--full` flag name removed, so that users are not confused between two similarly named flags going forward.

#### Acceptance Criteria
1. When ユーザーが `sava list` に `--full` フラグを指定したとき、the List Command shall 未定義フラグとしてエラーを報告する。
2. The List Command shall `--full` という名前のフラグを提供しない。

### Requirement 3: ヘルプ・ドキュメントの整合
**Objective:** As a sava user consulting command help or the README, I want the documentation to reflect the renamed flag, so that I am not misled by stale references to `--full`.

#### Acceptance Criteria
1. When ユーザーが `sava list --help` を実行したとき、the List Command shall フラグ名として `--full`ではなく `--full-list` を表示する。
2. The sava README shall `--full` ではなく `--full-list` を使用例・説明として記載する。
