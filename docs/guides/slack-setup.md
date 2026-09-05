# Slack初回インストール・セットアップ手順

- Status: Initial setup guide
- Date: 2026-08-03
- Scope: single-workspace Slack App、`/spotdiggz`、Production `https://spotdiggz.vercel.app`

## 結論

Spot-DiggzはSlack CLIでdeployするSlack-hosted Appではない。既存のGo/Vercel applicationをSlackの外部HTTP endpointとして使うため、Slack CLIのaccount認証は必須ではない。Slack Appはrepositoryの`slack-manifest.json`をApp Manifest画面へ読み込んで作成・更新する。

本人がSlack画面で行う操作は、Appの作成または選択、workspaceへのinstall、Signing SecretとBot User OAuth Tokenの取得、owner member IDの確認、scope変更後のreinstallである。秘密値はchat、Git、Issue、command引数、shell historyへ記録しない。

## 利用フロー

```text
owner
  -> /spotdiggz
  -> Spot-DiggzがSlack署名、workspace ID、user IDを検証
  -> 条件入力モーダル（出発地、時間、交通手段、レベル、目的、気分）
  -> modal submissionを即時ack
  -> backgroundで地点検索と決定論的推薦
  -> ephemeralな候補（最大3件）と「公式情報」「ここに行く」
  -> 候補は保存しない
```

Slackの過去message、DM、channel historyは取得しない。入力した地点文字列、解決した座標、候補施設もDB、application log、metricsへ保存しない。retry重複処理を防ぐsource event keyと処理状態だけを最大1時間保持するため、Slack Freeのmessage保持期間はこのフローに影響しない。

## 1. ローカル事前準備

PowerShellでrepository rootから実行する。

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\install-integration-clis.ps1
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\check-integration-cli-prerequisites.ps1
```

Vercel CLIはProduction secret設定とdeployに使う。Slack CLIはManifestのlocal validationや、すでに認証済みの既存App更新にだけ任意で使える。HTTP連携のために`slack login`を完了する必要はない。

## 2. Slack AppをManifestから作成または更新する

1. [Slack API: Your Apps](https://api.slack.com/apps)を開く。
2. 新規の場合は`Create New App`、`From an app manifest`を選ぶ。既存Appの場合は`App Manifest`を開く。
3. 対象workspaceを選び、`slack-manifest.json`の内容を読み込ませる。
4. 次が一致することを確認して保存する。

| 項目 | 値 |
| --- | --- |
| App名 | `spot-diggz` |
| Slash Command | `/spotdiggz` |
| Request URL | `https://spotdiggz.vercel.app/integrations/slack/commands` |
| Bot scopes | `commands`, `chat:write` |
| Interactivity | 有効。同じRequest URL |
| Event Subscriptions | 不要 |
| Socket Mode | 無効 |

5. `OAuth & Permissions`から`Install to Workspace`または`Reinstall to Workspace`を実行する。
6. scope、command、Interactivityを変更した場合も必ずreinstallする。
7. Slack clientを再読み込みする。

`configure-slack-app.ps1`はSlack CLI認証済みの既存AppにManifestを反映する任意の補助手段である。同名Appを自動作成させず、対象Appを確認してから実行する。

## 3. IDとcredentialを確認する

- `SLACK_BOT_TOKEN`: `OAuth & Permissions`のBot User OAuth Token（`xoxb-`で始まる）。
- `SLACK_SIGNING_SECRET`: `Basic Information`のApp Credentials。
- `SLACK_TEAM_ID`: workspace URLの`T...`部分。設定scriptはBot Tokenの`auth.test`結果とも照合する。
- `SLACK_OWNER_USER_ID`: owner本人のprofileから`Copy member ID`で得る不変ID。

表示名やメールアドレスを認可に使わない。これらの値は回答へ貼らず、次のsecure promptへ本人が直接入力する。

## 4. Vercel Production環境変数とmigration

