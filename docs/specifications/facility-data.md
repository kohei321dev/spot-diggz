# Facility Data Specification

- Status: Current
- Related requirements: R-004、R-006、R-008–R-010、R-014–R-016、NFR-001–NFR-003
- Related decisions: ADR-0002、ADR-0008、ADR-0010、ADR-0011、ADR-0013、[DR-0018](../decisions/0018-api-first-product-definition.md)

## Source of Truth

公開facility catalogの正本は[`../../data/facilities.json`](../../data/facilities.json)です。Gitでreviewし、application起動時とCIでvalidateします。runtime external searchからcatalogへ自動追加しません。

## Current geographic coverage

現行catalogの収録地域とschema・validatorの都道府県許可値は、大阪府、兵庫県、和歌山県、奈良県、徳島県です。2026-07-19調査基準の公開catalogは31施設（大阪府24施設）です。これは現在のデータと実装の制約であり、プロダクトのターゲット地域の制限ではありません。地域を限定しない定義へ訂正しても、この制約の解除や全国データの一括登録を実施したことにはなりません。収録範囲の変更時は、検証・更新できる施設データとschema・validator・testを合わせて整備します。

## Required facility fields

- ID、施設名、住所、都道府県、市区町村、公開施設座標
- 利用可能な競技
- 通常営業時間、曜日別休業、一般利用状態、営業時間の根拠、必要な注意事項
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

- 動的情報は営業時間、料金、予約、休場、利用rule等で、`dynamicVerifiedAt`から30日以内を推薦条件にします。
- 安定情報は名称、住所、地域、設備、access等で、`stableVerifiedAt`から180日以内を推薦条件にします。
- 期限超過recordは参照可能ですが推薦しません。fresh recordが0件なら`/readyz`は503です。
- 未来の検証時刻、必須translation欠落、不正な休場形式、未検証status、重複ID等は起動時に拒否します。
- `schedule_check_required`は存在・所在地を参照できますが、日付別予定を確認しない限り推薦しません。

## Curated media

YouTube動画は任意で、施設ごとに0または1件です。recordはvideo ID、通常watch URL、選定日、確認日、選定理由を持ちます。Instagram/X profileも任意で、施設・platformごとに0または1件のHTTPS URLと公式性確認日を持ちます。

Mediaは施設事実のsourceまたは推薦scoreに使用しません。任意iframe URL、post URL、hashtag URL、許可外hostを拒否します。規約、権利、公式性、安全性に問題が生じた場合は外部linkを含むrecord全体をcatalogから除外します。

## Recommendation input and output data

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
- 週次失敗時は公式`sourceUrl`を再確認し、確認した属性だけの時刻と休場を更新します。
- 開発・E2E fixtureでProduction catalogの鮮度検査を代替しません。
- 候補発見の履歴は[`../research/discovery/`](../research/discovery/)、現在の契約は本書とcatalogを参照します。
