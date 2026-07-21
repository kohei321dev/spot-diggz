# spot-diggz Product Baseline

- Status: Active baseline（MVP release readiness）
- Date: 2026-07-21
- Product branch: `main`
- Related research: [市場・需要調査レポート](market_demand_research_2026-07.md)

## 1. Product Vision

気分・目的・スキル・時間・検索位置・交通手段から、利用者が「今日の自分に合うスケートセッション」を決め、実際に滑りに行けるようにする。

地図を提供すること自体を目的にせず、検証済み施設情報と説明可能な推薦で、スケートに行く前の意思決定を支援する。

## 2. Target Users

### Primary

大阪府、兵庫県、和歌山県、奈良県、徳島県でスケートボードを始めた初心者。仕事・学校の後や限られた休日に、合法的かつ安心して練習できる施設を自力で判断しにくい利用者を対象とする。

### Secondary

- 5府県の復帰者
- 5府県を旅行中の訪日スケーター
- 施設の利用ルール、交通、持ち物を日本語・英語で確認したい利用者

### Excluded from the initial validation

- 地元の主要施設と利用条件を既に把握している上級者
- 利用可否が確認できないストリートスポットを探す利用者
- 全国規模のユーザー投稿地図やスケートSNSを求める利用者
- 保護者を一次利用者とする親子向け市場

### Initial Market

MVPの地理scopeは、大阪府、兵庫県、和歌山県、奈良県、徳島県の5府県とする。公開catalogはこのscopeにある合法な公共・民間施設のうち、公式情報で必須属性を確認できた施設に限定する。

代表起点は大阪、神戸・姫路、和歌山、奈良、徳島の主要駅を用意する。Google Geocodingを有効にした環境では日本国内の任意の駅名・住所も検索できるが、推薦候補の地理scopeは5府県から拡大しない。

移動可否は施設の固定属性にせず、出発地点、出発時刻、交通手段、利用可能時間ごとに判定する。全国展開は初期検証と更新運用が成立した後に判断する。

## 3. User Problem

利用者は、施設の住所だけでなく、次の条件を同時に判断する必要がある。

- 今日の目的に合うか
- 自分のレベルで利用できるか
- 営業時間内に到着し、帰路を含めて滑走時間を確保できるか
- 現在地から現実的に移動できるか
- 臨時・定例の休場期間に該当しないか
- 利用可能な施設か、ルール違反にならないか
- 情報が十分に新しく、根拠を確認できるか

既存の地図・検索サービスで住所と経路は確認できるため、spot-diggzは「場所を表示すること」ではなく「候補を比較して行き先を決めること」を支援する。

## 4. Jobs To Be Done

> 仕事・学校の後に滑りたいと思ったとき、限られた時間と交通手段で、今から合法的かつ安心して練習できる施設を短時間で決めたい。

訪日利用者については、次のJobを追加する。

> 旅行中に滑れる場所を、利用ルールや地域のマナーを誤解せずに日本語・英語で選びたい。

## 5. Product Hypothesis

「今日の目的」「気分」「レベル」「利用可能時間」「検索位置」「交通手段」を入力すると、鮮度期限内の検証済み施設から理由付きで候補を提示するサービスは、単なる施設一覧よりも目的地決定と実際の訪問を促進する。

この仮説は未検証であり、実利用テストで判断する。

## 6. User Flow

```text
目的・気分・レベル・時間・検索位置・交通手段を入力
    ↓
鮮度超過・休場・時間外・移動超過・レベル不適合を除外
    ↓
目的・初心者適性・設備・移動条件・滑走可能時間で評価
    ↓
おすすめを最大3件、理由・情報源・注意事項付きで提示
    ↓
アイコンと短いラベルで詳細・現在地・外部導線を確認し、必要な場合だけ動画を表示
    ↓
外部ナビで実経路を確認して訪問
    ↓
情報の誤りを任意で報告
```

Google連携がない場合、またはGoogle Routesが失敗した場合は、交通手段別の固定速度を使う直線距離概算へ縮退する。結果は `travelEstimateKind` と注意文で実経路か概算かを区別する。

## 7. Requirements: What / Why

