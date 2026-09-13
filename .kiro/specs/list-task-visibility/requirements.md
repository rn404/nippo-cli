# Requirements Document

## Project Description (Input)
日々 `sava list` で当日のタイムラインを見るとき、完了済みタスクとメモが未完了タスクと時系列でフラットに混在表示され、最も頻度の高い「今日やることを確認する」というユースケースで見づらくなっている。

この課題に対し、以下の変更を行う:
- フラグなしの `sava list [date]` では完了済みタスクを非表示にし、未完了タスクとメモだけを表示する（デフォルト挙動の変更）
- `--full` フラグを指定したときだけ、完了済みタスクを含めた全件を表示する
- 一覧は「完了済みタスク → 未完了タスク → メモ」の順にグルーピングし、各グループ内は既存通り作成日時昇順で並べる
- `--task` フラグを指定すると、メモを除いたタスクのみを表示する（`--full` と組み合わせ可能）

対象は `internal/command.listOneDay` のアイテム表示ロジック。`-s/--stat` の集計・表示、`-a/--all`（stat無し、ファイル名一覧）、`--tag` フィルタの既存ロジックは変更しない。

詳細な背景・スコープ・制約は `.kiro/specs/list-task-visibility/brief.md`（discoveryセッションの成果物）を参照。

## Introduction
`sava list` の当日表示（日次・非stat）において、完了済みタスクをデフォルトで非表示にし、`--full` で明示的に全件表示できるようにする。あわせて `--task` フラグでメモを除外できるようにし、表示順序を「完了済みタスク → 未完了タスク（未着手・着手中） → メモ」の固定順に変更する。これにより、最も頻度の高い「今日やるべきことを素早く確認する」というユースケースの視認性を改善する。

## Boundary Context (Optional)
- **In scope**: `sava list [date]`（stat無しの日次表示）における、完了済みタスクのデフォルト非表示、`--full` フラグ、`--task` フラグ、ステータス別の表示順序
- **Out of scope**: `-s/--stat` の集計内容・出力フォーマット、`-a/--all`（stat無し、ファイル名一覧）の出力、`--tag`/`--or` フィルタ自体のロジック、`diff`/`del`/`start`/`end` など他コマンド
- **Adjacent expectations**: `--tag` フィルタは本機能の可視性ルール（デフォルト非表示・`--full`・`--task`）と併用可能であり、すべての条件を同時に満たすアイテムのみが表示される。`-s/--stat` および `-a/--all`（stat無し）は本機能の影響を受けず、既存の出力をそのまま維持する。

## Requirements

### Requirement 1: デフォルト表示での完了済みタスクの非表示
**Objective:** As a sava user, I want the default `list` output to hide completed tasks, so that I can quickly see what still needs my attention today.

#### Acceptance Criteria
1. When `--full` フラグなしで `sava list [date]` を実行したとき、the List Command shall closedタスクを表示対象アイテムから除外する。
2. While `--full` フラグが指定されていない間、the List Command shall openタスク・startedタスク・メモを表示対象アイテムに含める。
3. When 対象日の全アイテムがclosedタスクであり、かつ `--full` フラグが指定されていないとき、the List Command shall その日にログが1件も無い場合と同じ「アイテムなし」メッセージを表示する。

### Requirement 2: `--full` フラグによる完了済みタスクの表示
**Objective:** As a sava user, I want an explicit way to see completed tasks along with everything else, so that I can review the full day's record when I need it.

#### Acceptance Criteria
1. When `--full` フラグが指定されたとき、the List Command shall closedタスクを、openタスク・startedタスク・メモとともに表示対象アイテムに含める。
2. Where `--full` フラグが指定されている場合、the List Command shall closed状態を理由にアイテムを除外しない。

### Requirement 3: `--task` フラグによるメモの除外
**Objective:** As a sava user, I want to view tasks only, so that memos do not clutter my task-focused review.

#### Acceptance Criteria
1. When `--task` フラグが指定されたとき、the List Command shall メモを表示対象アイテムから除外する。
2. When `--task` フラグが `--full` フラグなしで指定されたとき、the List Command shall openタスクとstartedタスクのみを表示し、メモとclosedタスクの両方を除外する。
3. When `--task` フラグと `--full` フラグの両方が指定されたとき、the List Command shall メモを除外したまま、closed・open・startedのタスクを表示する。

### Requirement 4: ステータス別の表示順序
**Objective:** As a sava user, I want displayed items ordered by status in a fixed sequence, so that I can scan completed work, active work, and notes in a predictable way.

#### Acceptance Criteria
1. When the List Command が対象日のアイテムを表示するとき、the List Command shall closedタスクがopen・startedタスクより前に、open・startedタスクがメモより前に来るよう表示アイテムを並べる。
2. Within 各ステータスグループ（closedタスク／open・startedタスク／メモ）の内部において、the List Command shall アイテムを作成日時の昇順に並べる。
3. The List Command shall ステータス別の並び順をアイテムの並び順のみで表現し、グループ間に見出しを追加しない。

### Requirement 5: 既存フィルタ・他モードとの共存
**Objective:** As a sava user, I want the new visibility flags to work alongside existing filters, so that my existing habits (tag filtering, stats, file listing) keep working the same way.

#### Acceptance Criteria
1. When `--tag` フィルタが、デフォルトの可視性ルール・`--full` フラグ・`--task` フラグのいずれかと組み合わされたとき、the List Command shall 指定された条件をすべて同時に満たすアイテムのみを表示する。
2. Where `-s`/`--stat` フラグが指定されている場合、the List Command shall 既存のサマリー出力を維持し、closedタスクの非表示・`--full`・`--task` の挙動をその出力には適用しない。
3. Where `-a`/`--all` フラグが `-s`/`--stat` なしで指定されている場合、the List Command shall 既存のファイル名一覧出力を維持し、closedタスクの非表示・`--full`・`--task` の挙動をその出力には適用しない。
