# Requirements Document

## Project Description (Input)
今日書いたメモ・タスクのcontentに誤字や言い回しの誤りがあったとき、修正する手段が無い。現状、削除して再追加するとhashが変わり、タスクの場合は`StartedAt`/`Closed`などの状態も失われてしまう。

この課題に対し、以下の変更を行う:
- `sava edit <hash> <new content>` で、contentを直接指定して書き換えられるようにする（`add`/`todo`と同じ直接指定スタイル）
- `sava edit <hash>`（content省略時）は、現在のcontentを書き込んだ一時ファイルを `$EDITOR`（`$VISUAL`をフォールバック）で開き、保存・終了後の内容で書き換える（`git commit`がメッセージ省略時にエディタを開く挙動と同じ発想）
- エディタでの保存時、内容が編集前のcontentと完全に同一、または空文字列になった場合は、エラーではなく「中断」として扱う（`git commit`の空メッセージ中断と同様。これはcontentの妥当性検証ではなく、エディタ上で何も書かずに保存してしまった事故を防ぐための、アイテムのライフサイクル上自然な振る舞い）
- 直接指定モードでは、空文字列を含めcontentの妥当性検証は行わない（`add`/`todo`が現在検証を行っていないことと一貫させる）
- 対象アイテムは当日ログ内のメモ・タスクいずれも、状態に関わらず編集可能
- hash・CreatedAt・Closed/StartedAt等の状態は変更せず、UpdatedAtだけ更新する
- 過去の日のアイテムは編集できない（既存の「過去ログは不変」方針を維持）
- 編集履歴・元contentの保持は行わない（UpdatedAtの更新で十分）

対象は `internal/log`/`internal/command`/`cmd/sava` に新設する `edit` コマンド一式（エディタ起動・一時ファイル管理を含む）。既存の`tag`（タグ編集）、`start`/`end`（状態変更）のロジックは変更しない。

詳細な背景・スコープ・制約は `.kiro/specs/edit-command/brief.md`（discoveryセッションの成果物）を参照。

## Introduction
当日ログのメモ・タスクのcontentを、直接指定またはエディタ起動のいずれかの方法で書き換えられる`edit`コマンドを追加する。hashや状態（完了・着手中等）は保持し、UpdatedAtのみ更新する。過去ログは対象外とし、既存の「過去ログは不変」という方針を維持する。

## Boundary Context (Optional)
- **In scope**: 当日ログ内アイテムのcontent編集（直接指定モード・エディタモードの両方）
- **Out of scope**: 過去ログの編集、タグ編集（既存`tag`コマンド）、状態変更（既存`start`/`end`コマンド）、編集履歴・元contentの保持
- **Adjacent expectations**: 編集はUpdatedAtのみを変更し、hash・CreatedAt・Closed・StartedAt・Tagsといった他のフィールドは既存コマンドの責務のまま変更しない

## Requirements

### Requirement 1: 直接指定モードでのcontent編集
**Objective:** As a sava user, I want to directly specify new content when editing an item, so that I can quickly fix typos without opening an editor.

#### Acceptance Criteria
1. When `sava edit <hash> <new content>` のように新しいcontentが指定されたとき、the Edit Command shall 当日ログ内の対象アイテムのcontentを指定された内容に書き換える。
2. When 指定された新しいcontentが空文字列であっても、the Edit Command shall 書き換えを拒否せず、そのまま適用する。
3. The Edit Command shall 編集後も、対象アイテムのhash・CreatedAt・Closed・StartedAtを変更しない。
4. When contentの書き換えが行われたとき、the Edit Command shall 対象アイテムのUpdatedAtを現在時刻に更新する。

### Requirement 2: エディタモードでのcontent編集
**Objective:** As a sava user, I want to edit an item's content in my preferred text editor when I omit the new content, so that I can comfortably edit multi-line memos.

#### Acceptance Criteria
1. When `sava edit <hash>` のように新しいcontentを指定せずに実行されたとき、the Edit Command shall 対象アイテムの現在のcontentが書き込まれた状態のファイルを、エディタで編集できるようにする。
2. Where `$EDITOR` 環境変数が設定されている場合、the Edit Command shall そのエディタを起動する。
3. Where `$EDITOR` が設定されておらず `$VISUAL` が設定されている場合、the Edit Command shall `$VISUAL` で指定されたエディタを起動する。
4. Where `$EDITOR` も `$VISUAL` も設定されていない場合、the Edit Command shall `vi` を起動する。
5. When エディタが正常終了し、保存後の内容が編集前のcontentと異なり、かつ空文字列でないとき、the Edit Command shall その内容で対象アイテムのcontentを書き換える。
6. When エディタが正常終了したが、保存後の内容が編集前のcontentと完全に同一であるとき、the Edit Command shall 編集を中断したことを示すメッセージを表示し、エラーにはせず正常終了する。
7. When エディタが正常終了したが、保存後の内容が空文字列であるとき、the Edit Command shall 編集を中断したことを示すメッセージを表示し、エラーにはせず正常終了する。
8. If エディタがゼロ以外の終了コードで終了したとき、the Edit Command shall 編集を中断し、エラーを返す。

### Requirement 3: 当日ログのみを対象とする境界
**Objective:** As a sava user, I want edit to only affect today's still-mutable log, so that past, frozen logs stay immutable and the system stays simple.

#### Acceptance Criteria
1. The Edit Command shall 当日ログに存在するアイテムのみを編集対象とする。
2. When 指定されたhashが当日ログに存在しないとき（過去日のhashまたは存在しないhashのいずれであっても）、the Edit Command shall エラーを返す。

### Requirement 4: 対象アイテムの種別・状態への非依存
**Objective:** As a sava user, I want to edit a memo or a task regardless of its lifecycle state, so that fixing a typo never forces me to also change its status.

#### Acceptance Criteria
1. The Edit Command shall メモ・タスクいずれのアイテムも編集対象とする。
2. While 対象アイテムがどのような状態（未着手・着手中・完了）であっても、the Edit Command shall そのcontentを編集可能にする。
