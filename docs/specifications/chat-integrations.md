# Chat Integrations Specification

- Status: Current
- Related requirements: R-002–R-005、R-008、R-017–R-020
- Related decisions: ADR-0015、ADR-0016

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

- Slash Command retryはHMAC化source event keyで重複処理を防ぎます。
- 保存するのはsource key、処理状態、必要なfacility ID等の最小情報だけです。
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
