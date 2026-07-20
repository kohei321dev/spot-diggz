# Observability, SLI and SLO Design

- Status: MVP implemented baseline; production wiring pending
- Date: 2026-07-20
- Scope: application JSON logs, in-process Prometheus metrics, product events, catalog freshness

## 1. 目的

利用者が「今日どこで滑るか」を決める主要flowを、HTTP成功だけでなく、推薦結果、catalog鮮度、訂正受付まで観測する。MVP runtimeに実装したsignalと、production基盤が必要な未実装signalを分ける。

答える問い:

- applicationはrequestを受け付け、推薦を返せたか
- 条件に合う候補がなかったのか、処理errorだったのか
- 利用者は入力、結果表示、source、navigationまで進んだか
- catalogにfresh / staleな施設が何件あるか
- 訂正報告を受け付け、retention purgeを継続できているか
- release後にHTTP error率とlatencyが変化したか

## 2. 実装状態

| Signal | MVP state | Boundary |
| --- | --- | --- |
| JSON log | implemented | stdoutへ出力 |
| request correlation | implemented | 全HTTP responseの `X-Request-ID` とaccess log |
| HTTP metrics | implemented | route template、method、status class、duration |
| recommendation metrics | implemented | `success` / `no_results` / `error` とduration |
| product event metrics | implemented | allowlist eventの集計だけ |
| catalog metrics | implemented | `/readyz` と `/metrics` 時点の件数とfresh / stale snapshot |
| Prometheus endpoint | implemented | unauthenticated `GET /metrics` |
| distributed trace | not implemented | 単一processのMVPではrelease gate外 |
| Google provider metrics | implemented | 固定provider・success/error別のrequest数とduration |
| dashboard / alert delivery | [未検証] | production platform未選定 |
| long-term telemetry retention | [未検証] | metrics backend未選定 |

## 3. JSON log contract

applicationはGo `slog` のJSON handlerを使う。共通fieldは次のとおりである。

| Field | Source | Description |
| --- | --- | --- |
| `time` | runtime | event time |
| `level` | runtime | `INFO` / `WARN` / `ERROR` |
| `event_name` | message置換 | 安定したevent identifier |
| `service` | constant | `spotdiggz-api` |
| `environment` | `APP_ENV` | default `development` |
| `version` | `APP_VERSION` | default `unknown` |

### Startup and lifecycle events

- `facility_catalog_loaded`: `count`
- `facility_catalog_load_failed`: `path`, `error`。serverは起動しない
- `correction_store_initialization_failed`: `error`。serverは起動しない
- `correction_service_initialization_failed`: `error`。serverは起動しない
- `correction_retention_purge_failed`: `error`。retention保証を調査する
- `google_maps_integrations_enabled`
- `google_maps_integrations_disabled`: `travel_estimate=straight_line`
- `google_routes_initialization_failed`
- `travel_provider_initialization_failed`
- `geocoder_initialization_failed`
- `http_server_started`: `addr`
- `http_server_failed`
- `http_server_shutdown_failed`

### Access log

`http_request` は次を記録する。

- `request_id`
- `method`
- `route`: `/api/facilities/{facilityId}` 等のroute template
- `status`
- `duration_ms`

raw URL、query string、request body、response body、client IP、User-Agentは記録しない。facility IDを含む実pathではなくroute templateを使うため、location-search queryはaccess logへ入らない。

### Correction log

`correction_received` は `request_id`、`report_id`、`facility_id`、`category` だけを記録する。`details`、`evidenceUrl`、`contact`、`contactConsent` は記録しない。

### 禁止情報

- `GOOGLE_MAPS_API_KEY` 等のsecret
- 正確な起点座標、location-search query、完全な住所
- correctionの自由入力、根拠URL、連絡先
- request / response body
- browser storage値

禁止情報が出た場合は収集先accessを制限し、[Security Baseline](../security/security-baseline.md) のincident手順で扱う。

## 4. Prometheus metrics

`GET /metrics` はPrometheus text exposition formatを返す。application認証はないため、productionではprivate network、sidecar、ingress allowlist等で制限する。

### HTTP

| Metric | Type | Labels | Meaning |
| --- | --- | --- | --- |
| `spot_diggz_http_requests_total` | counter | `route`, `method`, `status_class` | route template別request数 |
| `spot_diggz_http_request_duration_seconds` | histogram | `route`, `method`, `status_class` | HTTP duration |

`status_class` は `2xx`、`4xx`、`5xx` 等へ正規化する。facility ID、request ID、query、event本文をlabelにしない。429はPOST routeの `4xx` として観測する。

### Recommendation

| Metric | Type | Labels | Meaning |
| --- | --- | --- | --- |
| `spot_diggz_recommendations_total` | counter | `result` | recommendation attempt数 |
| `spot_diggz_recommendation_duration_seconds` | histogram | `result` | recommendation processing duration |

`result`:

- `success`: 1件以上を返した
- `no_results`: 正常処理だが0件
- `error`: input validationまたはproviderを含む処理error

HTTP 400のinvalid session inputもrecommendation `error` に含む。rate limitはrecommendation handlerへ入る前に返すため、このcounterではなくHTTP 429で観測する。

### External providers

| Metric | Type | Labels | Meaning |
| --- | --- | --- | --- |
| `spot_diggz_external_requests_total` | counter | `provider`, `result` | Google provider request数 |
| `spot_diggz_external_request_duration_seconds` | histogram | `provider`, `result` | Google provider request duration |

