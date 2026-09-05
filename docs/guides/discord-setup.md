# Discord初回インストール・セットアップ手順

- Status: Initial setup guide
- Date: 2026-08-01
- Scope: single-guild Discord application、`/spotdiggz`、Production `https://spotdiggz.vercel.app`

## 1. 完了条件

次をすべて満たした状態を完了とする。

- 許可したDiscord serverへ`spot-diggz` applicationがGuild Installされている
- `/spotdiggz` guild commandが登録されている
- interactionが`https://spotdiggz.vercel.app/integrations/discord/interactions`へ送信される
- Ed25519署名、application ID、guild ID、owner user IDをspot-diggzが検証する
- ownerのcommandにはephemeralなdeferred responseの後、推薦結果が返る
- owner以外には利用拒否がephemeralで返る
- Gateway、Message Content Intent、message historyを使用しない

この連携はGitHub accountとDiscord accountを自動的に結び付けない。`GITHUB_OWNER`の本人に対応するDiscord user IDを、運用者が`DISCORD_OWNER_USER_ID`へ設定する。

## 2. 前提

- Discord serverへapplicationを追加できる`Manage Server`相当の権限がある
- Production URLの`/healthz`と`/readyz`が200を返す
- Vercel Project `spotdiggz`のProduction環境変数を設定し、再deployできる
- GitHub owner認証用のProduction環境変数は設定済みである
- chatの既定起点として、駅など公開された代表地点の緯度・経度を決めている
- PowerShell 5.1以降が使える

DiscordはInteractions Endpoint URLの保存時に署名付き`PING`を送る。先にDiscord環境変数を設定してProductionを再deployしてからendpointを登録する。

## 3. Discord applicationを作成する

