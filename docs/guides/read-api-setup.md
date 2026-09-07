# 読み取りAPIのローカル起動と資格情報管理

- Status: Implemented source setup; external provisioning incomplete
- Specification: [読み取りAPI](../specifications/read-api.md)
- Decision: [DR-0021](../decisions/0021-read-api-contract.md)、[DR-0022](../decisions/0022-domestic-catalog-quality.md)

## 前提

Goは`go.mod`のversionを使う。新APIにはVercel CLI、GitHub OAuth App、DB、Slack CLIは不要。Botの接続・Cloud Run公開・Google設定は別作業。`.env.local`はGoが自動で読み込まない。secretは管理された環境注入または非表示入力を使い、chat、引数、shell履歴、Gitへ貼らない。

`APP_MODE=api`、`API_OWNER_ID`、`API_CLIENTS_JSON`を設定して`go run ./cmd/api`。検索には`GOOGLE_MAPS_API_KEY`とGeocoding provider、分類・根拠・鮮度を満たすcatalogが必要。API modeはRoutes/Maps JavaScriptを使わない。新規API有効化・課金・key登録はこの手順の自動処理では行わない。

## 資格情報の発行・更新・失効

1. API管理者が各Botを許可するownerへ対応付ける。owner/client IDは人名・platform member IDそのものを使わず内部識別子にする。
2. clientごとに暗号学的乱数32 bytesを生成し、64桁小文字hexへ変換する。平文tokenをBot側secret storeへ保管し、API側へはその文字列のSHA-256 digestを設定する。
3. scope=`facilities:read`、タイムゾーン付きのexpiresAt、revoked=falseを設定する。MVPは自動発行・refresh serverを設けない。期限は運用者が明示して管理する。
4. ローテーションは新しいclient ID/digestで短い重複期間を設け、Bot切替後に古いcredentialをrevoked=trueまたは設定から除去する。
5. 設定は起動時snapshot。失効・更新は全instance/revisionの再起動/置換、旧revisionへtrafficが残っていないこと、旧token401・新token成功を検証して完了とする。漏えい時はまず失効を反映し、旧tokenを再利用しない。単一processの変更を全体の即時失効と表現しない。

Cloud Run Secret Manager/IAM、注入とrevision切替の具体コマンド・権限は#318で整備する。本書でsecret登録/deployを実施済みとはしない。

## PowerShellでの一時ローカル検証

以下は使い捨ての1時間credentialを**メモリだけ**に作る例。出力やファイル保存をしない。PowerShell transcript/画面共有を停止したprivate terminalで実行し、変数の内容を表示しない。本番credential発行として流用しない。

```powershell
$apiTestBytes = New-Object byte[] 32
$apiTestRng = [System.Security.Cryptography.RandomNumberGenerator]::Create()
$apiTestRng.GetBytes($apiTestBytes)
$apiTestRng.Dispose()
$apiTestToken = ([BitConverter]::ToString($apiTestBytes)).Replace('-', '').ToLowerInvariant()
$apiTestHasher = [System.Security.Cryptography.SHA256]::Create()
$apiTestDigest = ([BitConverter]::ToString($apiTestHasher.ComputeHash([Text.Encoding]::UTF8.GetBytes($apiTestToken)))).Replace('-', '').ToLowerInvariant()
$apiTestHasher.Dispose()
$env:APP_MODE = 'api'
$env:API_OWNER_ID = 'local-owner'
$apiTestClients = @(@{
  clientId = 'local-test'
  ownerId = $env:API_OWNER_ID
  tokenSha256 = $apiTestDigest
  scope = 'facilities:read'
  expiresAt = [DateTime]::UtcNow.AddHours(1).ToString('o')
  revoked = $false
})
$env:API_CLIENTS_JSON = ConvertTo-Json -InputObject $apiTestClients -Compress
go run ./cmd/api
```

起動terminalを止めるときはCtrl+C。token変数は終了時に破棄する。別terminalへ秘密値をコピーする代わりに、通常の回帰確認には次を使う。provider/clock/catalog/credentialをテスト内で閉じ、実Google接続や課金を行わない。

```powershell
go test ./internal/apiauth ./internal/facility ./internal/nearby ./internal/httpapi ./cmd/api ./cmd/catalogcheck
```

別terminalで認証不要のlivenessのみ確認する:

```powershell
Invoke-RestMethod http://localhost:8080/healthz
```

既存本番catalogはgenre未分類かつ鮮度再確認が必要なので、`/readyz`の503や検索の`data_unavailable`を隠すために確認日だけ更新しない。fixtureはテスト用であり本番配備しない。

## Catalogの公開前確認

配備対象のcatalogに対して実行する。既定のfile以外を使う場合は`-path`をその公開catalogのpathへ変更する。このcommandはcatalogや外部設定を変更しない。

```powershell
go run ./cmd/catalogcheck -path data/facilities.json -require-searchable
```

全recordが実行時点から既定168時間後もdynamic 30日・stable 180日の鮮度内であり、同時点の検索掲載候補が1件以上ある場合だけ成功する。候補にはgenre・滑走根拠・一般利用状態も必要。`hoursStatus=unknown`は除外し、根拠のある`not_applicable`のstreetは含められるが、hours欠落を24時間利用可とは表示しない。[catalog保守手順](catalog-maintenance.md)で出典・日英説明・確認時刻を揃える。

APIの`/readyz`は認証構成・地点provider・現時点の候補を確認するだけで、実Google通信、168時間先の鮮度、現在の滑走可否を保証しない。#314では本番catalogの再調査・分類は未実施であり、通常CIや新schemaの成功を公開gateの成功と扱わない。

## 実接続の確認項目（公開前gate）

- HTTPS経由でBearer tokenをheaderに付けた検索/詳細が成功し、未認証・期限切れ・失効済みが401。
- queryはJSON bodyのみ。bodyを出力する`-Verbose`/debug、header dump、proxy body loggingを使わない。
- 別Botのcredentialだけを失効しても、許可された他Botは利用可能。
- Bot側で非ownerをAPI呼出し前に拒否。tokenが正しいだけでは人間の認可確認にならない。
- 地点provider未設定/障害503と候補0件を区別。既存dataの出典・genre・鮮度を検証し、実検索で確認。
- 旧Web/Bot/metrics routeがAPI modeで公開されず、Cloud Run直アクセスもGateway保護を迂回できない。
- log/traceにquery・token・検索中心が残らない。実運用の通知/監視/rollbackは#318で検証。

## Rollback

新modeへ切り替えていない既存環境は変更不要。切替済みなら認証を保つ設定と、互換なschema・binary・catalog snapshotの組で前revisionへ戻す。新fieldや旧5府県外recordを旧binaryが拒否し得るため、binaryだけを戻さず、現在時刻でcatalog gateとAPI smokeを再実行する。漏えい済みtokenを復活させるrollbackは行わない。catalogや訂正DBを削除しない。予算超過による自動停止は追加しない。
