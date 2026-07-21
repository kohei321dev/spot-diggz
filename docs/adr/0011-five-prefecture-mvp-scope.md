# ADR-0011 MVPの地理scopeを5府県へ拡大する

- Status: Accepted
- Date: 2026-07-20
- Supersedes: [ADR-0002](0002-facility-data-source-and-freshness.md)の大阪都市圏scope
- Supersedes: [ADR-0009](0009-session-recommendation-ui.md)の大阪市・堺市代表地点scope
- Related: [Product Baseline](../product_baseline.md)

## Context

初期Product Baselineは大阪市・堺市を利用者の起点とし、大阪都市圏の20〜30施設を候補にしていた。MVP release readinessでは、同じ意思決定flowを府県境をまたぐ利用者にも検証し、地理scopeをUI、catalog schema、運用責任のすべてで一致させる必要がある。

任意地点検索を日本国内全体へ開いても、検証済みcatalogと更新責任が全国対応になるわけではない。検索できる起点と、推薦できる施設scopeを分けて定義する。

## Decision

1. MVPの施設catalogと利用者検証の地理scopeを、大阪府、兵庫県、和歌山県、奈良県、徳島県の5府県とする。
2. 代表起点は大阪・なんば・堺・なかもず・神戸・姫路・和歌山・奈良・徳島の9駅とする。
3. Google Geocodingで5府県外の日本国内起点を検索できても、推薦候補は5府県の検証済み施設に限定する。
4. catalog recordへ `prefecture` と `municipality` を必須化する。MVP公開dataの `prefecture` は5府県のallowlistに含める。
5. 5府県の合計20〜30施設を初期運用目標とする。府県ごとの件数を均等化することより、公式source、初心者適性、移動可能性、鮮度を優先する。
6. 候補発見と公開を分け、`status=verified`、source、日英必須属性、鮮度時刻を満たす施設だけを公開する。
7. 移動可能性は行政境界ではなく、起点、時刻、交通手段、利用可能時間からproviderで判定する。
8. 全国展開と、5府県外の施設追加は本ADRのscope外とする。追加時は更新工数と需要証拠を再評価する。
9. scope縮小が必要な場合はcatalog recordと代表起点を減らす。API schemaと推薦ruleを変えずにrollbackできる状態を保つ。

## Alternatives

### 大阪市・堺市だけで検証を続ける

運用負荷は最小だが、府県境をまたぐ移動、長距離route、地域ごとの施設情報差を検証できない。

### 近畿地方全体または全国へ拡大する

利用者候補は増えるが、source確認、英訳、30日鮮度、訂正対応の運用負荷が需要検証前に過大になる。

### 任意地点検索結果に応じて施設をruntime収集する

地理的な網羅性は高まるが、未検証施設を推薦へ混ぜ、出典・鮮度方針に反するため採用しない。

## Consequences

### Positive

- 代表起点と公開施設scopeを明示できる
- 府県境をまたぐ実経路providerとfallbackを検証できる
- `prefecture` / `municipality` でcatalog品質とcoverageを確認できる
- 全国対応を暗黙に約束せず、更新責任を限定できる

### Negative

- source確認、日英translation、30日鮮度、訂正reviewの対象が増える
- 20〜30施設では府県ごとのcoverageに偏りが生じる
- 徳島を含む長距離移動では、straight-line fallbackの誤差が大きくなる可能性がある
- 大阪都市圏向けの既存discovery文書だけでは5府県全体の候補台帳を表せない

## Verification

- 公開catalogの全recordが `prefecture` と `municipality` を持つこと
- 公開catalogの `prefecture` が5府県allowlistに含まれること
- release対象catalogに5府県すべての検証済み施設が含まれること
- 9つの代表起点を日本語・英語で選択できること
- 5府県の代表起点から推薦が決定論的に動作すること
- 府県別の件数とstale件数をrelease reviewで確認すること
- 施設確認・更新工数が合計週2時間以内か計測すること

## Revisit Conditions

- 5府県のいずれかで需要または検証済み施設を確保できない
- 更新工数が4週間継続して週2時間を超える
- 推薦利用の大半が一部府県に集中し、他府県の維持価値がない
- 5府県外から継続需要と運用ownerを確保できる
- provider costまたはfallback精度が長距離scopeに適さない
