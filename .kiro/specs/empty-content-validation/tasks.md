# Implementation Plan

- [x] 1. content空文字列検証の実装とユニットテスト
  - `internal/log`に、既存の`ErrEmptyTag`と同型のsentinel error（`ErrEmptyContent`）を追加する
  - `Add`関数の先頭でcontentをトリムし、トリム後に空文字列であればItemを作成せず`ErrEmptyContent`を返すようにする（Task/Memo共通で同じ基準を適用する）
  - トリム後に空でないcontentは、既存動作通りトリムせず元の文字列のままItemとして保存する
  - 空文字列・空白のみのcontentを渡した場合（Task/Memoそれぞれ）に`ErrEmptyContent`が返り、ログの`Items`が変化しないことを検証するユニットテストを追加する
  - 前後に空白を含む非空contentを渡した場合、Itemが作成され、contentがトリムされずそのまま保存されることを検証するユニットテストを追加する
  - Observable: `go test ./internal/log/...`で新規テストがパスし、空文字列・空白のみのcontentが`ErrEmptyContent`で拒否される一方、非空contentは従来通り（トリムなしで）保存されることが確認できる
  - _Requirements: 1.1, 1.2, 1.3, 2.1, 2.2_
  - _Boundary: log.Add_

- [ ] 2. command層での副作用スキップの統合テスト
  - `command.Add`に空文字列contentを渡した際、エラーが呼び出し元にそのまま伝播し、ログファイルへの書き込みが発生しないことを検証する統合テストを追加する
  - `command.Todo`にタスク開始・タグ付けオプションを同時に指定した状態で空文字列contentを渡した際、エラーが伝播し、タスク開始・タグ付け・タグindexの再構築が一切実行されないことを検証する統合テストを追加する
  - Observable: `go test ./...`がリポジトリ全体でgreenになり、空文字列content指定時に`command.Add`/`command.Todo`がログファイルを変更せず、タスク開始・タグ付けも実行していないことがテストで確認できる
  - _Requirements: 1.1, 1.2, 1.4_
  - _Boundary: command.Add, command.Todo_
