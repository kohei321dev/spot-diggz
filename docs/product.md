# SpotDiggz Product

- Status: Current
- Last reviewed: 2026-09-05
- Product branch: `main`
- Baseline evidence: remote `main` `ed514394042d86833189637b433d21eac2fcd684`
- Migrated from: [`product_baseline.md`](product_baseline.md) under Issue #305

## Product statement

SpotDiggzは、気分・目的・skill・時間・検索位置・交通手段から、利用者が「今日の自分に合うskate session」を決め、実際に滑りに行けるようにする日本語・英語対応serviceです。

地図を提供すること自体を目的にせず、検証済み施設情報と説明可能な推薦で、候補を比較して行き先を決めることを支援します。

## Problem

利用者は施設の住所だけでなく、目的、level、営業時間、移動時間、休場、利用rule、情報の鮮度と根拠を同時に判断する必要があります。既存の地図・検索serviceだけでは、この判断を短時間でまとめて行うことが難しい点を解決します。

## Target users

### Primary

大阪府、兵庫県、和歌山県、奈良県、徳島県でskateboardを始めた初心者です。仕事・学校の後や限られた休日に、合法的かつ安心して練習できる施設を自力で判断しにくい利用者を対象とします。

private MVPではGitHub owner allowlistの1名だけへ利用を許可します。これは複数userへ公開する前のsecurity boundaryであり、市場上のtarget userを変更するものではありません。

### Secondary

- 5府県の復帰者
- 5府県を旅行中の訪日skater
- 施設の利用rule、交通、持ち物を日本語・英語で確認したい利用者

### Initial validation exclusions

- 地元の主要施設と利用条件を既に把握している上級者
- 利用可否が確認できないstreet spotを探す利用者
- 全国規模のuser投稿地図やskate SNSを求める利用者
- 保護者を一次利用者とする親子向け市場

## Value

- 条件を一度入力し、検証済み施設から最大3件へ候補を絞れます。
- 推薦理由、到着・滑走目安、公式情報、注意事項を比較できます。
- 必要な場合だけ施設詳細と動画を開き、外部navigationへ進めます。
- 情報源と検証時刻を確認し、誤りを任意で報告できます。

## Core experience

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

4週間の需要検証では次を判断材料にします。

- 対象利用者10人以上が実際の行き先決定に利用する
- 5人以上が4週間以内に2回以上利用する
- 3人以上が既存serviceだけでは得られなかった行動差を示す
- 5人以上が推薦結果から外部navigationを開く
- 3人以上が実際に到着して滑走できたと回答する
- 20〜30施設の確認・更新作業が週2時間以内に収まる
- 公開catalogのsource、日英必須属性、検証時刻保有率が100%である

navigation利用が5人未満、到着・滑走が3人未満、既存serviceとの差がない、更新が週2時間を超える、またはprovider費用・privacy・運用負荷が上限を超える場合は、追加実装を止めて仮説かscopeを見直します。

## Scope

- 大阪府、兵庫県、和歌山県、奈良県、徳島県の検証済み施設catalog
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

- iOS・Android native application
- 全国施設の一括登録
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
- Missing evidence: 5府県catalog更新を週2時間以内に維持できるか、実Google trafficの精度・費用・fallback率、厳密なcorrection purge、metrics基盤、英語利用者獲得、media確認工数の実測。
- Required decision: ownerが需要検証とProduction運用の証拠を集め、scopeまたは運用を継続・変更するか判断する。

## Related documents

- Requirements: [`requirements.md`](requirements.md)
- Specifications: [`specifications/README.md`](specifications/README.md)
- Architecture: [`architecture.md`](architecture.md)
- Security: [`security.md`](security.md)
- Decisions: [`decisions/README.md`](decisions/README.md)
- Research: [`research/README.md`](research/README.md)
