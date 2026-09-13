# nippo-cli (`sava`)

## Motivation / Background
エンジニアはほとんどの時間を console をみて過ごしている.
作業時間やちょっとした思考をメモしておくのに他ツールとのスイッチはコストが高いと考えます.
作業と並行して気軽にメモを残していけるツールはメリットが大きいと思います.
(作業時間計測も工数管理の観点では非常に関心のあることだと思います)

以上のことを簡単に示すと
- cli ツール作りたい
- 独り言メモツールしたい(日報補助ツール)

構想は gist に
https://gist.github.com/rn404/decf010fc48d7d8688116af0f4427b44

## Install

[Releases](https://github.com/rn404/nippo-cli/releases/latest) からビルド済み
バイナリを取得する（Go 不要）. 対応プラットフォーム: `darwin_amd64` /
`darwin_arm64` / `linux_amd64` / `linux_arm64`.

```
# 例: macOS (Apple Silicon)。他プラットフォームでは darwin_arm64 を置き換える
gh release download --repo rn404/nippo-cli --pattern "*_darwin_arm64.tar.gz" --pattern "checksums.txt"
grep darwin_arm64.tar.gz checksums.txt | shasum -a 256 -c -
tar -xzf sava_*_darwin_arm64.tar.gz
mv sava_*_darwin_arm64/sava ~/.local/bin/sava
```

`gh` が無い場合は [Releases](https://github.com/rn404/nippo-cli/releases/latest)
から手動で取得する.

ソースからビルドする場合:

```
go install github.com/rn404/nippo-cli/cmd/sava@latest
```

## Release lifecycle

バージョンの単一の真実は `cmd/sava/version.txt` (`go:embed` でバイナリに埋め込み).

1. Actions で「Release PR」workflow を dispatch し, bump レベル (patch / minor / major) を選択
2. `version.txt` を更新した Release PR が自動で作られる (本文には変更点一覧が自動生成される)
3. PR 本文をリリースノートとして整えて, **マージ = リリース承認**
4. マージを検知して release workflow が起動し, テスト・バイナリビルド・タグ作成・GitHub Release 公開まで自動実行 (マージ時点の PR 本文がそのままリリースノートになる)

リリース処理の実体は `scripts/` にあり, Makefile 経由でローカルでも実行できます
(例: `make release-build TAG=v0.1.0` で `dist/` にバイナリを生成).

## Usage

```
# Add a memo (default)
sava add <message>

# Add a TODO item
sava todo <message>

# Add a TODO item and start it right away
sava todo -s <message>

# Start an existing TODO item
sava start <hash>

# Finish one or more TODO items
sava end <hash>...

# Delete item (searches the last 30 days by default)
sava del <hash>

# Delete a specific day's item directly, or search every log ever
sava del <date>:<hash>
sava del --deep <hash>

# Add item with tags (memo or TODO) / manage tags afterwards
sava add -t <tag>[,<tag>...] <message>
sava todo -t <tag>[,<tag>...] <message>
sava tag <hash> <tag>...
sava tag -d <hash> <tag>...
sava tag --list

# Show elapsed time between two items (each given as <date>:<hash>)
sava diff <date>:<hashA>...<date>:<hashB>
sava diff <date>:<hashA> <date>:<hashB>

# List today's log items (closed tasks are hidden by default)
sava list

# Also show completed tasks (closed -> open -> memo order)
sava list --full

# Show tasks only, excluding memos (combine with --full to include closed tasks)
sava list --task
sava list --task --full

# Show full (multi-line) content instead of just the first line
sava list --full-text

# Filter by tags (multiple tags match all; --or matches any)
sava list -t <tag>[,<tag>...]
sava list -t <tag>,<tag> --or

# List items of a specific day / all log files / summaries
sava list <yyyy-MM-dd>
sava list yesterday
sava list -a
sava list -s
sava list -a -s

# Delete logs past the storage period (30 days)
sava clear

# Delete all logs (with confirmation; use -y to skip prompts)
sava clear -a
```

`list` の各行は Markdown のチェックリスト構文で描画されます（未着手 `- [ ]` / 着手中
`` - [ ] `in-progress` `` / 完了 `- [x]` / メモは `-` のみ）。hash は常に行末の
`` (`hash`) `` に統一されているので、`grep -oE '`[a-f0-9]{8}`'` のようなパターンで
pipe からも一貫して抜き取れます。複数行の content はデフォルトでは先頭行のみを表示し、
`--full-text` で全文を表示します。

ログは `~/.log/sava/<yyyy-MM-dd>.json` に 1 日 1 ファイルで保存されます.
フォーマットの仕様サンプルは `testdata/log-format/` にあります.

タグ操作時には `~/.log/sava/index.json` (タグから日付ファイルへの逆引きキャッシュ)
が再生成されます. 壊れても全ログから再構築できるキャッシュです.

`sava diff` と `sava del --deep`（あるいは30日より前を含む検索）は、hash から
日付を逆引きするのではなく `<date>:<hash>` を要求 / 全ログを直接走査します.
これは hash の一意性が保証されるのは同一日のログ内だけであるためです.

### Objects
* LogFile > Log > Item (Task, Memo)

### Architecture
* cmd/sava -- internal/command -- internal/{logfile, log, view, index} -- internal/model

## Development

```
# Run from source
go run ./cmd/sava <command>

# Test / format / vet / lint at once
make check

# Individual targets
make test
make fmt
make vet
make lint   # golangci-lint: version is pinned in .mise.toml (mise install)
make build
```

## History

もともと Deno / TypeScript で実装されていましたが、Go に移行しました.
経緯は `docs/go-migration-plan.md` を参照してください.
