# Go CLI マイグレーション計画

## 概要

このドキュメントは、nippo-cli（`sava` コマンド）を Deno / TypeScript 実装から Go 実装へ移行するためのマイグレーション計画を記載しています。

### 移行の目的（想定されるメリット）

- **シングルバイナリ配布**: ランタイム（Deno）不要で `go install` や GitHub Releases 経由の配布が可能になる
- **起動速度**: CLI ツールとして起動オーバーヘッドが小さくなる
- **クロスコンパイル**: macOS / Linux / Windows 向けバイナリを容易に生成できる
- **依存の安定性**: Deno v2 / std / cliffy などのエコシステム変化に追従するコストがなくなる

### 前提（決定事項）

- 外部の利用者はいない（作者のみ）。**移行完了後、Deno 版は廃止する**
- 既存ログデータとの互換性維持は必須要件としない。フォーマットは結果的に互換となるが、旧データの救済処理（hash の再採番など）は行わない

## 現状分析

### プロジェクト構造（2026-07 時点）

- **総ファイル数**: TypeScript 22 ファイル（ソース 21、テスト 1）、約 990 行
- **エントリポイント**: `src/sava.ts`（cliffy v0.20.1 によるコマンド定義）
- **レイヤ構成**: Command → Feature → LogFile（README 記載のアーキテクチャ）

| レイヤ | ファイル | 役割 |
|--------|---------|------|
| entry | `src/sava.ts` | CLI 定義（cliffy） |
| commands | `add.ts` / `end.ts` / `delete.ts` / `list.ts` / `clear.ts` | 各サブコマンドの制御 |
| features | `logFile.ts` / `log.ts` / `generate.ts` / `hash.ts` / `path.ts` | ファイル I/O、ログ操作、表示整形 |
| models | `LogFile.ts` / `MemoItem.ts` / `TaskItem.ts` / `LogFileName.ts` / `Date.ts` / `factory/LogFileName.ts` | ドメインモデル |
| utils | `formatDate.ts` / `formatTime.ts` / `homeDir.ts` | 汎用ユーティリティ |
| const | `const.ts` | 定数（保存先、保持期間など） |

### コマンド仕様（移行で維持すべき外部仕様）

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

### データ仕様（最重要の互換性ポイント）

- **保存先**: `$HOME/.log/sava/`（`HOME` / `USERPROFILE` 未設定時はカレントディレクトリ）
- **ファイル名**: `yyyy-MM-dd.json`（1 日 1 ファイル）
- **フォーマット**: インデント 2 スペースの JSON

```json
{
  "hash": "...",
  "freezed": false,
  "items": [
    {
      "hash": "...",
      "createdAt": "ISO 8601 文字列",
      "updatedAt": "ISO 8601 文字列",
      "content": "本文",
      "closed": false
    }
  ]
}
```

- `closed` キーの**有無**でタスク（あり）とメモ（なし）を判別している
- `freezed: true` のファイルは更新不可

### 移行前に判断が必要な既存実装の問題点

調査の過程で見つかった、単純移植すると問題を引き継ぐ箇所については、以下の通り。修正対象とする。

1. **ハッシュ生成のバグ（重大・退行）**: 初期実装では `std@0.77.0/hash` の
   `createHash` を使用しており、その `Hash.toString()` はデフォルトで hex
   ダイジェストを返すため**正常に動作していた**。Deno v2 移行
   （commit `53cbd7e` / `cb9fc4b`）で `node:crypto` に置き換えた際、
   `node:crypto` の Hash は `toString()` でダイジェストを返さない
   （`digest('hex')` の呼び出しが必要）ため、それ以降に作成されたアイテムの
   hash は壊れており、`end <hash>` / `del <hash>` で個別アイテムを特定できない。
   Go 版では新方式の ID 生成に置き換える（「ハッシュ（アイテム ID）の扱い」参照）。
2. **`compareDatesInDescent` の名前と実装の不一致**: 名前は降順だが実装は昇順ソート。実態に合わせて修正する。
3. **`requiredDateFormatHash` / `isDateString` の日付検証が緩い**: `new Date()`
   のパース依存のため、`yyyy-M-d` などもゆるく通る。Go では
   `time.Parse("2006-01-02", ...)` による厳密な検証に置き換える。修正する。
