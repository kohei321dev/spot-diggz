# SpotDiggz Architecture

- Status: Current
- Last reviewed: 2026-09-05
- Requirements: [`requirements.md`](requirements.md)
- Decision authority: [Accepted Decision Records](decisions/README.md)
- Migrated from: [`architecture/quality-attributes.md`](architecture/quality-attributes.md) under Issue #305

## System context

SpotDiggzは、GitHub owner認証、Web UI、HTTP API、Slack/Discord command adapter、facility catalog、決定論的推薦、外部provider adapter、訂正store、observabilityを1つのGo applicationとしてdeployするモジュラーモノリスです。

```text
Browser
  -> GitHub OAuth owner authentication
  -> embedded Web UI / owner-protected HTTP API
       -> session input validation
       -> read-only facility catalog JSON
       -> deterministic recommendation
            -> Google Routes adapter (optional)
            -> straight-line adapter (fallback)
       -> Google Geocoding adapter (optional)
       -> correction store (Neon/PostgreSQL in Production, file locally)
       -> rate limit / log / metrics
  -> YouTube player (施設詳細を利用者が開いた後だけbrowserから直接)
  -> official Instagram / X profile (利用者の外部遷移後だけ)

Slack /spotdiggz
  -> HMAC署名 + team/user allowlist
  -> modal input -> same recommendation engine -> ephemeral response

Discord /spotdiggz
  -> Ed25519署名 + application/guild/user allowlist
  -> configured defaults -> same recommendation engine -> ephemeral response
```

## Components

| Component | Responsibility | Inputs | Outputs / state | Must not own |
| --- | --- | --- | --- | --- |
| `internal/webui` | 埋込静的asset、browser entry、詳細の段階表示 | HTTP response、API data | HTML/CSS/JS、product event | 推薦rule、第三者content収集 |
| `internal/httpapi` | route、HTTP validation、response/error、security header | HTTP request | HTTP response | provider固有request、永続化format |
| `internal/owneraccess` | OAuth state/PKCE、owner allowlist、session | GitHub OAuth callback、cookie | 署名済みsession | 推薦rule、OAuth token保存 |
| `internal/chatbot` | Slack/Discord署名、owner認可、ephemeral response | platform request | modalまたは推薦response、短期Slack状態 | message history、account linking、ranking |
| `internal/session` | 条件のdomain validation | purpose、mood、level、time、transport、origin | validated session | HTTP/Google固有型 |
| `internal/facility` | catalog model/load/validation、freshness、営業時間、休場、media metadata | JSON snapshot、時刻 | validated facility set | user request、credential、任意iframe |
| `internal/recommendation` | hard condition、score、stable order、reason | session、facility、travel result、clock | 最大3件の候補 | network、AI、catalog mutation |
| `internal/travel` | provider interface、Google Routes、fallback | origin、施設座標、transport | travel estimate | ranking |
| `internal/geocoding` | Google Geocoding境界 | location query | origin candidate | catalog scope拡張、query保存 |
| `internal/correction` | report validation、保存、90日purge、診断 | correction request | receipt、PostgreSQL/file state | catalog自動更新 |
| `internal/ratelimit` | process内token bucket | route request | allow/deny | identity、distributed quota |
| `internal/observability` | structured logとPrometheus metrics | stable event/result metadata | log、metrics | body、正確な位置、contact |
| `cmd/api` | configuration、dependency composition、lifecycle | environment、catalog | running application | domain rule |

依存方向はruntime adapterからdomainへ向けます。推薦はtravel provider interfaceに依存し、Google HTTP形式へ直接依存しません。Geocoding結果は起点候補であり、runtimeで5府県catalogを増やしません。訂正はowner review前にcatalogを書き換えません。

## Data and ownership

| Data | Source of Truth | Lifecycle |
| --- | --- | --- |
| Facility catalog | `data/facilities.json` in Git | imageへread-onlyで含め、変更はreviewとCIを通す |
| API contract | [`specifications/facility-catalog.openapi.yaml`](specifications/facility-catalog.openapi.yaml) | endpoint変更と同じPRで更新する |
| Correction report | Neon/PostgreSQL in Production、JSON Lines in local/CI | `deleteAfter`を持ち、起動時と1時間ごとに90日超過分をpurgeする |
| Slack request state | PostgreSQL | retry防止に必要な最小状態だけを最大1時間保持する |
| Owner/platform mapping and credentials | runtime environment / provider secret store | Git、image、logへ保存しない |
| Search origin and location query | request memory only | request終了後に破棄し、application storeへ保存しない |

## Runtime behavior and failure isolation

### Startup

1. Catalogのschema、source、座標、translation、時刻、休場、mediaを検証します。
2. Correction storeを初期化し、期限超過reportをpurgeします。
3. GitHub owner認証を構成します。Productionで必須設定が欠ける場合は起動を拒否します。
4. Slack/Discord設定がある場合は、完全性、platform mapping、既定条件を検証します。部分設定は起動を拒否します。
5. Google keyがあればadapterを、なければstraight-line推薦とdisabled geocodingを構成します。
6. fresh施設数を評価し、HTTP serverとretention workerを開始します。

構造不正なcatalogまたはstore初期化失敗はstartup failureです。期限超過catalogは参照できますが、fresh施設が0件なら`/healthz`は200、`/readyz`は503です。

