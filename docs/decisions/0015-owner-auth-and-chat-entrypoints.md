# ADR-0015 GitHub owner認証とSlack・Discord推薦入口

> Slackの入力・推薦応答フローは[ADR-0016](0016-slack-guided-recommendation.md)で更新した。固定起点と`response_url`だけを使うSlack設計は廃止し、Discordの既定値設計は維持する。

- Status: Accepted
- Date: 2026-08-01
- Type: Security
- Related Issues: Incomplete — repository historyから対応Issueを特定できていない
- Related Pull Requests: [#304](https://github.com/kohei321dev/spot-diggz/pull/304)
- Affected Docs: `requirements.md`, `architecture.md`, `security.md`, `specifications/chat-integrations.md`
- Supersedes: None
- Superseded By: [ADR-0016](0016-slack-guided-recommendation.md)（Slack入力・response flowのみ）
- Related: [ADR-0007](0007-go-modular-monolith-runtime.md)
- Related: [ADR-0009](0009-session-recommendation-ui.md)
- Related: [Security](../security.md)

## Context

spot-diggzのProduction URLとHTTP APIはapplication認証を持たず、URLへ到達できる利用者が施設参照、地点検索、推薦、訂正報告を実行できる。個人利用MVPとしては、公開URLを維持しながら、saydeckと同様に許可したGitHub ownerだけがWeb UIとAPIを利用できる状態へ変更したい。

同じownerはSlackとDiscordからも推薦を要求したい。Slack Freeの90日制限は過去のmessage/file historyの表示・検索・retentionに関する制限であり、新しいslash commandをapplicationが受信して応答するflowとは分けて扱う。spot-diggzはSlackまたはDiscordの過去messageを推薦入力、認証、状態復元に利用しない。

BrowserはGitHub OAuthを利用できるが、Slack/Discord webhookの呼出元userをGitHub OAuth cookieで認証することはできない。各platformのrequest署名を検証し、GitHub ownerと対応付けた不変のplatform user IDを認可する必要がある。

## Decision

1. Web UIと`/api/*`はGitHub OAuth web application flowで認証する。GitHubから取得した`login`が`GITHUB_OWNER`と一致するuserだけをownerとして許可する。
2. GitHub認証要求にはrandom `state`とPKCE S256を使用する。callback後はGitHub access tokenを保存せず、owner判定後に12時間有効なHMAC-SHA256署名済みHttpOnly session cookieを発行する。
3. OAuthではpublic profileによる本人識別だけを行い、repositoryやemailのscopeを要求しない。
4. `APP_ENV=production`では`APP_BASE_URL`、`AUTH_SECRET`、`GITHUB_CLIENT_ID`、`GITHUB_CLIENT_SECRET`が揃わない場合に起動を拒否する。`DEV_AUTH_BYPASS=1`はproductionでは常に無効とし、local UI/E2E確認だけに使う。
5. `/healthz`と`/readyz`はplatform liveness/readiness用にapplication認証なしで維持する。`/metrics`も既存方針どおりapplication認証を付けず、production network/ingressで制限する。
6. Slackは`POST /integrations/slack/commands`で`/spotdiggz`を受ける。raw body、`X-Slack-Request-Timestamp`、`X-Slack-Signature`をSigning Secretで検証し、5分を超えるrequestをreplayとして拒否する。`team_id`と`user_id`が設定済みowner mappingと一致する場合だけ推薦する。
7. Slackには3秒以内にephemeralな受付responseを返し、推薦結果はallowlistしたSlack `response_url`へephemeral responseとして送る。applicationはSlack history APIを呼ばず、command textやresponseを永続化しない。
8. Discordは`POST /integrations/discord/interactions`で`/spotdiggz`を受ける。raw bodyとtimestampをDiscord application public keyのEd25519署名で検証し、5分を超えるrequestをreplayとして拒否する。`application_id`、`guild_id`、user IDが設定済みowner mappingと一致する場合だけ推薦する。
9. Discordには3秒以内にephemeral deferred responseを返し、interaction tokenの有効時間内にoriginal responseを推薦結果で更新する。message history、bot token、Gateway接続はMVPに追加しない。
10. Slack/Discordはbrowser geolocationを利用できないため、server-side環境変数で設定した代表起点と既定の目的、気分、level、利用時間、交通手段を使う。代表起点には自宅等の正確な個人位置を設定せず、駅等の公開地点を使い、値をlog、response、metricsへ出力しない。
11. platform user IDの対応付けはsingle-owner MVPではsecret storeの環境変数で管理する。複数user、self-service linking、権限変更監査が必要になった時点で、GitHub identityを正本とするidentity tableとone-time linking flowへ移行する。
12. 推薦は既存の決定論的`recommendation.Engine`を再利用し、Slack/Discord専用の推薦ruleやAI生成を追加しない。

## Alternatives

### Slack・Discordでも毎回GitHub OAuthを要求する

platform commandからbrowserを開いてone-time codeを戻すlinking flowが必要になり、single-owner MVPには状態管理と失敗経路が過剰になる。複数user対応時の移行案として残す。

### 共通API keyをcommand textまたはheaderで渡す

Slack/Discord UIでは利用者がsecretを扱う必要があり、message historyやclient logへ漏洩しやすい。platform署名とuser ID認可より弱いため採用しない。

### Slack message event・Discord Gatewayで自然文を常時監視する

履歴、channel scope、bot permission、重複event、再接続、privacy boundaryが増える。明示的なslash commandだけでJobを満たせるため採用しない。

### queueで非同期処理する

耐久性は上がるが、現時点はsingle-owner、小規模catalog、短時間の決定論的推薦である。response失敗率やtimeoutを計測し、process内非同期処理で不足した場合に再評価する。

## Consequences

### Positive

- ProductionのWeb UIとAPIをGitHub ownerだけに制限できる。
- Slack/Discordともplatformが署名した現在のrequestと不変user IDで認可できる。
- Slack Freeのhistory期間に依存せず、新しいcommandへ毎回推薦を返せる。
- Web、Slack、Discordで同じcatalog、鮮度、営業時間、推薦ruleを再利用できる。
- OAuth token、message本文、検索位置を永続化しない。

### Negative

- GitHub OAuth App、Slack App、Discord Applicationの作成とProduction secret設定は手動作業になる。
- Slack user/team ID、Discord user/guild/application IDを初回に安全に確認して設定する必要がある。
- platform accountとGitHub accountの対応は環境変数による運用判断であり、self-serviceの暗号学的linkingではない。
- processが受付response後に停止した場合、Slack/Discordの遅延responseが失われる可能性がある。実測で必要性が出た場合はdurable queueを導入する。
- chat入口は固定の代表起点・既定条件を使うため、browser版の全条件入力より柔軟性が低い。

## Verification

- Productionで認証設定欠落時に起動を拒否する。
- GitHub OAuthのstate、PKCE、owner一致、非owner拒否、cookie改ざん拒否を自動testする。
- 未認証の`/api/*`は401、認証未設定のdevelopmentは503、Web UIは認証開始または設定案内へredirectする。
- Slack署名、5分replay window、team/user allowlist、`response_url` host allowlist、ephemeral delayed responseをtestする。
- Discord Ed25519署名、PING、application/guild/user allowlist、ephemeral deferred responseとoriginal response更新をtestする。
- access log、error log、metricsにOAuth token、cookie、Signing Secret、platform interaction token、message body、owner platform user ID、代表起点座標を出力しない。
- Production正式domainでGitHub owner login、非owner拒否、Slack command、Discord commandを実資格情報でsmoke testする。

## References

- [GitHub OAuth web application flow](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps)
- [Slack request signature verification](https://docs.slack.dev/authentication/verifying-requests-from-slack)
- [Slack slash commands](https://docs.slack.dev/interactivity/implementing-slash-commands/)
- [Discord interactions: receiving and responding](https://docs.discord.com/developers/interactions/receiving-and-responding)
