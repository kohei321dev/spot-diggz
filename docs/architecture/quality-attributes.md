# Quality Attributes and Architecture Guardrails

- Status: Active MVP architecture
- Date: 2026-07-20
- Decision authority: [Product Baseline](../product_baseline.md) and Accepted ADRs

## 1. Architecture summary

spot-diggzは、Web UI、HTTP API、facility catalog、決定論的推薦、外部provider adapter、訂正store、observabilityを1つのGo applicationとしてdeployするモジュラーモノリスである。facility catalog用のdatabase、queue、cacheは分離しないが、Productionの訂正reportだけは永続性のためmanaged Neonへ保存する。

```text
Browser
  -> embedded Web UI / HTTP API
       -> session input validation
       -> facility catalog (read-only JSON snapshot)
       -> deterministic recommendation
            -> Google Routes adapter (optional)
            -> straight-line adapter (fallback)
       -> Google Geocoding adapter (optional)
       -> correction store (Neon/PostgreSQL in production, file fallback locally)
       -> rate limit / observability
```

production成果物は `CGO_ENABLED=0` の静的な単一binaryを含むscratch OCI imageである。imageはGoogle HTTPS用CA bundleとnon-root UID `65532` が書き込めるlocal fallback directoryを持つ。facility catalogはimage内のread-only snapshot、Productionのcorrection reportはNeon/PostgreSQLへ置く。

## 2. Module boundaries

| Module / package | Responsibility | Must not own |
| --- | --- | --- |
| `internal/webui` | 埋め込み静的assetとbrowser entry point | 推薦rule、catalog更新 |
| `internal/httpapi` | route、HTTP validation、response/error contract、security header | provider固有request、永続化format |
| `internal/session` | purpose、mood、level、time、transport、originのdomain validation | HTTPやGoogle型 |
| `internal/facility` | catalog model/load/validation、freshness、営業時間・休場判定 | user request、外部API credential |
| `internal/recommendation` | hard condition、score、stable ordering、reason生成 | network、AI、catalog mutation |
| `internal/travel` | travel provider interface、Google Routes、straight-line fallback | recommendation ranking |
| `internal/geocoding` | Google Geocoding境界とlocation result | catalog scope拡張、query保存 |
| `internal/correction` | report validation、file/PostgreSQL保存、90日retention purge、read-only file store診断 | catalog自動更新 |
| `internal/ratelimit` | process内token bucket | user identity、distributed quota |
| `internal/observability` | Prometheus metricsとstable label | request body、正確な位置、contact |
| `cmd/api` | dependency composition、configuration、lifecycle | domain rule |

依存方向はHTTP / runtime adapterからdomainへ向ける。Recommendationはtravel provider interfaceに依存し、Google HTTP形式へ直接依存しない。Geocoding結果は起点候補であり、5府県catalogをruntimeで増やさない。Correctionは運用者review前にcatalogを書き換えない。

## 3. Quality attributes

| Priority | Attribute | MVP contract | Verification |
| --- | --- | --- | --- |
| P0 | Catalog trust | verified record、公式source、検証時刻、日英必須属性だけを公開 | startup validation、catalog tests、source review |
| P0 | Freshness safety | dynamic 30日・stable 180日の両方がfreshな施設だけを推薦 | freshness unit tests、`/readyz`、fresh/stale gauge |
| P0 | Privacy | origin/queryを非保存・非logging。Google有効時の送信を明示 | log tests、store inspection、privacy review |
| P0 | Determinism | 同じcatalog、input、時刻、provider結果から同じ順位・理由を返す | clock/provider injection、stable-order tests |
| P0 | Recoverability | Google障害時もstraight-line推薦を継続し、artifact rollbackでreportを失わない | provider failure tests、Runbook smoke、volume-preserving rollback |
| P1 | Modifiability | UI、HTTP、domain、provider、storeをmodule境界で変更できる | package review、focused tests、ADR review |
| P1 | Observability | HTTP、推薦、product event、catalog freshnessを低cardinalityで測る | metrics tests、dashboard/alert review |
| P1 | Security | input/size/rate/secret/container controlを多層化する | CI scans、HTTP tests、threat review |
| P1 | Performance | 小規模catalogの主要flowを同期処理で完了する | HTTP duration histogram、p95 observation |
| P1 | Accessibility / i18n | mobileで主要flowを日英利用できる | Playwright desktop/mobile、manual keyboard/screen-reader review |
| P1 | Operability | static binary、scratch image、health/readiness、smoke/rollbackを揃える | CI image smoke、Runbook exercise |

## 4. Runtime behavior and failure isolation

