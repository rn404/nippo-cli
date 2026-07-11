
このCLIはなにか？

もともとの目的はユーザーの工数管理のためのものである
あとからカレンダーをスキャンしてその分のログも差し込んで統計出せると嬉しい気がする
そうするともう少し MemoItem ひとつひとつのライフサイクルを見直したほうがいいかも？

今となっては AI Agent の外部記憶としての補助ツール、ユーザーとのコラボレーション補助ツールとしての小さい部品となれば嬉しい気がする

## 既存のコマンド
```
sava add <contents>        # タスク追加
sava add -m <contents>     # メモ追加
sava end <hash>            # タスク完了
sava del <hash>            # アイテム削除
sava list [date]           # 当日（または指定日）のログ表示
sava list -s [date]        # 統計表示
sava list -a               # 全ログファイル一覧
sava list -a -s            # 全ログファイル統計（10 件超で確認プロンプト）
sava clear                 # 保持期間（30 日）超過のログ削除
sava clear -a              # 全ログ削除（確認プロンプトあり）
sava help                  # ヘルプ
```


`sava end` のときに自動でメモを追加する
* タスクの開始と終了時の記録のためにメモしたい、と思ったがupdateAtをとっているので不要そう
* むしろ、タスク開始のコマンドも足していいかもしれない `sava start <hash>`
* タスクの記録時に同時に作業着手するオプション足すといいかも
* そうすると中断も欲しいか？ 厳密な作業時間の計測をしたい人向けなのでこれは優先度が低い

`sava diff <hash>...<hash>` の実装をしたい
* もとは hash 同士指定したらその間の時間をアウトプットしてもらえる、工数管理向けの機能で考えていた
* 詳細仕様を検討してもいいかもしれない

MemoItem への tag づけ機能
*  複数の tag を自由につけることができるようにし、アイテムのフィルタリングなどもできるようにする
* 存在する tag は後で管理するのが大変になるので、tagづけするコマンドが走ったらインデックス相当のファイルを吐き出しておくといいかも
* 複数タグ付けができる
* list時には曖昧検索、AND, OR検索をサポートしたい

---

## 再設計の決定事項 (2026-07-11)

### 方針
* データモデルは「フィールド追加」方式を採用する
  * `Item` に `startedAt` / `tags` を `omitempty` で追加し、旧 Deno 実装の JSON との互換を維持する
  * 状態は `closed` / `startedAt` の組合せで導出する（status enum への移行は見送り）
* タグ付けは専用コマンド (`sava tag`) と `add -t` の両方をサポートする
* 実装順は start → tag → diff

### コマンド体系（目標）
```
sava add <contents>               # タスク追加
sava add <contents> -s            # タスク追加 + 即着手
sava add <contents> -m            # メモ追加
sava add <contents> -t <tag>      # タグ付きで追加（-t 複数指定可）
sava start <hash>                 # タスク着手（startedAt を記録）
sava end <hash>                   # タスク完了
sava del <hash>                   # アイテム削除
sava tag <hash> <tag>...          # タグ付与（インデックス更新）
sava tag -d <hash> <tag>...       # タグ除去
sava tag --list                   # 既存タグ一覧
sava list [date] [-t <tag>]       # タグフィルタ（複数は AND、--or で OR）
sava diff <hashA>...<hashB>       # 2アイテム間の経過時間
sava clear [-a]                   # 既存のまま
```

### インデックスファイル
* `~/.log/sava/index.json` に `tag → [{date, hash}]` と `hash → date` の逆引きを保持
* あくまでキャッシュ扱い: 壊れたら全ログから再構築できる
* `list` のタグフィルタと `diff` の日またぎ hash 解決の両方が乗る

### diff の仕様（Phase 3 で確定）
* `sava diff <hashA>...<hashB>`（`..` 区切り、ハッシュ2引数もサポート）
* `A.createdAt` と `B.createdAt` の距離（絶対値）を出力。順序を入れ替えても結果は同じ
* 1タスク内の作業時間は `startedAt` / `updatedAt` で足りるため、diff は「アイテム間の距離」を測る道具と位置づける
* hash はインデックスの `hash → date` 逆引きで日またぎ解決する
  * インデックスにない・古い場合は一度だけ再構築してリトライ（self-heal）
* 経過時間の表示は非ゼロ成分のみ: `1d 2h 30m`, `45m 10s`, `0s` など

### タグの仕様（Phase 2 で確定）
* タグは任意のアイテム（Task / Memo）に付けられる
* タグは trim され重複排除される。空文字と空白を含むタグはエラー
* `sava tag` が操作できるのは当日ログのアイテムのみ（end / del と同じ制約）
* `list -t` は単日表示専用。`-a` との併用はエラー（インデックス活用は今後の課題）
* インデックスはタグ操作時に全ログから再生成される。del / clear 後は古くなり得るが、
  読む側（`tag --list`、将来の `diff`）が再生成・self-heal する方針
* `clear -a` は index.json も削除する

### レガシー互換の廃止 (2026-07-11)
* Go 移行完了により「旧 Deno 実装との互換」という枠組みを廃止
* 今後の不変条件は「旧バージョンの自分が書いたファイルを読めること」
  （新フィールドは omitempty で追加する、で満たされる）
* `testdata/legacy-samples/` は現行フォーマットの仕様サンプル
  `testdata/log-format/` に置き換え、round-trip テストをフォーマット仕様テストとして再定義

### 進捗
* [x] Phase 1: `startedAt` + `sava start` + `add -s`（list に `[>]` マーク表示）
* [x] Phase 2: タグ（`tag` コマンド、`add -t`、`index.json`、`list -t` / `--or` フィルタ）
* [x] Phase 3: `diff`（index の hash → date 逆引きによる日またぎ解決、self-heal つき）
* [x] レガシー（Deno 互換）の廃止とテストの現行フォーマット移行
* 中断 (pause) は優先度低のため見送り
* `end` 時の自動メモは updatedAt があるため不要と判断