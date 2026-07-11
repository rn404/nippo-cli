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

```
go install github.com/rn404/nippo-cli/cmd/sava@latest
```

ビルド済みバイナリ (macOS / Linux) は
[Releases](https://github.com/rn404/nippo-cli/releases) からも取得できます.

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
# Add todo item
sava add <message>

# Add todo item and start it right away
sava add -s <message>

# Add memo item
sava add -m <message>

# Start todo item
sava start <hash>

# Finish todo item
sava end <hash>

# Delete item
sava del <hash>

# Add item with tags / manage tags afterwards
sava add -t <tag>[,<tag>...] <message>
sava tag <hash> <tag>...
sava tag -d <hash> <tag>...
sava tag --list

# Show elapsed time between two items (resolved across days)
sava diff <hashA>...<hashB>
sava diff <hashA> <hashB>

# List today's log items
sava list

# Filter by tags (multiple tags match all; --or matches any)
sava list -t <tag>[,<tag>...]
sava list -t <tag>,<tag> --or

# List items of a specific day / all log files / summaries
sava list <yyyy-MM-dd>
sava list -a
sava list -s
sava list -a -s

# Delete logs past the storage period (30 days)
sava clear

# Delete all logs (with confirmation; use -y to skip prompts)
sava clear -a
```

ログは `~/.log/sava/<yyyy-MM-dd>.json` に 1 日 1 ファイルで保存されます.

タグ操作時には `~/.log/sava/index.json` (タグ・hash から日付ファイルへの逆引きキャッシュ)
が再生成されます. 壊れても全ログから再構築できるキャッシュです.

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
