# SpotDiggz Requirements

- Status: Current
- Product: [`product.md`](product.md)
- Last reviewed: 2026-09-06

要求IDは移行前のProduct Baselineにある`R-NNN`を維持します。Issue、仕様、test、Decision RecordはこのIDを参照します。

## Product direction and implementation boundary

[DR-0018](decisions/0018-api-first-product-definition.md)に従い、対象は地域や経験レベルを限定しない「スケボーをしたい人」です。UI、Slack・Discordのbot・appなどから施設情報と推薦を利用できるAPIを整備します。

以下のR-001〜R-020は既存の要求・受入条件です。現在のWeb向けAPIと内部serviceを呼ぶchat adapterの契約を、独立したAPI client向けの整備が完了した証拠とは扱いません。現在のowner認証、入力の許可値、推薦可能なcatalog範囲は、この方針訂正だけでは変更しません。

- Status: Incomplete
- Missing evidence: 各UI・bot・app向けAPIの呼出契約、認証・認可、互換性、エラー・応答形式、整備の優先順序と受入テスト。
- Required decision: ownerが既存APIの再利用範囲と不足する契約を別Issueで明確にし、必要なDecision Recordを承認してから実装する。匿名公開や複数userへの開放はこの訂正に含めない。

## Functional requirements

### R-001: 推薦条件入力

- Requirement: 目的、気分、level、利用可能時間、検索位置、交通手段を入力できる。
- Acceptance: 6項目を選択でき、API境界で列挙値、時間、座標範囲、不明fieldを検証する。

### R-002: 最大3件の推薦

- Requirement: hard conditionを満たす施設を、安定した順序で最大3件提示する。
- Acceptance: 同じcatalog、入力、時刻、provider結果から同じ順位を返し、不適合施設を含めない。

### R-003: 説明可能な推薦

- Requirement: 各候補へ目的、設備、時間、移動条件に基づくおすすめ理由を表示する。
- Acceptance: 理由を構造化responseで返し、未確認事実またはAI生成事実を含めない。

### R-004: 出典と鮮度

- Requirement: 施設ごとに情報源と検証時刻を表示する。
- Acceptance: `sourceUrl`、`verifiedAt`、`dynamicVerifiedAt`、`stableVerifiedAt`をAPIと詳細表示から確認できる。

### R-005: 外部navigation

- Requirement: 推薦結果または施設詳細から外部navigationへ遷移できる。
- Acceptance: 公開施設座標を使うHTTPS linkを、用途を表すlabel付きで開ける。

### R-006: 訂正報告

- Requirement: 利用者が施設単位で情報の誤りを報告できる。
- Acceptance: 入力検証と同意条件を満たすreportだけを保存し、receiptを返す。ProductionはPostgreSQL、local/CIはfile fallbackを使える。

### R-007: 日本語と英語

- Requirement: 日本語と英語で主要導線を利用できる。
- Acceptance: UIと施設の主要なsource-backed事実を両言語で表示し、translation欠落をcatalog validationで拒否する。

### R-008: 検索位置の非保存

- Requirement: 正確な検索位置と地点検索文字列をapplicationで永続化しない。
- Acceptance: store、access log、metrics、responseに座標または検索文字列を残さない。Google有効時の外部送信はprivacy表示で区別する。

### R-009: 鮮度と休場の推薦判定

- Requirement: 情報鮮度と一回限り・毎年の休場期間を推薦判定に使う。
- Acceptance: dynamic 30日、stable 180日の両方がfreshで、`one_time` / `annual`休場に該当しない施設だけを推薦する。

### R-010: Route providerの縮退

- Requirement: Google Routesの未設定または障害時も基本推薦を継続する。
- Acceptance: straight-line計算へ自動fallbackし、responseで実経路と概算を区別する。地点検索は利用不能を明示する503を返す。

### R-011: 主要flowの観測

- Requirement: HTTP、推薦、allowlist済みproduct event、catalog freshnessを観測できる。
- Acceptance: 安定したevent名と低cardinality labelを持つmetrics/logを提供し、秘密情報、正確な位置、自由入力を含めない。

### R-012: 公開endpointの濫用制限

