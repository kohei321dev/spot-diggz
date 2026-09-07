# SpotDiggz Requirements

- Status: Current
- Product: [`product.md`](product.md)
- Last reviewed: 2026-09-07

要求IDは移行前のProduct Baselineにある`R-NNN`を維持します。Issue、仕様、test、Decision RecordはこのIDを参照します。

## Product direction and implementation boundary

[DR-0018](decisions/0018-api-first-product-definition.md)に従い、対象は地域や経験レベルを限定しない「スケボーをしたい人」です。UI、Slack・Discordのbot・appなどから施設情報と推薦を利用できるAPIを整備します。

以下のR-001〜R-020は移行前の要求・受入条件をID保持のため記載しています。新MVPでは[DR-0019の要求ID適用表](specifications/api-mvp.md#要求idの適用移行)を優先し、独自Web UI・iframe・Web OAuth必須・Discord固定条件を新しい提供要件として引き継ぎません。現在のWeb向けAPIと内部serviceを呼ぶchat adapterの契約を、独立したAPI client向けの整備が完了した証拠とは扱いません。現在のowner認証、入力の許可値、推薦可能なcatalog範囲は、この方針訂正だけでは変更しません。

- Status: Incomplete
- Missing evidence: Botの構文・受付後の実行/配送失敗対策、旧コード退役、Gateway制限機能、Cloud Run環境と予算通知の実設定。読み取りAPIの認証・失効・owner対応付け・JSON・互換modeはDR-0021で具体化。
- Required decision: ownerが既存APIの再利用範囲と不足する契約を別Issueで明確にし、必要なDecision Recordを承認してから実装する。匿名公開や複数userへの開放はこの訂正に含めない。

[DR-0019](decisions/0019-api-client-boundary.md)で独立HTTP API、Slack/Discord Bot、独自Web UI不要、Cloud Run、スポット追加は後続、検索中→結果/エラー、月額目安USD 3・メール通知・予算超過時停止なしを採用しました。正本は[MVP API提供契約](specifications/api-mvp.md)と[運用計画](operations/cloud-run-plan.md)。#312と[詳細案](research/api-client-contract-plan.md)で残る技術判断を追跡します。以下の旧Web条件や現行OpenAPIを実装完了の証拠にはしません。

## 新MVP入力の採用事項

[DR-0020](decisions/0020-mention-nearby-search.md)で、メンション＋場所名と任意genre/limit/sortによる周辺検索を採用し、検索範囲も指定可能にします。正本は[周辺検索仕様](specifications/nearby-search.md)。R-001の6条件必須、R-002の固定3件、R-003の目的別評価、R-018のmodal/固定起点は、[要求ID適用表](specifications/api-mvp.md#要求idの適用移行)の新MVP契約へ部分置換します。以下の旧ID本文は移行前の追跡情報です。

受入方向は、場所名だけで検索、genreでpark/streetを区別、指定範囲と件数を独立に適用、直線距離順、曖昧地点を勝手に確定しないことです。APIのlimit/radiusKm/応答/場所解決/privacyはDR-0021で具体化しテストする。Bot構文、genreの実データ移行、platform別メンション受信/owner限定返信は#312〜#316で継続する。

## 読み取りAPIの受入条件（DR-0021）

- R-001〜R-003: queryだけで検索し、任意genre/limit/radiusKm/sortを境界検証する。距離・IDの安定順、範囲/件数の独立適用、曖昧地点・0件・データ不足・provider障害の区別をHTTP testで確認する。
- R-004: genre・滑走根拠・鮮度を満たすrecordだけ候補へ出す。現在営業中/滑走可を保証しないことをresponseへ明示する。実データの再調査は後続。
- R-008: query・検索中心・credentialをlog/storeへ残さない。検索中心/元queryをresponseへ返さず、曖昧地点確認用ラベルと公開施設情報のみ認証済みclientへ返す。旧地点APIの退役は別途。
- R-017: per-client Bearer、owner mapping、facilities:read、期限・個別失効、設定欠落fail closedを検証する。Botでの人間の認可は別境界。API modeはWeb/OAuth/DBなしで起動する。
- 検証正本: [読み取りAPI](specifications/read-api.md#検証根拠)。HTTP provider stubと固定clockによるtestは実Google/Bot/Cloud Run E2Eの代替ではない。

## カタログ品質の受入条件（DR-0022）

以下の旧ID本文に加え、現行の施設データには[DR-0022](decisions/0022-domestic-catalog-quality.md)の受入条件を適用します。

- R-004、R-007、R-009、NFR-001: 国内47都道府県の正式名称を受理し、略称・不正値を拒否する。旧5府県recordの互換性と範囲外だった国内地域をtestする。名称の許可を全国データ収録と同一視しない。
- R-004、R-009: known/非該当/不明を分離する。根拠・一般利用状態・日英説明のない非該当、parkの非該当、非knownと時刻の矛盾を拒否する。不明・未分類・根拠不足・期限超過は検索対象にせず、根拠のあるstreetの非該当は周辺検索と詳細で扱う。旧即時滑走推薦では非knownを移動provider呼出し前に除外する。
- NFR-001、NFR-004: 検索・API readiness・公開前の検索掲載gateを同じ適格性判定にする。全件168時間先までの鮮度と検索掲載候補1件以上を検証し、code/schema/catalogの互換な組でrollbackする。実施設の追加・再確認は今回の成果に含めず、公開前gateとして維持する。
- 検証: `internal/facility/catalog-quality_test.go`、`internal/httpapi/read_api_test.go`、`internal/recommendation/engine_test.go`、`cmd/catalogcheck/main_test.go`、`npm run test:contracts`。手順は[カタログ保守](guides/catalog-maintenance.md)。

## Functional requirements (migration baseline)

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

公開catalogは2026-07-19調査基準で5府県31施設（大阪府24施設）です。これは現在の収録範囲であり、ターゲットを5府県に限定する要件ではありません。DR-0022のschemaとvalidatorは国内47都道府県を受理し、実catalogは変更しません。`hoursStatus=unknown`や`schedule_check_required`は詳細参照できますが、確認不足のまま検索・即時滑走推薦には含めません。

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

- APIのGoモジュラーモノリスと共通推薦engineを基本にします。独立clientからAPIを呼べることを必須とし、Botの物理配置・deploy単位の詳細は#312/#318で評価します。
- facility catalogはGit管理のread-only JSON snapshotです。Neon/PostgreSQLは移行前の訂正・短期状態storeであり、新MVPでの必要範囲・費用を#312/#318で確認します。既存データを今回削除しません。
- MVP推薦にAIを使用しません。
- external provider、queue、cache、service分割、catalog databaseは必要性の計測とDecision Recordなしに追加しません。
- permanent stagingは設けません。Cloud Run向け一時検証環境の要否は#318で確認し、旧Vercel Previewを既定としません。
- 費用は全体で月額USD 3を目安とし、予算アラートメールのみを使います。超過を理由とした自動停止・課金無効化・推薦拒否は行いません。
- secret、個人情報、正確な現在地をsource、artifact、logへ保存しません。

## Release gates

- 文書・JSON・OpenAPI、Go format/vet/test、MVP smoke、E2E、build、security scanが変更範囲に応じてPASSする。
- [DR-0017](decisions/0017-minimal-development-ci.md)に従い、通常CIはGo・契約・文書・secret・依存検証を実行する。本番catalogの実時間鮮度、E2E、container検証はrelease前の手動確認とし、通常CIの成功だけではrelease条件を満たしたと扱わない。
- catalogが現在のschema・validatorの制約を満たす。新APIの公開前には`go run ./cmd/catalogcheck -require-searchable`で全件が168時間先までfreshであり、その終端にも検索掲載可能なrecordが1件以上あることを確認する。既存5府県fixtureの検証や通常CIを現在の実データ検証の代替にしない。
- owner認証、Slack/Discord署名・owner認可、media allowlist、provider fallbackを変更範囲に応じて検証する。
- Production変更時はpost-deploy smoke、data migration、secret、network、rollbackを確認する。
- 未確認のProduction Discord、Google、metrics制限、custom domain、自動deploy、rollback exerciseを確認済みと扱わない。

## Traceability

次表は移行前の追跡情報です。新MVPの適用範囲と後続testは[API契約](specifications/api-mvp.md#要求idの適用移行)を参照します。旧UI testを新MVPの恒久gateとせず、変更は#313の差分評価で必要なAPI回帰検証へ整合させます。

| Requirements | Specification | Primary tests or checks | Decisions |
| --- | --- | --- | --- |
| `R-001`–`R-005`, `R-007`, `R-009`, `R-013`, `R-014`, `R-015` | [`web-ui.md`](specifications/web-ui.md), [OpenAPI](specifications/facility-catalog.openapi.yaml) | `internal/*_test.go`, `e2e/spot-diggz.spec.ts`, `npm run test:contracts` | ADR-0003, ADR-0009, ADR-0010, ADR-0013, ADR-0014 |
| `R-006`, `R-008`, `R-011`, `R-012`, `NFR-003`–`NFR-007` | [`facility-data.md`](specifications/facility-data.md), [OpenAPI](specifications/facility-catalog.openapi.yaml) | `go test -race ./...`, `make verify-mvp`, operations smoke | ADR-0007, ADR-0008, ADR-0012 |
| `R-017`–`R-020` | [`chat-integrations.md`](specifications/chat-integrations.md), [OpenAPI](specifications/facility-catalog.openapi.yaml) | `internal/owneraccess/*_test.go`, `internal/chatbot/*_test.go` | ADR-0015, ADR-0016 |
| Facility data, `NFR-001`, `NFR-002` | [`facility-data.md`](specifications/facility-data.md) | `make verify-catalog`, recommendation tests | ADR-0002, ADR-0003, ADR-0008, ADR-0011 |