4. **`getLogFile` 内の `updateLogFile` 呼び出しに `await` 漏れ**、
   `clear` の `forEach(async ...)` など、非同期処理の扱いに不備があるが、Go では同期処理になるため自然に解消される見込み。テストで期待通りの挙動になっていることを確認する。
5. **`clear` が実際にはファイルを削除できない（重大・Phase 0 実測で発見）**:
   `listLogFile` が `walk` の返す絶対パスを `logDir` と再結合するためパスが
   二重になり、`Deno.remove` が NotFound で失敗する。`clear` / `clear -a` とも
   「Deleted...」と表示するだけで削除は機能していない。Go 版ではパス解決を
   正しく実装する（意図した修正）。


## Go 実装の方針

### 技術選定

| 項目          | 選定                       | 理由                                      |
| ----------- | ------------------------ | --------------------------------------- |
| Go バージョン    | 1.24 系（移行時の最新安定版）        | `.mise.toml` で管理                        |
| CLI フレームワーク | `spf13/cobra`（採用確定）      | サブコマンド・フラグ・ヘルプ生成のデファクト |
| ハッシュ（アイテム ID） | `crypto/rand` によるランダム短縮 ID | 依存追加不要。詳細は「ハッシュ（アイテム ID）の扱い」参照 |
| JSON        | 標準 `encoding/json`       | `MarshalIndent` で 2 スペースインデント維持         |
| 日付          | 標準 `time`                | `Intl.DateTimeFormat` 依存を排除             |
| テスト         | 標準 `testing` + テーブル駆動    | 必要なら `google/go-cmp` を追加                |
| リリース        | `goreleaser`（任意）         | クロスコンパイルと GitHub Releases 配布            |

外部依存は cobra（+ goreleaser）程度に抑え、標準ライブラリ中心で構成します。

### 実装ポリシー（決定事項）

- **コマンド名**: バイナリ名・表示名とも `sava` に統一する（現行は実行名 `sava` と `APP_NAME = 'nippo-cli'` が不一致。リポジトリ名は `nippo-cli` のまま）
- **出力互換性**: 機能面でのデグレがないことを保証対象とし、表示テキストの完全一致は目指さない
- **確認プロンプト**: TTY での利用を前提とする。非対話環境（パイプ・CI）向けには、確認をスキップするオプション（`--yes` など）を用意する
- **エラー処理**: stderr に 1 行メッセージ + exit 1 を基本とする（スタックトレースは出力しない）
- **バージョニング**: Go 版初リリースを `v0.1.0` とする。利用者が現時点でいないため、厳密なリリース管理は行わない
- **コード品質 CI**: golangci-lint を導入する。既存の `sonar.yml` は TODO コメントをスキャンする CI であり無害なため、そのまま存続させる

### ディレクトリ構成（案）

```
.
├── cmd/
│   └── sava/
│       └── main.go          # エントリポイント（cobra ルートコマンド）
├── internal/
│   ├── command/             # add / end / del / list / clear の実装
│   ├── logfile/             # ログファイル I/O（features/logFile.ts 相当）
│   ├── log/                 # アイテム操作（features/log.ts 相当）
│   ├── model/               # Item / Log / LogFile 構造体
│   └── view/                # 表示整形（features/generate.ts 相当）
├── go.mod
├── .mise.toml               # deno → go へ更新
└── docs/
```

### モデル定義（案）

```go
type Item struct {
    Hash      string  `json:"hash"`
    CreatedAt string  `json:"createdAt"` // ISO 8601 (RFC 3339)
    UpdatedAt string  `json:"updatedAt"`
    Content   string  `json:"content"`
    Closed    *bool   `json:"closed,omitempty"` // nil = メモ、非 nil = タスク
}

type Log struct {
    Hash    string `json:"hash"`
    Freezed bool   `json:"freezed"`
    Items   []Item `json:"items"`
}
```

`closed` の有無によるタスク / メモ判別は `*bool` + `omitempty` で既存 JSON と完全互換にします。

### ハッシュ（アイテム ID）の扱い — 再検討結果

**経緯（なぜ壊れたか）**: 初期実装の `std@0.77.0/hash` では
`createHash('md5').update(x).toString()` が hex ダイジェストを返すため
正常に動作していた。Deno v2 移行で `node:crypto` に置き換えた際、
`node:crypto` の Hash には同等の `toString()` がなく `digest('hex')` が
必要なため、以後の hash は壊れている（呼び出し側は無修正のまま）。

