# spot-diggz

スケボーをしたい人が、その日の目的や条件に合う行き先を決めるためのサービス。対象地域を限定せず、UIやSlack・Discordのbot・appなどから利用できるAPIとして、施設情報と推薦機能を提供できるように整備していきます。

新MVPは **Slack/Discord → 自分のBotサーバー → SpotDiggz API** とし、独自Web UIは提供しません。APIホストはCloud Run。月額目安は全体でUSD 3、Google Cloudの予算アラートメールのみを使い、超過しても自動停止しません。条件送信後は「検索中」、完了後に結果またはエラーを返します。方針の正本は[API契約](docs/specifications/api-mvp.md)と[Cloud Run運用計画](docs/operations/cloud-run-plan.md)です。実装・公開・通知設定は未完了です。

## 公開状況

従来の公開先は2026-09-06にトップページ・health endpointともHTTP 404を確認しました。現在利用できる本番URLは未確認のため、アクセスリンクは掲載していません。[公開先と旧資料の整理状況](docs/operations/service-status.md)を参照してください。

以下はリポジトリに存在する実装・運用方針です。本番で現在提供中であることを示すものではありません。

常設のステージング環境は設けない。通常はローカルとCIで検証し、Cloud Run移行時に必要な一時検証環境は#318で評価する。旧Vercel Previewを新構成の既定にはしない。実際の公開変更は別途承認後に行い、本番環境でスモークテストを実施する。

spot-diggzは、施設を地図で眺めるだけではなく、「今日、今の自分がどこへ滑りに行くか」を決めるためのサービスです。利用目的、気分、レベル、使える時間、出発地点、交通手段から、検証済みの施設を理由付きで比較できます。

新MVPの利用の流れは[API契約](docs/specifications/api-mvp.md)を参照してください。[How To Use](docs/guides/how-to-use.md)は移行前Web実装の参照です。現在の要求・仕様・設計・運用文書は[Documentation](docs/README.md)からたどれます。

## 独立読み取りAPI（新MVPの第一実装）

`APP_MODE=api`で、専用Bearer認証付き`POST /api/facilities/search`と`GET /api/facilities/{facilityId}`を起動できます。query（場所名）と任意genre/limit/radiusKm/sortを受け付け、検証済み情報から直線距離順に返します。GitHub OAuth・DB・独自Web UIはこのmodeに不要です。[API仕様](docs/specifications/read-api.md)と[ローカル設定手順](docs/guides/read-api-setup.md)を参照してください。

実装済みなのはAPI部分です。Slack/Discordの新メンション接続、実データのgenre分類・鮮度再確認、Cloud Run/Gateway公開は未完了です。本番catalogの確認日を自動で更新せず、未分類/古いデータは検索から除外します。以下の旧Web構成は互換用legacy modeの説明であり、API modeの依存条件ではありません。

## 現在の状態

現在の実装は、Web UI、API、検証済み施設カタログ、決定論的な推薦を1つのGo applicationに含むモジュラーモノリスです。Slack・Discord adapterは内部の共通推薦serviceを呼び出しており、独立した各bot・appがHTTP APIを利用する構成への整備は今後の作業です。

現在のcatalogと都道府県の許可値は大阪府、兵庫県、和歌山県、奈良県、徳島県の5府県です。これは既存データと実装の制約であり、ターゲットの地域制限ではありません。2026-07-19調査基準の公開カタログには31施設（大阪府24施設）を登録しています。日付別の一般利用予定を確認できない施設はカタログ参照のみとし、推薦から除外します。

private MVPでは、Web UIと`/api/*`をGitHub OAuthで`GITHUB_OWNER`に一致するownerだけへ制限する。Slack `/spotdiggz`は条件入力モーダルを開き、出発地・時間・交通手段・レベル・目的・気分から最大3件をephemeral responseで返す。候補は保存せず、「公式情報」と「ここに行く」の外部導線だけを表示する。SlackとDiscordはplatform署名と設定済みworkspace/guild/user IDを検証し、過去messageを参照・保存しない。

