# Technology Stack

## Architecture

レイヤードな CLI 構成:
`cmd/sava` (CLI 定義) → `internal/command` (ユースケース) →
`internal/{logfile, log, index}` (永続化・ドメインロジック) →
`internal/model` (データ構造) 。`internal/view` は出力整形を横断的に担当する。

オブジェクトモデルは `LogFile > Log > Item (Task | Memo)` の3層。
`Item.Closed` の有無で Task/Memo を区別し、`StartedAt` の有無で
Task の着手状態を表す（詳細は `internal/model` のコメント参照）。

## Core Technologies

- **Language**: Go 1.26 系（バージョンは `.mise.toml` で固定）
- **CLI Framework**: [spf13/cobra](https://github.com/spf13/cobra)
- **Runtime**: 単一バイナリ、外部ランタイム不要

## Key Libraries

- `github.com/spf13/cobra`: サブコマンド構成の唯一の外部依存。
  それ以外は標準ライブラリのみで完結させる方針。

## Development Standards

### Type Safety

- Go の型システムをそのまま利用。JSON 互換のため `Item` の任意項目は
  ポインタ + `omitempty` で表現し、旧バージョンが書いたファイルとの
  前方互換性を保つ（`internal/model/model.go` のコメント参照）。

### Code Quality

- `golangci-lint`（バージョンは `.mise.toml` で固定、CI と同期）。
  有効化リンター: `gocritic`, `godot`, `gosec`, `misspell`, `nilnil`,
  `revive`, `unconvert`, `unparam`。
- `errcheck` は `fmt.Fprint*` への書き込みエラーを対象外にしている
  （stdout/stderr への出力失敗は基本的に対処不能なため）。
- テストファイル (`_test.go`) は `gosec` の対象外
  （fixture 読み込みや `t.TempDir()` への書き込みが主体のため）。

### Testing

- 各パッケージに対応する `*_test.go` を同居させる（`command_test.go` 等）。
- 副作用（ホームディレクトリ書き込み等）を避けるため、`command` 層の
  関数はログディレクトリと `io.Writer`/reader を明示的に受け取り、
  テストから差し替え可能にする設計を徹底する。

## Development Environment

### Required Tools

- Go / golangci-lint のバージョンは `.mise.toml` で固定（`mise install`）。

### Common Commands

```bash
# Dev:   go run ./cmd/sava <command>
# Build: make build
# Test:  make test
# Lint:  make lint
# All:   make check   (fmt + vet + test + lint)
```

## Key Technical Decisions

- **単一バイナリ配布**: `go install` に加え、GitHub Releases に
  プラットフォーム別ビルド済みバイナリを提供し、Go 未導入でも使える。
- **バージョンの単一の真実**: `cmd/sava/version.txt` を `go:embed` で
  埋め込み、リリース PR ワークフローが機械的に更新する。
- **1日1ファイルの JSON ログ**: ファイル形式は互換性の要であり、
  `testdata/log-format/` に仕様サンプルを置いて回帰を検知する。
- **ハッシュの一意性はスコープ限定**: item hash は同一日のログ内でのみ
  一意性を保証する設計。そのため `diff` や `del --deep` は
  `<date>:<hash>` 形式を要求するか全ログを直接走査し、hash から
  日付を逆引きする実装を持たない。

---
_Document standards and patterns, not every dependency_
