# spot-diggz MVP Release and Operations Runbook

- Status: Operational baseline; Vercel/Neon production smoke verified 2026-07-20
- Date: 2026-07-20
- Scope: single Go application, facility catalog, Neon correction store in production, local correction file fallback, optional Google providers

このRunbookは、release ownerが同一imageを起動し、smoke、監視、縮退、rollbackまで判断するための手順である。Vercel/Neonの初回構築とProduction smokeは完了しており、metrics制限、Google quota、カスタムドメインなど未設定の項目は本文で明示する。

## 1. 読者・前提

- 想定読者: repositoryと実行環境を変更できるrelease owner / on-call operator
- local前提: Go 1.25.12、Make、curl。またはOCI imageをbuild・runできるcontainer runtime
- production前提: immutable image、HTTPS ingress、既存Neon OrganizationのProject、secret injection、private metrics scrape、直前imageの参照
- optional権限: Google Cloud project、billing、Routes API / Geocoding API、制限済みserver-side credential
- 禁止: key、訂正報告本文、連絡先、正確な検索位置、raw access logをcommand history、ticket、Gitへ貼らない

## 2. 目的・成功条件

目的は、検証済みcatalogを使う主要flowを公開し、外部provider障害時もstraight-line推薦へ縮退し、問題時にdataを失わず直前imageへ戻せる状態を作ることである。

成功条件:

- `make test`、`make vet`、`make verify-catalog`、`make verify-mvp`、`make build` が成功する
- imageが起動し、`/healthz` が200、fresh施設が1件以上あるcatalogで `/readyz` が200を返す
- catalog件数が想定値で、5府県と鮮度fieldを含む
- recommendationが `google_routes` または `straight_line` を明示して返る
- productionではNeonのcorrection storeへ書け、期限切れreport purgeが失敗していない。local/CIではfile storeを使う
- `/metrics` が収集境界だけから取得できる
- rollback対象image、起動設定、volumeをrelease前に記録している

## 3. 手順

### 3.1 Pre-release verification

1. `git status --short --branch` でrelease対象branchと意図しないdirty fileを確認する。
2. `make test` でdomain、HTTP、provider、retentionのtestを実行する。
3. `make vet` でGo静的検査を実行する。
4. `make verify-catalog` でproduction catalogが168時間後も鮮度内であることを確認する。
5. `make verify-mvp` でembedded UIと推薦flowを実HTTP検証する。
6. `make build` で `CGO_ENABLED=0` の静的な単一application binaryをbuildする。
7. `docker build --tag spotdiggz-api:<release-sha> .` でproduction相当imageをbuildする。
8. `git diff --check` で差分の空白errorを確認する。

確認点: 失敗したcheckを再実行だけで隠さない。使用したcommit SHAとimage digestをrelease記録へ残す。

#### Catalog freshness revalidation

GitHub Actionsは毎週月曜00:17 UTC（09:17 JST）に `make verify-catalog` を実行する。このcheckは `data/facilities.json` の全施設が実行時点から168時間後もdynamic 30日・stable 180日の鮮度内であることを確認し、期限不足の施設IDと区分を表示して失敗する。

失敗時は次の順に対応する。

1. 出力された施設の `sourceUrl` を開き、営業時間、料金、休場、予約、設備、主要ルールを公式情報で再確認する。
2. 変更された事実と次の30日以内の不定期休場をcatalogへ反映する。
3. 実際に確認した属性だけの `dynamicVerifiedAt` または `stableVerifiedAt` を更新する。確認せず時刻だけを延長しない。
4. `make verify-catalog`、`make test`、`make verify-mvp` を実行し、data-only PRをreleaseする。

固定の `testdata/facilities.dev.json` やE2E fixtureは再現可能なtest専用であり、production catalogの再確認完了を示す証拠には使わない。

### 3.2 Configuration

| Variable | Required | Default | Operational rule |
| --- | --- | --- | --- |
| `PORT` | no | `8080` | ingress targetと一致させる |
| `FACILITY_CATALOG_PATH` | no | `data/facilities.json` | imageに含めた検証済みfileを使う |
| `CORRECTION_STORE_PATH` | no | `var/corrections.jsonl` | local/CIのfile fallback用。productionではNeonを使う |
| `DATABASE_URL` | production | unset | Neon/PostgreSQLのcorrection store。設定時はfile storeより優先 |
| `GOOGLE_MAPS_API_KEY` | no | unset | secret storeから注入し、API・送信元を制限する |
| `APP_ENV` | no | `development` | `production` 等の低cardinality値をlogへ付与する |
| `APP_VERSION` | no | `unknown` | release SHAまたはimage versionを付与する |

Googleを有効にしないreleaseでは `GOOGLE_MAPS_API_KEY` を設定しない。この状態はerrorではなく、推薦を `straight_line` で継続する既定の縮退modeである。