推薦は目的、気分、レベル、利用可能時間、検索位置、交通手段を受け取り、鮮度、休場期間、通常営業時間、移動時間、初心者適性、設備を決定論的に評価する。動的情報は30日、安定情報は180日を鮮度期限とし、期限超過施設は推薦しない。休場期間は一回限りの `one_time` と毎年繰り返す `annual` を扱う。

`GOOGLE_MAPS_API_KEY` がある場合はGoogle Routes APIのCompute Route Matrixを優先し、失敗時は直線距離の概算へ自動で縮退する。同じkeyでGoogle Geocoding APIによる地点検索も有効になる。keyがない場合、推薦は直線距離概算で動作し、任意地点検索は `503` を返す。正確な検索位置や検索文字列はapplicationで永続化せず、access logにも出力しない。ただしGoogle連携を有効にすると、推薦の起点座標はGoogle Routesへ、地点検索文字列はGoogle Geocodingへ送信される。

訂正報告は `DATABASE_URL` が設定されている場合にNeon/PostgreSQLへ保存し、未設定時はローカル開発用のJSON Lines file storeへフォールバックする。施設カタログ自体は、検証可能性と同一成果物の再現性を優先してGit管理JSONを正本とする。

## 実装済みと未実施の境界

- [事実] Go adapter、入力検証、Google失敗時のfallback、日英表示、訂正報告、rate limit、構造化access log、Prometheus metricsはローカル実装と自動テストの対象である。
- [事実] GitHub OAuth owner allowlist、署名済みsession、Slack HMAC署名とowner ID認可、Discord Ed25519署名とowner ID認可はローカル実装と自動テストの対象である。
- [事実] Production正式domainでのGitHub owner loginとSlack `/spotdiggz`のmodal・候補応答を2026-08-03にownerが確認した。Discord interactionは未検証である。
- [事実] 訂正報告は`DATABASE_URL`設定時にNeon/PostgreSQL、未設定時に `var/corrections.jsonl` へ保存する。任意の連絡先は明示同意がある場合だけ受理し、送信時に90日後の削除期限を付け、起動時と1時間ごとに期限超過分をpurgeする。
- [未検証] 実Google APIへの接続、quota・課金・key制限、production環境からのfallbackは、資格情報がないため確認していない。
- [事実] 既存Neon Organizationの`spotdiggz` Project、Vercel Project、migration、Productionのhealth/readiness、施設API、UIを2026-07-20に確認した。
- [未検証] `/metrics`のnetwork制限、Google API quota・課金・key制限、custom domain/DNS、GitHub `main` pushからの自動Production deployは未設定である。

## ドキュメント

- [文書入口・現在の正本](docs/README.md)
- [プロダクト](docs/product.md)
- [要求](docs/requirements.md)
- [仕様一覧](docs/specifications/README.md)
- [MVP API契約](docs/specifications/facility-catalog.openapi.yaml)
- [アーキテクチャ](docs/architecture.md)
- [セキュリティ・プライバシー](docs/security.md)
- [Decision Record一覧](docs/decisions/README.md)
- [開発・文書・release process](docs/process/development.md)
- [運用文書一覧](docs/operations/README.md)
- [公開先の確認状況](docs/operations/service-status.md)
- [利用・設定guide一覧](docs/guides/README.md)
- [調査資料](docs/research/README.md)

## 使うコマンド一覧

新APIの起動は[設定手順](docs/guides/read-api-setup.md)に従い環境を注入して`go run ./cmd/api`、ネットワーク不要の回帰確認は`go test ./internal/apiauth ./internal/nearby ./internal/httpapi ./cmd/api`です。`APP_MODE=api`を必ず明示してください。その他の以下のコマンドは主に現在残っている移行前実装用です。特にWeb UI・GitHub OAuth・Vercel設定/deploy用のものは新MVPのセットアップではありません。Cloud Run用手順は#318で整備し、旧公開先・secret登録・外部設定を確認なく実行しません。

### Git

