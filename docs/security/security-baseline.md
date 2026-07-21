# Security and Privacy Baseline

- Status: Active MVP baseline
- Date: 2026-07-21
- Scope: Web UI、HTTP API、verified facility catalog、検索位置、訂正報告、手動キュレーションmedia、公式SNS外部リンク、optional Google連携、CI/CD
- Related: [Product Baseline](../product_baseline.md)
- Related: [ADR-0010](../adr/0010-google-maps-provider-and-fallback.md)

## 1. MVP security posture

- MVPにはaccount、login、管理画面を設けない。公開APIであることを前提に、入力検証、request size上限、process内rate limit、network境界を組み合わせる。
- 施設選定は検証済みcatalogと決定論的ruleだけで行い、AI providerへdataを送信しない。
- 正確な検索位置はapplicationで保存・access log出力・response再掲をしない。Google連携を有効にした場合の外部送信は別のprivacy boundaryとして明示する。
- 訂正報告は目的をcatalog品質改善に限定し、任意contactは明示同意時だけ受理し、90日後の削除期限を持たせる。
- secret、個人情報、正確な位置、request bodyをsource、image、log、metrics、traceへ含めない。
- 動画はcatalogで手動確認したYouTube動画を施設ごとに最大1件に限る。SNSは公式性を確認したInstagram・Xプロフィールへの外部リンクだけとし、投稿・ハッシュタグ・フィードは取得・表示しない。
- Google/Apple Maps、SKEPA等の第三者サイトやSNSから、画像・動画・投稿をスクレイピング、保存、再配信しない。YouTube iframeは利用者の明示操作後だけ読み込み、自動再生しない。

## 2. Data classification and retention

| Class | Data | Purpose | Storage / retention |
| --- | --- | --- | --- |
| Public | 施設名、住所、公開座標、料金、ルール、source URL | catalog表示と推薦 | Git管理。sourceと検証時刻を追跡 |
| Public | 手動キュレーション済みYouTube動画ID・外部URL・確認metadata、公式SNSプロフィールURL | 任意の施設補助情報と外部導線 | Git管理。施設ごと動画は最大1件、SNSはplatformごと最大1件。動画・SNSは推薦根拠に使わない |
| Internal | 推薦rule、設定名、集計metrics、release metadata | application運用 | 認可された開発・運用主体だけが参照 |
| Sensitive location | 推薦起点の緯度経度、地点検索文字列 | 移動時間計算、geocoding | applicationでは非保存。request処理後に破棄 |
| Personal | 訂正details、evidence URL、任意contact email、consent | catalog訂正の確認 | Neon/PostgreSQL in production、correction file in local/CI。receiptから90日後に削除 |
| Secret | `DATABASE_URL`、`GOOGLE_MAPS_API_KEY`、platform credential | correction store、server-side Google API、deploy | secret storeのみ。Git、image、browser、logは禁止 |

訂正reportには `receivedAt` と `deleteAfter=receivedAt+90日` を保存する。storeは起動時と1時間ごとに `deleteAfter` を過ぎたreportをpurgeするため、正常時も期限到達から削除まで最大1時間の差がある。purge失敗時は `correction_retention_purge_failed` を監視し、retention incidentとして扱う。backupや複製も元reportの `deleteAfter` を超えて保持しない。

## 3. Trust boundaries and data flows

```text
Browser
  -> Go HTTP application
       -> read-only facility catalog JSON
       -> correction store (Neon/PostgreSQL in production, JSON Lines file locally)
       -> Prometheus metrics endpoint
       -> Google Routes API       (key設定時のみ)
       -> Google Geocoding API    (key設定時のみ)
  -- 利用者の明示操作後 --> allowlistしたYouTube iframe
  -- 利用者の明示操作後 --> 公式確認済みInstagram / Xプロフィール
CI/CD
  -> source / binary / image verification
  -> Vercel Container / Neon    (Production configured)
```

### Browser to application