まずVercelへlogin・project link済みであることを確認し、secret設定scriptを実行する。

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\configure-slack-vercel-env.ps1
```

scriptはBot TokenをSlack `auth.test`で検証し、workspace IDとの一致を確認した後、次をProductionのsensitive environment variableとして登録する。

- `SLACK_BOT_TOKEN`
- `SLACK_SIGNING_SECRET`
- `SLACK_TEAM_ID`
- `SLACK_OWNER_USER_ID`

Productionでは短期Slack request状態をNeonへ保存するため`DATABASE_URL`が必須である。migrationを適用する。
Slack modalの任意地点を解決するため、Productionには制限済みの`GOOGLE_MAPS_API_KEY`も必要である。このscriptは既存Google keyを変更しない。

```powershell
go run ./cmd/dbmigrate
```

このcommandは`0001`から`0003`までの管理済みmigrationを順番に適用する。`0003`は廃止した保存tableを削除し、Slack request状態を最小化する。接続文字列を出力しない。

環境変数とmigrationを反映した後にredeployする。secret設定とdeployを続けて行う場合は次でもよい。

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\configure-slack-vercel-env.ps1 -Deploy
```

## 5. 動作確認

### Endpointの存在

署名なしPOSTの期待結果は次のとおり。

- `404`: routeが未deploy、またはRequest URLが誤っている。
- `401`: routeと署名検証が動作している。署名なし診断として正常。
- `503`または起動失敗: Slack変数またはProductionの`DATABASE_URL`が不足している可能性がある。

### Owner E2E

1. 許可済みownerが`/spotdiggz`を実行する。
2. 条件モーダルが開く。
3. 出発地と条件を送信し、候補がephemeralで最大3件返る。
4. 候補に「公式情報」と「ここに行く」だけが表示され、保存操作がないことを確認する。
5. それぞれのリンクが正しい外部情報を開くことを確認する。
6. 別userまたは別workspaceから実行すると拒否される。

実資格情報によるこのE2Eはlocal自動testの代替ではなく、Production設定後に本人が行うsmoke testである。

## 6. トラブルシュート

| 症状 | 確認事項 |
| --- | --- |
| `/spotdiggz`が無効 | Slash Command保存後に`Reinstall to Workspace`し、Slackを再読み込みしたか |
| モーダルが開かない | `SLACK_BOT_TOKEN`、`chat:write`、App install、Production redeployを確認する |
| モーダル送信またはリンクbuttonが反応しない | Interactivityが有効で、Request URLがSlash Commandと同一か |
| ownerなのに拒否 | `SLACK_TEAM_ID`と`SLACK_OWNER_USER_ID`が不変IDで完全一致するか |
| 候補が返らない | Google Geocoding設定、catalog鮮度、Vercel logの安定したerror/event項目を確認する。request bodyや地点をlogへ追加しない |

受付後の処理は現在Go process内のbackground goroutineで実行する。source event keyと処理状態は最大1時間DBへ記録するが、地点や候補は記録しない。processが停止すると入力地点を復元できず、その検索は失敗または期限切れになる。実測で喪失が問題になる場合だけ、地点を暗号化して短期保持するdurable jobまたはqueueを別ADRで検討する。

## 7. 無効化とcredential漏洩時

1. Slack Appをworkspaceからuninstallする。
2. Vercel Productionから4つの`SLACK_*`変数を削除する。
3. redeployし、Slack integrationが初期化されないことを確認する。

Bot TokenまたはSigning Secretが漏洩した場合はSlack側でrotate/revokeし、Vercel secretを更新してredeployする。漏洩値をticket、chat、logへ貼らない。

## Codexへ渡す依頼文

```text
docs/guides/slack-setup.mdを最後まで読み、ローカル検査、migration、Vercel環境変数設定、redeploy、Production smokeを順に進めてください。

Slack Appの作成・install・reinstall、credential取得はSlack画面で本人が行います。Bot Token、Signing Secret、workspace ID、member IDはchatやcommand引数へ出さず、scripts/configure-slack-vercel-env.ps1のsecure promptへ私が直接入力します。Slack CLI認証を必須扱いしないでください。

外部状態を変更する直前に対象App、workspace、Vercel project、Production環境を表示して確認し、同名Appの新規作成やPreviewへの誤設定を避けてください。未実行または失敗したE2Eを成功扱いしないでください。
```

## 公式資料

- [App Manifests](https://docs.slack.dev/app-management/app-manifest/)
- [Installing with OAuth](https://docs.slack.dev/authentication/installing-with-oauth/)
- [Verifying requests from Slack](https://docs.slack.dev/authentication/verifying-requests-from-slack/)
- [Implementing slash commands](https://docs.slack.dev/interactivity/implementing-slash-commands/)
- [Handling user interaction](https://docs.slack.dev/interactivity/handling-user-interaction/)
- [Modals](https://docs.slack.dev/surfaces/modals/)
