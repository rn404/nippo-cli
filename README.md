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

ビルド済みバイナリ (macOS / Linux / Windows) は
[Releases](https://github.com/rn404/nippo-cli/releases) からも取得できます.
`v*` タグを push すると CI がバイナリをビルドしてリリースを作成します.

## Usage

```
# Add todo item
sava add <message>

# Add memo item
sava add -m <message>

# Finish todo item
sava end <hash>

# Delete item
sava del <hash>

# List today's log items
sava list

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

### Objects
* LogFile > Log > Item (Task, Memo)

### Architecture
* cmd/sava -- internal/command -- internal/{logfile, log, view} -- internal/model

## Development

```
# Run from source
go run ./cmd/sava <command>

# Test / format / lint
go test ./...
gofmt -l .
go vet ./...
golangci-lint run   # version is pinned in .mise.toml (mise install)
```

## History

もともと Deno / TypeScript で実装されていましたが、Go に移行しました.
経緯は `docs/go-migration-plan.md` を参照してください.
