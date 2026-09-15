# Requirements Document

## Project Description (Input)
add/todoコマンドでcontent引数を省略したとき、internal/editor.Resolveを使って$EDITOR（無ければ$VISUAL、無ければvi）を起動し、新規メモ/タスクを書けるようにする。current=""を渡してeditor.Resolveを流用する。エディタで無変更または空文字列のまま保存した場合はedit-commandの中断ロジックと同様に処理を中断し、何も作成しない（既存のadd/todoの空文字列content検証ロジックとの整合性を取る：直接指定は空文字列を拒否、エディタ経由は空で保存したら中断）。関連Linear issue: RN4-9 https://linear.app/rn404/issue/RN4-9/addtodoのdollareditor対応

## Introduction
`sava add`/`sava todo`は現在、contentをコマンドライン引数で直接指定する方法しか提供していない。複数行にわたるメモや、その場で考えながら書きたいタスクであっても、シェル引数として1行で入力せざるを得ない。本仕様は、`add`/`todo`でcontent引数を省略したときにエディタ（`$EDITOR`、フォールバックで`$VISUAL`、さらに`vi`）を起動し、既存の`sava edit`のエディタモードと同じ操作感でメモ・タスクを新規作成できるようにする。

## Boundary Context (Optional)
- **In scope**: `sava add`/`sava todo`でcontent引数省略時にエディタを起動すること、保存内容に応じてメモ・タスクを作成または中断すること、エディタ経由で作成した場合の`--tag`/`--start`オプションとの統合
- **Out of scope**: `sava edit`のエディタ挙動自体の変更（既存仕様のまま）。`add`/`todo`の直接指定モードにおける空文字列検証ロジックの変更（既存仕様のまま）
- **Adjacent expectations**: エディタでの中断判定基準（保存内容が編集前の内容と同一、または空文字列であれば中断）は、既存の`sava edit`のエディタモードと同じ基準を用いる。空白文字のみで構成された保存内容も、直接指定モードの空文字列検証と同じ「空白のみ＝空」という基準に基づき中断として扱う

## Requirements

### Requirement 1: content省略時のエディタ起動
**Objective:** As a sava のユーザー, I want `add`/`todo`のcontent引数を省略したときにエディタが起動されること, so that シェル引数の制約を受けずに、複数行の内容やその場で考えながら書く内容を残せる

#### Acceptance Criteria
1. When ユーザーが`sava add`をcontent引数なしで実行した場合, the Add Command shall 空の内容が書き込まれた状態のファイルを、エディタで編集できるようにする。
2. When ユーザーが`sava todo`をcontent引数なしで実行した場合, the Todo Command shall 空の内容が書き込まれた状態のファイルを、エディタで編集できるようにする。
3. Where `$EDITOR`環境変数が設定されている場合, the Add Command and the Todo Command shall そのエディタを起動する。
4. Where `$EDITOR`が設定されておらず`$VISUAL`が設定されている場合, the Add Command and the Todo Command shall `$VISUAL`で指定されたエディタを起動する。
5. Where `$EDITOR`も`$VISUAL`も設定されていない場合, the Add Command and the Todo Command shall `vi`を起動する。

### Requirement 2: 保存内容に応じたアイテムの作成または中断
**Objective:** As a sava のユーザー, I want エディタで何も書かずに保存した場合は何も作成されないこと, so that 誤って空のメモ・タスクを残してしまう事故を防げる

#### Acceptance Criteria
1. When エディタが正常終了し、保存後の内容が空文字列でも空白文字のみでもないとき, the Add Command shall その内容でメモを当日ログに追加する。
2. When エディタが正常終了し、保存後の内容が空文字列でも空白文字のみでもないとき, the Todo Command shall その内容でタスクを当日ログに追加する。
3. When エディタが正常終了したが、保存後の内容が空文字列であるとき, the Add Command and the Todo Command shall メモ・タスクを作成せず、中断したことを示すメッセージを表示して正常終了する。
4. When エディタが正常終了したが、保存後の内容が空白文字のみで構成されているとき, the Add Command and the Todo Command shall 空文字列の場合と同様に中断として扱い、メモ・タスクを作成しない。
5. If エディタがゼロ以外の終了コードで終了したとき, the Add Command and the Todo Command shall メモ・タスクを作成せず、エラーを返す。

### Requirement 3: エディタ経由での既存オプションとの統合
**Objective:** As a sava のユーザー, I want エディタ経由で作成したメモ・タスクにも`--tag`や`--start`が直接指定モードと同じように効くこと, so that 入力方法を変えても既存オプションの挙動が変わらない

#### Acceptance Criteria
1. When エディタ経由の`sava add`が`--tag`付きで実行され、かつメモが作成されたとき, the Add Command shall 直接指定モードと同様に、作成したメモへ指定されたタグを付与する。
2. When エディタ経由の`sava todo`が`--tag`付きで実行され、かつタスクが作成されたとき, the Todo Command shall 直接指定モードと同様に、作成したタスクへ指定されたタグを付与する。
3. When エディタ経由の`sava todo`が`--start`付きで実行され、かつタスクが作成されたとき, the Todo Command shall 直接指定モードと同様に、作成したタスクを着手状態にする。
4. When エディタでの入力が中断された、またはエラーになったとき, the Add Command and the Todo Command shall `--tag`や`--start`などのオプション処理を実行しない。

### Requirement 4: 直接指定モードとの共存・境界
**Objective:** As a sava のユーザー, I want content引数を指定したときは従来通りエディタを起動せず即座に追加されること, so that 既存のワンライナーでの使い方が変わらない

#### Acceptance Criteria
1. While content引数が指定されている場合, the Add Command and the Todo Command shall 従来通りエディタを起動せず、指定された内容をそのまま処理する。
2. The Add Command and the Todo Command shall 直接指定モードにおける既存の空文字列検証（空文字列・空白のみのcontentを拒否する）の挙動を変更しない。
