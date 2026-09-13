# Requirements Document

## Introduction
`sava add`/`sava todo`は現在、contentの空文字列チェックを行っておらず、`sava add ""`のように空文字列を直接指定すると空のメモ・タスクが記録されてしまう。本仕様は、`add`/`todo`の直接指定（引数指定）モードにおいて空文字列のcontentを拒否し、意図しない空アイテムの記録を防ぐことを目的とする。

## Boundary Context (Optional)
- **In scope**: `sava add`/`sava todo`の直接指定（コマンドライン引数によるcontent指定）モードにおける空文字列検証
- **Out of scope**: `sava edit`の直接指定モード（既存の設計判断により空文字列を許容する方針のまま据え置き、本仕様では変更しない）。content省略時に`$EDITOR`を起動する新規フロー（別チケットのスコープ）
- **Adjacent expectations**: 既存のタグ検証（トリム後に空文字列となる場合はエラーとする）と同じ「空白のみで構成された文字列は空とみなす」判定基準に合わせる

## Requirements

### Requirement 1: 直接指定contentの空文字列検証
**Objective:** As a sava のユーザー, I want `add`/`todo`コマンドの直接指定モードで空のcontentを拒否してほしい, so that 意図しない空メモ・空タスクが記録されるのを防げる

#### Acceptance Criteria
1. When ユーザーが`sava add`を空文字列のcontentで実行した場合, the Add Command shall エラーを返し、メモをログに追加しない。
2. When ユーザーが`sava todo`を空文字列のcontentで実行した場合, the Todo Command shall エラーを返し、タスクをログに追加しない。
3. When 指定されたcontentが空白文字のみで構成されている場合, the Add Command and the Todo Command shall 空文字列の場合と同様に扱い、エラーを返す。
4. If contentの検証に失敗した場合, then the Add Command and the Todo Command shall タグ付け（`--tag`）やタスク開始（`--start`）など、その他のオプション処理を実行しない。

### Requirement 2: 非空contentの既存動作維持
**Objective:** As a sava のユーザー, I want 空でないcontentを指定したときは従来通りメモ・タスクが記録されること, so that 今回の変更によって既存の正常系が壊れない

#### Acceptance Criteria
1. While contentが空文字列でも空白のみでもない場合, the Add Command shall 従来通りメモをログに追加する。
2. While contentが空文字列でも空白のみでもない場合, the Todo Command shall 従来通りタスクをログに追加する。