| ID | What（要求） | Why（理由） | MVP受入条件 |
| --- | --- | --- | --- |
| R-001 | 目的・気分・レベル・時間・検索位置・交通手段を入力できる | 推薦条件を明確にするため | 6項目を選択し、境界で列挙値・座標を検証できる |
| R-002 | 条件に合う施設を最大3件提示する | 候補過多を避けるため | hard conditionを満たす候補を安定順序で返す |
| R-003 | 各候補のおすすめ理由を表示する | 推薦結果を信頼して比較するため | 目的、設備、時間、移動の根拠を構造化して返す |
| R-004 | 情報源と検証時刻を表示する | 鮮度と信頼性を判断するため | 施設ごとにsource URLと3種の検証時刻を返す |
| R-005 | 外部ナビへ遷移できる | 独自ナビを作らず訪問につなげるため | 施設座標をGoogle Maps等で開ける |
| R-006 | 情報の誤りを報告できる | catalogを継続更新するため | 施設単位の報告をfile storeへ保存しreceiptを返す |
| R-007 | 日本語と英語で主要導線を利用できる | 訪日利用者を検証するため | UIと施設の主要事実に日英表示がある |
| R-008 | 正確な検索位置をapplicationで保存しない | 位置情報riskを抑えるため | 座標・検索文字列をstore、access log、responseへ残さない |
| R-009 | 情報鮮度と休場期間を推薦判定に使う | 古い・休場中の候補を避けるため | dynamic 30日、stable 180日、`one_time` / `annual` を判定する |
| R-010 | 外部route障害時も基本推薦を継続する | 単一provider障害を主要flowへ波及させないため | Google Routes失敗時にstraight-lineへ自動fallbackする |
| R-011 | 主要flowとcatalog品質を観測できる | release後の利用者影響を判断するため | request、推薦、allowlist event、catalog freshnessをmetrics化する |
| R-012 | 公開write endpointの濫用を制限する | 個人運用の可用性を守るため | request size、strict JSON、process内token bucketを適用する |
| R-013 | 主要操作と外部導線を、用途を表すアイコンと短いラベルで表示する | テキストリンクだけでは現在地取得、公式情報、経路、訂正報告の用途を素早く判別しにくいため | 現在地取得、外部ナビ、公式情報、訂正報告は一貫したアイコンと表示名を持つ。アイコンだけで状態・操作を伝えず、キーボード操作、アクセス可能な名前、44 CSS px以上の操作領域を満たす |
| R-014 | 施設ごとに手動キュレーション済みYouTube動画を最大1件、任意の補助情報として初期表示できる | 滑走環境を想像し、候補を比較しやすくするため | catalogで検証済みのYouTube動画だけを施設ごとに0または1件、YouTube privacy-enhanced playerで初期表示する。自動再生はせず、利用者は同じトグル操作で動画を閉じて再表示できる。埋込不可・削除・通信失敗時は推薦表示を継続し、通常のYouTube外部リンクへ縮退する |
| R-015 | 公式性を確認済みのSNSプロフィールへの外部リンクを表示できる | 施設が公開する追加情報への導線を、投稿の収集なしで提供するため | 初期対象はInstagramとXとし、施設ごとにplatformごと最大1件のHTTPSプロフィールURLだけを表示する。リンクは用途を表すブランドアイコンと表示名を持ち、投稿、ハッシュタグ、フィードをapplication内へ表示しない |
| R-016 | 動画・SNSの取得と表示を、手動確認可能な運用境界に限定する | 著作権、利用規約、第三者通信、情報鮮度のriskを管理するため | Google/Apple Maps、SKEPA等の第三者サイト、SNSから画像・動画・投稿を自動収集、保存、スクレイピング、再配信しない。任意URLをiframeへ渡さず、許可したYouTube動画IDから固定の埋込先を構成する。ownerがprovider規約、埋込可否、著作権・肖像権上の利用可否を確認し、確認日を記録する |

R-008は外部送信を禁止する要求ではない。`GOOGLE_MAPS_API_KEY` を設定した場合、推薦の起点座標・施設座標・交通手段はGoogle Routesへ、地点検索文字列はGoogle Geocodingへ送信される。MVPは即時検索だけを扱い、Routesの `departureTime` を省略してproviderのrequest時刻を使う。UIとprivacy文書でこの境界を明示する。

## 8. Facility Data Requirements

公開施設は次の属性を持つ。

