# DR-0023: Bot共通の認証付き検索APIクライアントを先行実装する

- Status: Accepted
- Date: 2026-09-08
- Type: Architecture
- Related Issues: [#315](https://github.com/kohei321dev/spot-diggz/issues/315)、#312、#316
- Related Pull Requests: [#323](https://github.com/kohei321dev/spot-diggz/pull/323)
- Affected Docs: `specifications/bot-api-client.md`, `specifications/chat-integrations.md`, `specifications/api-mvp.md`, `specifications/nearby-search.md`, `requirements.md`, `architecture.md`, `security.md`
- Supersedes: None（DR-0019/DR-0021のHTTP利用を具体化。Bot文法・platform・保存・配置の未確定は維持）
- Superseded By: None

## Context

ownerがPR #322のマージ後、実装継続を依頼した。基準remote mainは`54805a38b0779c1e00feac90dca40abaaf237af7`。読み取りAPIとcatalog品質のコードはあるが、旧Botは内部推薦を呼び、新MVPのメンション受信/owner限定返信/非同期方式は未確定である。

#315の第一単位として、決定済みAPI契約だけを利用する共通clientを追加する。設計Issue #312にruntime実装を混ぜず、#315/#316を全完了とはしない。以下は委ねられた継続実装内の具体化であり、外部接続や新たな認証・保存方式の承認ではない。

## Decision

1. 既存Go構成の`internal/botsearch`に構造化入力→Bearer HTTP POST→構造化結果を置く。旧handlerへ接続せず、文法・表示・配送・runtime設定の追加を含めない。#316でも再利用できる。
2. 運用者設定のHTTPS originとDR-0021のclient専用tokenを使い、送信先を固定する。cookieや内部serviceで認証を代替せず、全redirectを拒否する。
3. 通信全体8秒・body最大1 MiB・header最大16 KiBを初期上限とする。自動retry、cache、queue、検索/会話stateを持たない。BotのACK後実行/配送保証とは分ける。
4. 入力は既存`nearby.DecodeInput`、施設の値/根拠はcatalog validatorを再利用する。responseの厳密な項目名・必須項目・重複・型を別途確認し、欠落をゼロ値の施設事実へ変換しない。APIの時計/地点による選定を再計算しない。
5. 正常5statusとHTTP/通信失敗を分け、失敗は固定errorと空の結果だけを返す。診断にquery・endpoint・token・raw responseを含めず、候補不足を不存在と断定しない。

## Rationale

未確定なplatform transportや保存方式を選ばず、採用済みの独立HTTP境界を実行検証できる。実API handlerとTLS loopbackを通して契約差を検出し、通信の安全策を各adapterへ重複実装しない。

## Alternatives Considered

- 旧内部推薦serviceを流用: 独立APIを呼ぶ要求を満たさず不採用。
- platform受信・非同期・保存を同時採用: 認可/配送/保存の未決定を隠すため今回の範囲外。
- TypeScript Chat SDKや新runtimeを導入: 共通HTTP部品だけには不要。既存Go/共有型の保守を優先する。
- redirect・未知fieldを黙って受理: credential転送や意図しない契約差を見逃しやすいため、本clientでは拒否する。

## Consequences

依存・route・設定値・DB・実施設dataは追加しない。platformへの返信はまだできず、#315/#316には接続が残る。APIの加算的field変更もclientとの互換test・配備順の確認が必要。上限変更は仕様・testを同じPRで更新する。

## Security and Privacy

API tokenはclientの権限であり、人間の本人確認ではない。真正性・owner認可済みadapterからのみ呼ぶ。tokenはruntimeメモリに必要だが、通常のclient文字列表示では伏せる。入力・候補・確認ラベルをlog/storeへ保存せず、upstream診断をerrorへ含めない。表示のエスケープとowner限定配送は別途実装する。

## Migration and Rollback

未接続部品の追加のみで、API/legacyの動作・保存データ・外部設定は変更しない。コードとdocsをcommit単位でrevertできる。接続後はAPI/client互換性と失効credentialを確認し、認証迂回を復旧方法にしない。

## Validation

unit/race testで設定・入力・redirect・timeout・取消し・HTTP failure・巨大/不正response・必須field欠落・Unicode紛らわしいkey・非漏えいを確認する。実API handler、合成catalog、固定clock、Geocoding stub、TLSでBearer、5status、genre/limit/radius/距離順、非該当hoursを検証する。Go format/vet/race/build、OpenAPI/文書、diff/secret検査を行う。実Slack/Discord/Google/Cloud Runの成功とは扱わない。

## Revisit Conditions

Bot受信/返信を接続するとき。API field拡張、client別上限・自動retry・保存・認証・配置・公開を変更するとき。

## References

- [読み取りAPI](../specifications/read-api.md)
- [Bot共通APIクライアント](../specifications/bot-api-client.md)
- [Go HTTP Client: timeoutとredirect](https://pkg.go.dev/net/http#Client)
- [Go JSON decoder](https://pkg.go.dev/encoding/json#Decoder)
