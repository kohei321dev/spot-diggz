# SpotDiggz Product

- Status: Current
- Last reviewed: 2026-09-06
- Product branch: `main`
- Baseline evidence: remote `main` `0a8cd3ba38cd67654c2bde0c4e4d2fc0d75302f9`（実装）、2026-09-06のownerによるProduct定義の訂正（[DR-0018](decisions/0018-api-first-product-definition.md)）
- Migrated from: [`product_baseline.md`](product_baseline.md) under Issue #305

## Product statement

SpotDiggzは、スケボーをしたい人が、その日の目的や条件に合う行き先を決めるためのサービスです。対象地域を限定せず、UIやSlack・Discordのbot・appなどから呼び出せるAPIとして、施設情報と推薦機能を提供できるように整備していきます。

地図を提供すること自体を目的にせず、検証済み施設情報と説明可能な推薦で、候補を比較して行き先を決めることを支援します。

## Problem

利用者は施設の住所だけでなく、目的、level、営業時間、移動時間、休場、利用rule、情報の鮮度と根拠を同時に判断する必要があります。既存の地図・検索serviceだけでは、この判断を短時間でまとめて行うことが難しい点を解決します。

## Target users

ターゲットは「スケボーをしたい人」です。関西や特定の府県、初心者などに限定しません。経験レベル、目的、時間、検索位置、交通手段は、利用者を対象外にする区分ではなく、候補を選ぶための条件です。

private MVPではGitHub owner allowlistの1名だけへ利用を許可します。これは複数userへ公開する前のsecurity boundaryであり、市場上のtarget userを変更するものではありません。

現在の施設データの収録範囲と入力できる条件には実装上の制約があります。ターゲットを限定しないことは、全地域・全条件への対応済み、未検証spotの推薦、匿名公開を意味しません。

## Delivery model

