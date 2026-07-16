# spot-diggz

スケートボード利用者が、その日の目的や条件に合うセッション先を決めるための多言語Webサービス。

## 現在の状態

このリポジトリは作り直しの初期段階にある。旧iOS、Android、Rust API、React UI、GCP／Firebase構成は現行ツリーから削除し、必要な場合はGitのコミット履歴から参照する。

現在は、検証済み施設カタログ、選択式のセッション入力、決定論的な推薦API、スマートフォン対応Web UIを1つのGo applicationとして実装している。現行の本番カタログは未検証施設を混入させないため空であり、公式情報の確認後に登録する。

## ドキュメント

- [プロダクト要求基準](docs/product_baseline.md)
- [市場・需要調査](docs/market_demand_research_2026-07.md)
- [開発ワークフロー](docs/development_workflow.md)
- [Discovery Sprint 0検証計画](docs/discovery/sprint-0-validation-plan.md)
- [大阪都市圏の施設候補](docs/discovery/osaka-facility-candidates.md)
- [ADR一覧](docs/adr/)
- [MVP API契約](docs/api/facility-catalog.openapi.yaml)
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
- `make run-dev`: 3件の開発用ダミー施設を読み込んでAPIとWeb UIを起動する。
- `make verify-mvp`: ダミーデータでUI配信から推薦responseまでの主要flowを実HTTP検証する。

`make run-dev`の起動後、`http://localhost:8080/`で条件入力から推薦結果まで確認できる。APIは `GET /healthz`、`GET /api/facilities`、`GET /api/facilities/{facilityId}`、`POST /api/recommendations`。施設一覧は `activity` クエリで競技を絞り込める。カタログファイルは `FACILITY_CATALOG_PATH` で変更できる。

推薦入力の目的、気分、レベル、利用可能時間、交通手段は列挙値として検証する。検索起点は現在地または指定地点の座標を受け取るが、保存・access log出力・responseへの再掲はしない。外部経路providerの選定前であるため、移動時間は直線距離による概算であり、実経路は結果画面の外部ナビで確認する。

開発用データは `testdata/facilities.dev.json` に置き、施設名、住所、出典を含めてすべてダミーである。通常起動と本番Docker imageではこのファイルを使用しない。

### ドキュメント検証

- `git grep -n "検索語" -- ':!docs/market_demand_research_2026-07.md'`: 現行ツリーの参照や古いパスを検索する。

## 次の作業

条件入力からダミー施設の推薦、情報源表示、外部ナビまでの開発用flowを実装済み。MVPリリースには、Issue #280の一次情報による施設登録、実経路providerの選定、日本語・英語表示、訂正報告、主要flowのE2E検証が残る。
