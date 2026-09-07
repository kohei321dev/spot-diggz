# Chat Integrations Specification

- Status: Current
- Related requirements: R-002–R-005、R-008、R-017–R-020
- Related decisions: ADR-0015、ADR-0016、DR-0019、[DR-0020](../decisions/0020-mention-nearby-search.md)

## 新MVPの採用済み方針

[API契約](api-mvp.md)に従い、Slack/Discord → 自分のBotサーバー → 認証付きSpotDiggz APIへ移行します。独自Web UIは使いません。位置・条件の送信後に「検索中」を表示し、完了後に結果、失敗時にもエラーと次の行動をownerへ返します。API呼出し・platform署名/ID認可・非保存を別々に検証します。

入力の正本は[周辺検索仕様](nearby-search.md)です。Slack/Discord共通でメンション＋場所名を基本にし、genre/limit/sortと検索範囲を任意指定します。旧slash/modal・6条件必須・固定3件を新MVPの基本入力にしません。場所確認、非owner拒否、Bot返信によるループ防止を検証します。

Slack無料プランの通常Appという方針は維持します。platform別のメンション受信transport・必要scope/intent・owner限定返信、受付後処理・配送失敗・retry・冪等性は#312/#315/#316で詳細化します。API接続は[読み取りAPI](read-api.md)のclient別Bearer/JSON/構造化statusを使います。Bot側のowner本人確認は省略できず、資格情報の実登録は後続です。旧Discord HTTP Interaction専用という制約との適合も再評価し、別transportの採用や環境変更を今回承認したとはしません。既存署名・ephemeral方式の流用可否やgoroutineによる完了保証を検証なしで宣言しません。

以下の全節は移行前実装の説明です。「Discordの固定起点」「内部engine呼出し」「mention非対応」「最大3件」を新MVPの受入条件にしません。実装変更時にAPI契約・本書・setup guide・manifestを同じPRで更新します。メンションはまだ実行できる新機能ではありません。

## Shared boundary

- SlackとDiscordはWebのGitHub sessionを共有しません。
- Platform署名とimmutableなworkspace/application/guild/user IDを検証し、設定済みownerだけを認可します。
- 過去message、channel history、DM historyを読みません。
- RecommendationはWebと同じ決定論的engineを使用し、最大3件を返します。
- Responseはownerだけが見られるephemeral形式です。
- 候補、地点、座標、interaction token、message本文を恒久保存しません。
- 保存Lists、保存button、保存APIを提供しません。

## Slack `/spotdiggz`

1. Slackがform-encoded Slash Commandを`POST /integrations/slack/commands`へ送ります。
2. Applicationはraw body、`X-Slack-Signature`、timestampを使ってHMAC-SHA256を検証します。5分を超えるrequestを拒否します。
3. `team_id`と`user_id`をallowlistへ完全一致させます。
4. Slash Commandを即時ackし、条件入力modalを開きます。
5. 利用者は出発地、時間、交通手段、level、目的、気分を入力します。
6. Modal submissionを即時ackし、backgroundで地点解決と推薦を実行します。
7. 最大3件を`chat.postEphemeral`で返します。各候補はおすすめ理由、移動・滑走目安、公式情報、「ここに行く」を含みます。

Slack Appの最小scopeは`commands`と`chat:write`です。Socket Mode、message event、DM history、app mentionは現在のflowに使用しません。設定手順は[`../guides/slack-setup.md`](../guides/slack-setup.md)を参照します。

## Slack idempotency and retention

- Modal submissionのretryは、modalの`View.ID`から生成したHMAC化source event keyで重複処理を防ぎます。同じkeyのrequest状態がstoreにある間は、推薦処理を再実行しません。最初のSlash Commandのretryは、このstoreによる重複排除の対象ではありません。
- 保存するのは`request_id`、`source_event_key`、処理状態（`generating`、`delivered`、`failed`）、作成・更新・期限の時刻だけです。候補facility IDは保存しません。
- 状態保持は最大1時間です。
- 入力地点、座標、候補本文、Slack interaction token、Bot Token、request bodyはstoreまたはlogへ残しません。
- Background処理は12秒でtimeoutし、失敗時は利用者が再実行できるerrorを返します。

## Discord `/spotdiggz`

1. DiscordがJSON Interactionを`POST /integrations/discord/interactions`へ送ります。
2. Applicationはraw body、timestamp、Ed25519署名を検証し、5分を超えるrequestを拒否します。
3. `application_id`、`guild_id`、member/user IDを設定へ完全一致させます。
4. PINGへPONGを返します。Commandは3秒以内にephemeral deferred responseを返します。
5. Server-sideの公開代表起点と設定済み既定条件から推薦します。
6. Discord APIの固定originにあるinteraction webhookを使ってoriginal responseを更新します。mention展開は無効です。

Discordの代表起点には駅等の公開地点を使用し、自宅等の正確な個人位置を設定しません。設定手順は[`../guides/discord-setup.md`](../guides/discord-setup.md)を参照します。

## Authorization failures

- 署名不正、timestamp超過、ID欠落、allowlist不一致はfail closedです。
- 表示名、email、deprecated verification token、共通API keyだけで認可しません。
- 利用者向けresponseには内部設定値、credential、署名検証詳細を含めません。

## External verification status

- [確認済み運用] Production正式domainのSlack modal・候補応答は2026-08-03にownerが確認し、PR #304に記録されています。
- Status: Incomplete
- Missing evidence: Production Discord interactionのPING、owner command、non-owner拒否、log/metricsのprivacy確認。
- Required decision: ownerがDiscord App設定後にProduction E2Eを行い、結果をrelease記録と本書へ反映する。
