# Facility Data Specification

- Status: Current
- Related requirements: R-004、R-006、R-008–R-010、R-014–R-016、NFR-001–NFR-003
- Related decisions: ADR-0002、ADR-0008、ADR-0010、ADR-0011、ADR-0013、[DR-0018](../decisions/0018-api-first-product-definition.md)、[DR-0022](../decisions/0022-domestic-catalog-quality.md)

## Source of Truth

公開facility catalogの正本は[`../../data/facilities.json`](../../data/facilities.json)です。Gitでreviewし、application起動時とCIでvalidateします。runtime external searchからcatalogへ自動追加しません。

## Current geographic coverage

[DR-0020](../decisions/0020-mention-nearby-search.md)で新MVPは`genre=skatepark` / `genre=street`による周辺検索を採用しました。streetはパーク外で滑走可能な根拠・利用ルールを確認した場所であり、パーク内のstreet sectionとは別です。DR-0021でmodel/OpenAPIへ任意`genre`と`skatingPermissionSourceUrl`を追加しました。genreを指定するrecordには滑走根拠のHTTPS URL（userinfo/明示port不可）を必須とし、旧recordの欠落は互換性のため受理しますが周辺検索から除外します。DR-0022で地域表現と営業時間の確認状態を整備しました。本番catalogの分類・確認日は変更していません。新入力は[周辺検索仕様](nearby-search.md)を優先し、未確認情報を無条件利用可として補完しません。

schema・validatorは日本の47都道府県の正式名称を完全一致で受理します。略称、前後空白、任意文字列、海外の州名は拒否します。市区町村・公開住所は必須で、住所・座標が地域と実際に一致することは出典で確認します。国内Geocoding/JSTの実装境界であり、プロダクトのターゲット地域を限定するものではありません。

実catalogの収録は大阪府、兵庫県、和歌山県、奈良県、徳島県のままです。2026-07-19調査基準の31施設（大阪府24施設）を保持し、全国のデータ収録や現在の鮮度を保証しません。許可する地域名と、根拠を確認して収録済みの範囲を区別します。

## 周辺検索掲載の追加条件

[読み取りAPI](read-api.md)はgenre/滑走根拠、dynamic 30日・stable 180日、営業時間状態がunknownでなく、一般利用が日付確認必須でないrecordを対象にします。出典の存在だけで滑走可と推測せず、curatorが許可・利用ルールを確認して分類します。現時点の営業時間/休場はfilterせず、注意情報を返し訪問前確認を要求します。検索・API readiness・追加release gateは`facility.IsNearbySearchable`で同じ条件を使います。

## 営業時間の確認状態（DR-0022）

| `hoursStatus` | 必須根拠・一般利用状態 | `hours` / `hoursBasis` | 周辺検索 / 旧即時滑走推薦 |
| --- | --- | --- | --- |
| `known`または旧recordの省略 | 既存の出典・有効な曜日別時刻。`hoursSourceUrl`は任意 | 有効なhours必須、hoursBasisは既存enum | 他の品質条件を満たせば検索対象 / 従来の営業時間等を判定 |
| `not_applicable` | streetのみ。営業時間制度が非該当であるHTTPSの`hoursSourceUrl`、滑走根拠、明示的なregular/limited、日英availabilityNote | なし。入力は省略/空を受理、responseは省略 | 他の品質条件を満たせば検索対象 / 対象外 |
| `unknown` | 分類済みpark/street、schedule_check_required、日英availabilityNote。hoursSourceUrlは任意 | なし。入力は省略/空を受理、responseは省略 | 対象外 / 対象外。認証済み詳細参照は可能 |

`not_applicable`は「24時間利用可」ではありません。営業時間を公開していないだけなら`unknown`です。省略を非該当へ変換せず、架空の`00:00–24:00`を補いません。新URL fieldはHTTPS・userinfo/明示port不可です。非knownと時刻/営業時間根拠enumの同時指定、parkの非該当、必要な説明・根拠の欠落は起動時に拒否します。

料金、予約、ルール、日英必須属性は引き続き必要です。空白だけのrule/featureは拒否し、不明を無料・登録不要と補完しません。施設の存在や滑走許可等を確認できない候補は公開catalogへ昇格させず、候補資料で`Incomplete`と根拠不足を記録します。hoursのunknownは未検証施設を`status=verified`にできる例外ではありません。

