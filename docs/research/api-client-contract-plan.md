# API提供契約とclient境界の実装前評価

- Status: Proposed — 採用前の設計案。現行仕様を上書きしない。
- Date: 2026-09-06
- Issue: [#312](https://github.com/kohei321dev/spot-diggz/issues/312)
- Baseline: remote `main` `1b4797ce488961bf02572573999f71d0b1d7e9d1`
- Related decision: [DR-0019](../decisions/0019-api-client-boundary.md)（Proposed）
- Implementation authority: ownerは#312〜#318の実装着手と関連docsの更新を依頼した。未提示だったclient認証・呼出境界の採用承認、merge、deploy、secret登録とは区別する。

## 先に決めること

「APIとして提供する」は、APIとbotを別々のプロセスで動かすことまで必ずしも意味しない。一方、#313・#315・#316には独立clientからHTTP APIを呼ぶことを想定した記述がある。これを既存の内部service呼出しに読み替えて、全Issueを実装完了にしてはならない。

| 案 | 初期の呼出経路 | 認証情報と責任 | 新計画への影響 |
| --- | --- | --- | --- |
| A: 同一Goアプリ内の各HTTP入口と共通処理を整備 | Web→JSON API、Slack/Discord→署名付きHTTP入口→共通の型付きapplication処理 | 現行GitHub owner session、platform署名とowner ID認可を維持。新しいAPI利用資格は発行しない | 単一アプリの維持に適する。独立bot/appによる汎用HTTP API呼出しは未実装と明記し、#313・#315・#316の該当範囲をowner承認後に訂正する必要がある |
| B: 独立bot/appからのHTTP API利用も初回から実装 | Web・独立bot/app→認証付きJSON API。platform署名の検証者とAPI呼出主体を分ける | clientごとの利用資格、許可範囲、ownerへの対応付け、期限、失効、鍵更新、漏えい時対応を新しく決定する | 起票済みのHTTP client整備という表現に沿う。安全境界と資格情報管理方式の設計承認が必要 |

小規模なowner利用を保ち、新しい秘密情報管理を増やさずに始めるならAを提案する。ただし、Bが必要な要求をAで満たしたとは扱わない。API提供方針だけを理由に新しい認証方式、DB、OAuth providerを導入しない。

**採用案は未確定。** この判断が必要なのは構成・安全境界であり、#312〜#318への着手許可をもう一度求めるものではない。

## 観測した実装と変更対応表

| 対象 | 現行コードの事実 | 再利用・変更方針 | 後続Issue |
| --- | --- | --- | --- |
| Web認証 | `internal/owneraccess/manager.go`がGitHub OAuth、owner一致、署名済みsession、未認証API拒否を処理 | 再利用。設定欠落・非owner拒否を維持。Bの資格情報をbrowserへ配布しない | #313、#317 |
| 施設参照 | `internal/httpapi/server.go`に一覧JSONと施設ID指定の取得API | 既存pathとJSONを維持。施設一覧ページの新設とは分ける | #313 |
| 地点検索 | `POST /api/locations/search`はqueryを受け、任意Google providerへ渡す | 既存契約を基本にする。未設定・失敗は利用不能。chat側も未検証の地点を作らない | #313、#315、#316 |
| 推薦 | `POST /api/recommendations`とchatの`Service`が同じengineを呼ぶ | engineは再利用。Aなら入力検証・期限・エラー・観測を共通化。Bなら認証付きclient契約を実装 | #313 |
| 訂正・event | `POST /api/corrections`と`POST /api/events`が既存UIで使われる | 未承認の削除をしない。botへ不要なwrite権限を与えない | #313、#317 |
| Slack | `slack_interactive.go`がmodal入力を地点解決し、内部serviceへ渡す | 入力・ephemeral返信は再利用。A/Bを決めずにHTTP client対応済みとしない | #315 |
| Discord | `discord.go`は条件optionを持たず、server設定の既定条件を使用 | 位置・条件の明示入力とエラー案内が必要。入力方式は下記提案を承認後に実装 | #316 |
| 地域 | OpenAPIと`internal/facility/catalog.go`に5府県の固定許可値 | 対象ユーザーとは切り分ける。地域表現の変更は検証・互換性と一緒に扱う | #314 |
| 経験レベル | `session/model.go`の許可値はbeginner、returning、intermediate | 上級者がターゲット外とはしない。未対応の入力条件を勝手に既存値へ変換しない | #312、#313 |
| Web UI | `app.js`が既存JSON APIへfetchする | 必要な契約適合・エラー表示だけ変更。全面的な作り直しはしない | #317 |
| 公開先 | `operations/service-status.md`では現在の公開先がIncomplete | 旧hostの実行を再開せず、確認済み環境と明示承認後だけ外部設定を行う | #318 |

コードからの観測は本番稼働の証拠ではない。採用前の方針は実装済み仕様ではない。

## JSON API契約の共通部分（提案）

### 再利用するpathと意味

| Method / path | 入力 | 応答・意味 |
| --- | --- | --- |
| `GET /api/facilities` | 任意のactivity絞込 | 検証済みcatalogの参照。全件が今すぐ推薦可能とは限らない |
| `GET /api/facilities/{facilityId}` | 施設ID | 施設事実、出典、確認時刻、利用条件。存在しなければ404 |
| `POST /api/locations/search` | JSONのquery | 地点候補。外部provider未設定・失敗は503。入力queryをURLやlogへ含めない |
| `POST /api/recommendations` | 目的・気分・level・使える分数・交通手段・origin | 最大3件の根拠付き推薦。0件を含む正常結果と処理障害を区別する |
| `POST /api/corrections` | 検証・同意条件を満たす訂正report | 保存成功時だけ202とreceipt。既存保持・purge契約を維持する |
| `POST /api/events` | allowlist済みevent名 | 最小限の集計受付。自由入力や位置を送らない |

- 現行OpenAPIを契約の正本として維持し、pathの一括改名や根拠のない`/v1`追加はしない。
- 既存fieldの意味・必須性・型・enumを変えるときはclient影響と移行をレビューする。enum追加を常に無害な変更とは扱わない。
- 現行の`error.code` / `error.message`、HTTP status、`X-Request-ID`、`Cache-Control: no-store`を基本にし、互換性のない形式へ一括変更しない。
- 認証欠落、不正入力、候補なし、外部依存障害を別の状態として扱う。JSON APIの認証失敗でHTML loginを返さない。
- 正確なoriginと地点queryをapplicationへ保存・log出力・入力のまま応答再掲しない。Google有効時の外部送信はprivacy境界として明示する。
- 推薦理由は検証済み属性と条件から決定論的に生成し、stale・未検証事実を混入させない。
- HTTP入口とchat入口で期限・入力検証・安全なerror・所要時間観測を共有する。共有するためだけの自己HTTP呼出しや本番認証bypassを追加しない。

### 認証・認可の不変条件

1. WebはGitHub owner認証を維持する。認証済みでもownerでなければ許可しない。
2. Slack/Discordはraw bodyとtimestampの署名検証、および設定済みimmutableなplatform ID認可を維持する。
3. 表示名・メール・clientが申告したowner名だけで認可しない。共通API key一つで全入口の本人確認を代用しない。
4. Bでは、platformの本人確認を誰が実施し、APIがどの証拠・信頼関係でownerを認可するかを明示する。botが任意のowner headerを指定すれば通る構成にしない。
5. productionの設定欠落はfail closed。local専用bypassを本番の疎通手段へ流用しない。
6. credential、Cookie、署名、token、位置、本文をlog、artifact、browser assetへ出さない。

### 未確定の安全境界

- Status: Incomplete
- Missing evidence: A/Bの採用判断。Bの場合、許可するclient種別、資格情報発行・保存・期限・失効・更新方式、owner対応付け、CORS/redirect許可範囲、監査方法。
- Required decision: ownerが初期提供に独立bot/appからのHTTP API呼出しを含めるか判断する。Bなら方式案のレビュー後に認証を実装し、Aなら独立client未対応をIssueと仕様に明記する。

## 地域・入力・データの後続提案

- 地域非限定の対象定義を、全世界の住所形式や多言語データが実装済みという意味にしない。
- #314では現行の日本国内の住所表現を保ち、都道府県許可値を5府県固定から国内の正式な都道府県集合へ変える案を検討する。海外住所・国別timezone等は別の契約設計が必要。
- 施設の追加は公式sourceと更新責任を確認した小さな単位にする。初回は旧#303が指定した`OSK-F017`の再確認を提案する。まだ公式情報の再調査・更新は行っていない。
- 既存の動的30日・安定180日の鮮度、一般利用未確認を推薦しない条件を維持する。確認日時だけを更新して検証を通さない。
- levelはターゲット区分ではなく推薦条件。追加する選択肢の意味と適合ルールが決まるまでは、無条件に`advanced`を追加したり中級へ読み替えたりしない。
- Discordは、公開代表起点の暗黙利用から、出発地を必須・その他条件を既存値から選べるcommand option方式へ変える案を提案する。詳細は#316で公式仕様と照合して確定する。Slackの既存modalは維持する。

## 安全境界・回帰テスト表

| Case | 必須結果 | 主な対象Issue |
| --- | --- | --- |
| Web sessionなし・不正・期限切れ・非owner | JSON APIを拒否。認証を要求する安定したerror | #313、#317 |
| production認証設定欠落 / bypass指定 | 起動拒否または認可拒否。bypassで通らない | #313、#318 |
| Slack/Discord署名改ざん、古い/未来timestamp、ID欠落・不一致 | 重い処理・返信前に拒否。署名や識別子をlogに出さない | #315、#316 |
| Bの偽client、偽owner申告、期限切れ・失効資格情報 | APIで拒否。platform側検証を迂回できない | #313、#315、#316 |
| 不正JSON、過大body、不明field、不正座標・enum | 安定した4xx。値を診断logへ転載しない | #313 |
| 同一入力・catalog・固定clock・provider結果 | 同じ候補・順位・根拠。UIによって選定を変えない | #313、#315〜#317 |
| 候補なし、stale、一般利用未確認、休場 | 捏造しない。条件変更/公式確認など次の行動を伝える | #313〜#317 |
| Google未設定・timeout・失敗 | 既定の地点検索利用不能/経路概算を区別し、安全に応答する | #313、#315、#316 |
| platform retry、連打、配送失敗 | 重複処理方針、期限、retry可否を明示。履歴本文を保存しない | #315、#316 |
| 訂正store失敗・保持期限 | 保存失敗を成功扱いしない。既存同意・retention・purgeを維持 | #313、#318 |
| log/metrics/trace/store/browser assetの検査 | 位置、query、自由入力、secret、個人IDが混入しない | #313〜#318 |
| 日英、mobile、keyboard、位置情報拒否 | シンプルな主要操作とerror案内を維持 | #317 |

ネットワークを必要としないunit/HTTP/署名fixtureテストと、承認済み実環境のowner/非owner E2Eを分けて記録する。CI成功を外部App設定や公開稼働の証拠としない。

## 実装・文書更新の順序

1. #312: A/Bと契約を採用し、DR-0019をレビューする。必要なIssue範囲の訂正もここで合意する。
2. #313 / #314: 既存実装を再利用してAPI契約・施設データ制約を整合させる。変更前に競合を確認する。
3. #315 / #316 / #317: 承認したAPI境界へ各入口を適合させ、表示・入力の差分を検証する。
4. #318: 確認済みの環境・公開範囲で結合と運用を検証する。外部変更は別途承認する。

各IssueのPRで、対応するrequirements、OpenAPI/説明仕様、architecture/security、guide、operations、Decision Recordを同時に更新する。後でまとめて現行仕様を追記する運用にはしない。採用前の本書はresearchに置き、採用内容は正規文書へ反映する。

## 今回の確認範囲

- remote main、open #312〜#318、open PRなし、clean working treeを確認した。
- 上表のコード、現行OpenAPI、DR-0015/0018、requirements、security、processと照合した。
- runtime、依存、catalog、設定、外部App、公開環境には変更していない。
- 生のprovider response、credential、`.env`の値、位置・個人情報は収集・転載していない。
- API本体と各入口の改修完了、実環境検証の完了は未確認。現在は設計レビュー段階。
