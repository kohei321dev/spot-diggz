# API提供契約とclient境界の実装前評価

- Status: Proposed details; product direction accepted
- Date: 2026-09-06
- Issue: [#312](https://github.com/kohei321dev/spot-diggz/issues/312)
- Pull Request: [#319](https://github.com/kohei321dev/spot-diggz/pull/319)（提供/予算方針）、[#320](https://github.com/kohei321dev/spot-diggz/pull/320)（周辺検索入力の合意）
- Baseline: remote `main` `1b4797ce488961bf02572573999f71d0b1d7e9d1`
- Related decision: [DR-0019](../decisions/0019-api-client-boundary.md)、[DR-0020](../decisions/0020-mention-nearby-search.md)（方針Accepted、技術詳細Incomplete）
- Implementation authority: ownerは新計画の実装着手とdocs更新を依頼し、今回検討内容のdocs/Issue整合を指示した。未提示のcredential・保存方式、外部設定・merge・deployの承認とは分ける。

## 最新の適用範囲

PR #320はremote main `2f3982ea40ddade04770139d02b3ba26549448ac`へマージ済み。ownerの実装再開依頼に基づき[DR-0021](../decisions/0021-read-api-contract.md)で読み取りAPIのBearer/期限/失効/owner mapping、JSONの数値・応答・場所解決・品質filter、明示的API modeを具体化した。[読み取りAPI](../specifications/read-api.md)とOpenAPIがこの部分の正本であり、以下の「文書のみ」「認証/数値未確定」はその時点の調査記録として読む。

#312全体を閉じず、#313の第一実装単位だけを先行する。Bot文法/受信/owner限定返信/配送保証、旧コード退役、streetの欠落属性表現、実catalog分類、Cloud Run/Gatewayの実設定は未完了。新規保存・外部変更の権限は拡大しない。

## 決定済み方針と本書の役割

独立clientが呼べるAPI（旧案B）、Slack/Discord → 自分のBotサーバー → SpotDiggz API、独自Web UI不要、APIホストCloud Run、スポット追加は後続、検索中→結果/エラー通知、月額目安USD 3・メール通知のみ・予算超過で停止しない方針は承認された。

正本は[MVP API提供契約](../specifications/api-mvp.md)と[Cloud Run運用計画](../operations/cloud-run-plan.md)。本書は残る詳細案を扱い、同一アプリ内の内部service共有のみ（旧案A）を推奨しない。旧提案はGit履歴に保持する。

2026-09-07追記: PR #319は`b22fd4035c3aaf039955cfa8fdd226b5e0328e44`へマージ済み。今回の入力正本は[周辺検索仕様](../specifications/nearby-search.md)です。メンション＋場所名と任意genre/limit/sort・検索範囲は採用方向、limitの提案値・radius構文/範囲・返却schemaは詳細未確定として分けます。以下のコード観測は冒頭Baseline時点からruntime変更がない移行前の説明です。

## 観測した実装と変更対応表

| 対象 | 移行前コードの事実 | 再利用・変更方針 | Issue |
| --- | --- | --- | --- |
| 認証 | `internal/owneraccess/manager.go`がGitHub OAuthとowner sessionを検証 | owner限定を維持し独立API client認証を追加。旧Web OAuthを新MVPの必須方式にせず、廃止範囲は互換性とともに確認 | #312/#313 |
| 施設参照 | `internal/httpapi/server.go`に一覧JSONとID取得API | 施設情報APIを再利用。施設一覧画面やListsを作らない | #313 |
| 地点検索 | `POST /api/locations/search`と任意Google provider | query検証・非保存を再利用。不明な地点を推測しない | #313/#315/#316 |
| 推薦 | HTTP handlerとchatのServiceが同じengineを呼ぶ | 検証済みデータ・距離計算を再利用し、genre/範囲/limit/距離順の周辺検索へ。6条件の即時滑走判定を基本検索へ流用しない | #313/#315/#316 |
| 訂正・event | 既存Webがcorrections/events APIを使う | 機能・権限・データ・互換性を棚卸し。Web不要を理由に既存データを削除しない | #312/#313 |
| Slack | modal入力→地点解決→内部service、goroutine、短期状態store | メンション＋場所名/任意optionへ移行。通常Appの受信/署名/権限・検索中/owner限定返信とAPI接続を検証 | #315 |
| Discord | commandはserver側の固定起点・既定条件 | メンション＋場所名/任意optionへ移行。受信transportと既存HTTP Interaction専用制約の適合、真正性・owner限定返信を検証 | #316 |
| 地域 | OpenAPIとcatalog validatorに5府県の許可値 | ユーザー定義と分離しschema/validator/fixtureを整合する | #314 |
| 経験レベル | beginner/returning/intermediateの許可値 | ターゲット外と未対応入力を区別。未承認enumを足さない | #312/#313 |
| Web UI | 埋込asset、Web login、UI向けtestが残る | #317の整備を対象外とし、旧Web廃止の対象表と必要なAPI/CI回帰検証を#313で用意する | #313 |
| 公開先 | 旧Vercel設定と実公開先Incomplete | Cloud Runへ向けた構成・費用・通知・安全境界を確認。既存scriptを実行しない | #318 |

これは基準commitでのコード観測であり、本番稼働の証拠ではない。今回runtime・catalog・OpenAPIは変更しない。

## JSON API契約の詳細案

既存path、JSON、`error.code`/`error.message`、HTTP status、`X-Request-ID`、`Cache-Control: no-store`を基本にする。一括改名や根拠のないversion追加はしない。

| Method / path | 再利用案 | 未確定 |
| --- | --- | --- |
| GET /api/facilities | 検証済みcatalog参照。全件が推薦可能とは限らない | client権限・認証 |
| GET /api/facilities/{facilityId} | 施設事実・出典・確認時刻・利用条件、不存在404 | client権限・認証 |
| POST /api/locations/search | JSON query、地点候補、未設定/失敗503 | 曖昧な地点の確認方法 |
| POST /api/recommendations | 旧契約は6条件・origin、最大3件。新MVPは周辺検索であり互換性評価が必要 | 新検索path/field、genre/limit/sort/任意範囲、0件/未収録/情報不足の識別field・文面 |
| POST /api/corrections | 既存の同意・保存・receipt契約を保護 | 新MVP公開範囲と必要な権限 |
| POST /api/events | allowlist済み最小集計 | 旧Web廃止後の必要性と公開範囲 |

enum・型・必須性の変更は互換性をレビューする。認証失敗はHTML loginではなく構造化errorにする。origin/queryのlog・store非保存を維持する。場所候補APIとR-008のresponse再掲禁止の差は未解決であり、場所確認に必要なAPI/Bot間データと人間向け表示を分けてレビューする。Googleへの送信はprivacy境界として説明する。

## 未確定の詳細と担当

| 項目 | 提案・確認内容 | 担当 |
| --- | --- | --- |
| API資格情報 | clientごとの発行・保存・期限・失効・更新・監査、owner対応付け。表示名や任意owner headerだけでは許可しない | #312/#313 |
| 入力 | メンション＋場所名、任意genre/limit/sort/検索範囲。構文詳細、limitの提案値、radius構文/既定値/上下限、広域地名の基準点を確定する。旧6条件必須には戻さない | #312/#313/#315/#316 |
| 返信・privacy | 応答field/必須性/不明値、地点確認と正確な位置の非再掲、platform別メンション受信/owner限定返信を確認する | #312/#315/#316 |
| 候補不足 | 条件を勝手に緩めず、未対応・根拠不足・条件不一致を区別して案内する案。返信code/文面は未確定 | #312/#313 |
| 地域 | 5府県固定を国内の正式な都道府県集合にする案。海外表現の対応済み宣言はしない | #314 |
| スポット | park/street分類・既存recordの互換性、streetの滑走可否根拠、検索と即時滑走の品質filterを詳細化。実施設追加は後続、実公開前の鮮度確認は維持 | #312/#314/#318 |
| 非同期処理 | Cloud RunのCPU・lifecycle、Bot配置、deadline・retry上限・冪等性・配送失敗を検証 | #312/#315/#316/#318 |
| 状態store | 既存の最小metadata/最大1時間を基準に、再起動をまたぐ要否と費用を評価。位置・token保存への変更は別判断 | #312/#318 |
| APIG | Google Cloud API Gatewayを調査候補とし、正式名称・client別回数/頻度/IP制限の適合、直アクセス迂回防止を確認 | #312/#318 |

- Status: Incomplete
- Missing evidence: 上表の詳細schema、方式選定とテスト結果、公開環境と費用の実測。
- Required decision: 技術案を具体化し、未提示の認証・保存・外部サービス変更をownerがレビューする。方針の再承認や実装着手許可の取り直しと混同しない。

## 安全境界・回帰テスト表

| Case | 必須結果 | Issue |
| --- | --- | --- |
| API資格なし、不正/期限切れ/失効資格、偽owner | 拒否。ブラウザcookieや申告headerだけで通さない | #313 |
| Production設定欠落・bypass | fail closed。Web認証を外すだけの匿名公開はしない | #313/#318 |
| platform署名改ざん・期限外・ID欠落/不一致 | 重い処理前に拒否。署名・IDをlogへ出さない | #315/#316 |
| 不正JSON・body過大・不明field・不正座標/enum | 安定した4xx。入力を診断logへ転載しない | #313 |
| 同じ入力・catalog・固定clock/provider | 同じ順位・根拠を返す | #313/#315/#316 |
| genre/limit/検索範囲/距離sort | 指定範囲と件数を別々に適用し、近い順に返す。不正/未知optionを無視しない | #313/#315/#316 |
| 曖昧な場所・Bot自身の返信・非ownerメンション | 場所を勝手に確定せず、再帰起動や無認可検索・公開返信をしない | #315/#316 |
| 0件・stale・未検証・一般利用未確認・休場 | 推測で候補を作らず、障害と正常0件を区別 | #313/#314 |
| Google未設定・timeout・障害 | 地点検索利用不能と経路概算を区別 | #313/#315/#316 |
| 受付→成功/失敗 | 検索中表示の後に結果または安全なエラーと次の行動 | #315/#316 |
| retry・連打・process停止・返信API障害 | 重複抑止、期限、配送不能の観測・復旧を検証。無限retryをしない | #315/#316/#318 |
| log/metrics/trace/store検査 | 位置・query・本文・secret・個人IDを残さない | #313〜#318 |
| 旧Web廃止 | 未使用route/asset/OAuth/testの範囲を確認し必要なAPI保護・回帰testを維持 | #313 |
| 費用通知 | USD 3目安とメール通知を確認し、超過による自動停止がない | #318 |

## 実装・文書更新の順序

1. #312: 採用済み方針に沿って技術詳細・安全境界・互換性・受入testを確定する。
2. #313/#314: APIと旧Web境界、地域制約と品質を整合する。実スポット追加は別の後続作業。
3. #315/#316: BotをAPIへ接続し、入力・検索中・結果/エラー返信を検証する。
4. #318: Cloud Run公開条件、費用・メール通知、実データ・owner/非owner E2E、運用を確認する。

#317はWeb UI不要の判断でNot planned。#318の環境調査は先行できるが、公開判定は対象実装と別途の外部変更承認後に行う。各実装PRでrequirements、OpenAPI、chat仕様、architecture/security、guide、operationsを同時更新する。