- Requirement: 公開write endpointの入力サイズ、形式、頻度を制限する。
- Acceptance: route別body limit、strict JSON、列挙値検証、process内token bucketを適用し、429または安定したerror codeを返す。

### R-013: 認識可能でaccessibilityのある操作

- Requirement: 主要操作と外部導線を、用途を表すiconと短いlabelで表示する。
- Acceptance: iconだけで意味や状態を伝えず、keyboard操作、accessible name、44 CSS px以上の操作領域を満たす。

### R-014: 手動選定YouTube動画

- Requirement: 施設ごとに手動選定済みYouTube動画を0または1件、任意の補助情報として表示できる。
- Acceptance: 施設詳細を利用者が開いた後だけprivacy-enhanced playerを生成し、自動再生しない。同じ操作で閉じて再表示でき、埋込失敗時は推薦を維持して通常linkへ縮退する。

### R-015: 公式SNS profile

- Requirement: 公式性を確認済みのInstagramまたはX profileへ外部遷移できる。
- Acceptance: 施設・platformごとに最大1件のHTTPS profile URLだけをlabel付きで表示し、投稿、hashtag、feedをapplication内へ表示しない。

### R-016: 外部mediaの手動管理

- Requirement: 外部mediaの取得と表示を手動確認可能な境界に限定する。
- Acceptance: 第三者site・SNSをscraping、保存、再配信せず、任意iframe URLを受け付けない。ownerが規約、埋込可否、権利、公式性、確認日を記録する。

### R-017: GitHub owner認証

- Requirement: Web UIと`/api/*`を許可したGitHub ownerだけが利用できる。
- Acceptance: OAuth `state`とPKCEを検証し、`GITHUB_OWNER`一致時だけ最大12時間の署名済みHttpOnly sessionを発行する。未認証APIは401、Production設定欠落は起動失敗とする。

### R-018: Slack・Discord推薦入口

- Requirement: 同じownerがSlackとDiscordから推薦を要求できる。
- Acceptance: Slack HMAC署名とteam/user ID、Discord Ed25519署名とapplication/guild/user IDを検証する。Slackはmodal条件、Discordは設定済み既定条件から最大3件をephemeral responseで返す。

### R-019: Message history非依存

- Requirement: chat連携をmessage historyへ依存させない。
- Acceptance: 過去message APIを呼ばず、地点、座標、推薦文、interaction tokenを永続化しない。Slack retry防止用のHMAC化source keyと処理状態等だけを最大1時間保持する。

### R-020: 推薦候補を保存しない

- Requirement: Slackを含む推薦候補をLists等へ保存しない。
- Acceptance: 候補は公式情報と「ここに行く」の外部導線を提供し、保存button、保存API、保存tableを持たない。

## Facility data requirements

公開施設は、ID、日英名称・住所、市区町村・都道府県、公開座標、競技、営業時間・休業、一般利用状態と根拠、注意事項、休場期間、料金・予約・登録、初心者適性、設備・路面・照明・屋内外、安全rule、access、source、status、confidence、検証時刻を持ちます。schemaと制約の正本は[`specifications/facility-data.md`](specifications/facility-data.md)です。

公開catalogは2026-07-19調査基準で5府県31施設（大阪府24施設）です。これは現在の収録範囲であり、ターゲットを5府県に限定する要件ではありません。現行schemaとvalidatorの地域制約は別の実装変更まで維持します。`schedule_check_required`は参照できますが、日付別予定の確認なしでは推薦しません。

## Non-functional requirements

### NFR-001: Catalog trust and freshness

- Quality attribute: 正確性と鮮度安全性
- Measurement: source・日英必須属性・検証時刻保有率100%、freshness gauge、weekly horizon check
- Acceptance: 構造不正は起動拒否、全件staleはreadiness 503、stale施設は推薦から除外する。

### NFR-002: Determinism

- Quality attribute: 再現性
- Measurement: clock/provider注入とstable-order test
- Acceptance: 同じ入力と依存結果から同じ候補・順位・理由を返す。

### NFR-003: Privacy and security

- Quality attribute: data最小化と多層防御
- Measurement: auth/authorization、log、body limit、CSP、secret scan、container test
- Acceptance: [`security.md`](security.md)のrelease gateを満たす。

