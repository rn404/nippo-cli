# Research & Design Decisions

## Summary
- **Feature**: `edit-command`
- **Discovery Scope**: Extension（既存の当日ログ変更コマンド群への追加）＋ 新規の小さな責務（エディタ起動）
- **Key Findings**:
  - `cmd/sava` は現在 `internal/command` と `internal/logfile`（`logfile.Dir()`のみ）しかimportしておらず、`internal/log`/`internal/view` には直接依存していない。CLI層は薄く保たれている。
  - `Start`/`Tag`/`AddTags`等の既存の当日限定コマンドは、いずれも自分のループでhashを検索して直接ミューテートしており、共有の「hash検索」ヘルパーは存在しない。
  - 読み取り専用の操作（`list`）は`ensureToday`ではなく`logfile.Stat`を使っており、キャリーフォワード（副作用のある書き込み）を起こさない。この区別は新設する読み取り専用ルックアップにも適用すべき。

## Research Log

### cmd/sava の依存関係調査
- **Context**: エディタ起動ロジックをどの層に置くべきか判断するため、CLI層が現在何にアクセスできるかを確認した。
- **Sources Consulted**: `cmd/sava/commands.go` のimport文、`internal/command/command.go` の `ensureToday`
- **Findings**: `cmd/sava` は `internal/command` と `internal/logfile` のみをimport。`internal/log`/`internal/view`/`internal/model` には触れていない。
- **Implications**: エディタ起動（`internal/editor`として新設）は `internal/log`/`model` に依存しない、文字列のみを扱う独立パッケージにすれば、既存の依存方向（`structure.md`: cmd → command → log/view/...）を壊さずに済む。`cmd/sava` は「直接指定か、省略か」を判定し、省略時は `internal/editor` で解決したcontentを得てから、従来通り `internal/command` だけを呼ぶ薄いグルーとして留まる。

### 読み取り専用ルックアップの副作用調査
- **Context**: エディタモードでは、エディタを開く前に「現在のcontent」を取得する必要があるが、この取得自体が当日ログに書き込み副作用（キャリーフォワード等）を起こすべきではない。
- **Sources Consulted**: `internal/command.listOneDay`（`logfile.Stat`を使用）、`internal/command.ensureToday`（`Add`/`Start`/`Tag`等が使用、キャリーフォワードを書き込む）
- **Findings**: 読み取り専用の`list`は`logfile.Stat`を使い、書き込み系コマンドは`ensureToday`を使う、という区別が既存コードベースに既に存在する。
- **Implications**: 新設する読み取り専用の「当日ログからhashでアイテムを取得する」関数（`command.TodayItem`）は `logfile.Stat` を使い、`ensureToday`は使わない。実際の編集確定（`command.Edit`）は既存の`Start`/`Tag`と同様に`ensureToday`を使う。これにより、エディタを開くだけの操作でキャリーフォワードが走ってしまう、という意図しない副作用を避けられる。

### エディタプロセスのテスト容易性
- **Context**: 実際のエディタプロセス（vi等）を起動するコードは、テスト環境で決定的に検証しづらい。
- **Sources Consulted**: `internal/log.uniqueID`（`next func() string` を注入してテストから衝突を制御できるようにしている既存パターン）
- **Findings**: このコードベースには「テストしづらい外部要素は関数注入で切り離す」という既存の慣習がある。
- **Implications**: `internal/editor.Resolve` は実際のプロセス起動処理（`launch func(editorName, path string) error`）を引数として受け取る設計にし、本番用の実装（`exec.Command`をラップしたもの）は `cmd/sava` 側で渡す。これにより `internal/editor` のテストは偽の`launch`（ファイル内容を書き換えるだけ、特定の終了コードを返す等）で完全に決定的に行える。

## Architecture Pattern Evaluation