## Required facility fields

- ID、施設名、住所、都道府県、市区町村、公開施設座標
- 利用可能な競技
- 営業時間の確認状態に応じた時刻または非該当/不明の根拠・説明、一般利用状態、必要な注意事項
- `one_time`の`YYYY-MM-DD`範囲と`annual`の`MM-DD`範囲
- 料金、予約、利用登録
- 初心者適性、section、路面、照明、屋根、屋内外
- helmet、防具、年齢等の利用rule
- 最寄り駅、駐車場等のaccess
- `sourceUrl`、`sourceType`、`status=verified`、`confidence`
- `verifiedAt`、`dynamicVerifiedAt`、`stableVerifiedAt`
- 日本語のsource-backed情報と`englishTranslation`

正確なfield名、型、enum、API responseは[`facility-catalog.openapi.yaml`](facility-catalog.openapi.yaml)を正とします。

## Freshness and availability

- 動的情報は営業時間・hoursStatusの根拠、滑走許可・一般利用状態、料金、予約、休場、利用rule等で、`dynamicVerifiedAt`から30日以内を推薦条件にします。
- 安定情報は名称、住所、地域、設備、access等で、`stableVerifiedAt`から180日以内を推薦条件にします。
- 期限超過recordは参照可能ですが推薦しません。API modeの`/readyz`は検索対象となるfresh recordが0件なら503です。legacy modeの鮮度判定とは区別します。
- 未来の検証時刻、必須translation欠落、不正な休場形式、未検証status、重複ID等は起動時に拒否します。
- `schedule_check_required`は存在・所在地を参照できますが、日付別予定を確認しない限り推薦しません。

## Curated media

YouTube動画は任意で、施設ごとに0または1件です。recordはvideo ID、通常watch URL、選定日、確認日、選定理由を持ちます。Instagram/X profileも任意で、施設・platformごとに0または1件のHTTPS URLと公式性確認日を持ちます。

Mediaは施設事実のsourceまたは推薦scoreに使用しません。任意iframe URL、post URL、hashtag URL、許可外hostを拒否します。規約、権利、公式性、安全性に問題が生じた場合は外部linkを含むrecord全体をcatalogから除外します。

## 移行前の推薦・訂正データ（legacy mode）

以下の固定3件・旅程計算・訂正storeは保持した旧実装です。API modeの入出力は[読み取りAPI](read-api.md)を参照してください。旧旅程計算はknown以外を移動provider呼出し前に除外します。

- Origin座標と地点queryはrequest処理中だけ保持し、catalog・correction store・log・metricsへ保存しません。
- Recommendationはfresh facilityとrequest条件から計算し、最大3件を返します。
- Google Routes使用時は起点座標、施設座標、交通手段を送信します。失敗時はstraight-line estimateへ縮退します。
- Google Geocoding使用時は地点queryとcountry/language/region制約を送信します。key未設定または失敗時は503です。

## Correction report

- ProductionはNeon/PostgreSQL、local/CIはJSON Lines fileを使用します。
- `details`は10〜1000文字、`evidenceUrl`は任意のHTTPS URLです。
- contactは任意で、`contactConsent=true`のときだけ受理します。
- reportは`receivedAt`と`deleteAfter=receivedAt+90日`を持ちます。
- 起動時と1時間ごとに期限超過分をpurgeします。catalogへ自動反映しません。
- file storeは32 MiB上限で、書込・sync失敗または上限超過時は503を返します。

## Validation and maintenance

- `make verify-catalog`は公開catalogが実行時点から168時間後もfreshであることを検査します。
- 新APIの公開前には`go run ./cmd/catalogcheck -require-searchable`で上記に加え同じ期間の終端でも検索掲載可能なrecordが1件以上あることを検査します。fixtureの合格を実データの鮮度証拠としません。
- 週次失敗時は公式`sourceUrl`を再確認し、確認した属性だけの時刻と休場を更新します。
- 開発・E2E fixtureでProduction catalogの鮮度検査を代替しません。
- 候補発見の履歴は[`../research/discovery/`](../research/discovery/)、現在の契約は本書とcatalogを参照します。
- 担当、確認根拠、時刻、日英説明、昇格・再検証・rollbackの手順は[カタログ保守](../guides/catalog-maintenance.md)を参照します。
