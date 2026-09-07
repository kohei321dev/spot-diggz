# DR-0021: 読み取りAPIのBearer認証と周辺検索の最小契約

- Status: Accepted
- Date: 2026-09-07
- Type: Architecture
- Related Issues: [#313](https://github.com/kohei321dev/spot-diggz/issues/313)、#312、#314
- Related Pull Requests: Incomplete — 実装PR作成後に追記
- Affected Docs: `specifications/read-api.md`, `specifications/nearby-search.md`, `specifications/facility-catalog.openapi.yaml`, `security.md`, `architecture.md`, `requirements.md`, `guides/read-api-setup.md`
- Supersedes: DR-0019のAPI認証方式未確定、DR-0020の検索APIの数値・応答未確定のみ。Bot受信/返信・外部運用は別途。
- Superseded By: None

## Context

ownerはPR #320のマージ後、Botごとの専用Bearer token、API側での読み取り権限・有効期限・個別失効、Bot側でのowner ID照合という提案を受けて、最小契約を具体化し検索APIから実装するよう依頼した。基準mainは`2f3982ea40ddade04770139d02b3ba26549448ac`。全MVPの設計完了を待たず、#313の独立読み取りAPIを検証可能な実装単位とする。#312/#313全体を完了とはしない。

## Decision

1. 明示的な`APP_MODE=api`で独立読み取りAPIを起動する。旧Web/OAuth/訂正store/Bot endpointを登録・起動しない。未指定の旧modeは互換性のため維持し、既存保存データや旧コードの削除は後続とする。
2. client別に32 byte乱数を64桁小文字hexにしたBearer tokenを発行する。APIにはSHA-256 digestとclient ID、owner ID、scope、expiresAt、revokedだけを設定し、平文tokenはBot側secret storeだけに置く。期限・失効・owner mapping・scopeをrequestごとに検証する。config未設定/不正ならAPI mode起動を拒否する。失効反映は設定更新と全稼働instance/revisionの再起動・置換が必要であり、即時失効とはしない。
3. API権限は`facilities:read`のみ。tokenは信頼するBot/clientの識別であって人間の本人確認ではない。Botでplatform本人確認とowner認可を必須にする。HTTP header以外のtoken、cookie、申告owner headerでAPI認証を代用しない。
4. `POST /api/facilities/search`は場所名queryと任意genre/limit/sort/radiusKmを受け付ける。実装初期値はlimit=5（1〜10）、radiusKm=10（0.1〜50）、sort=distanceのみ。件数・半径は別々に適用し、丸め前の距離とfacilityIdで安定整列する。これは調査済みの最適値ではなく、ownerから委ねられた最小実装の変更可能な初期制約である。
5. 場所解決はAPI内で行い、複数候補なら場所ラベルを返して再入力を要求する。先頭を自動採用せず、検索基準座標や元queryをresponseへ再掲しない。認証済みclientへ確認に必要なラベルのみ返す。Botの選択UI/文法は後続とする。
6. 検索は検証済み・鮮度内・genreと滑走可能な根拠が確認できるrecordを対象とする。既存recordへgenreを推測付与しない。genre/根拠欠落や期限超過は情報不足として区別し、本番catalogの追加・再調査は行わない。現在営業中かの判定はしないが、時間・休場・ルールを返し「今滑れる」を保証しない。
7. 検索成功・該当なし・データ不足・場所未特定・場所確認待ちを構造化statusで区別する。外部依存障害は503であり空候補へ変換しない。独立APIは同期・上限timeoutで完結し、queueや検索履歴を追加しない。

## Rationale

DB、OAuth認証基盤、Bot常駐配置を検索APIの着手条件にせず、JSON契約とHTTP境界を先に検証できる。公開中の旧契約を一括変更せず、明示的modeとendpoint表で互換性・rollbackを保つ。Bearer tokenの漏えいは権限行使につながるため最小scope・期限・digest保存・HTTPSを併用する。

## Alternatives Considered

- 全体設計・全Botの完成後にAPI実装: ownerが求める小さな実装開始を妨げる。
- GitHub browser cookieをBotで共有: サーバー間契約と本人確認の責務を混同するため不採用。
- OAuth認可serverや短命JWT発行基盤を新設: 現在の単一owner・少数の信頼clientに対して追加運用が大きい。将来の動的client追加や即時失効要求時に再評価する。
- 全APIを無条件公開/共通tokenで全owner許可: 不採用。
- 未分類の旧施設を自動でskateparkにする: 事実の推測となるため不採用。

## Security and Privacy

TLS終端・Gateway直アクセス防止・実secret設定は#318の公開gate。local test以外で平文HTTPを使わない。token digest比較は固定長で定時間比較し、エラーにtoken/client ID/入力/provider responseを含めない。位置/queryは処理メモリのみ。R-008の曖昧性は新APIでは「検索座標と元queryを返さず、場所確認に必要なラベルだけ返す」と限定して解消し、旧地点APIの削除や契約変更は別途扱う。

## Consequences

API modeにはWeb、OAuth、旧推薦、訂正/event、Bot endpoint、公開metricsを登録しない。既存詳細APIのpathを再利用するが、API modeではBearer必須。旧modeの認証とrouteは変更しない。新検索で使うgenre/滑走根拠は加算的schemaとし、未分類の本番recordは検索対象へ昇格しない。新APIを実装してもMVPの実データ・Bot・本番公開は未完了である。

## Migration and Rollback

無指定時は旧modeのまま。API modeへの切替・新credential登録は別途明示的に行う。code/configを互換な組でrevertし、旧DBとcatalogを保持する。token失効時は古いdigestを受け入れる全revisionを停止/置換する。予算超過は停止理由にしない。

## Validation

固定clock/catalog/地理providerのunit・独立HTTP test、未認証/期限/失効/非owner mapping/scope/設定欠落、JSON境界、距離・genre・半径・limit、曖昧地点・データ不足・provider障害、log/response非漏えい、旧mode非回帰を検証する。Go format/vet/race/test/build、OpenAPI・文書・secret/依存checkを行う。Cloud Run/Bot E2Eは後続。

## Revisit Conditions

即時失効、複数owner/匿名公開、細分化scope、動的client登録、半径/件数の需要、全国genreデータ、Bot transport/返信・公開条件が変わるとき。

## References

- [Bearer token usage](https://datatracker.ietf.org/doc/html/rfc6750)
- [Go constant-time comparison](https://pkg.go.dev/crypto/subtle#ConstantTimeCompare)
- [DR-0020](0020-mention-nearby-search.md)