- TLS terminationはproduction ingressで行う。production URLとcertificateはdeploy時に確認する。
- JSON write endpointは `Content-Type: application/json`、strict JSON、型・長さ・enum・座標範囲を検証する。
- request body上限はrecommendation 16 KiB、location search 1 KiB、correction 8 KiB、event 1 KiBとする。
- process内token bucketはrecommendation 60/min burst 10、location search 30/min burst 5、correction 30/min burst 10、event 300/min burst 60とする。
- `X-Content-Type-Options` 等のsecurity headerをapplicationで設定し、HTMLは埋め込み静的assetだけを配信する。
- client提供のrequest ID、IP、自由入力をmetrics labelに使わない。
- 現在地取得、外部ナビ、公式情報、訂正報告を含む主要操作は、アイコンだけで意味や状態を伝えない。表示名、アクセス可能な名前、keyboard操作、44 CSS px以上の操作領域を受入条件とする。

process内rate limitはinstance間で共有されず、source単位でもない。production公開時はingressのrequest/body制限、DDoS対策、必要なIP policyを追加し、application limiterだけを濫用対策の根拠にしない。

### Browser to third-party media and social services

- YouTube iframeは、catalogで許可した動画IDから固定の埋込URLを構成する。利用者が明示操作するまでiframe、providerサムネイル、player scriptを読み込まない。任意URLをiframeの`src`へ渡さない。
- embedは自動再生を許可しない。技術的な埋込失敗、動画削除、埋込禁止時は、推薦結果と施設情報を継続表示し、通常のYouTube外部リンクだけを提示する。
- SNSは公式性を確認済みのInstagramまたはXプロフィールURLだけを許可する。外部リンクはallowlistしたHTTPS host、`noopener`、`noreferrer`を使い、投稿、ハッシュタグ、フィードをapplication内へ埋め込まない。
- BrowserがYouTube iframeまたはSNS外部リンクを開くと、利用者のnetwork metadata等が当該providerへ送信され得る。applicationは閲覧履歴、動画再生状態、SNS投稿を受信・保存しない。
- ownerは公開前と定期見直し時に、provider規約、埋込可否、著作権・肖像権上の利用可否、privacy表示を確認し、確認日と判断根拠をcatalog review記録へ残す。規約・権利・安全性に懸念が生じたrecordは、iframeと外部リンクの両方を無効化する。

### Application to Google

`GOOGLE_MAPS_API_KEY` が未設定ならGoogleへrequestしない。設定時はserverからHTTPSで次を送信する。

| Provider | Sent data | Not sent by design |
| --- | --- | --- |
| Routes Compute Route Matrix | 正確な起点座標、公開施設座標、交通手段 | 訂正report、contact、任意event |
| Geocoding | 利用者の地点検索文字列、country/language/region制約 | 訂正report、推薦条件一式 |

applicationで非保存であっても、Googleがrequestを受信する。利用者向けprivacy表示では「保存しない」と「外部送信しない」を区別する。provider側の利用規約、retention、telemetryはproduction有効化前にownerが確認する。

即時検索ではGoogle Routesの `departureTime` を省略し、providerのrequest時刻を使う。Google requestは4秒timeout、hostごとの同時HTTP connectionは4本とし、Routes失敗時はrequest全体をstraight-line計算へ縮退する。GeocodingはJSON bodyのPOSTで受け、失敗またはkey未設定時に `503 location_search_unavailable` を返し、代表地点またはbrowser Geolocationへ戻れるようにする。

### Application to correction store

- Productionの`DATABASE_URL`はVercel/Neonのsecret storeからruntime注入し、Neon側の暗号化、権限、backup、90日retention設定を運用で確認する。`CORRECTION_STORE_PATH`はlocal/CIのfile fallback用である。
- file fallbackを使う場合、既定directoryはUID `65532` が書き込める。新規directoryは `0700`、fileは `0600` で作る。既存volumeのownerとmodeはdeploy時に検査する。
- file fallbackは起動時にstore fileを作成またはopenして書込とsyncを確認する。store総量は32 MiBを上限とし、超過時はappendせず `503 correction_store_unavailable` を返す。Neonでは同じ期限条件をSQL deleteで適用する。
- `details` は10〜1000文字、`evidenceUrl` はoptional HTTPS URL、contact emailはoptionalで `contactConsent=true` の場合だけ受理する。
- logにはreport ID、facility ID、category等の最小metadataだけを残し、details、evidence URL、contactを含めない。
- fileの手編集、別の一時fileへの迂回保存、retentionを超えるbackupを禁止する。

