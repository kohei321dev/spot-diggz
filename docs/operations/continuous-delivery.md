# Continuous Delivery Pipeline Design

- Status: CI and rollback artifact implemented; Vercel/Neon Production smoke verified
- Date: 2026-07-20
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

## 4. Implemented CI

### 4.1 Go verification

Ubuntu 24.04とGo 1.25.12で次を実行する。

1. `gofmt` 差分検査
2. `go vet ./cmd/... ./internal/...`
3. `go test -race ./...`
4. `make verify-catalog`
5. `make verify-mvp`
6. `CGO_ENABLED=0 GOOS=linux go build -trimpath`
7. `file` によるstatically linked確認
8. `govulncheck` のsource scan
9. `govulncheck -mode=binary` のbinary scan

Go testはcatalog schema、production catalogの事実回帰、freshness horizon command、休場、provider fallback、geocoding、correction retention、rate limit、metrics、HTTP contractを含む。workflowは毎週月曜00:17 UTC（09:17 JST）にも起動し、production catalogが168時間後も鮮度内かを検査する。

### 4.2 Browser E2E

Go verification後、Node.js 24とPlaywright Chromiumでdesktop / mobileの主要flowを実行する。

- `npm ci`
- `npm run test:contracts`
- `npx playwright install --with-deps chromium`
- `npm run test:e2e`

failure時のscreenshot、trace、videoとHTML reportは7日保持のCI artifactにする。E2Eは固定test catalogとlocal serverを使い、実Google APIへ接続しない。固定fixtureの時刻調整はE2E再現性のためであり、production catalogの鮮度検査とは分離する。

### 4.3 Container security and smoke

Go verification後に同じcommitからimageをbuildする。

1. scratch imageをbuildする
2. containerをlocalhostだけへpublishする
3. `/healthz` が `{"status":"ok"}` になるまで待つ
4. `/readyz` が `{"status":"ready"}` になるまで待つ
5. 訂正APIへCI専用reportをPOSTし、同じbinaryの `correctioncheck` でstoreを検証する
6. CycloneDX SBOMを生成し、30日保持のartifactにする
7. imageのHIGH / CRITICAL vulnerabilityをTrivyで検査する
8. scan済みimageをDocker archive、image ID、archive SHA-256として30日保持する

readinessはcatalogにdynamic 30日・stable 180日の両方が鮮度内の施設が1件以上ある場合だけ成功する。emptyまたは全件staleのimageはsmokeを通過しない。

### 4.4 Repository and supply-chain checks

- Trivy filesystem scan: vulnerability、secret、misconfiguration
- Gitleaks full-history scan
- pull request dependency review: moderate以上でfail
- third-party GitHub Actionsはcommit SHAへ固定
- workflow tokenは `contents: read` を既定にする

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

## 6. Artifact flow

Target flow:

```text
commit
  -> Go / race / smoke / vuln checks
  -> Playwright desktop + mobile
  -> static binary + scratch image
  -> image smoke + SBOM + image scan
  -> immutable registry artifact
  -> approved environment configuration
  -> deployment
  -> post-deploy smoke + observation window
  -> promote or rollback
```

[事実] CIはbinary、image、SBOMを検証し、scan済みDocker image archive、image ID、archive checksumをcommit SHA単位のartifactとして30日保持する。このarchiveは `sha256sum -c` と `docker load` で同一成果物をpreviewまたはrollback演習へ渡せる。

[事実] Vercel Project `spotdiggz`へのCLI deploy、Vercel Container build、Vercel alias、既存Neon OrganizationのProject、migration、Production smokeは2026-07-20に確認した。GitHub `main` pushからの自動Production deployは、変更をmainへmergeした後に別途確認する。

[未検証] image署名・provenance、private metrics scrape、Google API quota監視、custom domain/DNS、GitHub main pushからの自動deployは未設定である。CI artifactの保持期限を越える長期rollbackにはVercelまたは別registryのimmutable artifact運用が必要になる。

## 7. Configuration and secret gate

Non-secret:

- `PORT`
- `FACILITY_CATALOG_PATH`
- `CORRECTION_STORE_PATH`
- `APP_ENV`
- `APP_VERSION`

Secret:

- `DATABASE_URL`
- `GOOGLE_MAPS_API_KEY`

production keyはserver-side secret storeから注入し、Routes API / Geocoding APIと送信元を制限する。fork PR、E2E、image buildへproduction keyを渡さない。Googleを使わないreleaseではkeyを設定せず、straight-lineを明示的な既定modeとする。

## 8. Data and compatibility

facility catalogはimage内のread-only JSON snapshotである。catalog変更はcodeと同じCIを通し、未来時刻、未検証status、日英欠落、不正休場形式を拒否する。`make verify-catalog` は全公開施設について、実行時点から168時間後もdynamic 30日・stable 180日の両方が鮮度内であることを要求する。期限内でない施設IDと区分を出力して非0で終了する。

週次checkの失敗は、データを自動延命せず再調査を開始するsignalである。各施設の公式 `sourceUrl` で営業、料金、休場、予約、設備、主要ルールを確認し、確認済み属性だけの検証時刻と将来の `one_time` 休場を更新する。更新後は `make verify-catalog` と全CIを通してdata-only releaseを行う。`testdata/` やE2E用の固定fixtureでproduction checkを代替しない。

correction storeは32 MiB上限のJSON Lines persistent fileである。applicationは起動時にfileの書込・sync可否を確認し、同じbinaryのread-only `correctioncheck` で破損行を本文非表示のまま診断できる。rollback時は同じvolumeをmountし、fileを旧imageへcopy、truncate、schema downgradeしない。保存形式を破壊的に変える場合はmigration、backup、forward-fixを別途設計する。

## 9. Post-deploy smoke

[MVP Runbook](mvp-runbook.md) に従い、次を確認する。

- `/healthz`: process liveness
- `/readyz`: fresh施設が1件以上。empty / all-staleは503
- facility API: 5府県、日英、鮮度field
- recommendation API: 最大3件、origin非再掲、provider種別
- location search: Google有効時200、無効時503
- metrics: catalog、HTTP、recommendation、event
- correction store: Neon接続とretention worker errorなし。file fallbackではvolume writable

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