- ID、施設名、住所、都道府県、市区町村、公開施設座標
- 利用可能な競技
- 通常営業時間と曜日別休業
- 一般利用状態（`regular`、`limited`、`schedule_check_required`）と営業時間の根拠（`official`、`conservative`）
- 一般利用が限定される場合の利用者向け注意事項
- `one_time` の `YYYY-MM-DD` 範囲と `annual` の `MM-DD` 範囲
- 料金、予約、利用登録の要否
- 初心者適性、セクション、路面、照明、屋根、屋内外
- ヘルメット、防具、年齢等の利用ルール
- 最寄り駅、駐車場等のアクセス情報
- `sourceUrl`、`sourceType`、`status=verified`、`confidence`
- `verifiedAt`、`dynamicVerifiedAt`、`stableVerifiedAt`
- 日本語のsource-backed情報と `englishTranslation`
- 任意の補助mediaとして、手動キュレーション済みYouTube動画0または1件。動画ID、通常のYouTube URL、選定日、確認日、選定理由を持つ
- 任意の公式SNSプロフィールとして、InstagramまたはXごとに0または1件。platform、HTTPS URL、公式性を確認した日を持つ

動的情報は営業時間、料金、予約、休場、利用ルール等とし、`dynamicVerifiedAt` から30日以内を推薦条件とする。安定情報は名称、住所、地域、設備、アクセス等とし、`stableVerifiedAt` から180日以内を推薦条件とする。両方が期限内の施設だけを推薦する。

期限超過はcatalog構造の破損ではないため、起動時loadと施設参照APIは許可する。推薦からは除外し、`spot_diggz_catalog_freshness` で `stale` として観測する。未来の検証時刻、欠落した日英必須属性、不正な休場形式、未検証statusは起動時に拒否する。

動画とSNSは施設の営業時間、利用可否、設備等の事実を裏付けるsourceではなく、推薦順位にも使わない補助情報とする。動画の技術的な埋込失敗時は通常の外部リンクへ縮退する。一方、規約違反、権利侵害のおそれ、公式性喪失、不適切コンテンツが判明した場合は、外部リンクを含むmedia record全体をcatalogから除外する。

2026-07-19調査基準の公開catalogは5府県31施設（大阪府24施設）である。初期は合計20〜30施設程度を運用目標とする。候補と公開確定を分け、全国施設の一括登録は初期検証後に判断する。`schedule_check_required` の施設は存在・所在地の参照対象だが、日付別の予定確認なしでは推薦しない。

## 9. Deterministic Recommendation and AI Boundary

MVPではAIを推薦処理へ使用しない。除外、順位付け、推薦理由は、検証済みcatalogと決定論的ruleだけで生成する。

### MVP後にAIを再評価できる領域

- 自然文を検索条件へ変換する
- 検証済み推薦理由を利用者向けに説明する
- 複数施設の比較を要約する
- source-backed情報の翻訳案を作る

### AIへ委ねない領域

- 営業時間、料金、利用可否の事実生成
- 未確認スポットの公開可否判断
- 出典、検証時刻、鮮度状態の改変
- hard conditionを上書きする施設選定
- 正確な位置情報の保存判断

## 10. MVP Scope and Release Boundary

### Included

- 5府県の検証済み施設catalog
- Go製モジュラーモノリスと単一OCI image
- スマートフォン対応Web UI
- ワンタップ推薦と詳細条件入力
- 最大3件の決定論的推薦
- dynamic 30日 / stable 180日の鮮度判定
- `one_time` / `annual` 休場判定
- 日本語・英語のUIと施設translation
- 情報源表示と外部ナビ
- アイコンと短いラベルによる主要操作・外部導線
- 手動キュレーション済みYouTube動画の初期表示と開閉トグル、公式確認済みInstagram・Xプロフィールへの外部リンク
- optionalなGoogle Routes / Google Geocoding
- straight-line fallback
- 90日保持metadata付き訂正報告file store
- allowlist product event、request ID、JSON access log、Prometheus metrics
- liveness、fresh施設が1件以上あることを確認するreadiness、rate limit、security header、smoke、rollback Runbook
- `CGO_ENABLED=0` の単一binaryと、CA bundle・非root書込directoryを持つscratch image

### Excluded

- iOS・Android native application
- 全国施設の一括登録
- 無審査のユーザー投稿と自動反映
- 利用可否不明のストリートスポット
- Instagram等からの自動収集
- Google/Apple Maps、SKEPA等の第三者サイトからの画像・動画の取得、保存、再配信
- SNS投稿、ハッシュタグ、フィードの収集・表示
- 利用者投稿の画像・動画、および任意URLのiframe埋込
- 独自地図・独自turn-by-turn navigation
- account、chat、follow、ranking、payment
- AIによる施設選定