### Metrics and logs

- `/metrics` はapplication認証を持たない。productionではprivate network、service-to-service policy、またはingress allowlistで制限する。
- access logはpath template、status、duration、request ID等を記録し、query string、request body、正確な位置、contact、API keyを記録しない。
- traceを追加する場合もrequest/response bodyをattributeへ保存しない。
- `APP_VERSION` とdeploy時刻をrelease記録に残し、incidentとartifactを関連付ける。

## 4. Secrets and Google credential

- `GOOGLE_MAPS_API_KEY` はserver-side secret storeからruntime注入する。`.env`、command履歴、CI artifact、test fixture、container layerへ保存しない。
- production keyはRoutes APIとGeocoding APIだけに制限し、利用可能なら固定egress IP等のapplication restrictionを設定する。
- quotaとbilling alert、利用量監視、owner、rotation、失効手順を有効化前に定義する。
- compromiseの疑いがある場合はkeyをprovider側で失効し、environmentから削除してapplicationを再起動する。推薦はstraight-lineへ縮退し、地点検索は停止する。
- scratch imageにはGoogle HTTPSのcertificate検証に必要なCA bundleだけを追加し、shellやpackage managerは含めない。

## 5. Catalog integrity and application boundary

- 公開catalogは `status=verified`、source、日英必須属性、検証時刻を持つrecordだけに限定する。
- applicationは起動時にschema、重複ID、座標、URL、未来時刻、translation、休場形式を検証し、構造的に不正なcatalogでは起動しない。
- media recordは、YouTube動画が施設ごとに0または1件であること、動画ID・通常URL・選定日・確認日を持つこと、SNS URLが許可platformのHTTPSプロフィールでplatformごとに最大1件であることを検証する。任意iframe URL、投稿URL、ハッシュタグURLは拒否する。
- dynamic 30日 / stable 180日の期限超過recordはload可能だが推薦しない。fresh recordが0件なら `/readyz` は503を返す。
- correctionをcatalogへ自動反映しない。運用者が公式sourceを確認し、data変更を通常のreviewとCIへ通す。
- `one_time` / `annual` 休場、通常営業時間、provider結果は当日の公式情報を保証しない。UIはsourceと検証時刻を提示する。

## 6. Build and supply chain

- `CGO_ENABLED=0` で静的な単一binaryをbuildし、digest固定したbuilder、scratch / non-root UID `65532` で実行する。
- Go test、race、vet、format、JSON/OpenAPI contract、MVP smoke、Playwright E2E、source/binary vulnerability scan、filesystem/image scan、secret scanをCI gateにする。
- `package-lock.json` とGo module metadataをversion管理し、third-party GitHub Actionsをcommit SHAへ固定する。
- imageはSBOMを生成し、同一digestを環境間で昇格する。Vercel Container build・Production deployは確認済みだが、署名、provenance、digest昇格運用は未設定である。
- image内catalogはread-only、local fallback時のcorrection directoryだけをwrite可能にする。Productionのcorrection reportはNeonへ保存する。
- `.dockerignore` はallowlist方式とし、`.git`、`.env`、`var`、log、test artifactをbuild contextへ送らない。

## 7. Threats and controls