`provider` は `google_routes` と `google_geocoding`、`result` は `success` と `error` の固定値だけを使う。座標、検索文字列、交通手段、error本文をlabelへ含めない。`google_routes` の `error` はstraight-line fallbackを開始した回数であり、その後の推薦成否はrecommendation metricと組み合わせて判断する。

### Product journey

`spot_diggz_product_events_total{event=...}` は次のallowlistだけを数える。

- `input_started`
- `input_completed`
- `recommendation_completed`
- `result_displayed`
- `source_opened`
- `navigation_opened`
- `correction_submitted`

`POST /api/events` は上記のうち `correction_submitted` を除くevent名だけを受け付ける。`correction_submitted` は訂正APIが正常保存後にserver側でのみ加算し、clientからの同名eventは400で拒否する。利用者ID、session ID、locationを持たないため、個人単位の追跡や厳密なcohort funnelは行わない。

### Catalog

| Metric | Type | Labels | Meaning |
| --- | --- | --- | --- |
| `spot_diggz_catalog_facilities` | gauge | none | load済み施設数 |
| `spot_diggz_catalog_freshness` | gauge | `state=fresh|stale` | dynamic 30日とstable 180日の両方で分類した件数 |

gaugeはapplication起動時に初期化し、`/readyz` と `/metrics` の各requestで現在時刻を使って再計算する。recommendationもrequestごとに同じ鮮度期限を適用する。

### Histogram buckets

HTTP、recommendation、external providerは、0.005、0.01、0.025、0.05、0.1、0.25、0.5、1、2.5、5、10秒の固定bucketを使う。

## 5. SLI

### Recommendation availability

```text
good = success + no_results
total = success + no_results + error
availability = good / total
```

429は別のHTTP availability / abuse signalとして併記する。invalid inputをavailabilityから除外するかは、dashboardでHTTP 400とprovider 503を分離できるようになってから再評価する。

### Recommendation latency

`spot_diggz_recommendation_duration_seconds` の `success` と `no_results` を対象にp50 / p95 / p99を算出する。Google Routesの4秒timeoutとfallbackを含むend-to-end値であり、原因の切り分けにはexternal provider durationとresultを併用する。

### Facility freshness

```text
freshness = catalog_fresh / catalog_facilities
```

release gateではmetricsだけでなく、全recordのsource、`dynamicVerifiedAt`、`stableVerifiedAt`、日英必須fieldをtestする。

### User journey proxy

- input completion proxy = `input_completed / input_started`
- result display proxy = `result_displayed / input_started`
- navigation proxy = `navigation_opened / result_displayed`
- source-open proxy = `source_opened / result_displayed`

event送信はbest effortで、network error時に主要flowを失敗させない。これらは需要のproxyであり、実際に到着・滑走できた人数ではない。

### Correction acceptance

`route=/api/corrections` の2xx、4xx、5xxから受付状態を算出する。retention purge失敗はmetric未実装のため `correction_retention_purge_failed` logをalert sourceとする。

## 6. Provisional SLO

MVPにSLAは設定しない。production trafficを2〜4週間収集するまで、次をrelease判断の仮目標とする。

| SLI | Objective | Window |
| --- | ---: | --- |
| Recommendation availability | 99.0%以上 | rolling 30 days |
| Recommendation latency | p95 5秒以下 | rolling 7 days |
| Catalog freshness | 100% | release時 |
| Source・検証時刻・日英必須field | 100% | release test |
| Correction 5xx rate | 1.0%未満 | rolling 7 days |

straight-lineだけのlocal deterministic testはp95 500ms以下を目標にできるが、production SLOと同一視しない。

## 7. Alert and operator action

| Condition | Initial action |
| --- | --- |
| recommendation 5xxまたはavailability低下 | route、release version、provider modeを確認 |
| recommendation p95が5秒超 | Google timeout / fallback、process resourceを確認 |
| catalog staleが1件以上 | sourceを再確認しdata-only release |
| `/api/corrections` 5xx | volume mount、容量、permissionを確認 |
| `correction_retention_purge_failed` | retention breach riskとして即時調査 |
| 429急増 | client loop、濫用、ingress制限を確認 |
| `/healthz` 非200 | trafficを停止し直前imageへrollback |
| `/readyz` 503かつhealth 200 | empty / all-staleを確認し、fresh catalogへ更新または直前imageへrollback |
| Google usage / billing急増 | keyを削除してstraight-lineへ縮退 |

具体的なsmoke、縮退、rollbackは [MVP Runbook](mvp-runbook.md) を正とする。

## 8. Retention and access

| Data | MVP handling |
| --- | --- |
| application logs | [未検証] backend未選定。production目標30日、禁止情報なし |
| metrics | in-memory。process restartでreset。backend retention未選定 |
| traces | 収集しない |
| correction reports | 32 MiB上限のfile store。90日後の `deleteAfter`、起動時と1時間ごとにpurge |
| product events | metric aggregateのみ。raw event recordなし |

telemetry backendを導入する場合は、access control、retention、削除、release markerを先に定義する。

## 9. Known gaps and next gate

- retention purge失敗のcounterがなくlog alertに依存する
- process CPU、memory、Go runtime metricがない
- trace IDとdistributed tracingがない
- metricsがprocess restartでresetする
- production dashboard、alert、scrape、retentionは資格情報とplatform選定が必要で未実施

production公開前に最低限、private metrics scrape、release version、5xx / stale / purge failure alert、dashboard、post-deploy observation windowを設定する。