### Startup

1. facility JSONを読み、schema、source、座標、translation、時刻、休場形式を検証する。
2. correction storeを初期化し、`deleteAfter` 超過reportをpurgeする。file fallback時はfileの作成・書込・sync可否も確認する。
3. keyがあればGoogle adapter、なければstraight-line recommendationとdisabled geocodingを構成する。
4. catalogのfresh件数を評価し、HTTP serverと1時間ごとのretention workerを開始する。

構造的に不正なcatalogまたはcorrection store初期化失敗はstartup failureとする。期限超過catalogはloadできるが、fresh施設が0件なら `/healthz` は200、`/readyz` は503である。

### Request path

- Recommendationは候補のfreshness、休場、営業時間、時間、levelを除外し、provider結果を使って最大3件をstable orderで返す。
- Google Routesは4秒timeoutとし、HTTP / decode / element failure時はrequest全体をstraight-lineへfallbackする。
- GeocodingはGoogle専用で、JSON bodyのPOSTと専用rate limitを通し、key未設定またはprovider failure時に503を返す。代表地点・browser geolocationはUI側の代替経路である。
- Correctionは32 MiBのstore上限内でappendとsyncに成功した場合だけ202を返す。reportには90日後の `deleteAfter` を付け、起動時と1時間ごとにpurgeする。
- `/metrics` はapplication内で認証せず、deployment networkで制限する。

## 5. Deployment boundaries

```text
immutable image
  - static Go binary
  - embedded UI
  - verified catalog snapshot
  - CA bundle
runtime configuration
  - PORT / catalog path / correction path / environment / version
  - GOOGLE_MAPS_API_KEY (optional secret)
persistent state
  - correction JSON Lines volume
external state
  - Google Routes / Geocoding (optional)
  - Prometheus-compatible collector
```

- 同一image digestを環境間で昇格し、環境ごとにrebuildしない。
- root filesystemとcatalogをread-onlyにし、correction directoryだけを書込可能にする。
- livenessは `/healthz`、readinessはfresh施設が1件以上であることを確認する `/readyz` を使う。
- Google-only rollbackはkey削除と再起動、application rollbackは直前imageと同じcorrection volumeを使う。
- scratch imageにはshellがないため、container内部での手作業を運用手段にせず、endpoint、log、metrics、volume inspectionで診断する。
- 同じapplication binaryの `correctioncheck` subcommandはstoreを変更せず、本文を出力しない診断境界として利用できる。

## 6. Fitness functions and release checks

Current automated checks:

- `gofmt`、`go vet`、`go test -race ./...`
- catalog schema/freshness/closure/translation validation
- deterministic recommendationとprovider fallback tests
- HTTP endpoint、strict JSON、body limit、rate limit、error code tests
- correction consent、file mode、retention purge tests
- metrics / privacy-sensitive logging tests
- `make verify-mvp` の実HTTP smoke
- Playwright desktop/mobile E2E
- static binary check、scratch image health/readiness smoke
- source/binary dependency、filesystem/image、secret scans

Release environment checks:

- 5府県すべてを含み、fresh施設が1件以上ある
- `/metrics` がpublic Internetから到達できない
- correction volumeのowner、mode、暗号化、backup、retentionを確認した
- Google有効時はkey制限、quota/billing alert、実通信、privacy表示を確認した
- post-deploy smoke、observation window、rollbackを実施した

[事実] Vercel ContainerとNeonの接続、migration、health/readiness、facility API、訂正API、desktop/mobile E2EをProductionで確認した。[未検証] metrics network restriction、alert、Google quota、custom domain、main pushからの自動deployは未設定である。

## 7. Guardrails and evolution triggers

- AIをMVP推薦へ追加しない。追加時は事実生成、位置送信、評価、fallbackを新ADRで決める。
- facility catalog databaseはoperator更新、履歴、同時更新がJSON運用を超えた時だけ検討する。訂正reportの永続化はProductionでNeonを利用する。
- distributed rate limitは複数instance運用またはabuse実測後に導入する。
- queue、cache、service分割は独立scale、障害分離、release cadence、SLOの必要性を計測してから採用する。
- provider追加は品質、cost、privacy、利用規約、fallback semanticsをADRで比較する。
- 5府県外へのscope拡大は需要証拠、catalog owner、更新工数を満たしてから決める。

関連する運用詳細は [Observability](../operations/observability.md)、[Continuous Delivery](../operations/continuous-delivery.md)、[MVP Runbook](../operations/mvp-runbook.md)、privacy controlは [Security Baseline](../security/security-baseline.md) を正とする。
