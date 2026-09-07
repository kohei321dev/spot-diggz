# 独立読み取りAPI

- Status: Implemented in source; production not deployed
- Decision: [DR-0021](../decisions/0021-read-api-contract.md)、[DR-0022](../decisions/0022-domestic-catalog-quality.md)
- Tracking: [#313](https://github.com/kohei321dev/spot-diggz/issues/313)の第一実装単位。全MVP完了ではない。
- Contract: [OpenAPI](facility-catalog.openapi.yaml)

## 起動と認証

`APP_MODE=api`を明示する。未指定/`legacy`は移行前実装を維持し、それ以外は起動失敗。API modeはGitHub OAuth、訂正DB/file store、保持worker、旧Bot handlerを初期化しない。`DATABASE_URL`や`DEV_AUTH_BYPASS`はAPI認証へ影響しない。

`API_OWNER_ID`と`API_CLIENTS_JSON`を起動時に検証する。owner ID/client IDは個人情報でない小文字英数字・`-`・`_`の1〜64文字。client設定は1〜32件、全体32 KiB以下とする。

| Client field | 契約 |
| --- | --- |
| `clientId` | Bot/clientごとの一意ID |
| `ownerId` | 設定された`API_OWNER_ID`と完全一致 |
| `tokenSha256` | 32 byte乱数を64桁小文字hexへ変換したtoken文字列のSHA-256、小文字hex64桁。重複不可 |
| `scope` | `facilities:read`のみ |
| `expiresAt` | タイムゾーン付きRFC 3339。現在時刻と等しい場合も期限切れ |
| `revoked` | trueなら拒否。省略時false |

設定欠落、不正JSON、未知field、owner/scope不一致、重複client ID/digestは起動拒否。APIにはtokenの平文を保存しない。requestでは`Authorization: Bearer <token>`のみを受理し、期限・失効を毎回評価する。重複Authorization header、cookie、queryのtoken、申告owner headerで代用できない。全資格情報が期限切れ/失効でも設定構文は有効だが、利用はすべて401となる。readinessは有効tokenの実呼出しを代替しない。

これはclientの認証であり、人間をAPIが直接認証するものではない。信頼するBotがplatform本人確認とowner ID認可を行う。HTTPS、secret注入、全revisionを含む失効反映は[設定手順](../guides/read-api-setup.md)と#318で扱う。

## Route allowlistと互換性

| Method / path | API mode | 旧mode |
| --- | --- | --- |
| `GET /healthz` | 認証不要、liveness | 維持 |
| `GET /readyz` | 認証設定・地点provider・検索対象となる鮮度内recordがある場合200、それ以外503 | 旧鮮度判定を維持 |
| `POST /api/facilities/search` | Bearer必須、新検索 | 未登録 |
| `GET /api/facilities/{facilityId}` | Bearer必須、既存Facility schema | 旧owner session必須 |
| その他（Web、OAuth、一覧、旧推薦、地点候補、訂正、event、Bot、metrics） | 登録しない。unknown routeはJSON 404 | 旧route・保護を維持 |

旧asset・schema・DB・route実装そのものは削除していない。これは新MVPへ独自Web UIを提供する判断ではなく、段階移行の互換性・rollback用である。旧コードの退役判断は#313の残項目。詳細取得は古い/未分類recordも返すため、検索可能・現在営業中と同一視しない。

## 周辺検索JSON

```json
{"query":"大阪駅","genre":"skatepark","limit":5,"radiusKm":10,"sort":"distance"}
```

これはpublic placeの入力例で、現在の収録・検索成功を保証しない。必須は`query`のみ。前後空白除去後1〜120文字、制御文字不可。limit既定5/1〜10、radiusKm既定10/0.1〜50（小数可）、sort既定distance/同値のみ。genre省略は両種、明示値はskatepark/streetのみ。未知・重複field、null、不正型、範囲外、URL query、不正UTF-8を拒否する。bodyはapplication/json、最大4096 bytes。

APIがGoogle Geocodingで場所を解決し、曖昧ならラベル候補を最大5件返す。検索中心の座標・元queryは返さない。clientは住所/市区町村などでqueryを具体化して再送し、先頭候補を自動採用しない。opaqueな選択tokenや会話stateは作らない。Botの選択UIは後続。

検索対象は`status=verified`、genre・HTTPSの滑走根拠あり、dynamic 30日/stable 180日以内、hoursStatusがunknownでなく、一般利用が`schedule_check_required`でないスケートボードrecord。未分類/期限超過は除外する。根拠のあるstreetの`not_applicable`は検索できるが、営業時間・休場期間の現時点判定はしない。返却する利用条件/休場/出典と`availabilityNote`により、「今滑れる」を保証しない。

施設の地域は国内47都道府県の正式名称。`hoursStatus`省略は従来のknown相当で有効な時刻を持つ。非該当/不明のresponseは`hours`と`hoursBasis`を省略し、状態・日英availabilityNoteを返す。非該当には`hoursSourceUrl`も必須。clientは欠落を24時間利用可や無料に変換しない。条件は[施設データ仕様](facility-data.md#営業時間の確認状態dr-0022)を参照。詳細取得では不明のrecordも参照できるが検索候補ではない。

Haversineによる直線距離kmを丸めず、`distanceKm <= radiusKm`で絞る。距離昇順、同値はfacilityId昇順、最後にlimitを適用。genre/radiusの自動緩和なし。移動/到着/滑走時間を生成しない。

| HTTP / status | 意味・次の操作 |
| --- | --- |
| 200 / `ok` | `matches`に`facility`と`distanceKm`。最大limit件。訪問前に出典とルールを確認 |
| 200 / `no_matches` | 半径内に掲載可能なrecordはあるがgenreが不一致。条件変更を利用者が選ぶ |
| 200 / `data_unavailable` | 半径内の登録がない、または0件かつ根拠/鮮度不足recordがある。実在スポットがないとは断定しない |
| 200 / `location_not_found` | 地点未特定。queryを具体化して再入力 |
| 200 / `location_ambiguous` | `locationCandidates[].label`で場所確認。matchesは空 |
| 400 / 安定error code | 入力を修正。詳細はOpenAPI |
| 401 / `invalid_token` | credentialの期限・失効・設定を確認 |
| 413 / `request_too_large`、415 / `unsupported_media_type` | bodyサイズ/Content-Typeを修正 |
| 429 / `rate_limited` | Retry-After: 60。再試行回数はclient側でも制限 |
| 503 / `location_search_unavailable` | provider未設定/障害/timeout。後で再実行。正常0件へ変換しない |

すべての成功responseにstatus、matches（空でも配列）、distanceKind=`straight_line`、適用limit/radiusKm/sort、availabilityNoteを返す。施設の公開所在地/座標はFacility内に含むが、検索中心とは別物である。UI/Bot向けの短縮表示はclientの責務。

## 制限・観測・運用

- 同期処理、地点解決timeout 5秒、API自身のprovider retryなし。queue・DB・検索履歴・cacheは追加しない。provider障害時の回路遮断は未導入、timeoutとprocess-local bucketで局所的に制限する。
- 認証後のprocess-local共有bucketは検索30回/分burst5、詳細60回/分burst10。複数instanceの全体quota、client別quota、送信元IP制限ではない。#318のGateway検討を代替しない。
- X-Request-ID、Cache-Control: no-storeを返す。HTTP件数/所要時間とGoogle境界の結果/所要時間を既存registryで計測。request ID/status/countだけの検索完了logを追加し、本文、header、検索座標、provider生responseを記録しない。
- API modeに公開metrics routeを設けない。privateなexport/収集、分散trace、アラート、Cloud Run/Gateway logのマスキングは#318の公開gate。application内計測を実運用監視の完了としない。
- provider設定の存在だけで外部疎通成功としない。readinessは実Googleへprobeせず、資格情報/データ/API呼出しのsmokeが別途必要。
- 本番catalogのgenre・根拠・検証日は変更していない。#314/DR-0022で地域制約・営業時間状態・品質検証と[保守手順](../guides/catalog-maintenance.md)を整えた。実データの分類・再調査は後続で、公開前には`catalogcheck -require-searchable`を実行する。架空fixtureを本番へ入れない。

## 検証根拠

`internal/apiauth/credentials_test.go`、`internal/nearby/*_test.go`、`internal/httpapi/read_api_test.go`、`cmd/api/read_api_test.go`、`internal/facility/genre_test.go`、`internal/facility/catalog-quality_test.go`、`cmd/catalogcheck/main_test.go`で正常/境界/権限/外部障害/非漏えい、地域、営業時間状態と検索/詳細/readinessの整合を固定時刻と合成データで確認する。実Bot・実Google・Cloud RunのE2Eは未実施。
