# Bot共通APIクライアント

- Status: Implemented in source; platform adapters not connected
- Decision: [DR-0023](../decisions/0023-bot-api-client.md)
- Tracking: [#315](https://github.com/kohei321dev/spot-diggz/issues/315)の第一実装単位。#316でも共用予定。両Issueの全体完了ではない。
- API contract: [読み取りAPI](read-api.md)、[OpenAPI](facility-catalog.openapi.yaml)

## 責務と未接続の境界

`internal/botsearch`は構造化した`nearby.Input`を認証付きHTTP APIへ送り、検証した`nearby.Response`または固定errorを呼出元へ返すGo部品である。旧`internal/chatbot`の内部推薦serviceを呼ばず、独立APIを利用する。

この部品はplatform本人確認をしない。将来のadapterが真正性・owner ID・Bot宛て入力を検証した後だけ呼ぶ。API credentialが通ることを人間の認可と同一視しない。現時点では`cmd/api`や旧Slack/Discord handlerへ接続しておらず、新メンションは実行できない。

メンション文法、結果の短縮・翻訳・platform用エスケープ、検索中表示、owner限定配送、Bot配置、ACK後実行、配送失敗・重複eventの扱いは含めない。施設JSONを直接chatへ貼る実装でもない。

## 呼出し契約

- `NewClient(baseURL, token)`は運用者が注入するHTTPS originと、DR-0021のclient専用64桁小文字hex tokenを受け取る。secret storeや環境変数名の追加・実登録は行わない。
- originはpathなし（末尾`/`は可）。userinfo、query、fragment、空host、不正port、平文HTTPを拒否する。検索messageから送信先を選ばない。
- `Search(ctx, input)`は`POST /api/facilities/search`だけを呼ぶ。tokenはAuthorization header、検索条件はJSON bodyのみ。cookie、Web session、owner自己申告headerで代用しない。
- `nearby.DecodeInput`と同じ入力検証を再利用する。queryの前後空白を除き、不正UTF-8・制御文字・空白だけの場所・不正enum/数値をHTTP送信前に拒否する。
- `nearby.Input`には適用済み数値を渡す。JSONのoption省略を扱う呼出元は`nearby.DecodeInput`で既定値を適用する。構造体の`Limit=0`などをclientが省略と推測しない。場所解決と候補選定はAPIに任せる。
- clientを再利用でき、検索ごとの状態は保持しない。終了時の`CloseIdleConnections()`は空き接続を閉じる。本文・候補・確認ラベル・tokenをlog/storeへ記録しない。

## 通信と応答検証

- TLS証明書検証を有効にし、すべてのredirectを拒否する。同じoriginやsubdomainにも追従せず、URL変更は設定変更として扱う。
- DNS接続からbody読取りまで8秒と、呼出元contextの短い方を適用する。APIの5秒上限に通信分を加えた実装初期値であり、Bot配送やCloud Runの実行保証ではない。
- application側の自動retryをしない。POSTの再送用body・idempotency headerを設定せず、429や障害時に自動再検索しない。配送の冪等性・queueを追加した意味ではない。
- HTTP 200だけを検索応答とし、`application/json`・UTF-8、body最大1 MiB、response header最大16 KiBを検証する。巨大bodyを途中まで成功として返さない。
- 共有Go型のJSON tagに沿って、厳密な項目名、重複・未知field、必須項目の存在、型、nullを検証する。座標やbooleanの欠落を0/falseへ変換しない。必須配列は配列、任意fieldの欠落は省略として扱い、nullを施設事実へ変換しない。`matches`は0件でも空配列とする。
- 5つの正常status、候補数、曖昧ラベル（2〜5件・各300文字以内）、直線距離、適用limit/radiusKm/sort、注意文を確認する。矛盾する応答は全体を拒否する。
- 候補は有限かつ0以上・半径内の距離、距離/ID順、ID重複なし、指定genreとの整合、skateboard対応を確認する。Facilityの値・根拠は共有catalog validatorで確認し、未分類・営業時間unknown・日付別一般利用確認必須を候補として受理しない。
- 鮮度や検索中心からの距離をclientの時計/推測座標で再計算しない。APIの判定責任を維持する。営業時間非該当の根拠・日英注意文を保持し、hours欠落から24時間利用可・無料を補わない。

未知fieldを拒否するため、APIの加算的schema変更もclientの更新・契約test・互換な配備を同じ変更で評価する。これは本clientの保守制約であり、OpenAPI自体の変更ではない。

## 結果とエラー

正常な`ok`、`no_matches`、`data_unavailable`、`location_not_found`、`location_ambiguous`はそのまま返す。並べ直し、曖昧地点の先頭自動採用、genre/範囲の緩和は行わない。

| client error | 条件 | adapterが案内すべき次の行動 |
| --- | --- | --- |
| `ErrInvalidConfig` | 未初期化、origin/token形式不正 | 運用者が設定を確認。値は表示しない |
| `ErrInvalidInput` | 送信前の入力不正 | 入力を修正。元queryは不要に再掲しない |
| `ErrRejected` | HTTP 400/413/415 | 入力とAPI/client互換性を確認 |
| `ErrUnauthorized` | HTTP 401 | 運用者がcredential・期限・失効を確認 |
| `ErrForbidden` | HTTP 403 | 運用者が許可・Gateway等の制限を確認 |
| `ErrRateLimited` | HTTP 429 | 時間を置いて利用者が再実行。APIの現行案内は60秒 |
| `ErrUnavailable` | HTTP 5xx、TLS/DNS/接続/読取り障害 | 後で再実行。続く場合は運用者が確認 |
| `ErrTimeout` | contextまたはHTTP timeout | 時間を置いて再実行 |
| `ErrCanceled` | 呼出元の取消し | 検索結果や成功扱いにしない |
| `ErrInvalidResponse` | redirect、想定外HTTP、形式/サイズ/契約不整合 | 運用者が設定・API/client互換性を確認 |

error時は部分候補を返さず、正常0件へ変換しない。固定文字列のsentinelだけを返し、upstream body/header・URL・token・内部診断をwrapしない。これはadapter用の分類であり、利用者への配送実装ではない。

## 検証と運用への引継ぎ

`go test -race ./internal/botsearch`で設定、入力、redirect、HTTP failure、timeout/取消し、巨大・不正JSON、必須field欠落、非漏えいを検証する。`integration_test.go`は実際の`httpapi.NewReadAPI`とBearer認証をTLS loopbackで呼び、固定clock・合成catalog・stub Geocodingで全正常statusとDR-0022を確認する。実secret・本番catalogの改変・外部接続は不要。

- Status: Incomplete
- Missing evidence: #312のBot文法・受信transport・owner限定配送・非同期契約、#315/#316の入口接続、#318のsecret注入・Gateway/Cloud Run・実機E2E。実データのgenre/鮮度再確認も別gate。
- Required decision: agentが未確定技術案を具体化し、認証・保存・公開・運用の意味が変わる部分はownerが確認する。外部設定・secret登録・deployは別途承認する。

現時点のrollbackは本部品と同時更新docsのcommit revertで、API/legacy route・DB・外部設定は変わらない。将来接続後はAPI/client互換性と旧credential失効を確認し、認証bypassや内部推薦への無断fallbackで復旧しない。
