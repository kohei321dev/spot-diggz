# ADR-0009 選択式セッション検索とWeb UIをGo binaryへ含める

- Status: Accepted
- Date: 2026-07-16
- Related: [Product Baseline](../product_baseline.md)
- Related: [ADR-0003](0003-recommendation-engine-before-ai.md)
- Related: [ADR-0007](0007-go-modular-monolith-runtime.md)
- Related: [ADR-0008](0008-facility-catalog-api-and-storage.md)
- Origin/provider decisions partially superseded by: [ADR-0010](0010-google-maps-provider-and-fallback.md)
- Geographic scope partially superseded by: [ADR-0011](0011-five-prefecture-mvp-scope.md)

## Context

MVPでは、利用者が目的、気分、レベル、利用可能時間、検索起点、交通手段を指定し、検証済み施設から最大3件を選べる画面が必要である。

正確な現在地は保存しない。駅名や住所の自由入力を座標へ変換するには、外部geocoding providerの利用規約、送信データ、rate limit、障害時動作を決める必要がある。経路時間についてもproviderが未選定であり、実経路と同等の精度を断定できない。

初期トラフィックは小さく、Web UIだけのために別runtime、package manager、deployable unitを追加する必要性は確認されていない。

## Decision

1. MVPの主要画面は、依存libraryを持たないHTML、CSS、JavaScriptで実装し、Goの`embed`でapplication binaryへ含める。
2. UIとAPIを同一origin、同一processから配信し、frontend固有のbuild pipelineは追加しない。
3. 推薦入力は`POST /api/recommendations`で受け付け、目的、気分、レベル、利用可能時間、交通手段を列挙値として境界で検証する。
4. 検索起点は次の2種類とする。
   - browserのGeolocation APIで利用者が許可した現在地
   - UIに登録した大阪市・堺市の代表地点
5. APIは起点の緯度経度をrequest処理中だけ利用する。位置情報を永続化せず、access logへ出力せず、responseにも含めない。
6. 推薦はADR-0003のDeterministic Layerで行う。AI concierge、自然文入力、個人別説明はMVPへ含めない。
7. 外部route providerを選定するまで、移動時間は直線距離と交通手段別の固定速度から算出した開発用概算とする。片道の概算移動時間が利用可能時間の3分の1を超える施設は除外する。UIとAPI responseで概算であることを明示し、外部ナビで実経路を確認できるようにする。
8. 営業時間は日本標準時の到着見込み時刻で判定する。営業時間を判定できない施設は推薦しない。
9. 初期の気分選択肢は「じっくり練習」「気軽に滑る」「新しいセクションに挑戦」とし、利用者検証で見直す。

## Alternatives

### React等のfrontend frameworkを採用する

画面数や状態管理が増えた場合は有効だが、現時点の1画面と1つの主要flowにはbuild依存と更新対象が増える。必要性が確認できるまで採用しない。

### 自由入力した駅名・住所を外部APIで即時geocodingする

任意地点を扱えるが、provider、privacy、利用規約、rate limit、fallbackが未決定である。MVPの最初の実装では代表地点を選択し、provider選定後に追加する。

### AIで入力解釈と施設選定を行う

対話的な体験を作れるが、施設選定の再現性と根拠を弱める。MVPでは採用せず、決定論的な候補を説明するadapterとして後から評価する。

## Consequences

### Positive

- UI、API、施設fixtureを1つのbinaryとcontainerで確認できる
- 位置情報をdatabaseやbrowser storageへ保存せずに検索できる
- 選択値と推薦ruleを単体・HTTP・browser testで再現できる
- AIや外部providerの障害なしに主要flowを提供できる

### Negative

- 代表地点以外の駅名・住所はUIから指定できない
- 移動時間は実経路、待ち時間、渋滞を反映しない
- frontendの規模が増えた場合、asset管理と型共有を再検討する必要がある
- 気分の選択肢はproduct仮説であり、利用者検証前には確定できない

## Verification

- APIが未知の選択値、範囲外座標、過大bodyを拒否する
- 推薦が初心者適性、営業時間、概算移動時間で除外し、最大3件を安定した順序で返す
- responseが推薦理由、情報源、確認日、概算移動時間の注意書きを含む
- access logとresponseへ起点座標を出力しない
- desktopとmobileで6条件を選択し、推薦結果と外部ナビを表示できる
- `go test`、`go vet`、`go build`、脆弱性検査を既存CIで実行できる