**現行設計の問題点（正しく hex 化しても残る問題）**:

- ID の元が `md5(createdAt)` のため、同一タイムスタンプで作成された
  アイテム同士は衝突する
- 用途は `end` / `del` でアイテムを特定する短い ID であり、
  暗号学的ハッシュ（md5）である必要がない

**Go 版の方式（決定）**: md5 の踏襲はやめ、代替方式へ移行する。

- `crypto/rand` で生成するランダム短縮 ID（hex 8 文字）を採用する
- JSON のキー名は `hash` のまま維持し、スキーマ変更はしない
- ファイル（Log）側の `hash` フィールドは現状どこからも参照されていないが、
  フォーマット維持のため同方式で生成を続ける（削除は移行完了後に別途検討）

**旧データの扱い（決定）**: 利用者がいないため、再採番などの移行処理は行わない。
旧ファイルの読み込み・表示はフォーマット互換のため可能だが、壊れた hash を持つ
旧アイテムへの `end` / `del` は保証しない。

## マイグレーション手順

### ブランチ運用（決定事項）

- 移行用の `migrate` ブランチを作成し、Phase ごとのブランチを `migrate` に向けてマージしていく
- 移行完了後、デフォルトブランチを切り替えて Go 版を main に置く

### Phase 0: 仕様の固定（サンプル採取）

- [x] **0.1** 現行 Deno 版の入出力仕様を記録（2026-07-05 完了）
  - 各コマンドの出力サンプルを `testdata/legacy-samples/outputs.md` に保存
  - サンプルのログ JSON を `testdata/legacy-samples/2026-07-05.json` に保存
  - 実測により hash バグ（`"[object Object]"` が保存される）と `clear` のパス二重化バグ（前述 5）を確認
- [x] **0.2** 互換性の範囲を決定（2026-07-05 決定済み）
  - **維持する**: コマンド体系、ログ JSON フォーマット、保存先パス、出力される情報（文言の完全一致は目指さない）
  - **修正する**: ハッシュ生成バグ、日付検証、ソート順の名称不一致（前述 4 点すべて）
  - **既存ログファイルの hash は移行対象外**: 利用者がいないため再採番は行わず、新規アイテムから新方式の ID を採用する（「ハッシュ（アイテム ID）の扱い」参照）

### Phase 1: Go プロジェクトの土台作り

- [ ] **1.1** `go mod init`、ディレクトリ構成の作成
- [ ] **1.2** `.mise.toml` に go を追加（deno と一時併存）
- [ ] **1.3** cobra でコマンドの骨組み（`add` / `end` / `del` / `list` / `clear`）を定義
- [ ] **1.4** CI に Go 用ジョブを追加（`gofmt` / `go vet` / `go test` / `golangci-lint`）
  - 既存の `wc_ci.yml`（Deno 用）と並走させる

### Phase 2: ドメイン層の移植

- [ ] **2.1** `internal/model`: `Item` / `Log` / `LogFile` 構造体と JSON 互換の確認
  - Deno 版が書き出した実ファイルを `testdata/` から読み込むラウンドトリップテスト
- [ ] **2.2** `internal/logfile`: 読み書き・一覧・新規作成（`~/.log/sava` 解決を含む）
- [ ] **2.3** `internal/log`: addItem / deleteItem / finishTaskItem / listItems の移植
  - ここで ID 生成を新方式（`crypto/rand` によるランダム短縮 ID）に置き換える
- [ ] **2.4** `internal/view`: 出力整形の移植（Phase 0 のサンプルと比較し、表示される情報に過不足がないことを確認）

### Phase 3: コマンド層の移植

- [ ] **3.1** `add`（`-m` フラグ含む）
- [ ] **3.2** `end` / `del`
- [ ] **3.3** `list`（`-a` / `-s` / 日付指定、確認プロンプト含む）
- [ ] **3.4** `clear`（`-a`、確認プロンプト含む）
- [ ] **3.5** 確認プロンプトのスキップオプション（`--yes` など）の追加（非対話環境向け）
- [ ] **3.6** `help` / `--version`（`v0.1.0`、表示名 `sava`）

### Phase 4: 検証と切り替え