### Implementation status

| State | Boundary |
| --- | --- |
| [事実] Local implementation | Go modules、HTTP API、UI、provider adapter、fallback、file/PostgreSQL store、migration、rate limit、metrics、unit/component test |
| [未検証] External integration | 実credentialを使うGoogle Routes / Geocoding、quota、課金、key restriction、provider側telemetry |
| [事実] Production operation | Vercel Project、既存Neon OrganizationのProject、migration、health/readiness、facility API、UI、correction API、desktop/mobile E2E |
| [未検証] Production operation | metrics network restriction、alert、Google quota、custom domain/DNS、GitHub main pushからの自動deploy、rollback exercise |

Google外部APIの有効化、metrics制限、custom domain、main pushからの自動deployは、資格情報・課金設定・platform権限を持つ人間の承認後に実施する。ローカルtestだけでなく、今回確認済みのProduction smokeをrelease根拠として記録する。

## 11. Success Metrics

4週間の需要検証で、機能数やpage viewより実際の行動を重視する。

- 対象利用者10人以上が実際の行き先決定に利用する
- 5人以上が4週間以内に2回以上利用する
- 3人以上が既存serviceだけでは得られなかった行動差を示す
- 5人以上が推薦結果から外部navigationを開く
- 3人以上が実際に到着して滑走できたと回答する
- 20〜30施設の確認・更新作業が週2時間以内に収まる
- 公開catalogのsource、日英必須属性、検証時刻保有率が100%である

次のいずれかが確認された場合は追加実装を止め、仮説またはscopeを見直す。

- navigationを開いた利用者が5人未満、または到着して滑走できた利用者が3人未満
- 利用者が最終的に知人、Instagram、Google Mapsだけで判断する
- 既存serviceと比べて行動上の差が出ない
- 施設情報の更新作業が週2時間を超える
- provider費用、privacy、運用負荷が継続可能な上限を超える

検証手順とevent定義は [Discovery Sprint 0検証計画](discovery/sprint-0-validation-plan.md) と [Observability設計](operations/observability.md) を参照する。

## 12. Risks and Assumptions

- 初心者・復帰者の課題が継続利用するほど強いとは限らない
- 5府県へ広げることで施設確認と英訳の運用負荷が増える
- 動的情報30日の更新が週2時間の上限に収まらない可能性がある
- 通常営業時間が新しくても、当日の貸切・天候・混雑は保証できない
- straight-line fallbackは実経路、待ち時間、渋滞を反映しない
- Google連携時は正確な起点座標または検索文字列が外部providerへ送信される
- process内rate limitは複数instance間で共有されず、強い濫用対策ではない
- YouTube動画とSNSプロフィールは第三者が管理する情報であり、現在の施設状態や安全性を保証しない。動画の選定、埋込可否、権利・規約確認を継続運用できるかは未検証である
- YouTube iframeは動画を持つ推薦結果の初期表示時に第三者通信を発生させる。providerの規約、telemetry、保持期間はowner確認が必要である
- productionのcorrection reportはNeon/PostgreSQLへ保存する。`DATABASE_URL`未設定のlocal/CIではfile storeへフォールバックするため、container置換をまたぐ永続性はproduction設定に依存する
- correctionの期限切れpurgeは起動時と1時間ごとのため、失敗logを監視しないと90日retentionを保証できない。Neonでは`delete_after`を条件に削除する
- metrics endpointはapplication認証を持たないため、ingressまたはnetworkで制限する必要がある

## 13. Open Questions

- 5府県20〜30施設の更新作業を週2時間以内に維持できるか
- 実Google trafficの精度、quota、費用、fallback率は許容範囲か
- `GOOGLE_MAPS_API_KEY` をAPI・送信元制限したproduction構成をどのplatformで運用するか
- Neonの`correction_reports`削除を定期job、アプリworker、store移行のどれで厳密に保証するか
- metrics収集、alert、dashboardをどの基盤で運用するか
- 英語利用者をどのchannelで獲得し、translation品質を誰が確認するか
- 動画の選定・確認・失効対応を、既存の20〜30施設の更新作業と合わせて週2時間以内に維持できるか