## Revisit Conditions

- 任意の駅名・住所指定が利用者検証の必須条件になる
- route providerの精度、cost、利用規約を確認して採用できる
- UIの画面数、状態、team ownershipが増え、専用frontend toolchainの便益が上回る
- AI conciergeがMVP後の検証対象として優先される

## Evolution 2026-07-17: ワンタップ推薦と段階的な条件開示

### Final Goal (unchanged)

利用者の条件から、今から実際に滑りに行ける検証済み施設を決定論的に提示し、外部ナビで訪問へつなげる。

### Means / Implementation (changed)

| 論点 | ADR確定時 | 実装変更後 | ゴール影響 |
| --- | --- | --- | --- |
| 主要操作 | 6条件を表示したformを送信 | 保存済みまたは既定の6条件を使うワンタップ操作を追加 | 入力負担を軽減 |
| 条件変更 | 常時表示 | `条件を変更`の内側で段階的に開示 | 6条件の選択可能性を維持 |
| 気分 | selectで1項目を選択 | 3つの気分別buttonから直接推薦を開始 | 推薦ruleは変更なし |
| 結果 | 最大3件を同じcard形式で表示 | 1件目を`今日の一択`、残りを代替候補として表示 | 意思決定を優先 |
| 時間表示 | 片道の概算移動時間 | 到着、営業終了、往復移動込みの滑走可能時間も表示 | 実行可能性を明確化 |
| 設定保存 | 保存なし | 代表地点と選択値だけをbrowser local storageへ保存 | 正確な現在地は非保存を維持 |

### Why Changed

- [事実] 6条件を毎回選択する画面は、短時間で行き先を決めるJobに対して操作数が多い。
- [事実] 既存の推薦APIは完全な6条件を受け取るため、UI側で既定値と保存値を組み立てればAPIを分岐させずに再利用できる。
- [判断] 主要導線をワンタップにし、詳細条件を二次導線へ移す。推薦の再現性、入力検証、位置情報の非保存は変更しない。

### Exception Handling

- 適用範囲: MVPの単一画面Web UI
- 解除条件: 利用者検証で初回設定の不足または条件変更の発見性低下が確認された場合、onboardingまたは表示方法を見直す。
- Owner / Watch: Sprint Reviewで`recommendation_completed`、`result_displayed`、`navigation_opened`と利用者観察を確認する。

## Evolution 2026-07-20: 任意地点検索と実経路provider

### Final Goal (unchanged)

正確な検索位置をapplicationへ保存せず、今から滑りに行ける検証済み施設を決定論的に提示する。

### Means / Implementation (changed)

| 論点 | ADR確定時 | 実装変更後 | ゴール影響 |
| --- | --- | --- | --- |
| 代表地点 | 大阪市・堺市の4地点 | 5府県の9地点 | 地理scopeをADR-0011へ拡大 |
| 任意地点 | provider未決定のため対象外 | Google Geocodingによる駅名・住所検索 | 任意起点を追加 |
| 移動時間 | straight-line概算のみ | Google Routesを優先し、失敗時straight-line | 精度を改善し可用性を維持 |
| privacy | applicationで非保存 | 非保存を維持し、Googleへの外部送信を明示 | trust boundaryを追加 |
| 設定保存 | 代表地点と選択値 | localeだけ | data minimizationを強化 |

### Why Changed

- [事実] server-side provider adapter、timeout、fallback、地点検索APIを独立moduleとして実装できた。
- [事実] `GOOGLE_MAPS_API_KEY` がない環境でも推薦はstraight-lineで継続し、地点検索だけを利用不可にできる。
- [判断] Google利用は明示的なopt-inとし、送信データ、縮退表示、無効化方法をADR-0010とRunbookで管理する。

### Exception Handling

- 適用範囲: `GOOGLE_MAPS_API_KEY` を設定した環境だけGoogleへ送信する。
- 解除条件: keyを削除して再起動すると、推薦はstraight-lineのみ、地点検索は `503` へ戻る。
- Owner / Watch: release ownerが `travelEstimateKind`、location search smoke、Google usageを確認する。
