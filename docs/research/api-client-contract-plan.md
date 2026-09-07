# API提供契約とclient境界の実装前評価

- Status: Proposed details; product direction accepted
- Date: 2026-09-06
- Issue: [#312](https://github.com/kohei321dev/spot-diggz/issues/312)
- Pull Request: [#319](https://github.com/kohei321dev/spot-diggz/pull/319)
- Baseline: remote `main` `1b4797ce488961bf02572573999f71d0b1d7e9d1`
- Related decision: [DR-0019](../decisions/0019-api-client-boundary.md)（方針Accepted、技術詳細Incomplete）
- Implementation authority: ownerは新計画の実装着手とdocs更新を依頼し、今回検討内容のdocs/Issue整合を指示した。未提示のcredential・保存方式、外部設定・merge・deployの承認とは分ける。

## 決定済み方針と本書の役割

独立clientが呼べるAPI（旧案B）、Slack/Discord → 自分のBotサーバー → SpotDiggz API、独自Web UI不要、APIホストCloud Run、スポット追加は後続、検索中→結果/エラー通知、月額目安USD 3・メール通知のみ・予算超過で停止しない方針は承認された。

正本は[MVP API提供契約](../specifications/api-mvp.md)と[Cloud Run運用計画](../operations/cloud-run-plan.md)。本書は残る詳細案を扱い、同一アプリ内の内部service共有のみ（旧案A）を推奨しない。旧提案はGit履歴に保持する。

## 観測した実装と変更対応表

| 対象 | 移行前コードの事実 | 再利用・変更方針 | Issue |
| --- | --- | --- | --- |
| 認証 | `internal/owneraccess/manager.go`がGitHub OAuthとowner sessionを検証 | owner限定を維持し独立API client認証を追加。旧Web OAuthを新MVPの必須方式にせず、廃止範囲は互換性とともに確認 | #312/#313 |
| 施設参照 | `internal/httpapi/server.go`に一覧JSONとID取得API | 施設情報APIを再利用。施設一覧画面やListsを作らない | #313 |
| 地点検索 | `POST /api/locations/search`と任意Google provider | query検証・非保存を再利用。不明な地点を推測しない | #313/#315/#316 |
| 推薦 | HTTP handlerとchatのServiceが同じengineを呼ぶ | engineを再利用しBotを認証付きHTTP APIへ接続する | #313/#315/#316 |
| 訂正・event | 既存Webがcorrections/events APIを使う | 機能・権限・データ・互換性を棚卸し。Web不要を理由に既存データを削除しない | #312/#313 |
| Slack | modal入力→地点解決→内部service、goroutine、短期状態store | 通常App/HTTP入口・modal・署名を再利用。利用者に見える検索中、API結果/エラー、停止・配送失敗の検証を追加 | #315 |
| Discord | commandはserver側の固定起点・既定条件 | 明示的な位置・条件入力とAPI接続へ移行。option/modalの具体方式は未確定 | #316 |
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
| POST /api/recommendations | 条件・origin、最大3件、根拠 | 0件/未収録/情報不足の識別field・文面 |
| POST /api/corrections | 既存の同意・保存・receipt契約を保護 | 新MVP公開範囲と必要な権限 |
| POST /api/events | allowlist済み最小集計 | 旧Web廃止後の必要性と公開範囲 |

enum・型・必須性の変更は互換性をレビューする。認証失敗はHTML loginではなく構造化errorにする。正確なorigin/queryをlog・store・応答に再掲しない。Googleへの送信はprivacy境界として説明する。

## 未確定の詳細と担当

| 項目 | 提案・確認内容 | 担当 |
| --- | --- | --- |
| API資格情報 | clientごとの発行・保存・期限・失効・更新・監査、owner対応付け。表示名や任意owner headerだけでは許可しない | #312/#313 |
| 入力 | 既存6条件を基本に必須/既定値/範囲を確定。出発地を必須にする案、Discord option案は未承認 | #312/#316 |
| 候補不足 | 条件を勝手に緩めず、未対応・根拠不足・条件不一致を区別して案内する案。返信code/文面は未確定 | #312/#313 |
| 地域 | 5府県固定を国内の正式な都道府県集合にする案。海外表現の対応済み宣言はしない | #314 |
| スポット | 実施設の追加・特定施設の再調査は後続。fixtureで品質制約を検証し、実公開前の鮮度確認は維持 | #314/#318 |
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