- 施設情報と条件に合う推薦をAPIとして提供し、UI、Slack・Discordのbotやappなどが利用できるようにします。
- 各利用側は条件の入力と結果の表示・返信を担い、施設情報と推薦の判断を共有します。
- 特定のWeb画面やchatサービスだけをプロダクトそのものとは定義しません。
- これは提供方針です。独立した各clientがHTTP APIを呼ぶ構成の完成や、新しいAPI認証方式の採用を宣言するものではありません。実装前の未確定事項は[Requirements](requirements.md#product-direction-and-implementation-boundary)で管理します。

## Value

- 利用するUI・bot・appから条件を伝え、検証済み施設を根拠付きで比較して行き先を決められることを目指します。
- 施設情報と推薦の判断をAPIから利用できるようにし、入口ごとに別の施設情報や推薦ロジックを持たずに済むようにします。
- 現行実装は最大3件の推薦理由、到着・滑走目安、公式情報、注意事項、情報源と検証時刻を提供し、誤りの任意報告も受け付けます。

## Current implementation flow

以下は現在のコードにある入口です。Web向けHTTP APIはありますが、Slack・Discord adapterは内部の共通推薦serviceを呼びます。各bot・appが独立してAPIを利用する提供形態とは区別します。

```text
Web: GitHub owner認証 -> 条件入力またはワンクリック推薦
Slack: 署名・owner認可済みcommand -> modalで6項目を入力
Discord: 署名・owner認可済みcommand -> server-sideの公開代表起点・既定条件を使用
  -> 各入口から目的・気分・level・時間・検索位置・交通手段を推薦engineへ渡す
  -> 鮮度超過・休場・時間外・移動超過・level不適合を除外
  -> 目的・初心者適性・設備・移動条件・滑走可能時間を評価
  -> 最大3件を理由・情報源・注意事項付きで比較
  -> 必要な施設詳細と動画だけを開く
  -> 公式情報を確認し、外部navigationで訪問
```

6項目の利用者入力はWebの詳細条件フォームとSlack modalで提供します。Discord commandでは条件入力を受け付けず、設定済みの条件で推薦します。施設詳細と動画はWebで確認できます。

Google連携がない場合、またはGoogle Routesが失敗した場合、推薦は交通手段別の固定速度による直線距離概算へ縮退します。結果は実経路か概算かを明示します。

## Product hypothesis and success measures

条件入力から鮮度期限内の施設を理由付きで提示するserviceは、単なる施設一覧より目的地決定と実際の訪問を促進する、という仮説を検証します。この仮説は未検証です。

以下は従来の4週間の需要検証計画であり、達成済みの数値ではありません。APIを利用する入口も含めた検証方法・対象者募集・計測方法の再確認が必要です。地域や経験レベルをターゲットの制限として引き継ぎません。

- 対象利用者10人以上が実際の行き先決定に利用する
- 5人以上が4週間以内に2回以上利用する
- 3人以上が既存serviceだけでは得られなかった行動差を示す
- 5人以上が推薦結果から外部navigationを開く
- 3人以上が実際に到着して滑走できたと回答する
- 20〜30施設の確認・更新作業が週2時間以内に収まる
- 公開catalogのsource、日英必須属性、検証時刻保有率が100%である

navigation利用が5人未満、到着・滑走が3人未満、既存serviceとの差がない、更新が週2時間を超える、またはprovider費用・privacy・運用負荷が上限を超える場合は、追加実装を止めて仮説かscopeを見直します。

## Product scope

- スケボーの行き先を決めるための、検証済み施設情報と条件に基づく推薦。
- UIやSlack・Discordのbot・appなどから利用できるAPIの整備。
- 情報源、鮮度、注意事項を伝え、根拠不足の場合は推薦できないことを明示すること。

## Current implementation and data coverage

以下は既存実装の範囲です。地域や技術構成そのものをプロダクトのターゲットとしません。施設の追加には出典・鮮度・更新責任の確認と、現行schema等の制約を変更する別作業が必要です。

- 現在のcatalogと都道府県の許可値は大阪府、兵庫県、和歌山県、奈良県、徳島県の5府県
- Go製モジュラーモノリスと単一OCI image
- smartphone対応の日本語・英語Web UI
- ワンクリック推薦、詳細条件入力、最大3件の決定論的推薦
- dynamic 30日、stable 180日の鮮度判定と休場判定
- 情報源、外部navigation、訂正報告
- 施設詳細内の手動選定済みYouTube動画と、公式確認済みInstagram・X profileへの外部link
- optional Google Routes / Geocodingとstraight-line fallback
- GitHub owner認証、Slack modal推薦、Discord command入口
- request ID、構造化log、Prometheus metrics、health/readiness、rate limit、runbook

### Permanent staging environment

- Status: Not applicable
- Reason: 現在は単一ownerの小規模private MVPであり、通常変更はlocalとCI、外部連携等は必要期間だけVercel Preview、本番反映後はProduction smokeで検証する運用がREADMEに明記されています。
- Revisit when: 複数人の継続的検証、長期間の外部受入、Productionと異なるdata・credential境界が必要になったとき。

## Non-goals

- 自社製iOS・Android native applicationの新規実装（APIを利用するappを対象外とする意味ではない）
- 今回の定義訂正に伴う全国施設の一括登録や、未収録地域への対応済み宣言
- 無審査user投稿またはcatalogへの自動反映
- 利用可否不明のstreet spot
- 第三者siteやSNSからの画像・動画・投稿の収集、保存、scraping、再配信
- SNS投稿、hashtag、feedのapplication内表示
- user投稿mediaまたは任意URLのiframe埋込
- 独自地図、独自turn-by-turn navigation
- 複数user account、会話履歴、follow、ranking、payment、保存Lists
- AIによる施設選定

## Current state

- [公開状況] 従来の公開先は2026-09-06にHTTP 404を確認しました。現在の公開URLは未確認です。[公開先の確認状況](operations/service-status.md)を参照してください。以下のProduction確認は過去の記録であり、現在の稼働を示しません。
- [観測した実装] Go modules、HTTP API、Web UI、GitHub owner認証、Slack/Discord adapter、provider fallback、file/PostgreSQL store、migration、rate limit、metrics、unit/component/E2E testがあります。
- [確認済み運用] Vercel/Neon、migration、health/readiness、facility API、correction API、desktop/mobile E2Eは2026-07-20の記録があります。
- [確認済み運用] Production正式domainのGitHub owner loginとSlack `/spotdiggz`のmodal・候補応答は2026-08-03にownerが確認し、PR #304に記録されています。
- [未確認] Discord Production interaction、実Google traffic、Google制限・quota・課金・provider retention、metrics network restriction・dashboard・alert、custom domain/DNS、`main`からの自動Production deploy、rollback exerciseは確認証拠が不足しています。

## Constraints and risks

- 施設事実はverified catalogへgroundingし、AI出力を事実源にしません。
- 正確な検索位置はapplicationへ保存しません。Google有効時は起点座標または検索文字列がGoogleへ送信されます。
- 当日の貸切、天候、混雑、第三者mediaの現在性は保証しません。訪問前に公式情報を確認します。
- process内rate limitはinstance間で共有されないため、強いabuse対策にはなりません。
- correction retention、metrics公開制限、media確認を継続運用する必要があります。

## Open questions

- Status: Incomplete
- Missing evidence: 各UI・bot・app向けAPI契約と認証・認可方式、対応する条件・データの拡張順序、入口別の需要検証方法、catalogの更新工数、実Google trafficの精度・費用・fallback率、厳密なcorrection purge、metrics基盤、英語利用者獲得、media確認工数の実測。
- Required decision: ownerがAPI提供に必要な契約・安全境界・実装範囲を別Issueと必要なDecision Recordで決める。地域限定ではなく、需要と検証・更新できるデータに基づいて整備の順序を決める。

## Related documents

- Requirements: [`requirements.md`](requirements.md)
- Specifications: [`specifications/README.md`](specifications/README.md)
- Architecture: [`architecture.md`](architecture.md)
- Security: [`security.md`](security.md)
- Decisions: [`decisions/README.md`](decisions/README.md)
- Research: [`research/README.md`](research/README.md)