### Request path

- Recommendationはfreshness、休場、営業時間、level、時間、移動条件で除外し、最大3件をstable orderで返します。
- Browser UIと`/api/*`はowner session、chat endpointはplatform署名とowner platform IDを要求します。
- Slackはmodal submissionへ即時ackし、background処理後にephemeral responseを返します。処理は12秒でtimeoutします。
- Google Routesは4秒timeoutとし、HTTP・decode・element failure時はrequest全体をstraight-lineへfallbackします。
- Geocoding未設定・失敗時は503を返し、UIは代表地点またはbrowser geolocationへ戻れます。
- Correctionはstoreへの書込成功時だけ202を返します。file storeは32 MiB上限です。
- YouTube iframeは、利用者が動画を含む施設詳細を開いた時点で初めて生成します。閉じている間は生成せず、自動再生しません。
- SNSはallowlist済み公式profileへの外部linkだけで、postやmediaを収集しません。

## Deployment

```text
immutable image
  - CGO_ENABLED=0 static Go binary
  - embedded Web UI
  - verified catalog snapshot
  - CA bundle
runtime configuration
  - port / environment / version / canonical base URL
  - GitHub, Slack, Discord owner authentication settings
  - optional GOOGLE_MAPS_API_KEY
persistent external state
  - Neon/PostgreSQL correction reports and Slack short-lived state
external services
  - GitHub OAuth, Slack API, Discord API
  - Google Routes / Geocoding (optional)
  - YouTube / Instagram / X (explicit browser action only)
```

リポジトリ内の配置設計はVercel ContainerとNeonを前提にしていますが、現在の本番稼働と公開先は未確認です。[公開先の確認状況](operations/service-status.md)を参照してください。運用再開時は通常localとCI、必要な変更だけ一時Preview、Production反映後のsmokeを行う方針です。permanent stagingは現scopeで設けません。

## Quality attributes

| Priority | Attribute | Contract | Verification |
| --- | --- | --- | --- |
| P0 | Catalog trust | verified record、公式source、日英必須属性、検証時刻 | startup/catalog tests、source review |
| P0 | Freshness safety | dynamic 30日・stable 180日がfreshな施設だけを推薦 | unit test、readiness、freshness metrics |
| P0 | Privacy | origin/queryの非保存・非logging、外部送信境界の明示 | log/store test、privacy review |
| P0 | Determinism | 同じ入力と依存結果から同じ順位・理由 | injected clock/provider、stable-order test |
| P0 | Recoverability | Google障害時のfallback、reportを失わないrollback | failure test、runbook smoke、rollback exercise |
| P1 | Modifiability | UI、HTTP、domain、provider、storeの責務分離 | package review、focused test |
| P1 | Observability | HTTP、推薦、event、catalog freshnessの低cardinality計測 | metrics test、dashboard/alert review |
| P1 | Security | input、auth、secret、container、media allowlistの多層防御 | CI scan、HTTP/E2E、threat review |
| P1 | Performance | 小規模catalogの同期処理、mediaの遅延読込 | duration histogram、browser network check |
| P1 | Accessibility/i18n | mobile・keyboard・日英の主要flow | desktop/mobile E2E、manual review |
| P1 | Operability | static binary、health/readiness、smoke、rollback | image smoke、runbook exercise |

## Reliability and observability

- livenessは`/healthz`、readinessは`/readyz`を使用します。
- `/metrics`はapplication認証を持たず、Production network境界で制限します。
- request、外部provider、recommendation、retentionは安定したevent名・error code・request IDで観測します。
- log・metrics・traceへrequest body、秘密情報、正確な位置、contactを含めません。
- 詳細は[`operations/observability.md`](operations/observability.md)と[`operations/mvp-runbook.md`](operations/mvp-runbook.md)を正とします。

## Incomplete operational evidence

- Status: Incomplete
- Missing evidence: Discord Production E2E、Google key restriction・quota・billing・fallback実測、metrics network restriction・dashboard・alert、custom domain/DNS、`main`自動deploy、rollback exercise、Neon backup/retentionの継続確認。
- Required decision: ownerが各Production gateの設定・実測後に、operationsとrelease記録へ結果を反映する。

## Architecture boundaries

- AIをMVP推薦へ追加しません。
- Facility catalog database、distributed rate limit、queue、cache、service分割は必要性の計測と新しいDecision Recordなしに追加しません。
- 外部media provider、複数動画、画像・投稿埋込、asset保存を追加する場合は、権利、privacy、CSP、失敗時縮退、運用ownerを新しいDecision Recordで決めます。
- 5府県外へのscope拡大は需要証拠、catalog owner、更新工数を確認してから判断します。

## Related decisions

- Index: [`decisions/README.md`](decisions/README.md)
- Runtime: [ADR-0007](decisions/0007-go-modular-monolith-runtime.md)
- Catalog: [ADR-0008](decisions/0008-facility-catalog-api-and-storage.md)
- Recommendation UI: [ADR-0009](decisions/0009-session-recommendation-ui.md)
- Deployment: [ADR-0012](decisions/0012-vercel-neon-deployment.md)
- Owner and chat boundary: [ADR-0015](decisions/0015-owner-auth-and-chat-entrypoints.md)
