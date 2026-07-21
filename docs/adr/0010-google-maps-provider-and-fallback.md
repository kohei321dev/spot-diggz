# ADR-0010 Google Mapsをoptional providerとして採用しstraight-line fallbackを維持する

- Status: Accepted
- Date: 2026-07-20
- Related: [Product Baseline](../product_baseline.md)
- Related: [ADR-0003](0003-recommendation-engine-before-ai.md)
- Related: [ADR-0009](0009-session-recommendation-ui.md)
- Related: [Security Baseline](../security/security-baseline.md)

## Context

決定論的推薦は、検索起点から各施設までの移動時間をhard conditionとscoreへ使う。straight-line概算だけでは、鉄道網、道路、待ち時間、渋滞等を反映できない。一方、外部route providerを必須にすると、credential、quota、課金、障害、privacyが主要flowの単一障害点になる。

任意の駅名・住所を扱うにはgeocodingも必要である。検索文字列と正確な起点座標はsensitive location dataであり、applicationで保存しない場合でも、外部providerを有効にすればproviderへ送信される。

## Decision

1. Google Maps連携は `GOOGLE_MAPS_API_KEY` の有無で明示的にopt-inする。未設定時はGoogleへrequestしない。
2. key設定時はGoogle Routes APIのCompute Route Matrixを移動時間providerのprimaryとする。起点座標、公開施設座標、交通手段をserverから送信する。MVPは「今から」の検索だけを扱うため `departureTime` を省略し、Google側のrequest時刻を使う。分単位へ丸めた過去時刻は送信しない。
3. Google Routes requestは4秒でtimeoutする。HTTP error、不正response、欠落・失敗したmatrix elementをprovider失敗として扱う。
4. Google Routesが失敗した場合は、request全体を交通手段別固定速度のstraight-line providerで再計算する。fallback結果は `travelEstimateKind=straight_line` と注意文で識別する。
5. key設定時はGoogle Geocoding APIで日本国内の駅名・住所を検索し、最大5件を返す。検索文字列はURL queryではなくJSON bodyでapplicationへ受け、Googleへ送信する。4秒timeout、provider error、未設定時は `503 location_search_unavailable` とする。
6. Geocoding自体には外部provider fallbackを設けない。UIは代表地点またはbrowser Geolocationへ戻れるようにする。
7. applicationは起点座標と検索文字列を永続化せず、query stringを含むURLやrequest bodyをaccess logへ出力せず、推薦responseへ起点を再掲しない。
8. Googleへ送信される事実をprivacy文書と利用者向け表示で明示する。「applicationで非保存」と「外部へ非送信」を同義に扱わない。
9. keyはserver-side secretとしてGit、image、browserへ含めない。productionでは送信元と利用APIを制限し、quota・課金alertを設定する。
10. Google API有効化、billing、credential作成、実通信、production deployは人間の承認を必要とする。本ADR時点では資格情報がないため未実施である。
11. Google連携のrollbackはkeyを削除して再起動する。推薦をstraight-lineだけで継続し、任意地点検索を無効化する。
12. process内token bucketをrecommendation 60/min burst 10、location search 30/min burst 5とし、Google hostごとの同時HTTP connectionを4本に制限する。production ingressとGoogle quotaによるsource別・費用上限は別途必須とする。

## Alternatives

### Googleを必須providerにする

全responseの移動種別を揃えられるが、credential・quota・障害で推薦全体が停止する。MVP availabilityを優先して採用しない。

### straight-lineだけを継続する

costとprivacy境界は最小になるが、5府県の鉄道・道路条件を反映せず、移動可能性の誤判定が増える。

### Google失敗時に503を返す

精度不明なfallbackを避けられるが、利用者が候補を一切得られない。概算表示と外部navigation確認を条件に採用しない。

### browserからGoogle APIを直接呼ぶ

server実装は減るが、web service credentialをuntrusted clientへ露出し、送信・error・quota制御も分散する。server-side boundaryを採用する。

### 別のroute/geocoding providerを採用する

vendor依存を下げられる可能性はあるが、MVPでは複数providerの契約・精度・cost比較を同時に運用しない。抽象化済みのprovider境界で将来再評価する。

## Consequences

### Positive

- Google有効時は交通手段とrequest時刻を使う実経路目安を利用できる
- Google障害またはkey未設定でも基本推薦を継続できる
- 任意地点検索を追加しつつ、application内の位置dataを最小化できる
- `travelEstimateKind` で利用者とtestがprovider種別を判別できる

### Negative

- 正確な起点座標または検索文字列がGoogleへ送信される
- credential、billing、quota、利用規約、provider側retentionが運用対象になる
- fallbackは実経路を反映せず、provider障害を利用者が注意文から判断する必要がある
- process内制限はinstance間・送信元間で共有されず、Google側quotaとingress制限が別途必要になる
- Geocodingにはserver-side fallbackがなく、任意地点検索だけは停止する

## Verification

### Local verification

- Google形式のstub responseをfacility IDごとの距離・時間へ変換できること
- out-of-order matrix elementをindexで正しく対応付けること
- 4 transport modeの即時requestで `departureTime` を送信しないこと
- Google HTTP failure時に全候補をstraight-lineへfallbackすること
- `travelEstimateKind` と注意文が実際に使ったproviderを示すこと
- location queryと起点座標がaccess log、store、responseへ含まれないこと
- key未設定時に推薦が動作し、地点検索が503を返すこと
- location searchがJSON bodyのPOSTだけを受け、rate limit後はproviderを呼ばないこと

### External verification

- [未検証] 制限済みkeyでRoutes APIとGeocoding APIへ実通信できること
- [未検証] 4 transport mode、5府県の代表起点、timeout、quota errorで期待どおり動くこと
- [未検証] Google usage・billing alertとkey restrictionが有効であること
- [未検証] productionからkey削除後にstraight-lineへrollbackできること

## Revisit Conditions

- fallback率またはGoogle error率がrelease目標を超える
- 実経路の精度が行き先決定を改善しない
- cost、quota、利用規約、privacyがMVPの許容範囲を超える
- providerごとのpartial resultを使う必要が生じる
- Google以外のproviderが品質・cost・privacyで明確に優位になる

## References

- [Google Routes API: Get a route matrix](https://developers.google.com/maps/documentation/routes/compute_route_matrix)
- [Google Geocoding API v3: Geocoding requests](https://developers.google.com/maps/documentation/geocoding/guides-v3/requests-geocoding)
- [Google Maps Platform security guidance](https://developers.google.com/maps/api-security-best-practices)