### NFR-004: Recoverability

- Quality attribute: provider障害とrelease障害からの復旧
- Measurement: provider failure test、health/readiness smoke、rollback exercise
- Acceptance: Google障害時に推薦を継続し、application rollbackでcorrection reportを失わない。

### NFR-005: Accessibility and localization

- Quality attribute: mobile、keyboard、screen reader、日英利用
- Measurement: desktop/mobile E2Eとmanual keyboard/screen-reader review
- Acceptance: R-007とR-013を満たす。

### NFR-006: Operability and observability

- Quality attribute: 診断、計測、復旧可能性
- Measurement: health/readiness、structured log、metrics、artifact metadata、runbook exercise
- Acceptance: request rate・error・duration、外部依存結果、catalog freshness、retention失敗を秘密情報なしで観測できる。

### NFR-007: Performance

- Quality attribute: 小規模catalogの対話的応答
- Measurement: HTTP duration histogramとE2E
- Acceptance: 数値SLOは[`operations/observability.md`](operations/observability.md)の現行目標に従う。Production測定証拠がない項目はrelease時に`Incomplete`として扱う。

## Constraints

- 単一deploy可能単位のGoモジュラーモノリスを維持します。
- facility catalogはGit管理のread-only JSON snapshot、Production correctionはNeon/PostgreSQLです。
- MVP推薦にAIを使用しません。
- external provider、queue、cache、service分割、catalog databaseは必要性の計測とDecision Recordなしに追加しません。
- permanent stagingは現在のscopeでは設けず、必要な変更だけ一時Vercel Previewを使います。
- secret、個人情報、正確な現在地をsource、artifact、logへ保存しません。

## Release gates

- 文書・JSON・OpenAPI、Go format/vet/test、MVP smoke、E2E、build、security scanが変更範囲に応じてPASSする。
- [DR-0017](decisions/0017-minimal-development-ci.md)に従い、通常CIはGo・契約・文書・secret・依存検証を実行する。本番catalogの実時間鮮度、E2E、container検証はrelease前の手動確認とし、通常CIの成功だけではrelease条件を満たしたと扱わない。
- catalogが現在のschema・validatorの制約を満たし、fresh施設が1件以上ある。現行の5府県に関する検証は既存catalogの検証であり、恒久的なターゲット地域の制限ではない。制約を変更する際はschema・validator・関連testも別作業で整合させる。
- owner認証、Slack/Discord署名・owner認可、media allowlist、provider fallbackを変更範囲に応じて検証する。
- Production変更時はpost-deploy smoke、data migration、secret、network、rollbackを確認する。
- 未確認のProduction Discord、Google、metrics制限、custom domain、自動deploy、rollback exerciseを確認済みと扱わない。

## Traceability

| Requirements | Specification | Primary tests or checks | Decisions |
| --- | --- | --- | --- |
| `R-001`–`R-005`, `R-007`, `R-009`, `R-013`, `R-014`, `R-015` | [`web-ui.md`](specifications/web-ui.md), [OpenAPI](specifications/facility-catalog.openapi.yaml) | `internal/*_test.go`, `e2e/spot-diggz.spec.ts`, `npm run test:contracts` | ADR-0003, ADR-0009, ADR-0010, ADR-0013, ADR-0014 |
| `R-006`, `R-008`, `R-011`, `R-012`, `NFR-003`–`NFR-007` | [`facility-data.md`](specifications/facility-data.md), [OpenAPI](specifications/facility-catalog.openapi.yaml) | `go test -race ./...`, `make verify-mvp`, operations smoke | ADR-0007, ADR-0008, ADR-0012 |
| `R-017`–`R-020` | [`chat-integrations.md`](specifications/chat-integrations.md), [OpenAPI](specifications/facility-catalog.openapi.yaml) | `internal/owneraccess/*_test.go`, `internal/chatbot/*_test.go` | ADR-0015, ADR-0016 |
| Facility data, `NFR-001`, `NFR-002` | [`facility-data.md`](specifications/facility-data.md) | `make verify-catalog`, recommendation tests | ADR-0002, ADR-0003, ADR-0008, ADR-0011 |
