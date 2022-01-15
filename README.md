## motivation
- cli ツール作りたい
- 独り言メモツールしたい(日報補助ツール)

## command 暫定
アプリの名前なににするかまだ未定

### Basic usage
```
command 'Something todo'
タスクの追加

command -m 'Something todo'
メモの追加

command -f <hash>
指定した todo を終了させる

command -d <hash>
指定した todo を削除する

```
- hash である必要あるんだろうか
  - 全部のログから特定の一つを削除すると考えると、hash が安全
- 数字で操作できたほうが楽？
  - なんか特定の日を削除するエイリアスあってもいいかも
  - 当日のやつは `today{0}` みたいな指定できるとかっこいい
  - 特定日時は `2021-12-22{0}` でどうだろう？

### Task Index
```
command list
その日のログを表示する

command -a list
保存されている全部のログを表示する

command -s list
保存されているサマリーを表示する

command -s -a list
これで全ログのサマリーを出す(想定)
```

### Other Options
```
command freeze
更新させなくするやつ
デフォルトで当日のログを凍結

command freeze 2021-12-1
特定の日を指定する

command -r freeze
解凍する
使い方はオプションなしと一緒で指定もできる

command edit
(TBD) 歴史改ざん

command clean
30日以上前のやつは削除する(デフォルト挙動)

command -a clean
全削除(確認ありだと嬉しいな)
```

## 一日のイメージ

```
command 'Aを実装する'
command 'Bを実装する'
command -m 'AAAはやらないことになった'
command -m 'BBBの実装はもう少し考えたほうがいいかも'
command 'Cのレビューをする'
command list // 確認してる
command -f <hash> // 終わったタスクを終了
command -d <hash> // 必要なくなったものを削除
command freeze //退勤マン
command list // 振り返り
```

- 未完了のタスクを翌日分のタスクに回すコマンドもほしいかも？
- 未完了のタスクだけを表示するコマンドも欲しいかも
- 当初予定では、メモ間の時間も計算できるようにしたかった(作業時間)
