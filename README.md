# spot-diggz

スケートボード利用者が、その日の目的や条件に合うセッション先を決めるための多言語Webサービス。

## 現在の状態

このリポジトリは作り直しの初期段階にある。旧iOS、Android、Rust API、React UI、GCP／Firebase構成は現行ツリーから削除し、必要な場合はGitのコミット履歴から参照する。

現在は、検証済み施設カタログを起点にGo製MVPを実装している。現行カタログは未検証施設を混入させないため空であり、公式情報の確認後に登録する。

## ドキュメント

- [プロダクト要求基準](docs/product_baseline.md)
- [市場・需要調査](docs/market_demand_research_2026-07.md)
- [開発ワークフロー](docs/development_workflow.md)
- [Discovery Sprint 0検証計画](docs/discovery/sprint-0-validation-plan.md)
- [大阪都市圏の施設候補](docs/discovery/osaka-facility-candidates.md)
- [ADR一覧](docs/adr/)
- [Facility Catalog API契約](docs/api/facility-catalog.openapi.yaml)
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

### Go API

- `make fmt`: Goコードを整形する。
- `make test`: 全Goテストを実行する。
- `make vet`: Go静的検査を実行する。
- `make build`: APIバイナリをビルドする。
- `make run`: ローカルAPIを起動する。

初期APIは `GET /healthz`、`GET /api/facilities`、`GET /api/facilities/{facilityId}`。施設一覧は `activity` クエリで競技を絞り込める。カタログファイルは `FACILITY_CATALOG_PATH` で変更できる。

### ドキュメント検証

- `git grep -n "検索語" -- ':!docs/market_demand_research_2026-07.md'`: 現行ツリーの参照や古いパスを検索する。

## 次の作業

施設カタログAPIの基盤は実装済み。次はIssue #280で大阪都市圏の候補23件を一次情報により検証し、`data/facilities.json`へ掲載可能な施設を登録する。その後、条件入力と決定論的推薦を実装する。