Vercel Productionでは`DATABASE_URL`でNeon/PostgreSQLを使い、訂正reportをcontainer filesystemへ保存しない。`DATABASE_URL`未設定のlocal/CIではapplication userが書き込めるfile storeを使う。file storeの既定pathは `/var/lib/spotdiggz/corrections.jsonl`、実行userはUID `65532` である。scratch imageには同UIDが所有する書込directoryとGoogle HTTPS用CA bundleを含む。
file storeを使う場合、applicationは起動時に作成・書込・sync可否を確認し、総量が32 MiBを超えるstoreでは起動しない。稼働中に上限へ達した場合は新規appendを拒否して503を返す。Neon側では`delete_after`を条件に期限切れreportを削除する。

### 3.3 Start

Local without Google:

```bash
APP_ENV=local APP_VERSION=dev make run
```

Local with a credential supplied by an approved secret source:

```bash
APP_ENV=local APP_VERSION=dev GOOGLE_MAPS_API_KEY='<secret>' make run
```

起動logで次を確認する。

- `facility_catalog_loaded`: 想定件数をloadした
- `google_maps_integrations_enabled` または `google_maps_integrations_disabled`: 意図したmodeである
- `http_server_started`: 想定portでlistenした
- `correction_store_initialization_failed` がない

### 3.4 Read-only smoke

`<base-url>` はlocalでは `http://localhost:8080`、productionでは承認済みHTTPS URLへ置き換える。

```bash
curl --fail --silent '<base-url>/healthz'
curl --fail --silent '<base-url>/readyz'
curl --fail --silent '<base-url>/api/facilities?activity=skateboard'
curl --fail --silent '<base-url>/metrics'
```

確認点:

- healthは常にlivenessとして `{"status":"ok"}` を返す
- readyはfresh施設が1件以上なら `{"status":"ready"}`。emptyまたは全件staleなら503と `{"status":"not_ready"}`
- facility responseに `prefecture`、`municipality`、`englishTranslation`、`dynamicVerifiedAt`、`stableVerifiedAt` がある
- metricsに `spot_diggz_catalog_facilities`、fresh/staleの両state、Google providerのsuccess/error seriesがある
- productionの `/metrics` はpublic Internetから到達できない

### 3.5 Recommendation smoke

```bash
curl --fail --silent \
  --header 'Content-Type: application/json' \
  --data '{"purpose":"basics","mood":"focused","level":"beginner","availableMinutes":120,"transport":"public_transit","origin":{"mode":"specified_location","latitude":34.7025,"longitude":135.4960}}' \
  '<base-url>/api/recommendations'
```

確認点:

- responseは最大3件で、起点座標を含まない
- 各候補にsource、鮮度時刻、到着・終了時刻、`travelEstimateKind` がある
- Google有効時の正常系は `google_routes`
- key未設定またはGoogle Routes失敗時は `straight_line` と概算注意文
- 通常営業時間、鮮度、休場情報は当日の公式sourceを置き換えない

### 3.6 Location-search smoke

Google有効時:

```bash
curl --fail --silent \
  --header 'Content-Type: application/json' \
  --data '{"query":"神戸駅"}' \
  '<base-url>/api/locations/search'
```

Google無効時は同じrequestが `503` と `location_search_unavailable` を返せば想定どおりである。検索文字列をapplication logで検索して、raw queryが記録されていないことをpreview環境で確認する。

### 3.7 Write-path smoke

訂正とeventはwriteである。productionではtest reportを作らず、承認済みpreview catalogの `<facility-id>` だけで行う。

```bash
curl --fail --silent \
  --header 'Content-Type: application/json' \
  --data '{"facilityId":"<facility-id>","category":"hours","details":"検証用の訂正報告です。実データへ反映しないでください。"}' \
  '<base-url>/api/corrections'

curl --fail --silent \
  --header 'Content-Type: application/json' \
  --data '{"event":"result_displayed"}' \
  '<base-url>/api/events'
```

確認点:

- correctionは `202`、`COR-` で始まるreport ID、UTCの `receivedAt` を返す
- storeは新規directoryを `0700`、fileを `0600` で作成し、reportに90日後の `deleteAfter` がある。総量は32 MiB以内で、既存mountはdeploy時にownerとmodeを別途確認する
- `correction_received` logはreport ID、facility ID、categoryだけを持ち、details、URL、contactを含まない
- eventは `202` を返し、`spot_diggz_product_events_total` が増える

### 3.8 Retention operation

file storeは起動時と1時間ごとに、`deleteAfter` を過ぎたreportを排他制御下で削除する。正常時でも期限到達から次のhourly purgeまで最大1時間の検知差がある。`correction_retention_purge_failed` を検知した場合、retentionを保証できないためrelease ownerへ通知する。

訂正storeを手作業で編集しない。稼働中のread-only診断は次を使う。

