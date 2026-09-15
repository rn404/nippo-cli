# Implementation Plan

- [x] 1. `--full` → `--full-list` へのコア実装（CLIフラグ・内部オプション・README）
  - `internal/command/command.go`: `ListOptions` の `Full` フィールドを `FullList` にリネームする（doc commentはそのまま維持）。`listOneDay` 内の `if opts.Full { ... }` を `if opts.FullList { ... }` に更新する
  - `cmd/sava/commands.go`: `newListCommand` 内のフラグ登録を `cmd.Flags().BoolVarP(&opts.Full, "full", "f", false, "include completed tasks")` から `cmd.Flags().BoolVarP(&opts.FullList, "full-list", "f", false, "include completed tasks")` に変更する（ショートハンド `-f` とヘルプ文言はそのまま維持）
  - `README.md`: `--full` の使用例（97, 99, 101行目）をすべて `--full-list` に更新する
  - Observable: `go build ./...` が成功し、`sava list --help` の出力に `--full-list` のみが表示され `--full` は表示されない
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 2.1, 2.2, 3.1, 3.2_
  - _Boundary: ListCommandFlags, ListOptionsVisibility_

- [x] 2. 既存テストの更新と回帰テストの追加
- [x] 2.1 (P) internal/command テストのフィールド名更新
  - `internal/command/command_test.go` 内の全 `ListOptions{..., Full: true, ...}` 参照（同ファイル内7箇所: `TestListToday_FullShowsClosedTasksInOrder`、`TestListToday_TaskAndFullShowsAllTasksNoMemos`、`TestListToday_FullOrdersClosedGroupByCreatedAtAscending`、`TestListToday_FullHasNoGroupHeadersBetweenSections`、`TestListToday_TagFilterCombinesWithVisibility`、`TestListStatAndAll_UnaffectedByNewFlags` 内2箇所）を `FullList: true` に更新する
  - `TestListStatAndAll_UnaffectedByNewFlags` 内のエラーメッセージ文言（`"...unaffected by Full/TasksOnly/FullText"`、2箇所）を `FullList/TasksOnly/FullText` に更新する
  - Observable: `go test ./internal/command/...` が全てパスする
  - _Requirements: 1.3, 1.4_
  - _Boundary: internal/command/command_test.go_

- [x] 2.2 (P) CLI統合テストの更新と未定義フラグの回帰テスト追加
  - `cmd/sava/root_test.go` の `TestListFullAndTaskFlags` 内、`mustExecute(t, "list", "--full")` および `mustExecute(t, "list", "--full", "--task")` の `"--full"` を `"--full-list"` に更新し、関数のdoc comment（`--full`/`-f` の表記）もあわせて更新する
  - `sava list --full`（旧フラグ名）を実行した際に `root.Execute()`（`execute` ヘルパー経由）がエラーを返すことを検証する新規テストケースを追加する
  - `TestFullTextFlag` に、`--full-list --full-text` を同時指定した場合に両方の効果（完了済みタスクを含める・contentを全文表示する）が独立して現れることを検証するケースを追加する
  - Observable: `go test ./cmd/sava/...` が全てパスし、`sava list --full` がエラーになることが新規テストで確認できる
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 2.1_
  - _Boundary: cmd/sava/root_test.go_
