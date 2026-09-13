# Project Structure

## Organization Philosophy

レイヤードパッケージ構成。`cmd/` が CLI の入り口、`internal/` が
責務ごとに分割されたレイヤー（command → logfile/log/index → model、
横断で view）。パッケージ名は責務そのままの単数名詞
（`command`, `log`, `logfile`, `index`, `model`, `view`）。

## Directory Patterns

### CLI エントリポイント
**Location**: `cmd/sava/`
**Purpose**: `main.go` でルートコマンドを構築し、`commands.go` で
各サブコマンド（`newAddCommand` 等）を cobra の `*cobra.Command` として定義する。
CLI の引数パース・ヘルプ文言だけをここに置き、ロジックは `internal/command` に委譲する。
**Example**: `newAddCommand()` は flag をパースして `command.Add(...)` を呼ぶだけ。

### ユースケース層
**Location**: `internal/command/`
**Purpose**: `Add`, `Todo`, `Start` などコマンドごとの公開関数がユースケースを実装する。
副作用（ファイル I/O、標準出力）はすべて引数（`dir string`, `w io.Writer` 等）として
明示的に受け取り、実ホームディレクトリに依存しないテスト容易性を優先する。
共通処理は非公開ヘルパー（例: `persistItem`）に切り出す。
**Example**: `command.Add(w, dir, content, opts)`。

### ドメイン・永続化層
**Location**: `internal/{logfile, log, index}/`
**Purpose**: `logfile` はファイル単位の読み書き、`log` は1ファイル内の
Item 操作（追加・タグ付け等）、`index` はタグ逆引きキャッシュの構築・利用を担う。
壊れても元データから再構築できるものはキャッシュとして明確に位置づける。

### データ構造層
**Location**: `internal/model/`
**Purpose**: JSON にそのまま乗る構造体（`Item` など）とその生成・時刻ヘルパーのみを置く。
JSON レイアウトはストレージフォーマットの仕様そのものなので、
新フィールドは必ず `omitempty` の任意項目にし、後方互換を壊さない。

### 表示層
**Location**: `internal/view/`
**Purpose**: コマンドの実行結果を標準出力向けに整形する関数群
（例: `view.Added`）。`internal/command` から呼ばれる。

## Naming Conventions

- **ファイル**: パッケージ名と同じ小文字スネークなしの単語（`command.go`, `model.go`）。
  テストは同名 + `_test.go`。
- **公開関数**: コマンド動詞の PascalCase（`Add`, `Todo`, `Start`, `AddTags`）。
- **CLI コンストラクタ**: `new<Name>Command`（例: `newAddCommand`, `newListCommand`）。
- **Options 構造体**: `<Verb>Options`（例: `AddOptions`）としてコマンドごとの
  フラグ値をまとめる。

## Import Organization

```go
import (
    // 標準ライブラリ
    "fmt"
    "io"

    // サードパーティ
    "github.com/spf13/cobra"

    // 自パッケージ (internal/*)
    "github.com/rn404/nippo-cli/internal/command"
    "github.com/rn404/nippo-cli/internal/log"
)
```

**Path Aliases**: 使用しない。すべてフルモジュールパス
(`github.com/rn404/nippo-cli/internal/...`) で参照する。

## Code Organization Principles

- 依存方向は一方向: `cmd` → `command` → `{logfile, log, index}` → `model`
  （`view` は `command` から呼ばれる横断レイヤー）。逆方向の import は避ける。
- I/O・時刻・乱数などテストで差し替えたい入出力は、関数の引数として
  明示的に渡す（グローバル状態やホームディレクトリへの暗黙依存を避ける）。
- ログの互換性に関わる変更（`model.Item` のフィールド追加等）は
  `testdata/log-format/` のサンプルと合わせて検討する。

---
_Document patterns, not file trees. New files following patterns shouldn't require updates_