1. [Discord Developer Portal](https://discord.com/developers/applications)を開く。
2. `New Application`を選ぶ。
3. Nameへ`spot-diggz`を入力して作成する。
4. `General Information`で次を控える。
   - `Application ID` → `DISCORD_APPLICATION_ID`
   - `Public Key` → `DISCORD_PUBLIC_KEY`

Public Keyは秘密鍵ではないが、設定値の改ざんを防ぐためProduction環境変数として管理する。

## 4. Guild IDとowner user IDを確認する

1. Discord clientの`User Settings`、`Advanced`で`Developer Mode`を有効にする。
2. 対象server iconを右クリックし、`Copy Server ID`を選ぶ。これが`DISCORD_GUILD_ID`である。
3. owner本人を右クリックし、`Copy User ID`を選ぶ。これが`DISCORD_OWNER_USER_ID`である。

username、display name、emailは認可に使用しない。

## 5. Guild Installを設定する

Developer Portalの対象applicationで`Installation`を開き、次を設定する。

1. Installation Contextsは`Guild Install`を有効にする。
2. `User Install`は無効にする。spot-diggzはDMまたはuser-installed interactionを許可しない。
3. Install Linkは`Discord Provided Link`を選ぶ。
4. Default Install SettingsのGuild Installへ`applications.commands` scopeだけを追加する。
5. 生成されたInstall Linkを開き、対象serverへ追加する。

spot-diggzはHTTP interaction responseだけを使うため、通常運用ではbot権限、Gateway intents、Message Content Intentを要求しない。

## 6. Production環境変数を設定する

Vercel Dashboardで`spotdiggz` Projectの`Settings`、`Environment Variables`を開き、Productionへ次を一度に設定する。

```dotenv
DISCORD_PUBLIC_KEY=<General InformationのPublic Key>
DISCORD_APPLICATION_ID=<Application ID>
DISCORD_GUILD_ID=<対象server ID>
DISCORD_OWNER_USER_ID=<owner本人のuser ID>

CHAT_DEFAULT_ORIGIN_LATITUDE=<公開代表地点の緯度>
CHAT_DEFAULT_ORIGIN_LONGITUDE=<公開代表地点の経度>
CHAT_DEFAULT_PURPOSE=basics
CHAT_DEFAULT_MOOD=focused
CHAT_DEFAULT_LEVEL=beginner
CHAT_DEFAULT_AVAILABLE_MINUTES=120
CHAT_DEFAULT_TRANSPORT=public_transit
```

注意:

- Bot TokenをVercel、`.env`、Git、GitHub Actionsへ保存しない
- chat起点に自宅、職場の入口、正確な現在地を使わない
- Discord設定4項目の一部だけを設定しない。部分設定ではapplicationが起動を拒否する
- SlackとDiscordを同時に使う場合、`CHAT_DEFAULT_*`は両platformで共通になる

設定後、Productionを再deployする。deploy logに`discord_integration_enabled`があり、`chat_integration_initialization_failed`がないことを確認する。secret、interaction token、platform IDをlogへ出力しない。

## 7. Interactions Endpoint URLを登録する

1. Developer Portalの`General Information`を開く。
2. `Interactions Endpoint URL`へ次を入力する。

```text
https://spotdiggz.vercel.app/integrations/discord/interactions
```

3. `Save Changes`を実行する。
4. Discordのendpoint検証が成功することを確認する。

保存時にDiscordは署名付き`PING`を送り、spot-diggzは署名と5分以内のtimestampを検証して`PONG`を返す。custom domainへ移行した場合はoriginだけを置き換え、pathは変更しない。

## 8. `/spotdiggz` guild commandを登録する

Discordのapplication commandはDeveloper Portal画面ではなくHTTP APIで登録する。MVPでは反映が即時のguild commandを使う。

### 8.1 Setup用Bot Tokenを取得する

1. Developer Portalの`Bot`を開く。
2. `Reset Token`を選び、setup用Bot Tokenを取得する。
3. tokenを画面、shell履歴、Git、Vercelへ貼らない。

Bot Tokenはcommand登録APIの認証にだけ使い、spot-diggz runtimeでは使用しない。

### 8.2 Repositoryの登録scriptを実行する

PowerShellでrepository rootから次を実行する。tokenはsecure promptへ貼り付けるためcommand historyへ残らない。

```powershell
powershell -File .\scripts\register-discord-command.ps1 `
  -ApplicationId '<DISCORD_APPLICATION_ID>' `
  -GuildId '<DISCORD_GUILD_ID>'
```

scriptは次のendpointへ`POST`し、同名commandがある場合はupsertする。

```text
https://discord.com/api/v10/applications/<application-id>/guilds/<guild-id>/commands
```

成功時はcommand名だけを表示する。Bot Tokenや各種IDは出力しない。今後commandを更新しない場合はDeveloper PortalでBot Tokenを再生成してsetup時のtokenを失効させ、新しいtokenを保存せず破棄できる。

## 9. Discord側の表示権限を絞る

application codeがowner以外を必ず拒否するため、この設定は補助的なUI制御である。
この設定は次節のnon-owner拒否testを完了した後に適用する。

1. Discord serverの`Server Settings`、`Integrations`を開く。
2. `spot-diggz`の`Manage`を開く。
3. 必要に応じてcommandを表示するuser、role、channelをowner用に制限する。

server AdministratorはDiscord側permissionを上書きできる場合があるが、spot-diggz側の`DISCORD_OWNER_USER_ID`検証は省略されない。

## 10. 疎通確認

1. owner本人が対象serverで`/spotdiggz`を実行する。
2. 3秒以内にephemeralなdeferred responseが開始されることを確認する。
3. 続いて最大3件の推薦、移動・滑走目安、公式情報URLが本人だけに表示されることを確認する。
4. owner以外のuserで同じcommandを実行し、`このアカウントではspot-diggzを利用できません。`が本人だけに表示されることを確認する。
5. application access logにrequest body、application/guild/user ID、interaction token、推薦本文がないことを確認する。

spot-diggzはDiscordの過去messageを取得、保存、状態復元に使用しない。

## 11. トラブルシューティング

| Symptom | Check |
| --- | --- |
| Interactions Endpoint URLを保存できない | Productionが再deploy済みか、Public KeyとURLが正しいか、`PING`へ200/PONGを返しているか確認する |
| `/spotdiggz`が候補に出ない | Guild Install先、guild command登録先のID、`applications.commands` scopeを確認する |
| `This interaction failed` | 3秒以内のdeferred response、Vercel起動状態、署名timestampとserver時刻を確認する |
| HTTP 401 | Public Key、署名headers、5分以内のtimestamp、別applicationのendpointでないことを確認する |
| ownerなのに拒否される | `DISCORD_APPLICATION_ID`、`DISCORD_GUILD_ID`、`DISCORD_OWNER_USER_ID`を不変IDで設定したか確認する |
| 受付後に結果が来ない | Google/provider timeout、Vercel process停止、Discord webhook更新失敗を確認する。interaction tokenをlogへ追加しない |

Discordはendpointへ無効な署名を使う定期的なsecurity checkを行う。無効署名に401を返せなくなるとendpointが削除される可能性があるため、署名検証を無効化しない。

## 12. 無効化・credential漏洩時

通常の無効化:

1. Discord serverの`Server Settings`、`Integrations`からapplicationを削除する。
2. Developer PortalのInteractions Endpoint URLを削除する。
3. Vercel Productionから`DISCORD_PUBLIC_KEY`、`DISCORD_APPLICATION_ID`、`DISCORD_GUILD_ID`、`DISCORD_OWNER_USER_ID`を削除する。
4. 再deployし、`discord_integration_enabled`が出ないことを確認する。

Bot Tokenが漏洩した場合はDeveloper Portalの`Bot`で即時`Reset Token`を行う。spot-diggz runtimeはBot Tokenを使用しないため、Production環境変数の追加は不要である。

## 13. 公式資料

- [Building your first Discord app](https://docs.discord.com/developers/quick-start/getting-started)
- [Interactions overview](https://docs.discord.com/developers/interactions/overview)
- [Receiving and responding to interactions](https://docs.discord.com/developers/interactions/receiving-and-responding)
- [Application commands](https://docs.discord.com/developers/interactions/application-commands)
- [Where can I find my User/Server ID?](https://support.discord.com/hc/en-us/articles/206346498-Where-can-I-find-my-User-Server-Message-ID)
