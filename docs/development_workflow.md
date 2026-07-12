# spot-diggz 開発ワークフロー

## 方針

spot-diggzでは、Lean Product Discoveryと短い開発スプリントを組み合わせる。

- Discovery: 市場、利用者課題、要求、仮説を検証する
- Delivery: 設計、実装、テスト、実利用可能な成果物を作る
- Review: 実際の利用結果を確認する
- Retro: 次の改善点を記録する

プロダクト要求が未検証の段階では、機能を大量に実装せず、検証可能な最小の縦切りを作る。

## スプリント

個人開発では1週間を基本とする。各スプリントには、作業量ではなく1つの検証可能なSprint Goalを設定する。

```text
Sprint Goal:
何を検証・実現するか

Deliverable:
実際に触れる成果物

Evidence:
利用者反応、テスト結果、計測値
```

### Sprint Planning

- Product BaselineとGitHub Issueを確認する
- Sprint Goalを1つ選ぶ
- 完了条件を決める
- 実装前に必要なADRを確認する

### Sprint Review

- 実際に動く画面またはAPIを確認する
- 利用者が目的地決定まで進めるか確認する
- 期待した証拠が得られたかを記録する

### Sprint Retrospective

次の3点を短く記録する。

- 継続すること
- 問題だったこと
- 次に試すこと

## Definition of Ready

Issueを着手する前に、最低限次を満たす。

- 対象ユーザーが定義されている
- WhatとWhyがProduct Baselineまたは関連Issueにある
- 完了条件が確認可能である
- 必要な設計判断がADRに記録されている、または不要と判断されている
- 外部データ・位置情報・AI出力のリスクが確認されている

## Definition of Done

- 実装と関連テストが完了している
- 主要な利用シナリオを手動または自動で確認している
- 施設情報には出典URLと確認日がある
- AIが未確認情報を断定しない
- 必要なドキュメントを更新している
- 変更の意図がIssueまたはコミットに記録されている

## ブランチ

```text
main                           新プロダクトの安定状態・統合ブランチ
archive/*                      明示的に残す旧実装・節目の退避
feature/*                      短期間の機能ブランチ
docs/*                         要求・設計・調査ドキュメント
```

新しい機能は`main`から短命なfeatureブランチを作り、検証可能な単位で統合する。過去の実装はGitのコミット履歴で参照し、特別な退避ブランチは必要な場合だけ作成する。

## ドキュメントの使い分け

| 内容 | 場所 |
|---|---|
| プロダクトの目的・対象・要求 | `docs/product_baseline.md` |
| 市場調査・利用者調査 | `docs/market_demand_research_2026-07.md` |
| 設計判断と代替案 | `docs/adr/` |
| 実装タスク・検証タスク | GitHub Issues |
| AIエージェントの作業規約 | `AGENTS.md` |
| 実装の詳細 | コード内コメント・各ディレクトリの設計資料 |

## 初期スプリント案

### Discovery Sprint 0

- Product Baselineのレビュー
- 初期都市圏の決定
- 20〜30施設のデータ項目確認
- 比較対象サービスの決定
- 利用者テストの質問と成功条件の決定

### Sprint 1: Facility Catalog

- 施設データモデル
- 公式情報の収集
- 20〜30施設の初期登録
- 施設詳細表示

### Sprint 2: Recommendation

- 条件入力
- 利用不可施設の除外
- ルールベースのスコアリング
- 最大3件の推薦と理由表示

### Sprint 3: Validation

- 外部ナビ連携
- 訂正報告
- 利用者テスト
- 実際の訪問・再利用の計測

### Sprint 4: AI and Localization

- 自然文の条件変換
- 推薦理由の説明
- 日本語・英語対応

この計画は初期案であり、各Sprint Reviewの証拠により変更する。