| Threat | Current control | Residual risk / next control |
| --- | --- | --- |
| malformed / oversized input | strict JSON、schema validation、body limit | fuzzとproduction trafficで境界を継続検証 |
| public endpoint abuse | route別token bucket、Google接続数上限、429 | source別ingress / edge limit、Google quota、DDoS対策が未設定 |
| location disclosure | non-persistence、query/body非logging | Google有効時の外部送信とprovider retention |
| correction PII leakage / disk exhaustion | consent、最小log、90日deadline、purge、32 MiB上限 | Neon backup / retention policyとlocal file fallbackのvolume運用が未検証 |
| API key leakage or abuse | server-side injection、secret scan、fallback | production key restriction / rotationは未検証 |
| stale or tampered catalog | source metadata、startup validation、readiness、CI review | official source自体の当日変更は検知できない |
| third-party media tracking / unavailable embed | 明示操作後の遅延load、自動再生禁止、allowlist URL、外部リンクへの縮退 | provider telemetry、動画削除、埋込可否、規約変更をownerが定期確認する必要がある |
| scraped or unlicensed media | 自動収集・保存・再配信の禁止、手動review記録 | curatorの判断誤り、著作権・肖像権・規約の解釈はowner確認が必要 |
| malicious external link / arbitrary iframe | catalog validation、platform/host allowlist、固定iframe URL、`noopener` / `noreferrer` | allowlistとprovider仕様の定期見直しが必要 |
| metrics disclosure | data minimization | production network restrictionは未実装 |
| container compromise | static binary、scratch、non-root、read-only catalog | platform sandbox / filesystem policyは未選定 |

## 8. Incident and rollback

1. accessを制限し、影響するdata class、request、release versionを特定する。
2. key漏えいなら失効、位置/contactのlog混入なら収集停止とaccess制限、correction漏えいならvolume accessを停止する。
3. 直前の検証済みimageへrollbackする。correction volumeをtruncate、上書き、旧imageへcopyしない。
4. Googleだけを無効化する場合はkeyを削除して再起動し、straight-line recommendationとlocation search 503を確認する。
5. retention failureは期限超過reportとbackupを特定してpurgeし、原因と削除結果を記録する。
6. mediaまたはSNS linkに規約、権利、安全性、公式性の問題が見つかった場合は、該当catalog recordからiframeと外部リンクを無効化し、公開状態と判断根拠を記録する。
7. 回帰test、検知rule、Runbook、脅威modelを更新する。

具体的なsmoke、例外判定、rollbackは [MVP Runbook](../operations/mvp-runbook.md) を正とする。

## 9. Release gate

- [ ] catalog validationとfreshness readinessがPASSした
- [ ] request validation、body limit、rate limitのtestがPASSした
- [ ] log / metricsに位置、query、contact、key、request bodyがないことを確認した
- [ ] ProductionのNeon secret、権限、backup、90日retentionを確認した。file fallbackを使う環境ではvolumeのowner、mode、暗号化、backupも確認した
- [ ] `/metrics` をpublic Internetから遮断した
- [ ] secret、dependency、source/binary、filesystem/image scanがPASSした
- [ ] Google有効時はAPI / application restriction、quota、billing alert、privacy表示を確認した
- [ ] Google無効時または障害時のfallbackを確認した
- [ ] icon controlが表示名、アクセス可能な名前、keyboard操作、44 CSS px以上の操作領域を満たすことをdesktop/mobile E2Eで確認した
- [ ] media schemaでYouTube動画が施設ごとに最大1件、SNSがplatformごとに最大1件であること、任意iframe URL・投稿URL・ハッシュタグURLが拒否されることを確認した
- [ ] YouTube iframeが明示操作後だけ読み込まれ、自動再生せず、埋込失敗時に通常リンクへ縮退することを確認した
- [ ] CSP等でiframeと外部リンクのoriginをallowlistし、provider規約、埋込可否、著作権・肖像権、privacy表示をownerが確認・記録した
- [x] production HTTPS、post-deploy smokeを確認した。[ ] rollback exerciseは未実施

[事実] ProductionのVercel secret injection、Neon migration、訂正APIのwrite path、health/readiness、UI/API smokeを確認した。[未検証] 実Google credential、provider側retention、metrics network policy、alert、custom domain、incident exerciseである。
