# Deno v2 マイグレーション計画

## 概要

このドキュメントは、nippo-cliプロジェクトをDeno v2に対応させるためのマイグレーション計画を記載しています。

## 現状分析（詳細調査結果）

### プロジェクト構造
- **総ファイル数**: 21のTypeScriptファイル（ソース18、テスト3）
- **依存関係管理**: 集約型（src/dependencies.ts、test/dependencies.ts）
- **設定ファイル**: `.config/deno.jsonc`（レガシー形式）、`.mise.toml`

### 現在の依存関係
**メイン依存関係（src/dependencies.ts）:**
```typescript
- std@0.100.0/path/mod.ts, flags/mod.ts
- std@0.120.0/fs/ensure_dir.ts, ensure_file.ts  
- std@0.77.0/hash/mod.ts
- std@0.123.0/fs/walk.ts
- dir@1.5.1/home_dir/mod.ts
- date_fns@v2.22.1/format/index.js
- cliffy@v0.20.1/command/mod.ts
```

**テスト依存関係（test/dependencies.ts）:**
```typescript
- std@0.121.0/testing/asserts.ts
```

### 影響を受けるファイル分析

**高影響（import書き換え必須）:**
- `src/sava.ts` - dependencies.tsから複数モジュール使用
- `src/features/logFile.ts` - createHash, ensureFile, walk使用
- `src/features/path.ts` - homeDir, join使用
- `test/log_test.ts` - 混合importパターン

**中影響（依存関係更新の影響）:**
- 全command系ファイル（18ファイル） - 内部importのみ
- 全model系ファイル - 内部importのみ

**低影響（設定変更のみ）:**
- `.config/deno.jsonc` → `deno.json`移行
- `scripts/format.sh` → deno taskへ移行

### 重大な問題点
1. **stdライブラリバージョン競合**: 5つの異なるバージョンが混在（0.77.0〜0.123.0）
2. **Cliffy大幅版数遅れ**: v0.20.1 → v1.x（破壊的変更多数）
3. **設定ファイル分散**: レガシー.config/形式とimport map未対応
4. **CI/CD不備**: テスト・lint・型チェックの自動化なし
5. **テスト依存関係分離**: src/testで異なる依存関係バージョン

## Deno v2の主要変更点

### 1. 標準ライブラリの変更
- JSR（JavaScript Registry）への移行
- モジュール構造の変更
- 新しいAPIの追加と古いAPIの廃止

### 2. 新機能
- Node.js互換性の向上
- パフォーマンスの改善
- 新しいAPI（Web標準準拠）

### 3. 破壊的変更
- 一部のAPIの削除や変更
- インポート方式の変更推奨

## マイグレーション手順（改訂版）

### Phase 1: 基本設定とCI整備
- [x] **1.1** deno.jsonファイルの作成
  - 既存の依存パッケージをimportsに指定
  - 基本的なTypeScript設定
  - runタスクのみ設定（他のタスクは後で追加）

- [ ] **1.2** ソースコードのimport文書き換え
  - deno.jsonのimportsマッピングに対応
  - src/dependencies.ts経由から直接インポートに変更

- [ ] **1.3** GitHub Actionsの設定
  - lint、format、typecheck用のワークフロー作成
  - Deno v2対応のCI環境構築

- [ ] **1.4** CI整備後のソースコード品質対応
  - lint、formatエラーの修正
  - 基本的な動作確認

### Phase 2: 依存関係の段階的更新
- [x] **2.1** 標準ライブラリの統合・更新（優先度：高）
  - 全stdライブラリを最新版に統一（現在：0.77.0〜0.123.0 → 統一版）
  - JSR移行（@std/fs, @std/path, @std/crypto, @std/testing）
  - 特に重要：createHash（std/hash廃止予定、懸念あり）、fs（API変更なし）、path、testing
  - ✓ @std/flags → @std/cli/parse-args移行完了

- [x] **2.2** サードパーティライブラリの更新（優先度：高）
  - **cliffy: v0.20.1 → v0.20.1**（v0.25.7から0.20.1にダウングレード済み）
  - **dir → Deno標準API移行**（Deno.env.get("HOME")使用）完了
  - **date_fns → Intl.DateTimeFormat移行**（標準API、依存関係削除）完了

- [x] **2.3** テスト環境の統合（優先度：中）
  - test/dependencies.tsとsrc/dependencies.tsの統合
  - テスト用依存関係の最新化

- [x] **2.4** 追加タスクの設定
  - deno.jsonにtest、fmt、lint、devタスクを追加
  - scripts/format.shの置き換え

### Phase 3: 本格的なコード修正
- [ ] **3.1** 廃止予定APIの置換
  - `Deno.cwd()` → 必要に応じて他のAPI検討
  - ファイルシステムAPI確認

- [ ] **3.2** 型定義の更新
  - TypeScript設定の最適化
  - 型安全性の向上

