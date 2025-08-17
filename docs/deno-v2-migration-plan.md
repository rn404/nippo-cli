# Deno v2 マイグレーション計画

## 概要

このドキュメントは、nippo-cliプロジェクトをDeno v2に対応させるためのマイグレーション計画を記載しています。

## 現状分析

### 現在の依存関係
プロジェクトで使用している主要な依存関係とバージョン：

```typescript
// src/dependencies.ts より
- std@0.100.0/path/mod.ts
- std@0.100.0/flags/mod.ts  
- std@0.120.0/fs/ensure_dir.ts
- std@0.120.0/fs/ensure_file.ts
- std@0.77.0/hash/mod.ts
- std@0.123.0/fs/walk.ts
- std@0.121.0/testing/asserts.ts (テスト用)
- dir@1.5.1/home_dir/mod.ts
- date_fns@v2.22.1/format/index.js
- cliffy@v0.20.1/command/mod.ts
```

### 問題点
1. **古いstdライブラリバージョン**: 複数の異なるバージョンのstdライブラリを使用
2. **バージョン不統一**: 0.77.0から0.123.0まで様々なバージョンが混在
3. **deno.jsonファイルの不在**: モダンなDeno設定ファイルがない
4. **型安全性の課題**: 古いAPIを使用している可能性

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

## マイグレーション手順

### Phase 1: 環境準備
- [ ] **1.1** deno.jsonファイルの作成
  - TypeScript設定
  - インポートマップ設定
  - タスク定義
  - リンティング・フォーマット設定

- [ ] **1.2** 最新のDeno v2インストール確認
  - mise経由でのDeno最新版確認
  - 動作テスト

### Phase 2: 依存関係の更新
- [ ] **2.1** 標準ライブラリの統合・更新
  - 全stdライブラリを最新版に統一
  - JSR移行が可能なものは移行検討

- [ ] **2.2** サードパーティライブラリの更新
  - cliffy: v1.0.0以降への更新
  - dir: 最新版またはDeno標準API（Deno.env.get("HOME")）への移行
  - date_fns: より軽量な代替案またはJavaScript標準Dateへの移行

- [ ] **2.3** インポート文の整理
  - src/dependencies.tsの整理・統合
  - test/dependencies.tsの更新

### Phase 3: コード修正
- [ ] **3.1** 廃止予定APIの置換
  - `Deno.cwd()` → 必要に応じて他のAPI検討
  - ファイルシステムAPI確認

- [ ] **3.2** 型定義の更新
  - TypeScript設定の最適化
  - 型安全性の向上

- [ ] **3.3** テストの修正
  - テスト用アサーション関数の更新
  - テスト実行方式の確認

### Phase 4: 設定とスクリプトの最適化
- [ ] **4.1** deno.jsonタスクの定義
  - `deno task test`
  - `deno task fmt`
  - `deno task lint`
  - `deno task dev`

- [ ] **4.2** 既存スクリプトの見直し
  - scripts/format.shのdeno.jsonタスク化

- [ ] **4.3** CI/CD設定の更新（該当する場合）

### Phase 5: テストと検証
- [ ] **5.1** 単体テストの実行・修正
- [ ] **5.2** 統合テストの実行
- [ ] **5.3** パフォーマンステスト
- [ ] **5.4** 動作確認

## 想定される課題と対策

### 1. 互換性の問題
**問題**: 古いAPIが削除されている可能性
**対策**: 公式ドキュメントとマイグレーションガイドを参照し、代替APIを使用

### 2. 依存関係の競合
**問題**: ライブラリの大幅なAPI変更
**対策**: 段階的な更新、必要に応じて代替ライブラリの検討

### 3. パフォーマンスの変化
**問題**: 新しいバージョンでのパフォーマンス変化
**対策**: ベンチマークテストの実施、最適化の検討

## スケジュール

| フェーズ | 想定工数 | 説明 |
|---------|---------|------|
| Phase 1 | 1-2時間 | 環境準備・設定ファイル作成 |
| Phase 2 | 3-4時間 | 依存関係の調査・更新 |
| Phase 3 | 2-3時間 | コード修正・API更新 |
| Phase 4 | 1-2時間 | 設定最適化・スクリプト整理 |
| Phase 5 | 2-3時間 | テスト・検証・デバッグ |
| **合計** | **9-14時間** | |

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

---
*最終更新: 2025-08-17*
*作成者: Claude Code*