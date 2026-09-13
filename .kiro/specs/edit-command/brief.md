# Brief: edit-command

## Problem
今日書いたメモ・タスクのcontentに誤字や言い回しの誤りがあったとき、修正する手段が無い。現状、削除して再追加するとhashが変わり、タスクの場合は`StartedAt`/`Closed`などの状態も失われてしまう。

## Current State
`internal/command`には`Start`/`End`/`Tag`など、当日ログのみを対象にした変更系コマンドが既に存在し、いずれも`ensureToday`→`log.XXX`（hash検索・フィールド更新・`UpdatedAt`更新）→`logfile.Update`という共通パターンに従っている。content自体を書き換える手段は無い。

## Desired Outcome
- `sava edit <hash> <new content>` で、contentを直接指定して書き換えられる（`add`/`todo`と同じ、直接指定スタイル）
- `sava edit <hash>`（contentを省略）では、現在のcontentを書き込んだ一時ファイルを `$EDITOR`（`$VISUAL`をフォールバック）で開き、保存・終了後の内容で書き換える（`git commit`がメッセージ省略時にエディタを開く挙動と同じ発想）
- エディタでの保存時、内容が編集前のcontentと完全に同一、または空文字列になった場合は、`git commit`の空メッセージ中断と同様に更新を中断する（エラー終了ではなく、中断メッセージを出して正常終了）
- 直接指定モードでは、空文字列を含めcontentの妥当性検証は行わない（`add`/`todo`と一貫させる）
- 対象アイテムは当日ログ内のメモ・タスクいずれも、状態に関わらず編集可能
- hash・CreatedAt・Closed/StartedAt等の状態は変更せず、UpdatedAtだけ更新される
- 過去の日のアイテムは編集できない（既存の「過去ログは不変」方針を維持）
- 編集履歴・元contentの保持は行わない（UpdatedAtの更新で「編集された」という事実が分かれば十分、という合意済み）

## Approach
既存の`Start`/`AddTags`と同じ当日ログ取得・永続化パターンを踏襲しつつ、contentの取得元を2系統（直接指定／エディタ）に分ける:
- `internal/log` に `Edit(l *model.Log, hash, newContent string) (model.Item, error)` を追加（hash検索→content書き換え→UpdatedAt更新。中断判定は呼び出し側で行うため、ここでは単純な書き換えのみ）
- `internal/command` に `Edit(w io.Writer, dir, hash, newContent string) error` を追加（`ensureToday`→`log.Edit`→永続化→確認出力）。新規content取得手段（エディタ起動）は、CLI層でcontentを確定させてからこの関数に渡す形にし、`command.Edit`自体はエディタの有無を知らない設計にする
- エディタ起動・一時ファイルの読み書き・中断判定（無変更／空）を新しいヘルパーに切り出す（配置先は design フェーズで決定: `cmd/sava`内、または新規パッケージ）
- `cmd/sava` に `edit` サブコマンドを追加。`content` 引数を任意にし（`cobra.RangeArgs(1, 2)`等）、省略時にエディタ起動ヘルパーを呼ぶ

## Scope
- **In**: 当日ログのメモ・タスクのcontent編集コマンド（直接指定モード＋エディタモード）
- **Out**: 過去ログの編集、タグ編集（既存`tag`コマンドの役割）、状態変更（既存`start`/`end`の役割）、編集履歴・元contentの保持

## Boundary Candidates
- `internal/log.Edit`（ドメインロジック：hash検索・content書き換え）
- `internal/command.Edit`（当日ログの取得・永続化・確認出力。contentの取得元には関与しない）
- エディタ起動・一時ファイル管理・中断判定（新しい責務。直接指定モードのみのテストではこの責務は不要なため、境界を分けておく）
- `cmd/sava` の `edit` サブコマンド登録（content引数の有無で分岐）

## Out of Boundary
- 過去ログ（frozen）の編集
- タグ・状態（started/closed）の変更
- 編集履歴・元contentの保持

## Upstream / Downstream
- **Upstream**: 既存の `ensureToday`/`logfile.Update`/`model.Item`
- **Downstream**: 特になし

## Existing Spec Touchpoints
- **Extends**: なし（新規境界）
- **Adjacent**: なし（`list-task-visibility`/`list-output-format`は表示に関する既存specで、本specのcontent編集とは無関係）

## Constraints
- 直接指定モードでは、空文字列を含めcontentの妥当性検証は行わない（`add`/`todo`が現在検証を行っていないことと一貫させる）
- エディタモードでの無変更・空文字列保存は、エラーではなく「中断」として扱う（git commit同様。contentの妥当性検証ではなく、アイテムのライフサイクル上自然な事故防止の振る舞い）
- hashが当日ログに存在しない場合（過去日のhash・存在しないhashいずれも含む）はエラーとする。過去日か存在しないかの区別が必要かは `/kiro-spec-requirements` で詳細化する
- `$EDITOR`/`$VISUAL`が未設定の場合のフォールバック方針（例: `vi`を既定にする、あるいは明示的にエラーにする）は design フェーズで決定する
- エディタはユーザーの実ターミナル（stdin/stdout/stderr）にそのまま接続する必要がある
- 一時ファイルは、エディタの起動失敗時も含めて確実に削除する