- `git status --short --branch`: 現在のブランチと作業ツリーを確認する。
- `git diff --check`: 空白エラー等を確認する。
- `git diff -- README.md docs/`: 文書差分だけを確認する。
- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\validate-docs.ps1`: 必須文書、命名、内部link、compatibility stub、Decision Recordを検証する。

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
- `powershell -File .\scripts\register-discord-command.ps1 -ApplicationId '<id>' -GuildId '<id>'`: Discordの`/spotdiggz` guild commandを登録または更新する。Bot Tokenはsecure promptへ入力し、引数やshell履歴へ含めない。
- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\install-integration-clis.ps1`: Slack CLIとVercel CLIをuser領域へ導入し、versionを確認する。
- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\check-integration-cli-prerequisites.ps1`: 外部変更をせず、CLI、user PATH、Slack Manifest、hook、Vercel project linkのlocal事前準備を検証する。
- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\configure-slack-app.ps1`: Slack CLI認証済みの場合だけ、選択した既存AppへManifestを反映する任意手順。HTTP連携自体にSlack CLI認証は不要。
- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\configure-slack-vercel-env.ps1 -Deploy`: Slack連携用の秘密値を非表示promptからVercel Productionへ設定し、deploy後にhealth/readinessを確認する。
- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\configure-slack-vercel-env.ps1 -PreflightOnly`: 秘密値や外部状態を変更せず、Vercel認証と`spotdiggz` Project linkを確認する。
- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\configure-github-vercel-env.ps1 -PreflightOnly`: GitHub OAuth用のVercel認証、Project link、callbackを外部変更なしで確認する。
- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\configure-github-vercel-env.ps1 -Deploy`: GitHub OAuth credentialをpromptからVercel Productionへ登録し、`AUTH_SECRET`を生成してProduction deployとhealth/readiness確認を行う。

### 起動

- `make run`: `data/facilities.json` を使い、`http://localhost:8080/` で起動する。OAuth未設定時は認証設定案内を表示する。
- `DEV_AUTH_BYPASS=1 make run`: ローカル画面確認だけを目的にGitHub認証を省略して起動する。Productionでは無効である。
- `set -a; . ./.env.local; set +a; make run`: 既存Neonの`spotdiggz` DBを使ってlocal起動する。`.env.local`はGitへcommitしない。
- `make run-dev`: `testdata/facilities.dev.json` のダミーデータで起動する。
- `PORT=8081 make run-dev`: listen portを変更して起動する。
- `CORRECTION_STORE_PATH=/tmp/spotdiggz-corrections.jsonl make run-dev`: 訂正file storeを一時pathへ変更する。
- `GOOGLE_MAPS_API_KEY='<secret>' make run`: Google RoutesとGoogle Geocodingを有効にする。実値はsecret storeから注入し、shell履歴やGitへ残さない。