- [ ] **4.1** 実データでの並行動作確認（Deno 版で作った `~/.log/sava` を Go 版で読み書き）
- [ ] **4.2** Phase 0 のサンプルと比較し、機能面のデグレがないことを確認
- [ ] **4.3** README の Usage を Go 版に更新
- [ ] **4.4** goreleaser 設定（任意）と `go install` 手順の整備

### Phase 5: Deno 資産の撤去

- [ ] **5.1** `src/`、`deno.json`、`deno.lock` の削除
- [ ] **5.2** CI から Deno ジョブ（`wc_ci.yml` ほか）を削除、paths-filter の対象を Go に更新（`sonar.yml` は存続）
- [ ] **5.3** `.mise.toml` から deno を削除
- [ ] **5.4** `docs/deno-v2-migration-plan.md` をアーカイブ扱いに（削除 or 注記）

## 想定される課題と対策

### 1. 既存ログファイルとの互換性（リスク: 低・方針決定済み）

**問題**: 既存ファイルの hash はバグにより壊れており、Go 版の新方式 ID とは互換がない。
**対策（決定）**: 利用者がいないため移行処理は行わない。フォーマット自体は互換のため旧ファイルの読み込み・表示（`list`）は可能とし、壊れた hash を持つ旧アイテムへの `end` / `del` は非サポートとする。

### 2. 出力・プロンプトの再現（リスク: 低）

**問題**: `confirm()`（y/N プロンプト）や `Intl.DateTimeFormat` の時刻表記（`HH:mm`、en-US の 24 時間表記）の再現。
**対策**: プロンプトは `bufio.Scanner` で自前実装（数行）し、TTY 前提とする。非対話環境向けにはスキップオプション（`--yes` など）を用意する。時刻は `time.Format("15:04")`、日付は `time.Format("2006-01-02")` で同等になる。タイムゾーンはローカルタイムを使用（現行と同じ）。文言の完全一致は目指さない（「実装ポリシー」参照）。

### 3. 日付まわりの挙動差（リスク: 低）

**問題**: 現行は `new Date(string)` の緩いパースに依存（`list 2026-7-5` なども通る）。Go の `time.Parse` は厳密。
**対策**: `yyyy-MM-dd` のみ受け付ける仕様として明文化（厳密化は意図した変更とする）。

### 4. Deno 版と Go 版の併存期間（リスク: 低）

**問題**: 移行中に main ブランチが「どちらが正か」曖昧になる。
**対策**: Go 版は `migrate` ブランチで開発し、移行完了まで main（Deno 版）を正とする。移行完了後にデフォルトブランチを切り替えて Go 版を main に置き、**Deno 版はその時点で廃止する**（利用者がいないため、切り替え時期の調整や告知は不要）。データディレクトリが共有されるため、並行利用しても実害はない。

## スケジュール

| フェーズ | 想定工数 | 説明 |
|---------|---------|------|
| Phase 0 | 1-2時間 | 仕様固定・ゴールデンテスト・互換方針の決定 |
| Phase 1 | 1-2時間 | Go プロジェクト土台・CI 併設 |
| Phase 2 | 3-4時間 | ドメイン層の移植とテスト |
| Phase 3 | 2-3時間 | コマンド層の移植 |
| Phase 4 | 1-2時間 | 検証・ドキュメント・配布整備 |
| Phase 5 | 1時間 | Deno 資産の撤去 |
| **合計** | **9-14時間** | |

## 参考資料

- [cobra](https://github.com/spf13/cobra)
- [goreleaser](https://goreleaser.com/)
- [Go time パッケージ（レイアウト仕様）](https://pkg.go.dev/time)
- 現行仕様: `README.md`、`docs/deno-v2-migration-plan.md`

## 次のステップ

1. `migrate` ブランチを作成し、Phase 0 から順次実行する（Phase ごとにブランチを切り `migrate` へマージ）
2. 問題が発生した場合は本計画書を更新する

※ 互換性の範囲・hash の扱い・実行前の判断ポイント（コマンド名、出力互換、CLI フレームワーク、プロンプト、エラー処理、バージョニング、CI、ブランチ運用）は 2026-07-05 にすべて決定済み。内容は「前提」「実装ポリシー」「ブランチ運用」の各セクションに反映済み

---
*最終更新: 2026-07-05*
*作成者: Claude Code*