```bash
docker exec '<container-name>' \
  /spotdiggz-api correctioncheck \
  -path /var/lib/spotdiggz/corrections.jsonl
```

commandはstoreを変更せず、report総数、期限切れ件数、破損行番号だけを出力する。report ID、details、evidence URL、contact、破損行本文は出力しない。

破損時は新規writeとtraffic切替を停止し、元volumeの暗号化snapshotを保持したままread-only mountしたcopyを検査する。検証済みbackupを同じretention条件で復元するか、security owner承認済みのone-time migrationをcopyへ適用し、`correctioncheck` 成功後にだけ切り替える。元fileを直接truncate・上書きしない。backupも元reportの `deleteAfter` より長く保持しない。

### 3.9 Deploy and post-deploy gate

1. 既存Neon OrganizationのProjectから取得した`DATABASE_URL`をVercel Productionへsecretとして注入する。
2. `npx vercel --prod`で`Dockerfile.vercel`を使うContainerをdeployする。
3. `/healthz`をliveness、`/readyz`をreadinessとして確認する。
4. 3.4から3.6のread-only smokeを実行し、訂正APIは承認済みのwrite-path smoke時だけ実行する。
5. `spot_diggz_http_requests_total`の5xx、recommendation result、catalog staleを観察する。
6. 問題時は`npx vercel rollback`で直前Productionへ戻し、Neon schemaを巻き戻さない。

Vercelでの初回構築、Neon Project、migration、Production smokeの詳細は [Vercel・Neonデプロイ手順](vercel-neon-deployment.md) を正とする。`/metrics`のpublic到達制限、Google quota、custom domainは未設定のため、別途release gateとする。

CI成功runは、Trivy scan後のDocker image archive、image ID、archive SHA-256を30日保持する。registryが未選定でも、このarchiveを検証して `docker load` すれば同じ成果物でpreviewまたはrollback演習を行える。production昇格では最終的にimmutable registry digestを記録する。

## 4. 例外・縮退・rollback

| Symptom | Immediate action | Decision |
| --- | --- | --- |
| 起動前に `facility_catalog_load_failed` | 新imageへtrafficを流さない | catalogを修正するか直前imageへrollback |
| `correction_store_initialization_failed` | 新imageへtrafficを流さない | volume owner、mount、容量を修正 |
| `/healthz` が非200 | traffic切替を停止 | 直前imageへrollback |
| `/readyz` が503、healthは200 | fresh件数と検証時刻を確認 | empty / all-stale catalogを更新するか、fresh catalogを含む直前imageへrollback |
| location searchが503 | 代表地点・現在地へ案内 | recommendationが正常なら限定的縮退を継続 |
| Google有効時に `straight_line` が継続 | Google usage、quota、credentialを確認 | keyを削除して明示的縮退、またはproviderを復旧 |
| recommendationの5xx増加 | 5xx routeとrelease markerを確認 | straight-lineも失敗する場合はrollback |
| correctionが503 | writeを停止しvolumeを保全 | mount、容量、permissionを修正。別の一時fileへ逃がさない |
| `correction_retention_purge_failed` | storeへのwrite影響と期限超過を確認 | privacy incident手順へescalateし、purgeを回復 |
| catalog staleが増加 | 該当施設を推薦対象外として維持 | sourceを再確認しdata-only release |
| 429増加 | client retryと濫用を確認 | rate limitを迂回せず、ingress制限を追加検討 |
| logへ位置・contact・keyが混入 | log accessを制限し収集を停止 | security incidentとして削除・失効・修正 |

Application rollback:

1. 新releaseへのtrafficを停止する。
2. 直前に記録したimage digestを同じNeon接続設定で起動する。local/CIのfile fallbackでは同じvolumeを使う。
3. Neon schemaやcorrection dataをrollbackに合わせて削除、truncate、旧imageへcopyしない。
4. 3.4と3.5を再実行する。
5. rollback時刻、原因、影響、data状態を記録する。

Google-only rollback:

1. `GOOGLE_MAPS_API_KEY` をenvironmentから削除する。
2. applicationを再起動する。
3. recommendationが `straight_line`、location searchが503であることを確認する。
4. key compromiseの疑いがある場合はprovider側で失効する。

最終確認:

- health/readiness、推薦、catalog、metricsを確認した
- Neon correction dataを保持した。file fallbackではcorrection volumeを保持した
- logに禁止情報がない
- external provider modeが利用者表示と一致する
- rollbackまたは縮退の理由をrelease記録へ残した

## 5. 関連参照

- [OpenAPI contract](../api/facility-catalog.openapi.yaml)
- [Observability design](observability.md)
- [Continuous delivery](continuous-delivery.md)
- [Security and privacy baseline](../security/security-baseline.md)
- [ADR-0010 Google provider and fallback](../adr/0010-google-maps-provider-and-fallback.md)
- [ADR-0011 five-prefecture scope](../adr/0011-five-prefecture-mvp-scope.md)