| Environment variable | Required | Default | Purpose |
| --- | --- | --- | --- |
| `APP_MODE` | API単体起動時yes | `legacy` | `api`は認証付き読み取りrouteのみ。未知値は起動拒否 |
| `API_OWNER_ID` | API mode yes | unset | 個人情報でない許可owner内部ID |
| `API_CLIENTS_JSON` | API mode yes | unset | client別digest・owner・scope・期限・失効設定。APIには平文tokenを入れない |
| `PORT` | no | `8080` | HTTP listen port |
| `FACILITY_CATALOG_PATH` | no | `data/facilities.json` | 起動時に読む検証済みcatalog |
| `CORRECTION_STORE_PATH` | no | `var/corrections.jsonl` | 訂正報告のJSON Lines file |
| `DATABASE_URL` | Production Slack利用時yes | unset | Neon/PostgreSQLの訂正報告と短期Slack request状態。設定時はfile storeより優先 |
| `GOOGLE_MAPS_API_KEY` | API検索/旧Slack利用時yes | unset | API modeはGeocodingのみ。legacyはRoutes / Geocoding |
| `APP_ENV` | no | `development` | JSON logのenvironment。image既定値は `production` |
| `APP_VERSION` | no | `unknown` | JSON logへ付けるrelease SHAまたはversion |
| `APP_BASE_URL` | legacy Production yes | unset | GitHub OAuth callbackを構成するpathなしの正式HTTPS origin。例: `https://<deployment-host>` |
| `AUTH_SECRET` | legacy Production yes | unset | owner sessionとOAuth stateへ署名する32 bytes以上のrandom secret |
| `GITHUB_CLIENT_ID` | legacy Production yes | unset | GitHub OAuth App client ID |
| `GITHUB_CLIENT_SECRET` | legacy Production yes | unset | GitHub OAuth App client secret |
| `GITHUB_OWNER` | no | `kohei321dev` | 利用を許可するGitHub login |
| `DEV_AUTH_BYPASS` | no | unset | local UI/E2E専用。`1`で認証を省略し、Productionでは常に無効 |
| `SLACK_BOT_TOKEN` | Slack利用時yes | unset | `views.open`と`chat.postEphemeral`に使うBot User OAuth Token |
| `SLACK_SIGNING_SECRET` | Slack利用時yes | unset | Slack request署名検証secret |
| `SLACK_TEAM_ID` | Slack利用時yes | unset | 許可workspace ID |
| `SLACK_OWNER_USER_ID` | Slack利用時yes | unset | GitHub ownerへ対応付けるSlack user ID |
| `DISCORD_PUBLIC_KEY` | Discord利用時yes | unset | interaction Ed25519 public key（hex） |
| `DISCORD_APPLICATION_ID` | Discord利用時yes | unset | 許可Discord application ID |
| `DISCORD_GUILD_ID` | Discord利用時yes | unset | 許可guild ID |
| `DISCORD_OWNER_USER_ID` | Discord利用時yes | unset | GitHub ownerへ対応付けるDiscord user ID |
| `CHAT_DEFAULT_ORIGIN_LATITUDE` / `CHAT_DEFAULT_ORIGIN_LONGITUDE` | Discord利用時yes | unset | Discord用の公開代表起点。Slackはモーダル入力を使う |
| `CHAT_DEFAULT_PURPOSE` | no | `basics` | Discord推薦の目的 |
| `CHAT_DEFAULT_MOOD` | no | `focused` | Discord推薦の気分 |
| `CHAT_DEFAULT_LEVEL` | no | `beginner` | Discord推薦のlevel |
| `CHAT_DEFAULT_AVAILABLE_MINUTES` | no | `120` | Discord推薦の利用可能時間 |
| `CHAT_DEFAULT_TRANSPORT` | no | `public_transit` | Discord推薦の交通手段 |

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

API modeのroute allowlistは[読み取りAPI](docs/specifications/read-api.md#route-allowlistと互換性)を参照してください。以下はlegacy modeです。

主要endpointは `GET /healthz`、`GET /readyz`、GitHub OAuth用`/auth/github/*`、owner sessionが必要な`/api/*`、Slack用`POST /integrations/slack/commands`、Discord用`POST /integrations/discord/interactions`、`GET /metrics` である。request、response、制限、error codeの正本は [OpenAPI](docs/specifications/facility-catalog.openapi.yaml) とする。

## 次の作業

通常CIはGo format/vet/race test/build、Go source/binary脆弱性検査、JSON/OpenAPI・文書検証、secret scan、PR dependency reviewを実行する。Docker・Playwright・成果物保存・週次scheduleは含めない。運用変更の理由は[DR-0017](docs/decisions/0017-minimal-development-ci.md)を参照する。

ownerはrelease前と定期保守で `make verify-catalog` を実行する。production catalogの再確認期限が7日以内に迫ると失敗するため、施設の `sourceUrl` を確認し、事実を再検証してから検証時刻と休場情報を更新する。固定の開発・E2E fixtureはこの鮮度判定に使用しない。E2Eとcontainer検証は[リリース手順](docs/process/release.md)で別途確認する。

release前に、5府県catalogの公式情報再確認、制限済みGoogle credentialでの実通信、永続volumeを伴う実デプロイ、privateなmetrics収集、デプロイ後smoke、rollback演習を完了する。外部API有効化と実デプロイは資格情報・課金設定・platform権限が必要なため、このリポジトリのローカル実装完了とは分けて判定する。
