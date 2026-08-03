# GitHub owner認証・Vercel Productionセットアップ

- Status: Initial setup guide
- Date: 2026-08-03
- Scope: private MVP、単一GitHub owner、`https://spotdiggz.vercel.app`

## 結論

GitHub OAuth Appの登録とClient Secretの生成はGitHub画面で本人が行う。Vercel Production環境変数の登録、`AUTH_SECRET`生成、Production deploy、health/readiness確認はrepositoryのPowerShellスクリプトからVercel CLIで行う。

GitHub OAuth Appはcallback URLを1つだけ持つため、SayDeck用Appのcallbackを書き換えず、Spot-Diggz用に別のOAuth Appを作成する。

## 1. Spot-Diggz用GitHub OAuth Appを作成する

1. GitHubのprofile menuから`Settings`を開く。
2. `Developer settings`、`OAuth Apps`を開く。
3. `New OAuth App`または`Register a new application`を選ぶ。
4. 次を入力する。

| 項目 | 値 |
| --- | --- |
| Application name | `Spot-Diggz Production` |
| Homepage URL | `https://spotdiggz.vercel.app` |
| Application description | `Private owner access for Spot-Diggz`（任意） |
| Authorization callback URL | `https://spotdiggz.vercel.app/auth/github/callback` |
| Enable Device Flow | 無効 |

5. `Register application`を実行する。
6. Client IDを控える。
7. `Generate a new client secret`を実行し、表示されたClient Secretを安全な一時保管先へコピーする。

Client Secretはchat、Git、Issue、command引数、shell historyへ貼らない。GitHub OAuth AppのClient Secretは認可codeをaccess tokenへ交換するcredentialであり、server-side secretとして扱う。

GitHub公式手順: [Creating an OAuth app](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/creating-an-oauth-app)

## 2. CLI preflight

PowerShellでrepository rootから実行する。

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass `
  -File .\scripts\configure-github-vercel-env.ps1 `
  -PreflightOnly
```

次が表示されれば、Vercel認証、Project link、正式callbackのlocal確認が完了している。

```text
github-vercel-preflight=PASS project=spotdiggz callback=https://spotdiggz.vercel.app/auth/github/callback
```

このpreflightは環境変数変更やdeployを行わない。

## 3. Vercel Production環境変数を設定する

最初はdeployせず設定だけ行う。

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass `
  -File .\scripts\configure-github-vercel-env.ps1
```

promptへ次を入力する。

1. `GitHub OAuth Client ID`: GitHub OAuth App画面のClient ID。通常入力でありshell historyには残らない。
2. `GitHub OAuth Client Secret`: GitHub OAuth App画面で生成したsecret。非表示入力であり、貼り付けても画面には表示されない。

scriptは次をProductionへ登録する。

| 変数 | 値・扱い |
| --- | --- |
| `APP_BASE_URL` | `https://spotdiggz.vercel.app` |
| `GITHUB_OWNER` | `kohei321dev` |
| `GITHUB_CLIENT_ID` | 入力したClient ID |
| `GITHUB_CLIENT_SECRET` | Sensitive value |
| `AUTH_SECRET` | localで暗号学的乱数48 bytesから生成したSensitive value |

`AUTH_SECRET`は実行ごとに新しく生成されるため、scriptを再実行すると既存のowner sessionは失効する。Client Secretをrotateする場合は、新しいsecretをVercelへ設定してdeploy・login確認した後、GitHub側の古いsecretを削除する。

## 4. Production deploy

設定完了後、デプロイ前検証を通してから次を実行する。

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass `
  -File .\scripts\configure-github-vercel-env.ps1 `
  -Deploy
```

このcommandはcredentialを再入力し、5変数を更新してから、現在の作業ツリーを`vercel --prod --yes`でアップロードする。未コミット変更もupload対象になる。deploy後に`/healthz`と`/readyz`が200であることを確認する。

## 5. Owner認証E2E

1. `https://spotdiggz.vercel.app`をprivate windowで開く。
2. GitHubへredirectされることを確認する。
3. 設定済みowner GitHub accountで認可する。
4. Spot-Diggzへ戻り、Web UIと`/api/*`を利用できることを確認する。
5. sign out後に保護画面へ戻ることを確認する。
6. 許可owner以外では`/auth/denied`へ進み、Web UIとAPIを利用できないことを確認する。
7. Slackで`/spotdiggz`を実行し、modal、候補、公式情報・経路リンクを確認する。

Spot-Diggzはpublic profileだけを取得し、repository、email、organization scopeを要求しない。認証後もGitHub loginを`GITHUB_OWNER`と比較して認可する。

## 6. Rollbackとcredential漏洩時

新deploymentのhealth/readinessまたはowner loginが失敗した場合は、直前の正常なProduction deploymentへVercel CLIでrollbackする。Neon migrationは巻き戻さない。

Client Secretが漏洩した場合はGitHub OAuth App画面で新しいsecretを生成し、Vercelへ設定・deploy・login確認後に古いsecretを削除する。`AUTH_SECRET`が漏洩した場合はこのscriptを再実行してrotateし、全sessionを失効させる。

## 公式資料

- [Creating an OAuth app](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/creating-an-oauth-app)
- [Authorizing OAuth apps](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps)
- [Best practices for OAuth apps](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/best-practices-for-creating-an-oauth-app)
