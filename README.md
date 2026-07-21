# spot-diggz

スケートボード利用者が、その日の目的や条件に合うセッション先を決めるための日本語・英語対応Webサービス。

## 本番版

- [spot-diggz を開く](https://spotdiggz.vercel.app)
- 本番URL: `https://spotdiggz.vercel.app`

常設のステージング環境は設けない。通常の変更はローカル環境とCIで検証し、外部サービス連携、データ移行、インフラ変更など本番との差分によるリスクがある変更では、必要な期間だけVercel Previewを明示的に作成して確認する。`main`への反映後は、本番環境でスモークテストを実施する。

spot-diggzは、施設を地図で眺めるだけではなく、「今日、今の自分がどこへ滑りに行くか」を決めるためのサービスです。利用目的、気分、レベル、使える時間、出発地点、交通手段から、検証済みの施設を理由付きで比較できます。

初めて使う場合は、[How To Use](docs/how-to-use.md) を参照してください。

## 現在の状態

MVPは、Web UI、API、検証済み施設カタログ、決定論的な推薦を1つのGo applicationとして提供するモジュラーモノリスである。地理scopeは大阪府、兵庫県、和歌山県、奈良県、徳島県の5府県とし、2026-07-19調査基準の公開カタログには公式情報で必須属性を確認した31施設を登録している。大阪府は24施設で、日付別の一般利用予定を確認できない施設はカタログ参照のみとし、推薦から除外する。

推薦は目的、気分、レベル、利用可能時間、検索位置、交通手段を受け取り、鮮度、休場期間、通常営業時間、移動時間、初心者適性、設備を決定論的に評価する。動的情報は30日、安定情報は180日を鮮度期限とし、期限超過施設は推薦しない。休場期間は一回限りの `one_time` と毎年繰り返す `annual` を扱う。

`GOOGLE_MAPS_API_KEY` がある場合はGoogle Routes APIのCompute Route Matrixを優先し、失敗時は直線距離の概算へ自動で縮退する。同じkeyでGoogle Geocoding APIによる地点検索も有効になる。keyがない場合、推薦は直線距離概算で動作し、任意地点検索は `503` を返す。正確な検索位置や検索文字列はapplicationで永続化せず、access logにも出力しない。ただしGoogle連携を有効にすると、推薦の起点座標はGoogle Routesへ、地点検索文字列はGoogle Geocodingへ送信される。

訂正報告は `DATABASE_URL` が設定されている場合にNeon/PostgreSQLへ保存し、未設定時はローカル開発用のJSON Lines file storeへフォールバックする。施設カタログ自体は、検証可能性と同一成果物の再現性を優先してGit管理JSONを正本とする。

## 実装済みと未実施の境界

- [事実] Go adapter、入力検証、Google失敗時のfallback、日英表示、訂正報告、rate limit、構造化access log、Prometheus metricsはローカル実装と自動テストの対象である。
- [事実] 訂正報告は`DATABASE_URL`設定時にNeon/PostgreSQL、未設定時に `var/corrections.jsonl` へ保存する。任意の連絡先は明示同意がある場合だけ受理し、送信時に90日後の削除期限を付け、起動時と1時間ごとに期限超過分をpurgeする。
- [未検証] 実Google APIへの接続、quota・課金・key制限、production環境からのfallbackは、資格情報がないため確認していない。
- [事実] 既存Neon Organizationの`spotdiggz` Project、Vercel Project、migration、Productionのhealth/readiness、施設API、UIを2026-07-20に確認した。
- [未検証] `/metrics`のnetwork制限、Google API quota・課金・key制限、custom domain/DNS、GitHub `main` pushからの自動Production deployは未設定である。

## ドキュメント

- [プロダクト要求基準](docs/product_baseline.md)
- [MVP API契約](docs/api/facility-catalog.openapi.yaml)
- [MVP運用Runbook](docs/operations/mvp-runbook.md)
- [ログ・メトリクス・SLO設計](docs/operations/observability.md)
- [継続的デリバリー設計](docs/operations/continuous-delivery.md)
- [Vercel・Neonデプロイ手順](docs/operations/vercel-neon-deployment.md)
- [How To Use](docs/how-to-use.md)
- [セキュリティ・プライバシー基準](docs/security/security-baseline.md)
- [品質特性・アーキテクチャ指針](docs/architecture/quality-attributes.md)
- [ADR一覧](docs/adr/)
- [市場・需要調査](docs/market_demand_research_2026-07.md)
- [開発ワークフロー](docs/development_workflow.md)
- [Discovery Sprint 0検証計画](docs/discovery/sprint-0-validation-plan.md)
- [大阪都市圏の施設候補](docs/discovery/osaka-facility-candidates.md)
- [技術書から採用した原則](docs/engineering/principles-and-sources.md)
- [エージェント運用規約](AGENTS.md)

## 使うコマンド一覧

### Git

- `git status --short --branch`: 現在のブランチと作業ツリーを確認する。
- `git diff --check`: 空白エラー等を確認する。
- `git diff -- README.md docs/`: 文書差分だけを確認する。

### Buildとtest

- `make fmt`: Goコードを整形する。
- `make test`: 全Goテストを実行する。
- `make vet`: Go静的検査を実行する。
- `make build`: `CGO_ENABLED=0` で静的な単一binary `bin/spotdiggz-api` をビルドする。
- `make verify-catalog`: production catalogが実行時点から168時間後もdynamic 30日・stable 180日の鮮度内であることを検査する。
- `make verify-mvp`: ダミーデータでUI配信と推薦APIの主要flowを実HTTP検証する。
- `npm ci`: lockfileどおりにPlaywright E2E依存をinstallする。
- `npm run test:contracts`: JSON dataとOpenAPIの構文・path・local referenceを検証する。
- `npx playwright install chromium`: local E2E用Chromiumをinstallする。
- `npm run test:e2e`: desktop / mobileの主要flowをheadless Chromiumで検証する。
- `npm run test:e2e:headed`: E2Eを画面表示付きで調査する。
- `docker build --tag spotdiggz-api:local .`: production相当のOCI imageをローカルでbuildする。
- `npx vercel --prod`: `Dockerfile.vercel` を使ってVercelへProduction deployする。初回はProjectへのlinkが必要。
- `psql "$DATABASE_URL" -f db/migrations/0001-correction-reports.sql`: Neonへ訂正報告テーブルmigrationを適用する。接続文字列はshell履歴やGitへ残さない。
- `go run ./cmd/dbmigrate`: `DATABASE_URL`を読み、Neonへ管理済みmigrationを適用する。接続文字列自体は出力しない。
- `go run ./cmd/api correctioncheck -path <corrections.jsonl>`: 訂正storeを変更せず、report件数・期限切れ件数・破損行を検査する。report本文や連絡先は出力しない。

### 起動

- `make run`: `data/facilities.json` を使い、`http://localhost:8080/` で起動する。
- `set -a; . ./.env.local; set +a; make run`: 既存Neonの`spotdiggz` DBを使ってlocal起動する。`.env.local`はGitへcommitしない。
- `make run-dev`: `testdata/facilities.dev.json` のダミーデータで起動する。
- `PORT=8081 make run-dev`: listen portを変更して起動する。
- `CORRECTION_STORE_PATH=/tmp/spotdiggz-corrections.jsonl make run-dev`: 訂正file storeを一時pathへ変更する。
- `GOOGLE_MAPS_API_KEY='<secret>' make run`: Google RoutesとGoogle Geocodingを有効にする。実値はsecret storeから注入し、shell履歴やGitへ残さない。

| Environment variable | Required | Default | Purpose |
| --- | --- | --- | --- |
| `PORT` | no | `8080` | HTTP listen port |
| `FACILITY_CATALOG_PATH` | no | `data/facilities.json` | 起動時に読む検証済みcatalog |
| `CORRECTION_STORE_PATH` | no | `var/corrections.jsonl` | 訂正報告のJSON Lines file |
| `DATABASE_URL` | no | unset | Neon/PostgreSQLの訂正報告保存先。設定時はfile storeより優先 |
| `GOOGLE_MAPS_API_KEY` | no | unset | Google Routes / Geocodingのserver-side credential |
| `APP_ENV` | no | `development` | JSON logのenvironment。image既定値は `production` |
| `APP_VERSION` | no | `unknown` | JSON logへ付けるrelease SHAまたはversion |

### Local smoke

application起動後に次を確認する。完全な手順とrollback条件は [MVP運用Runbook](docs/operations/mvp-runbook.md) を参照する。

- `curl --fail --silent http://localhost:8080/healthz`: processのlivenessを確認する。
- `curl --fail --silent http://localhost:8080/readyz`: dynamic 30日・stable 180日の両方が鮮度内の施設が1件以上あることを確認する。emptyまたは全件staleは503になる。
- `curl --fail --silent http://localhost:8080/api/facilities?activity=skateboard`: 公開catalogを確認する。
- `curl --fail --silent http://localhost:8080/metrics`: Prometheus形式のmetricsを確認する。
- `curl --fail --silent --header 'Content-Type: application/json' --data '{"query":"神戸駅"}' http://localhost:8080/api/locations/search`: Google連携時の地点検索を確認する。検索文字列をURLへ含めない。

開発用データは施設名、住所、出典を含めてすべてダミーである。通常起動とproduction imageでは使用しない。

production imageはscratchを使い、Google HTTPS通信用CA bundle、UID `65532` が書き込めるlocal fallback用訂正store directory、`CGO_ENABLED=0` の単一binaryだけを含む。Vercel ProductionではNeon/PostgreSQLを使うため、訂正reportをcontainer filesystemへ保存しない。通常のOCI運用でfile storeを使う場合だけ、`/var/lib/spotdiggz`へpersistent volumeをmountする。

## API

主要endpointは `GET /healthz`、`GET /readyz`、`GET /api/facilities`、`GET /api/facilities/{facilityId}`、`POST /api/locations/search`、`POST /api/recommendations`、`POST /api/corrections`、`POST /api/events`、`GET /metrics` である。request、response、制限、error codeの正本は [OpenAPI](docs/api/facility-catalog.openapi.yaml) とする。

## 次の作業

CIは毎週 `make verify-catalog` を実行し、production catalogの再確認期限が7日以内に迫ると失敗する。これは公式情報の再調査を自動化するものではないため、失敗時は施設の `sourceUrl` を確認し、事実を再検証してから検証時刻と休場情報を更新する。固定の開発・E2E fixtureはこの鮮度判定に使用しない。

release前に、5府県catalogの公式情報再確認、制限済みGoogle credentialでの実通信、永続volumeを伴う実デプロイ、privateなmetrics収集、デプロイ後smoke、rollback演習を完了する。外部API有効化と実デプロイは資格情報・課金設定・platform権限が必要なため、このリポジトリのローカル実装完了とは分けて判定する。