- [ ] **3.3** テストの修正
  - テスト用アサーション関数の更新
  - テスト実行方式の確認

### Phase 4: 最適化と検証
- [ ] **4.1** パフォーマンステスト
- [ ] **4.2** 統合テストの実行
- [ ] **4.3** 動作確認
- [ ] **4.4** ドキュメント更新

## 想定される課題と対策（調査結果ベース）

### 1. Cliffyライブラリの互換性修正（リスク：中）
**問題**: v0.20.1のDeno v2非互換（Deno.setRaw API変更）
**影響ファイル**: `src/sava.ts`（main CLIエントリポイント）

**互換性問題:**
- **Deno.setRaw API**: Deno v1.26以降で変更、v0.20.1では`TypeError`発生
- **現在の環境**: Deno 2.4.3のため確実に問題発生
- **影響範囲**: Cliffyのprompt機能全般

**今回の対応（v0.25.7への最小限更新）:**
- **API変更**: ほぼなし（互換性修正のみ）
- **既存コード**: そのまま使用可能
- **Import更新**: dependencies.ts内のバージョン指定のみ

**既存コード（src/sava.ts）への影響:**
```typescript
// 既存コードはそのまま動作
new Command()
  .name(APP_NAME)
  .default('help')  // ← v0.25.7でも使用可能
  .command('add <contents:string>', new Command()...)
```

**対策**: 
- dependencies.ts内のバージョン更新のみ
- 既存コードの変更不要
- 将来的なv1.x移行は別途検討

### 2. stdライブラリのJSR移行（リスク：低）
**問題**: 複数バージョン混在とJSR移行対応
**影響ファイル**: `src/features/logFile.ts`, `src/features/path.ts`, `src/models/*.ts`
**対策**: 
- JSR移行（@std/fs, @std/path, @std/crypto, @std/testing）
- createHashのみ懸念事項として注視（API変更可能性）
- import文の一括更新、実装ロジック変更なし

### 3. テスト環境の分離問題（リスク：中）
**問題**: test/src間での依存関係バージョン不一致
**影響ファイル**: `test/log_test.ts`
**対策**: 
- 依存関係管理の統合
- テスト実行環境の標準化

### 4. 設定ファイル移行（リスク：低）
**問題**: `.config/deno.jsonc` → `deno.json`移行
**影響ファイル**: `scripts/format.sh`
**対策**: 
- 設定内容の確認・移行
- 既存スクリプトのパス更新

## スケジュール

| フェーズ | 想定工数 | 説明 |
|---------|---------|------|
| Phase 1 | 2-3時間 | 基本設定・CI整備・品質対応 |
| Phase 2 | 3-4時間 | 依存関係の段階的更新 |
| Phase 3 | 2-3時間 | 本格的なコード修正・API更新 |
| Phase 4 | 1-2時間 | 最適化・検証・ドキュメント更新 |
| **合計** | **8-12時間** | |

## 参考資料

- [Deno 2.0 Release Notes](https://deno.com/blog/v2.0)
- [Deno Standard Library JSR](https://jsr.io/@std)
- [Deno Migration Guide](https://docs.deno.com/runtime/manual/references/migrate_deprecations)
- [deno.json Configuration](https://docs.deno.com/runtime/manual/getting_started/configuration_file)

## 次のステップ

1. Phase 1から順次実行
2. 各フェーズ完了後にコミット
3. 問題が発生した場合は本計画書を更新
4. 完了後に動作確認とドキュメント更新

## 残っている不明瞭な点（実装時に検討）

### **技術的な詳細**
- **deno.json設定**: 既存`.config/deno.jsonc`からの設定移行内容、TypeScript設定の具体的な内容、import mapsの具体的なマッピング構造
- **JSR移行の具体的なバージョン**: `@std/fs`、`@std/path`、`@std/crypto`、`@std/testing`の使用バージョン、JSRパッケージのSemVer範囲指定方針
- **createHashの代替API**: Web Crypto API vs @std/crypto vs Node.js互換の選択基準、既存のMD5ハッシュとの互換性保証

### **実装・運用面**
- **GitHub Actions設定**: CI/CDワークフローの具体的な内容、Deno v2でのlint/format/testの実行コマンド、権限設定（--allow-env等）の必要性
- **テストの実行方法**: 現在のテスト実行方法（コマンド、設定）、新しいJSR環境でのテスト実行確認
- **既存スクリプトの扱い**: `scripts/format.sh`の完全置き換え vs 併用、他に影響を受けるスクリプトファイルの有無

### **プロジェクト方針**
- **バージョン管理**: 各フェーズでのコミット単位、動作しない中間状態の許容範囲
- **エラー発生時の対応**: 各フェーズでの問題発生時のロールバック方針、想定外のAPI変更への対応手順
- **パフォーマンス・動作確認**: 移行後の動作確認項目、パフォーマンス劣化の許容範囲

---
*最終更新: 2025-08-17*
*作成者: Claude Code*