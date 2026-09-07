# MVP API提供契約

- Status: Accepted direction; implementation incomplete
- Last reviewed: 2026-09-06
- Decision: [DR-0019](../decisions/0019-api-client-boundary.md)
- Tracking: [#312](https://github.com/kohei321dev/spot-diggz/issues/312)、後続#313〜#316、#318

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

- Botはplatform署名・timestampと設定済みownerのIDを検証する。API側でもclient認証・権限を検証し、未設定時はfail closedにする。
- API keyによる呼出元識別やIP制限を、利用者本人の認可と同一視しない。credentialの具体方式、発行・期限・失効・更新は未確定。
- 利用者が出発地・位置と行きたい条件を入力する。端末の現在地を自動取得できるとは約束しない。必須項目、既定値、地点候補の曖昧性解消、Discordのoption/modal選択は#312/#316で詳細化する。
- 検証済みデータと共通engineを再利用し、AIで施設事実を補わない。既存の最大3件、出典・確認時刻、鮮度と休場判定を維持する。
- 実スポットデータの追加は後続作業。#314では地域固定制約・品質検証・更新手順を整え、実施設の追加件数や特定施設の再調査を今回の必須成果にしない。実運用前の鮮度確認は免除しない。
- 候補、会話履歴、正確な位置、地点query、返信tokenを永続保存しない。保存Listsは設けない。短期状態の既存制約を変更する場合は明示的な設計レビューが必要。

## 承認済み返信フロー

1. 明示的なcommandをトリガーに条件を入力する。Slackは通常AppのHTTP連携とmodalを基本にし、無料プランでの利用を前提とする。Slack-hosted workflow、Socket Mode、過去message監視を追加しない。
2. 署名・認可・入力の検証後、条件送信に対し先に「検索中」を表示する。platformへのHTTP ACKと利用者が見える受付表示の両方を検証する。
3. Botが認証付きAPIへ条件を渡し、完了後にownerだけへ結果を返す。利用者にWeb UIを開かせない。
4. 処理失敗時も利用者向けエラーと再実行など次の行動を返す。内部診断、入力本文、位置、secretは返信しない。

候補0件は処理障害と区別する。未収録・情報不足・条件不一致を区別する具体的なresponse code、条件変更案、返信文面はまだ設計案であり、#312で確定する。条件の自動緩和や未確認情報の補完を実装済みとしない。

platform障害などでエラー通知自体を配送できない場合もあるため、「必ず届く」とは保証しない。配送失敗の観測、retry上限、重複抑止、timeout、process停止時の扱いを実装前に決める。

## 要求IDの適用移行

既存IDは振り直さない。[Requirements](../requirements.md)の移行前受入条件に対して、DR-0019による以下の変更を適用する。

| 対象 | 新MVPでの扱い | 残る確認 |
| --- | --- | --- |
| R-001〜R-005、R-007、R-009〜R-012、R-015〜R-016 | 入力・推薦・出典・日英・外部導線・品質・安全性をAPI/Botへ引き継ぐ | JSON field・文面・互換性の詳細 |
| R-006 | 訂正機能・保存データを今回削除しない | APIの公開範囲・権限・Botからの導線を#312で決定 |
| R-008、R-019〜R-020 | 位置・履歴・候補の非保存を維持 | 非同期処理が保存制約に反しないこと |
| R-013〜R-014のWeb操作領域・iframe表示、NFR-005の独自Web E2E | 独自Web UI向け提供要件は対象外。Botの読みやすさ・日英・安全な導線は維持 | 旧UIコード・テストの廃止範囲を#313で確認 |
| R-017 | Web OAuth cookie必須を新API client認証の必須方式にしない。owner限定は維持 | 認証方式、旧OAuthの廃止範囲 |
| R-018 | Slack/Discordで位置・条件入力→検索中→API結果またはエラー通知 | Discordの具体的入力方式、非同期実行・配送 |

独自Web UIの新規提供:

- Status: Not applicable
- Reason: ownerがMVPはSlack/DiscordからAPIを利用し、独自Web UIは不要と決定したため。
- Revisit when: ownerが独自Web UIを新しい提供範囲として承認したとき。

## 実装前に残る事項

- Status: Incomplete
- Missing evidence: API認証方式、ownerへの対応付け、権限・失効、入力/responseの詳細schema、旧API互換性、Bot配置、受付後処理の実行保証・配送失敗対策、短期状態store、Cloud Run公開設定。
- Required decision: #312で技術案とテストを具体化し、認証・保存・運用の意味が変わる箇所をownerがレビューする。#318で環境と費用を検証する。すでに承認されたAPI提供・Web不要・返信フロー・予算通知方針は再び未決定に戻さない。

## 検証と文書更新

- #313: 独立clientからの契約test、未認証/非owner/期限切れ/失効、不正入力、固定clock/providerの推薦と障害test。旧Web廃止の差分と互換性を確認。
- #314: 地域表現・不正値・鮮度・未検証データのfixture test。架空fixtureを本番catalogへ混ぜない。
- #315/#316: 検索中→結果/エラー、API呼出し、署名/ID拒否、retry/二重配送、timeout、配送不能、履歴非依存・非保存test。
- #318: 公開前のowner/非owner E2E、実データ鮮度、log/metricsのprivacy、予算メール、rollback。外部変更は別途承認。
- 各実装PRで本書・OpenAPI・chat仕様・guide・operationsを更新する。現在の[OpenAPI](facility-catalog.openapi.yaml)は移行前実装の契約で、新API認証は未反映。
