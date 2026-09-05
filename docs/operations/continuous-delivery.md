# Continuous Delivery Pipeline Design

- Status: Minimal development CI; release verification is manual
- Date: 2026-09-05
- CI: GitHub Actions
- Deployment platform: Vercel Container (Services single `app` service)
- Correction store: Neon/PostgreSQL in Production, file fallback in local/CI

## 1. 目的

同じcommitから静的Go binaryとscratch OCI imageを再現し、code、catalog、UI、security、container起動をrelease前に検証する。CI成功はdeploy成功を意味しない。registry公開、production設定、smoke、観察、rollbackを完了してreleaseとする。

## 2. Delivery contract

- source、catalog、OpenAPI、CI、DockerfileをGitでversion管理する
- `main` と短命branchに同じrequired verificationを適用する
- `CGO_ENABLED=0` で静的な単一Go binaryをbuildする
- production imageはscratch、non-root UID `65532` で実行する
- imageには検証済みcatalog、Google HTTPS用CA bundle、non-root書込directoryを含める
- Productionのcorrection reportはimage外のNeon/PostgreSQLに保存し、local/CIではfile storeを使う
- secretをsource、image、artifact、logへ含めない
- deploy後にhealth、freshness-aware readiness、recommendation、metricsを確認する
- rollbackは直前imageと同じNeon接続設定を使い、reportやschemaを上書きしない。file fallbackでは同じcorrection volumeを使う

## 3. Branch and review

- default branchは `main`
- 共有branchへ直接commitしない
- branchは1つの目的に限定する
- PRへWHAT、WHY、受入条件、risk、verification、rollbackを書く
- required checkを無効化せず、失敗を再実行だけで隠さない
- `main` へのforce pushと履歴書き換えを禁止する

## 4. 通常CI

[DR-0017](../decisions/0017-minimal-development-ci.md)に基づき、PR、main push、手動実行で必要なsource検証を行う。短命branch pushと週次scheduleは削除し、同じrefの古いrunはcancelする。Vercelへの接続、deploy、Production credentialは使用しない。

### 4.1 Go verification

Go versionはgo.modから取得する。

1. gofmt差分検査
2. `go vet ./cmd/... ./internal/...`
3. `go test -race ./cmd/... ./internal/...`
4. `CGO_ENABLED=0 go build -trimpath -o bin/spotdiggz-api ./cmd/api`
5. govulncheckのsource / binary scan
6. `npm ci --ignore-scripts`、`npm run test:contracts`

Go testにMVPの実HTTP検証とcatalogルールのテストを含むため、同じMVP testの追加実行はしない。対象packageはcmdとinternalに限定し、localに残るdependency directory等を走査しない。catalog schema・事実回帰・鮮度ルールは固定時刻で検証し、本番データの現在時点の再確認期限は通常CIの成否へ混ぜない。

### 4.2 文書・supply chain

- docs-check: 必須文書、Decision Record、内部リンク。
- Gitleaks: Git履歴のsecret scan。
- PR dependency review: moderate以上の脆弱性を含む依存変更でfail。
- third-party ActionsはSHA固定、workflow権限はcontents: read。

### 4.3 通常CIから削除した処理

- Playwright browserのinstall、desktop/mobile E2E、report upload。
- Docker image build、smoke、Trivy image scan、SBOM、image archive upload。
- Trivy filesystem scan（Go脆弱性とsecretは上記toolで検証する）。
- 毎週のscheduleと`make verify-catalog`。
- branch pushとPRの二重実行。

Playwright、Dockerfile、catalogcheck等のsource・commandは維持する。通常CI成功は、browser、container、現在の本番catalog、外部環境の検証成功を意味しない。

## 5. Local commands

```bash
make test
make vet
make verify-catalog
make verify-mvp
make build
npm ci
npm run test:contracts
npx playwright install chromium
npm run test:e2e
docker build --tag spotdiggz-api:local .
```

`make build` は `CGO_ENABLED=0` を設定する。local環境にGo、Node.js、Playwright browser、container runtimeがない場合、未実行checkを完了扱いにしない。

## 6. Release verification and artifacts

release前にownerが同じcommitで次を実行し、結果をrelease記録へ残す。

1. `make verify-catalog`で本番catalogの実時間鮮度を確認する。失敗した施設は公式情報を再調査する。
2. UI変更時とrelease前に`npm run test:e2e`を実行する。固定fixtureを使い、Production credentialを渡さない。
3. OCIを使うreleaseではimageをbuildし、Trivy等でimage脆弱性とDocker設定を検査する。
4. container smokeは、まずダミーcatalog・developmentの認証bypassでlocal確認する。Production相当の認証・Neon接続は承認済みPreviewで確認する。Production credentialなしでProduction起動の成功を要求しない。
5. deploy先の直前成果物とrollback方法を確認する。CIはDocker archiveやSBOMを生成・保存しない。

containerの具体的な確認は[MVP Runbook](mvp-runbook.md)、Vercelの操作は[Vercel・Neon手順](vercel-neon-deployment.md)を参照する。

[過去の確認] Vercel CLI deploy、Container build、alias、Neon migration、Production smokeは2026-07-20に記録された。今回のCI整理では稼働環境の状態を再検証しておらず、Vercel等の接続が解除済みかどうかは未確認である。

- Status: Incomplete
- Missing evidence: 現在の外部環境との接続状態、image署名・provenance、digest昇格運用、CI artifactを使わないrollback演習。
- Required decision: ownerが次回release前にdeploy先と直前成果物を確認し、手動検証・復旧結果を記録する。

## 7. Configuration and secret gate

Non-secret:

- `PORT`
- `FACILITY_CATALOG_PATH`
- `CORRECTION_STORE_PATH`
- `APP_ENV`
- `APP_VERSION`
- `APP_BASE_URL`
- `GITHUB_CLIENT_ID`
- `GITHUB_OWNER`
- `DISCORD_PUBLIC_KEY`
- `DISCORD_APPLICATION_ID`
- `DISCORD_GUILD_ID`
- `DISCORD_OWNER_USER_ID`
- `CHAT_DEFAULT_ORIGIN_LATITUDE`
- `CHAT_DEFAULT_ORIGIN_LONGITUDE`
- `CHAT_DEFAULT_PURPOSE`
- `CHAT_DEFAULT_MOOD`
- `CHAT_DEFAULT_LEVEL`
- `CHAT_DEFAULT_AVAILABLE_MINUTES`
- `CHAT_DEFAULT_TRANSPORT`

Secret:

- `DATABASE_URL`
- `GOOGLE_MAPS_API_KEY`
- `AUTH_SECRET`
- `GITHUB_CLIENT_SECRET`
- `SLACK_BOT_TOKEN`
- `SLACK_SIGNING_SECRET`
- `SLACK_TEAM_ID`
- `SLACK_OWNER_USER_ID`

ProductionはGitHub owner認証が必須であり、`APP_BASE_URL`、`AUTH_SECRET`、`GITHUB_CLIENT_ID`、`GITHUB_CLIENT_SECRET`の欠落時は起動しない。GitHub OAuth Appのcallbackは`APP_BASE_URL + /auth/github/callback`に完全一致させる。`DEV_AUTH_BYPASS`をProductionへ設定しない。Slackを有効にする場合は4つの`SLACK_*`、`DATABASE_URL`、地点解決用`GOOGLE_MAPS_API_KEY`を設定する。Discordを有効にする場合は対象platformの全設定と公開代表起点を設定する。部分設定は起動失敗とする。

production keyはserver-side secret storeから注入し、Routes API / Geocoding APIと送信元を制限する。fork PR、E2E、image buildへproduction keyを渡さない。Googleを使わないreleaseではkeyを設定せず、straight-lineを明示的な既定modeとする。

## 8. Data and compatibility

facility catalogはimage内のread-only JSON snapshotである。catalog変更はcodeと同じCIを通し、未来時刻、未検証status、日英欠落、不正休場形式を拒否する。`make verify-catalog` は全公開施設について、実行時点から168時間後もdynamic 30日・stable 180日の両方が鮮度内であることを要求する。期限内でない施設IDと区分を出力して非0で終了する。

通常CIの週次checkは行わない。ownerはrelease前と定期保守でこのcommandを実行し、失敗した場合は再調査する。各施設の公式 `sourceUrl` で営業、料金、休場、予約、設備、主要ルールを確認し、確認済み属性だけの検証時刻と将来の `one_time` 休場を更新する。更新後は `make verify-catalog` と全CIを通してdata-only releaseを行う。`testdata/` やE2E用の固定fixtureでproduction checkを代替しない。

Productionのcorrection storeはNeon/PostgreSQL、local/CI fallbackは32 MiB上限のJSON Lines fileである。file fallbackではapplication起動時に書込・sync可否を確認し、同じbinaryのread-only `correctioncheck`で破損行を本文非表示のまま診断できる。rollback時は同じNeon接続設定または同じfile volumeを使い、reportをcopy、truncate、schema downgradeしない。保存形式を破壊的に変える場合はmigration、backup、forward-fixを別途設計する。

## 9. Post-deploy smoke

[MVP Runbook](mvp-runbook.md) に従い、次を確認する。

- `/healthz`: process liveness
- `/readyz`: fresh施設が1件以上。empty / all-staleは503
- facility API: 5府県、日英、鮮度field
- recommendation API: 最大3件、origin非再掲、provider種別
- location search: Google有効時200、無効時503
- metrics: catalog、HTTP、recommendation、event
- correction store: Neon接続とretention worker errorなし。file fallbackではvolume writable
- GitHub owner authentication: owner login成功、非owner拒否、sign-out後のAPI 401
- Slack `/spotdiggz`: owner commandの受付と遅延response、非owner拒否。過去message history権限は要求しない
- Discord `/spotdiggz`: PING、owner commandのdeferred response更新、非owner拒否。Gatewayまたはmessage history権限は要求しない

production write-path smokeは承認済みpreviewで行う。実利用者の訂正dataをtestに使わない。

## 10. Rollback

Rollback条件:

- health非200
- readiness 503かつfresh catalogへ即時修正できない
- recommendation 5xxまたはlatencyの継続悪化
- correction store初期化・書込・purge failure
- secret、location、contactのlog混入
- catalog内容またはtranslationの重大誤り

Application rollback:

1. 新releaseへのtrafficを停止する。
2. 直前image digestを同じNeon接続設定で起動する。file fallbackでは同じpersistent volumeを使う。
3. health、readiness、recommendation、metricsを再確認する。
4. rollback marker、原因、data状態を記録する。

Google-only rollbackは `GOOGLE_MAPS_API_KEY` を削除して再起動する。推薦はstraight-lineへ縮退し、location searchは503になる。詳細は [MVP Runbook](mvp-runbook.md) を正とする。

## 11. Remaining release decisions

- image署名、provenance
- Vercel Production/Previewのenvironment分離とPreview DB access policy
- Neonのbackup、retention、接続pool設定
- secret injectionとkey rotation
- private metrics scrape、dashboard、alert
- deploy方式、observation window、traffic rollback command
- production URLでのGoogle実通信とquota / billing alert

これらは資格情報とplatform権限が必要であり、local CI実装とは別のrelease gateとして扱う。
