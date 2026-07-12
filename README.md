# spot-diggz

スケートボード利用者が、その日の目的や条件に合うセッション先を決めるための多言語Webサービス。

## 現在の状態

このリポジトリは作り直しの初期段階にある。旧iOS、Android、Rust API、React UI、GCP／Firebase構成は現行ツリーから削除し、必要な場合はGitのコミット履歴から参照する。

現在は、実装より先に市場・利用者課題・要求・MVP境界を検証する。

## ドキュメント

- [プロダクト要求基準](docs/product_baseline.md)
- [市場・需要調査](docs/market_demand_research_2026-07.md)
- [開発ワークフロー](docs/development_workflow.md)
- [ADR一覧](docs/adr/)
- [エージェント運用規約](AGENTS.md)
- [技術書から採用した原則](docs/engineering/principles-and-sources.md)
- [品質特性・アーキテクチャ指針](docs/architecture/quality-attributes.md)
- [ログ・メトリクス・トレース・SLO設計](docs/operations/observability.md)
- [継続的デリバリー設計](docs/operations/continuous-delivery.md)
- [セキュリティ・プライバシー基準](docs/security/security-baseline.md)

## 使うコマンド一覧

### Git

- `git status --short --branch`: 現在のブランチと作業ツリーの状態を確認する。
- `git branch --show-current`: 現在のブランチ名を確認する。
- `git log --oneline --decorate -5`: 直近のコミットとブランチ位置を確認する。
- `git diff --check`: 空白エラーなど、コミット前の差分問題を確認する。

### ドキュメント検証

- `git grep -n "検索語" -- ':!docs/market_demand_research_2026-07.md'`: 現行ツリーの参照や古いパスを検索する。

## 次の作業

Discovery Sprint 0で対象地域を1つ決め、20〜30施設の検証済みカタログと、利用者が入力する条件を定義する。その後、最小の推薦体験をWebで検証する。
