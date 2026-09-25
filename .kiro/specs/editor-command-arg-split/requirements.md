# Requirements Document

## Project Description (Input)
`sava add`/`sava todo`/`sava edit` のエディタ起動処理（`internal/editor.Name`/`internal/editor.Resolve` と `cmd/sava/commands.go` の `runEditor`）は、`$EDITOR`/`$VISUAL` の値をそのまま `exec.Command` の実行ファイル名として渡している。

Go の `exec.Command` はシェルを経由しないため、値がスペース区切りの複数トークン（例: `code --wait`、VS Code を `core.editor` に設定する際の一般的な書き方）だと、「`code --wait`」という1つの実行ファイル名として `$PATH` を探しにいき、見つからずに失敗する。

実際に発生したエラー:
```
sava: exec: "code --wait": executable file not found in $PATH
```

このエラーで `add`/`todo`/`edit` コマンドが丸ごと失敗し、エディタ経由での入力が一切できなくなる（content を直接引数で渡すモードは影響を受けない）。

`$EDITOR`/`$VISUAL` がコマンド名+引数の複数トークンで設定されている環境でも、正しく分割してエディタを起動できるようにしたい。

## Introduction
`sava add`/`sava todo`/`sava edit` のエディタモードが `$EDITOR`/`$VISUAL` の値を解釈する際、コマンド名と引数からなる複数トークンの値（例: `code --wait`）を正しく解釈できず、エディタの起動自体に失敗する不具合を修正する。単一トークンの値（例: `vim`）や、値が未設定で `vi` にフォールバックする既存の挙動は変更しない。

## Boundary Context (Optional)
- **In scope**: `$EDITOR`/`$VISUAL` の値を「コマンド + 引数」として解釈し、その解釈結果でエディタプロセスを起動する処理（`add`/`todo`/`edit` に共通のエディタ起動基盤）
- **Out of scope**: エディタ選択の優先順位ロジック自体（`$EDITOR` → `$VISUAL` → `vi` というフォールバック順序は変更しない）、content を直接引数で渡す直接指定モード（エディタを経由しないため無関係）、単純な空白区切りを超えるシェル文法のフルサポート（クォート内の空白、`~`/`$VAR` 展開、パイプ、リダイレクトなど）
- **Adjacent expectations**: 編集対象の一時ファイルパスは常に1つの引数としてそのまま渡され、分割・解釈の対象にはならない

## Requirements

### Requirement 1: 複数トークンの $EDITOR/$VISUAL 値の解釈
**Objective:** As a sava user, I want `$EDITOR`/`$VISUAL` values that contain a command plus arguments (e.g. `code --wait`) to be interpreted correctly, so that I can use my preferred editor without `add`/`todo`/`edit` failing.

#### Acceptance Criteria
1. When `$EDITOR`（未設定の場合は `$VISUAL`）の値が空白区切りの複数トークンを含むとき、the Editor Launcher shall 先頭トークンを起動コマンド、残りのトークンをそのコマンドへの引数として解釈する。
2. When 複数トークンの値でエディタを起動するとき、the Editor Launcher shall 解釈済みの引数列の末尾に編集対象の一時ファイルパスを追加した上でコマンドを実行する。
3. The Editor Launcher shall `$EDITOR`/`$VISUAL` の値が単一トークン（引数を含まない）の場合、既存と同じ挙動（そのトークンのみを起動コマンドとして扱う）を維持する。
4. Where `$EDITOR` も `$VISUAL` も設定されていない場合、the Editor Launcher shall 引き続き `vi` を単一トークンの起動コマンドとして扱う。

### Requirement 2: 起動失敗時の挙動の維持
**Objective:** As a sava user, I want a clear failure when the configured editor still cannot be launched after parsing, so that I understand my `$EDITOR`/`$VISUAL` setting itself needs fixing.

#### Acceptance Criteria
1. If トークン分割後の起動コマンドが `$PATH` 上に見つからない、または起動時にエラーが発生したとき、the Editor Launcher shall `add`/`todo`/`edit` コマンドをエラーで中断する。
2. The Editor Launcher shall 起動コマンドが見つからない場合に、設定と異なる別のエディタへ自動的にフォールバックしない。

### Requirement 3: 全エディタ起動経路への一貫した適用
**Objective:** As a sava user, I want the same `$EDITOR`/`$VISUAL` interpretation to apply no matter which command launches the editor, so that the fix does not need to be re-applied per command.

#### Acceptance Criteria
1. The Editor Launcher shall `add`・`todo`・`edit` のいずれのコマンドからエディタが起動される場合でも、同一の解釈ロジックを適用する。
