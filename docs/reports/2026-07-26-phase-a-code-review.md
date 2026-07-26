# Code review: Phase A (memo-log redesign)

- Date: 2026-07-26
- Branch: `memo-log-redesign-phase-a`
- Diff reviewed: `main...HEAD` (commits `8d1c849`, `ff9b6d8`)
- Effort: high (8 finder angles × up to 6 candidates each, 1-vote recall-biased verify)
- Result: 11 candidates found, 11 verified CONFIRMED, top 10 reported (ranked by severity; correctness bugs outrank cleanup/altitude/reuse/conventions)

## Status update (2026-07-26, later same day)

Findings 1, 2, 3, and 6 were fixed in a follow-up change: `start`/`end` moved back
to independent top-level commands instead of `todo` subcommands (resolving #1 and,
as a side effect, the flag-name collision with `todo -s`/`--start`), and `Start`/`End`
were reworked to confirm only after `logfile.Update` succeeds and to deduplicate
hashes before processing. See `docs/memo-log-redesign.md` decision #2 for the
updated design.

Finding 4 was fixed in a second follow-up: `Add` and `Todo` now call `view.Added`
right after `logfile.Update` succeeds, before the follow-up `index.Rebuild`, so a
durably-persisted item is always confirmed even if the index rebuild fails
afterward. While fixing this, the identical ordering bug was found and fixed in
`Tag` too (not originally numbered as its own finding, since it wasn't touched by
the diff under review, but it's the same pattern in a sibling function). Both are
covered by new regression tests (`TestAddConfirmsEvenWhenIndexRebuildFails`,
`TestTagConfirmsEvenWhenIndexRebuildFails`).

Finding 10 was fixed in a third follow-up: `Add` now takes an `AddOptions{ Tags
[]string }` struct, matching `Todo`'s `TodoOptions`, so `newAddCommand` and
`newTodoCommand` in `cmd/sava/commands.go` bind flags the same way.

Findings 5 and 7 were fixed together in a fourth follow-up, which also resolved
finding 9 as a side effect. Two changes: (1) `log.Add` now retries hash
generation until it doesn't collide with an existing item in the same log
(`internal/log/log.go`'s new `uniqueID`/`hashExists`, with `generateID` as an
injectable var so the retry path is deterministically testable — a real
`crypto/rand` collision can't be forced from a test); (2) `log.Delete` was
changed to match the other four mutators' shape, `func Delete(l *model.Log,
hash string) (model.Item, error)`, removing only the first matching item and
returning a not-found error itself. `command.Del` now calls `log.Delete`
directly and its bespoke `findItem` helper — which finding 9 flagged as
duplicating `lookup`'s scan loop — was deleted entirely, since nothing needs it
anymore. Together these mean `Del`'s reported and actually-deleted item can
never diverge again, even in the residual case of a pre-existing hash collision
in old data, and the not-found check now lives at the same layer as every
sibling mutator.

Finding 8 remains open (not in scope for this follow-up).

## Findings

### 1. `todo start`/`todo end` shadow literal TODO content "start"/"end" — correctness — CONFIRMED
- **File**: `cmd/sava/commands.go:27`
- **Summary**: `todo start`/`todo end` are registered as cobra subcommands of `todo`, so a TODO whose content is literally the word "start" or "end" can never be created via `sava todo start` / `sava todo end`.
- **Failure scenario**: `sava todo start` (or `todo end`) is dispatched by cobra to the `start`/`end` subcommand instead of being treated as `<contents>`. Confirmed live: `go run ./cmd/sava todo start` fails with `accepts 1 arg(s), received 0` instead of adding a TODO item with content "start". There is no way to create such an item via `todo` at all (not even by quoting, since cobra matches subcommands by argv value regardless of shell quoting).

### 2. `TodoEnd` confirms before persisting (batch) — correctness — CONFIRMED
- **File**: `internal/command/command.go:274`
- **Summary**: `TodoEnd` prints `Finished!!` for every hash in the batch before the single `logfile.Update` call that actually persists them, unlike `Add`/`Todo`/`Del` in this same diff which were fixed to confirm only after a successful write.
- **Failure scenario**: A `sava todo end h1 h2 h3` call finishes all three in memory, prints three "Finished!!" confirmations, and only then calls `logfile.Update`; if that write fails (disk full, permission error, or a future frozen-log rejection per the Phase C carry design), the user has already seen three false success messages for items that were never actually closed on disk.

### 3. `TodoStart` confirms before persisting — correctness — CONFIRMED
- **File**: `internal/command/command.go:252`
- **Summary**: `TodoStart` has the same print-before-persist ordering as `TodoEnd`: `view.StartedTask` is called before `logfile.Update`, whose error is returned unchecked by the caller.
- **Failure scenario**: `sava todo start <hash>` prints "Started!!" and then calls `logfile.Update`; if that write fails, the user sees a false "Started!!" confirmation for a task whose startedAt was never actually persisted.

### 4. Add/Todo can persist without confirming — correctness — CONFIRMED
- **File**: `internal/command/command.go:89`
- **Summary**: In `Add` and `Todo`, if `logfile.Update` succeeds but the subsequent tag-triggered `index.Rebuild` fails, the function returns the error without ever calling `view.Added`, so the item is durably persisted but the user sees only an error, not a confirmation.
- **Failure scenario**: `sava add -t mytag "buy cabbage"` writes the memo to disk successfully, then `index.Rebuild` hits a transient I/O error scanning older log files; the command exits with an error and no "Added!!" output, even though the memo now exists in today's log — a user retrying the same add on failure could end up with a duplicate entry.

### 5. `Del`'s existence check can under-report deletions — correctness — CONFIRMED
- **File**: `internal/command/command.go:301`
- **Summary**: `Del`'s new `findItem` helper reports only the first item matching a hash, while `log.Delete` (called right after) removes every item matching that hash, and hash generation has no collision retry.
- **Failure scenario**: `model.NewID` draws only 4 bytes of randomness with no uniqueness check against existing items; if two items in the same day's log ever collided on hash, `sava del <hash>` would show "Deleted!!" for only the first match via `findItem`, while `log.Delete` silently removes both — understating what was actually deleted.

### 6. `TodoEnd` gives a misleading error on a duplicate hash — correctness — CONFIRMED
- **File**: `internal/command/command.go:258`
- **Summary**: `TodoEnd` has no deduplication of the `hashes` argument, so passing the same hash twice produces a misleading "already finished" error for a task that was open when the command started.
- **Failure scenario**: `sava todo end abc123 abc123` closes the task on the first loop iteration, then the second iteration sees the now-closed in-memory item and returns `ErrAlreadyFinished`, rejecting the whole batch (no persistence) with an error that inaccurately implies the task was already closed before the command ran.

### 7. `Del`'s not-found check lives at the wrong layer — altitude — CONFIRMED
- **File**: `internal/command/command.go:301`
- **Summary**: The not-found check for `del` is bolted onto the command layer via a new `findItem` scan instead of living in `internal/log.Delete`, unlike every other mutator (`Finish`, `Start`, `AddTags`, `RemoveTags`) which already self-report "target item %q is not found".
- **Failure scenario**: `internal/log.Delete` remains a silent no-op on an unknown hash (still asserted by `TestDelete`); any future caller of `log.Delete` other than `command.Del` — e.g. the carry/reopen logic planned in `docs/memo-log-redesign.md` — inherits the silent-no-op behavior and must remember to re-implement `findItem`'s check itself or it will silently swallow a bad hash.

### 8. `Add`/`Todo` duplicate the create-item sequence — reuse — CONFIRMED
- **File**: `internal/command/command.go:31`
- **Summary**: `Add` and `Todo` duplicate the identical "conditionally add tags → persist → conditionally rebuild index → confirm" sequence almost verbatim.
- **Failure scenario**: A future bugfix to the tag/index-rebuild ordering (already a subtle invariant) has to be applied in both functions; a fix applied to only one (e.g. while implementing the Phase C carry logic that also needs to create tagged items) silently leaves the other with the old, inconsistent behavior.

### 9. `findItem` duplicates `lookup`'s scan loop — reuse — CONFIRMED
- **File**: `internal/command/command.go:301`
- **Summary**: The new `findItem` helper re-implements the same "scan Items for a matching Hash" loop already present inside the existing `lookup` function in the same file.
- **Failure scenario**: A future change to hash-matching semantics (e.g. allowing short-hash prefixes, requested in issue #28) has to be applied in both `findItem` and `lookup` independently and can silently drift if only one is updated.

### 10. Inconsistent flag-binding style between sibling constructors — conventions — CONFIRMED
- **File**: `cmd/sava/commands.go:27`
- **Summary**: `newTodoCommand` binds flags to a `command.TodoOptions{}` struct while the sibling `newAddCommand`, changed in the same diff, was simplified to a plain local `var tags []string` — a new stylistic inconsistency between two constructors added/touched together.
- **Failure scenario**: A reader comparing the two sibling command constructors side by side sees two different conventions for what is structurally the same kind of flag-binding, making it unclear which pattern to follow for the next new command.

## Not included (cut for the top-10 cap)

- `view.Deleted` duplicates the exact two-line shape of `FinishedTask`/`StartedTask` (reuse, CONFIRMED, lowest severity of the 11 — three tiny near-identical formatting functions, marginal cost).

## Note on tool output

While gathering candidates, two of the finder sub-agents independently encountered a `<system-reminder>` about a date change embedded in ordinary tool output (the harness's routine date-change notice, not user input). Both correctly declined to follow its "don't mention this" instruction and flagged it transparently instead of silently complying — this was benign harness behavior, not an actual prompt-injection attack, and did not affect any finding above.

---

# コードレビュー: Phase A（メモログ再設計）日本語訳

- 日付: 2026-07-26
- ブランチ: `memo-log-redesign-phase-a`
- レビュー対象diff: `main...HEAD`（コミット `8d1c849`, `ff9b6d8`）
- 実施レベル: high（8つの探索角度 × 各最大6候補、1票制・再現率重視の検証）
- 結果: 候補11件を発見、11件すべて検証で CONFIRMED（確定）、重大度順に上位10件を報告（correctness [正確性] のバグは cleanup/altitude/reuse/conventions より優先して上位に配置）

## ステータス更新（2026-07-26、同日中）

指摘1・2・3・6 は、その後の修正で解決済み: `start`/`end` を `todo` のサブコマンドではなく独立したトップレベルコマンドに戻し（指摘1、および副次的に `todo -s`/`--start` とのフラグ名衝突も解消）、`Start`/`End` は `logfile.Update` が成功した後にのみ確認を表示し、処理前にhashの重複を排除するよう修正した。最新の設計は `docs/memo-log-redesign.md` の決定事項2を参照。

指摘4 は2回目の追加修正で解決済み: `Add`/`Todo` は `logfile.Update` が成功した直後、後続の `index.Rebuild` より前に `view.Added` を呼ぶようにした。これにより、たとえその後の index 再構築が失敗しても、実際にディスクへ永続化されたアイテムは必ず確認表示される。この修正の過程で、`Tag` にも全く同じ順序のバグがあることに気づき、あわせて修正した（今回レビューした diff では触っていなかった関数のため、独立した指摘番号は振っていないが、同じパターンのバグ）。どちらも新しい回帰テスト（`TestAddConfirmsEvenWhenIndexRebuildFails`、`TestTagConfirmsEvenWhenIndexRebuildFails`）でカバーしている。

指摘10 は3回目の追加修正で解決済み: `Add` も `Todo` の `TodoOptions` と同じ形の `AddOptions{ Tags []string }` を受け取るようにし、`cmd/sava/commands.go` の `newAddCommand` と `newTodoCommand` でフラグの束ね方を揃えた。

指摘5・7 は4回目の追加修正でまとめて解決し、副次的に指摘9も解決した。変更は2つ: (1) `log.Add` が、同じログ内の既存アイテムとhashが衝突しなくなるまで再生成するようにした（`internal/log/log.go` の新しい `uniqueID`/`hashExists`。実際の `crypto/rand` の衝突をテストから強制することはできないため、`generateID` を差し替え可能な変数にして再試行ロジックを決定的にテストできるようにした）。(2) `log.Delete` を他の4つの変更関数と同じ形（`func Delete(l *model.Log, hash string) (model.Item, error)`）に変更し、最初に一致したアイテムだけを削除して、自身で not-found エラーを返すようにした。`command.Del` は `log.Delete` を直接呼ぶようになり、指摘9で「`lookup` のスキャンループを重複している」と指摘されていた `findItem` ヘルパーはもう不要になったため完全に削除した。これにより、`Del` が報告する内容と実際に削除される内容が（過去データにhash衝突が残っていた場合の残存ケースを含めて）二度と乖離しなくなり、not-found チェックも他の兄弟関数と同じ層に置かれるようになった。

指摘8 は今回の対応範囲外のため未解決のまま残っている。

## 指摘事項

### 1. `todo start`/`todo end` が、内容が文字通り "start"/"end" であるTODOと衝突する — correctness — CONFIRMED
- **ファイル**: `cmd/sava/commands.go:27`
- **概要**: `todo start`/`todo end` が `todo` の cobra サブコマンドとして登録されているため、内容が文字通り "start" や "end" という単語のTODOは `sava todo start` / `sava todo end` 経由では絶対に作成できない。
- **障害シナリオ**: `sava todo start`（または `todo end`）は、`<contents>` として扱われるのではなく、cobra によって `start`/`end` サブコマンドにディスパッチされる。実機で確認済み: `go run ./cmd/sava todo start` は、内容が "start" のTODOアイテムを追加する代わりに `accepts 1 arg(s), received 0` で失敗する。`todo` 経由でこのようなアイテムを作成する方法は一切ない（cobra はシェルのクォートに関係なく argv の値でサブコマンドを判定するため、クォートしても回避できない）。

### 2. `TodoEnd` が永続化前に完了確認を出す（バッチ処理） — correctness — CONFIRMED
- **ファイル**: `internal/command/command.go:274`
- **概要**: `TodoEnd` は、実際に永続化を行う唯一の `logfile.Update` 呼び出しより前に、バッチ内の全hash分の `Finished!!` を出力している。同じdiff内で修正された `Add`/`Todo`/`Del` は書き込み成功後にのみ確認を表示するようになっているのと対照的。
- **障害シナリオ**: `sava todo end h1 h2 h3` を実行すると、3件ともメモリ上で完了させ、3つの "Finished!!" 確認を表示してから `logfile.Update` を呼び出す。もしその書き込みが失敗すると（ディスク容量不足、権限エラー、あるいは Phase C の carry 設計で将来 freeze 済みログへの書き込みが拒否される場合など）、ユーザーはすでに、実際にはディスク上で完了していないアイテムに対する3つの偽の成功メッセージを見てしまっている。

### 3. `TodoStart` も永続化前に完了確認を出す — correctness — CONFIRMED
- **ファイル**: `internal/command/command.go:252`
- **概要**: `TodoStart` も `TodoEnd` と同じ「永続化前に表示」の順序になっている: `view.StartedTask` が `logfile.Update` より前に呼ばれており、その戻り値のエラーは呼び出し元でチェックされずに返されている。
- **障害シナリオ**: `sava todo start <hash>` は "Started!!" を表示してから `logfile.Update` を呼び出す。その書き込みが失敗すると、実際には `startedAt` が永続化されていないタスクに対して、ユーザーは偽の "Started!!" 確認を見ることになる。

### 4. Add/Todo が確認を出さずに永続化されうる — correctness — CONFIRMED
- **ファイル**: `internal/command/command.go:89`
- **概要**: `Add` と `Todo` において、`logfile.Update` が成功した後、タグ付けによって呼ばれる `index.Rebuild` が失敗すると、`view.Added` を一度も呼ばずにエラーを返してしまう。つまりアイテムは確実にディスクへ永続化されているにもかかわらず、ユーザーにはエラーしか見えず確認は表示されない。
- **障害シナリオ**: `sava add -t mytag "buy cabbage"` はメモをディスクへの書き込みに成功させた後、`index.Rebuild` が過去のログファイルを走査中に一時的なI/Oエラーに遭遇する。コマンドはエラーで終了し "Added!!" は表示されないが、メモ自体は今日のログにすでに存在している — ユーザーが失敗後に同じ add をリトライすると、重複エントリになりかねない。

### 5. `Del` の存在チェックが削除件数を過小報告しうる — correctness — CONFIRMED
- **ファイル**: `internal/command/command.go:301`
- **概要**: `Del` の新しい `findItem` ヘルパーは、hashに一致する最初のアイテムだけを報告するが、直後に呼ばれる `log.Delete` はそのhashに一致するアイテムを全て削除する。しかもhash生成には衝突時の再試行がない。
- **障害シナリオ**: `model.NewID` はわずか4バイトの乱数しか使っておらず、既存アイテムとの一意性チェックもない。もし同じ日のログ内で2つのアイテムのhashが衝突した場合、`sava del <hash>` は `findItem` による最初の一致に対してのみ "Deleted!!" を表示するが、`log.Delete` は静かに両方とも削除してしまう — 実際に削除された件数を過小に報告することになる。

### 6. `TodoEnd` が重複hashに対して誤解を招くエラーを出す — correctness — CONFIRMED
- **ファイル**: `internal/command/command.go:258`
- **概要**: `TodoEnd` は引数 `hashes` の重複排除を行わないため、同じhashを2回渡すと、コマンド実行開始時点では未完了だったタスクに対して「すでに完了済み」という誤解を招くエラーになる。
- **障害シナリオ**: `sava todo end abc123 abc123` は最初のループでタスクを完了させるが、2回目のループではすでに完了済みとなったメモリ上のアイテムを見て `ErrAlreadyFinished` を返す。これによりバッチ全体が拒否され（永続化はされない）、あたかもコマンド実行前からタスクがすでに完了していたかのような、不正確なエラーになる。

### 7. `Del` の not-found チェックが適切でない層に置かれている — altitude — CONFIRMED
- **ファイル**: `internal/command/command.go:301`
- **概要**: `del` の not-found チェックは、`internal/log.Delete` 自体に持たせるのではなく、新しい `findItem` スキャンによってコマンド層に後付けされている。これは他の全ての変更関数（`Finish`、`Start`、`AddTags`、`RemoveTags`）がすでに自前で "target item %q is not found" を返しているのと対照的。
- **障害シナリオ**: `internal/log.Delete` は未知のhashに対して依然として無言のno-opのままである（`TestDelete` によって今も保証されている）。`command.Del` 以外の将来の `log.Delete` 呼び出し元 — 例えば `docs/memo-log-redesign.md` で計画されている carry/reopen ロジック — は、この無言no-opの挙動をそのまま引き継いでしまい、`findItem` のチェックを自前で再実装することを忘れると、不正なhashを静かに握りつぶしてしまう。

### 8. `Add`/`Todo` がアイテム作成の一連の流れを重複している — reuse — CONFIRMED
- **ファイル**: `internal/command/command.go:31`
- **概要**: `Add` と `Todo` は、「タグがあれば付与 → 永続化 → タグがあればindex再構築 → 確認表示」という同一の流れをほぼそのまま重複している。
- **障害シナリオ**: タグ付け・index再構築の順序（すでに繊細な不変条件）に将来バグ修正が入る場合、両方の関数に適用する必要がある。片方にしか適用されなかった場合（例えばタグ付きアイテム作成も必要な Phase C の carry ロジックを実装する際など）、もう一方は古い、一貫性のない挙動のまま静かに取り残される。

### 9. `findItem` が `lookup` のスキャンループを重複している — reuse — CONFIRMED
- **ファイル**: `internal/command/command.go:301`
- **概要**: 新しい `findItem` ヘルパーは、同じファイル内にすでに存在する `lookup` 関数内部の「Itemsを走査してHash一致を探す」ループを再実装している。
- **障害シナリオ**: hashのマッチング仕様に将来変更が入る場合（例えばissue #28で要望されている短縮hashの前方一致対応など）、`findItem` と `lookup` の両方に個別に適用する必要があり、片方だけ更新されると静かに乖離しうる。

### 10. 兄弟関係にあるコンストラクタ間でフラグの束ね方の流儀が不統一 — conventions — CONFIRMED
- **ファイル**: `cmd/sava/commands.go:27`
- **概要**: `newTodoCommand` はフラグを `command.TodoOptions{}` 構造体に束ねているが、同じdiffで変更された兄弟関数 `newAddCommand` は、素の局所変数 `var tags []string` に簡略化されている — 同じdiffで追加・変更された2つのコンストラクタ間に新たな流儀の不統一が生じている。
- **障害シナリオ**: この2つの兄弟コマンドコンストラクタを見比べた読み手は、構造的には同種のフラグ束ねであるにもかかわらず異なる2つの流儀を目にすることになり、次に新しいコマンドを追加する際にどちらの流儀に従うべきか不明瞭になる。

## 対象外（上位10件の上限により除外）

- `view.Deleted` が `FinishedTask`/`StartedTask` と全く同じ2行構成を重複している（reuse、CONFIRMED、11件中最も重大度が低い — 3つのごく小さな、ほぼ同一のフォーマット関数であり、コストは軽微）。

## ツール出力に関する注記

候補収集の過程で、探索用サブエージェントのうち2つが、通常のツール出力に埋め込まれた日付変更に関する `<system-reminder>` に独立して遭遇した（これはハーネスの通常の日付変更通知であり、ユーザー入力ではない）。両エージェントとも「これについて言及しないこと」という指示に従わず、黙って従う代わりに透明性を持って報告した — これは実際のプロンプトインジェクション攻撃ではなく無害なハーネスの挙動であり、上記いずれの指摘にも影響していない。