| Option | Description | Strengths | Risks / Limitations | Notes |
|--------|-------------|-----------|---------------------|-------|
| `internal/editor` を新設（採用） | contentの文字列のみを扱う、`log`/`model`に依存しない独立パッケージ | 依存方向を壊さない、単体テストが容易、責務が明確 | パッケージが1つ増える | Simplification原則とも整合（新規責務は新規パッケージに切り出し、既存パッケージを汚さない） |
| `cmd/sava` に直接実装 | CLI層にエディタ起動ロジックをインラインで書く | パッケージが増えない | `cmd/sava`の既存の「薄いグルー」という性質が崩れる。テストもCLI層のテストに混ざり重くなる | 不採用 |
| `internal/command` に実装 | command層に`Edit`と一緒にエディタ起動も持たせる | 新規パッケージ不要 | brief.mdで合意済みの「command.Editはエディタの存在を知らない」という設計方針に反する | 不採用 |

## Design Decisions

### Decision: エディタ起動を `internal/editor` という独立パッケージに切り出す
- **Context**: content取得手段（直接指定／エディタ）を`command.Edit`から隠す、というdiscoveryでの合意をどう実装するか。
- **Alternatives Considered**: 上記Architecture Pattern Evaluation参照。
- **Selected Approach**: `internal/editor.Resolve(current string, launch func(editorName, path string) error) (content string, ok bool, err error)` を新設。`cmd/sava`がcontent省略時にこれを呼び、結果を`command.Edit`にそのまま渡す。
- **Rationale**: 依存方向を壊さず、テスト容易性も確保でき、discoveryでの設計方針とも一致する。
- **Trade-offs**: パッケージが1つ増えるが、責務が単純（文字列→文字列の変換＋中断判定）なので複雑さの増加は小さい。
- **Follow-up**: `internal/editor`の単体テストは偽の`launch`関数で、保存内容の変更あり/無変更/空文字列/エディタ異常終了の4パターンを検証する。

### Decision: 当日ログの読み取り専用ルックアップは `logfile.Stat` を使う（`ensureToday`は使わない）
- **Context**: エディタモードで現在のcontentを取得する際、キャリーフォワードのような書き込み副作用を起こしてはならない。
- **Alternatives Considered**: (1) `ensureToday`を使う（既存の書き込み系コマンドと同じ） (2) `logfile.Stat`を使う（既存の`list`と同じ、読み取り専用）
- **Selected Approach**: 案2。
- **Rationale**: エディタを開くだけの操作（まだ何も編集していない時点）でログファイルが書き込まれる・キャリーフォワードが走るのは、ユーザーの意図しない副作用になる。`list`が既にこの区別を確立している。
- **Trade-offs**: 当日ログがまだ存在しない（today's file未作成）場合、`TodayItem`は「見つからない」を返す。これは直接指定モードで`command.Edit`を呼んだ場合（`ensureToday`が空ログを作ってから検索し、結局見つからずエラーになる）と、観測可能な結果（エラーになる）としては同じなので、ユーザー体験上の差異はない。
- **Follow-up**: なし。

### Decision: hash検索は既存コマンドと同様、各関数が自分のループを持つ（共有Findヘルパーは新設しない）
- **Context**: `log.Edit`（書き込み用）と `log`内の読み取り専用ルックアップ（`command.TodayItem`が使う）の両方でhash検索が必要になる。
- **Alternatives Considered**: (1) 共有の`log.Find(l, hash) (model.Item, error)`を新設し、読み取り用に使う (2) 各関数が自分でループする
- **Selected Approach**: 案1（読み取り専用の`log.Find`を新設）。`log.Edit`自体は`Start`と同様、インデックス付きループで検索→ミューテートを1箇所で行う（`Find`は使わない、既存パターンと同じ）。
- **Rationale**: `internal/command`パッケージは`model.Log.Items`を直接走査せず、常に`internal/log`の関数に委譲する、という既存の境界規則がある。読み取り専用ルックアップも例外にしない。
- **Trade-offs**: なし。
- **Follow-up**: なし。

## Risks & Mitigations
- `$EDITOR`に引数付きコマンド（例: `code --wait`）が設定されているケース — 簡易的な空白区切り（シェルクオート非対応）で分割する設計とし、完全なシェル互換は範囲外と明記する（README等で触れる必要があれば別途対応）。
- 一時ファイルの削除漏れ — `defer`で確実に削除し、エディタ起動失敗時も同様に削除する。

## References
- なし（外部ライブラリ・API調査は不要、Go標準ライブラリ`os/exec`のみ使用）
