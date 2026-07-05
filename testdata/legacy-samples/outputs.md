# Deno 版 CLI の出力サンプル（Phase 0 採取）

採取日: 2026-07-05 / Deno 2.4.3 / `HOME` を一時ディレクトリに向けて実測。
Go 版は表示テキストの完全一致を目指さず、**表示される情報の過不足がないこと**を基準とする（計画書「実装ポリシー」参照）。

## add / add -m

出力なし（正常終了、exit 0）。`~/.log/sava/yyyy-MM-dd.json` に追記される。

## list（当日）

```
    Today's logs are...

Task ->
- [ ] buy cabbage (17:43) [object Object]
- [ ] feed the shrimp (17:43) [object Object]

Memo ->
- shrimp looks happy today (17:43) [object Object]
```

- 時刻は `(HH:mm)` のローカルタイム 24 時間表記
- 行形式: `- [x| ] content (HH:mm) hash`（メモはチェックボックスなし）
- 末尾の hash は既存バグにより全アイテム `[object Object]`
- アイテムが 0 件のときは `There is no body...`

## list <date>

```
    Log for 2026-07-05 are...
```

以降は list と同じ。日付でない引数はエラー（現行はスタックトレース、exit 1）。

## list -s（当日の統計）

```
    Today's log stats are...

- 2026-07-05  Task: 2 (unfinished: 2), Memo: 1
```

- 日付の直後の `*` は freezed マーク（freezed 時のみ）

## list -a（ファイル一覧、日付昇順）

```
    The logs here are...

- 2026-05-01
- 2026-07-05
```

## list -a -s（全ファイル統計、10 件超で確認プロンプト）

```
    View all log statistics. There are 2 total.

- 2026-05-01  Task: 0 (unfinished: 0), Memo: 0
- 2026-07-05  Task: 0 (unfinished: 0), Memo: 0
```

## end <hash>

```
Finished!!
> buy cabbage (17:43)
```

- 最初に一致した未完了タスクを閉じる（既存バグにより hash は全件同一のため、実質先頭のタスク）

## del <hash>

出力なし。hash が一致する**全アイテム**を削除（既存バグにより実質全件削除）。

## clear（30 日超過分の削除）

```
    Delete logs that are past their storage period. ( Storage period: 30 days )

Deleted... 2026-05-01 logs.
```

**注意（既存バグ）**: 上記メッセージの後、パス二重化により `Deno.remove` が
NotFound で失敗し、**実際にはファイルは削除されない**（`clear -a` も同様）。
Go 版では正しく削除される（意図した修正）。

## help（引数なし実行も同じ）

```
  Usage:   nippo-cli
  Version: 0.0.1

  Options: -h, --help / -V, --version
  Commands: add <contents> / end <hash> / del <hash> / list [hash] / clear / help [command]
```

- 表示名が `nippo-cli` になっている（Go 版では `sava` に統一、v0.1.0）

## 保存 JSON フォーマット

`2026-07-05.json` を参照。キー順は `hash, freezed, items` /
アイテムは `hash, content, createdAt, updatedAt(, closed)`。
`closed` キーの有無でタスク / メモを判別。インデント 2 スペース、末尾改行なし。
`createdAt` / `updatedAt` は UTC の ISO 8601（ミリ秒 3 桁 + `Z`）。
ファイル名の日付はローカルタイム基準。
