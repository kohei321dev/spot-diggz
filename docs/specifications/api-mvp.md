# MVP API提供契約

- Status: Accepted direction; implementation incomplete
- Last reviewed: 2026-09-07
- Decision: [DR-0019](../decisions/0019-api-client-boundary.md)、[DR-0020](../decisions/0020-mention-nearby-search.md)
- Tracking: [#312](https://github.com/kohei321dev/spot-diggz/issues/312)、後続#313〜#316、#318

[DR-0021](../decisions/0021-read-api-contract.md)で読み取りAPIの最小契約を確定し、`APP_MODE=api`をソース実装した。[読み取りAPI](read-api.md)が認証・検索JSON・応答・互換性の正本。Bot接続・旧コード退役・Cloud Run公開は未完了。

## 提供範囲と責務

対象は地域・経験レベルを限定しない「スケボーをしたい人」。初期利用は許可済みownerのみとし、全国のスポット情報を提供済みとはしない。

```text
Slack / Discord
  -> 自分のBotサーバー（署名・owner ID検証、条件入力、受付・返信）
  -> 認証付きSpotDiggz HTTP API（client認証・認可、施設参照・推薦）
  -> 検証済みcatalog / 共通の決定論的推薦engine
```

BotとAPIは責務と信頼境界を分ける。APIを独立clientから呼べることが受入条件であり、内部serviceの呼出しだけで代用しない。物理的なサーバー数・deploy単位・リポジトリ分割は未確定。APIホスト先はCloud Runとする。

独自Web UIはMVPの対象外。#317の画面整備を行わない。将来の外部UI clientの可能性を否定するものではない。既存の施設参照API、メディア情報、訂正データまで削除する判断ではなく、旧Web/OAuth/専用routeの移行範囲は#312/#313で確認する。

## 認証・入力・施設情報

- Botは受信transportに適したplatform真正性・再送対策と設定済みownerのIDを検証する。旧HTTP入口の署名・timestamp検証を無条件に解除せず、メンション受信への適合は#312/#315/#316で確認する。API側でもclient認証・権限を検証し、未設定時はfail closedにする。
- API keyによる呼出元識別やIP制限を、利用者本人の認可と同一視しない。読み取りAPIはclient別Bearer token（digest保存）、owner対応付け、facilities:read、期限、設定反映後の個別失効を採用した（DR-0021）。
- [周辺検索仕様](nearby-search.md)に従い、メンションと場所名から周辺を探す。genre・limit・sortを任意指定し、検索範囲も指定可能にする。出発地・現在地・旧6条件は基本検索で必須にしない。場所が曖昧なら利用者へ確認する。
- 検証済みデータと決定論的処理を再利用し、AIで施設事実を補わない。固定の最大3件を新MVPへ引き継がず、候補数はlimitで指定し直線距離順に返す。出典・確認時刻・品質原則は維持する。検索掲載可否はDR-0021/DR-0022で具体化し、根拠のあるstreetの営業時間非該当と、不明の検索除外を分ける。当日滑走可否を保証しない。
- 実スポットデータの追加は後続作業。[DR-0022](../decisions/0022-domestic-catalog-quality.md)で国内地域表現・営業時間状態・品質検証と[更新手順](../guides/catalog-maintenance.md)を整え、#314の成果とする。実施設の追加件数や特定施設の再調査は含めず、実運用前の鮮度・検索掲載gateは免除しない。
- 候補、会話履歴、正確な位置、地点query、返信tokenを永続保存しない。保存Listsは設けない。短期状態の既存制約を変更する場合は明示的な設計レビューが必要。

## 承認済み返信フロー

1. 明示的なメンションと場所名をトリガーにする（DR-0020）。Slackは無料プランの通常Appを前提とし、slash command/modalを新MVPの基本入口にしない。Slack-hosted workflow、Socket Mode、履歴巡回を自動採用しない。platform別受信transport・scope/intent・返信方式は#312/#315/#316で検証する。
2. 真正性・認可・入力の検証後、条件送信に対し先に「検索中」を表示する。受信方式に必要なprotocol上の受付（HTTPの場合はACK）と利用者が見える受付表示を別々に検証する。
3. Botが認証付きAPIへ条件を渡し、完了後にownerだけへ結果を返す。利用者にWeb UIを開かせない。
4. 処理失敗時も利用者向けエラーと再実行など次の行動を返す。内部診断、入力本文の不要な再掲、正確な検索座標、secretを返信しない。新APIでは検索中心/元queryを返さず、確認用ラベルのみを返す（DR-0021）。旧地点APIとBot表示の退役/移行は別途。

候補0件・データ不足・条件不一致・地点未特定/曖昧・障害は[読み取りAPI](read-api.md)のstatus/HTTPで区別する。Botの返信文面と条件変更UIは後続。条件の自動緩和や未確認情報の補完は行わない。

platform障害などでエラー通知自体を配送できない場合もあるため、「必ず届く」とは保証しない。配送失敗の観測、retry上限、重複抑止、timeout、process停止時の扱いを実装前に決める。

## 要求IDの適用移行

既存IDは振り直さない。[Requirements](../requirements.md)の移行前受入条件に対して、DR-0019/DR-0020による以下の変更を適用する。

| 対象 | 新MVPでの扱い | 残る確認 |
| --- | --- | --- |
| R-001 | メンション＋場所名、任意genre/limit/sort/検索範囲。旧6条件必須を置換 | Bot文法は後続。API field/数値/場所解決はDR-0021 |
| R-002〜R-003 | 固定3件と目的別scoreを基本検索から外し、limitと直線距離順で比較する | APIはDR-0021で確定。Bot表示は後続 |
| R-004〜R-005、R-007、R-009〜R-012、R-015〜R-016 | 出典・日英・外部導線・品質・安全性をAPI/Botへ引き継ぐ | 検索と即時滑走判定の区別、JSON field・文面・互換性 |
| R-006 | 訂正機能・保存データを今回削除しない | APIの公開範囲・権限・Botからの導線を#312で決定 |
| R-008、R-019〜R-020 | 位置・履歴・候補の非保存を維持 | 非同期処理が保存制約に反しないこと |
| R-013〜R-014のWeb操作領域・iframe表示、NFR-005の独自Web E2E | 独自Web UI向け提供要件は対象外。Botの読みやすさ・日英・安全な導線は維持 | 旧UIコード・テストの廃止範囲を#313で確認 |
| R-017 | Web OAuth cookie必須を新API client認証の必須方式にしない。owner限定は維持 | API Bearer方式はDR-0021。旧OAuthコードの退役は後続 |
| R-018 | Slack/Discordでメンション＋場所名/任意option→検索中→API結果またはエラー通知 | platform別メンション受信、owner限定返信、非同期実行・配送 |

独自Web UIの新規提供:

- Status: Not applicable
- Reason: ownerがMVPはSlack/DiscordからAPIを利用し、独自Web UIは不要と決定したため。
- Revisit when: ownerが独自Web UIを新しい提供範囲として承認したとき。

## 実装前に残る事項

- Status: Incomplete
- Missing evidence: Bot構文・platform別メンション受信/owner限定返信、Bot配置、受付後処理・配送失敗対策、短期状態store、旧コード退役、Cloud Run公開設定。API側の認証/数値/応答/互換modeはDR-0021と読み取りAPI仕様で確定。
- Required decision: #312で技術案とテストを具体化し、認証・保存・運用の意味が変わる箇所をownerがレビューする。#318で環境と費用を検証する。すでに承認されたAPI提供・Web不要・返信フロー・予算通知方針は再び未決定に戻さない。

## 検証と文書更新

- #313: 独立clientからの契約test、未認証/非owner/期限切れ/失効、不正入力、固定clock/providerの推薦と障害test。旧Web廃止の差分と互換性を確認。
- #314: 地域表現・不正値・鮮度・未検証データのfixture test。架空fixtureを本番catalogへ混ぜない。
- #315/#316: 検索中→結果/エラー、API呼出し、署名/ID拒否、retry/二重配送、timeout、配送不能、履歴非依存・非保存test。
- #318: 公開前のowner/非owner E2E、実データ鮮度、log/metricsのprivacy、予算メール、rollback。外部変更は別途承認。
- 各実装PRで本書・OpenAPI・chat仕様・guide・operationsを更新する。[OpenAPI](facility-catalog.openapi.yaml)はAPI modeと保持したlegacy modeの契約を明示する。本番提供済みを意味しない。
