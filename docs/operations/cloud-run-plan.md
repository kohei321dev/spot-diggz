# Cloud Run運用・費用計画

- Status: Accepted direction; configuration incomplete
- Last reviewed: 2026-09-06
- Decision: [DR-0019](../decisions/0019-api-client-boundary.md)
- Issues: [#312](https://github.com/kohei321dev/spot-diggz/issues/312)、[#318](https://github.com/kohei321dev/spot-diggz/issues/318)

## 決定済み事項

- APIホスト先はCloud Run。Slack/DiscordのBotがAPIを呼び、独自Web UIを提供しない。
- API管理はApigeeではなく「APIG」を使いたいというownerの希望。Google Cloud API Gatewayを調査候補とするが、正式サービス名の確認と要件適合検証を残す。
- SpotDiggz全体の月額目安はUSD 3。API Gatewayの料金だけの予算ではない。
- Google Cloudの予算アラートをメールで通知する。通知先の実アドレスは公開文書・Issueへ記録せず、設定時に安全に確認する。
- 予算超過時も稼働を継続する。自動停止、課金無効化、予算起因の推薦拒否、spend capは設定しない。追加料金が発生し得ることをownerが了承している。
- 通常の認証・濫用防止・入力制限は別目的として維持する。「停止しない」は認可拒否・障害・provider固有上限を無効にする意味ではない。

## 費用の確認範囲

API Gateway、Bot/APIのCloud Run、ネットワーク、DB/短期状態、image保存/build、secret/log/監視、Google Maps等の外部APIを含めて見積もる。USD 3以内の実現はまだ未検証。無料枠を他projectで使用している可能性も確認する。

公式料金ではAPI Gatewayは請求先アカウント当たり月200万callまでリクエスト料金が無料で、それを超え10億callまでは100万call当たりUSD 3。通信料は別である。[API Gateway料金](https://cloud.google.com/api-gateway/pricing)

Cloud Runの無料枠は請求先アカウント内で集計され、実行時間・メモリ・region・課金方式で費用が変わる。無料枠があるだけで全体無料とは扱わない。[Cloud Run料金](https://cloud.google.com/run/pricing)

通常のalerts-only budgetは利用額の上限ではなく、通知にも使用量報告の遅延があり得る。本計画は通知のみを選び、USD 3ぴったりで止めることを約束しない。[予算と通知](https://docs.cloud.google.com/billing/docs/how-to/budgets)

## 技術検証が必要な事項

| 項目 | 状態・確認内容 | 担当Issue |
| --- | --- | --- |
| API認証とGateway迂回防止 | credential方式、権限、Cloud Run直アクセスの制限を設計・検証する | #312/#313/#318 |
| 回数・頻度・送信元IP | Gatewayのサービス上限と独自のclient別制限を混同しない。API keyのIP制限、実際のBot出口IP、必要な固定IP費用を検証する。IPを本人確認の代用にしない | #312/#318 |
| 受付後の処理 | 「検索中→結果/エラー」の実行・配送・retry・停止時を設計する。ACK後のgoroutineだけで完了保証とはしない | #312/#315/#316/#318 |
| 配置・課金・scale | Bot/APIの配置、region、最小/最大instance、CPU・メモリ、課金方式を試算。最小0・リクエスト課金は候補であり確定設定ではない | #318 |
| 永続状態 | 既存Neon/PostgreSQLの必要範囲・保持・費用を確認。DB削除、queueへの位置/token保存は未承認 | #312/#318 |
| メール通知 | budget scope、USD 3の目安、受信者、通知threshold、通知確認手順。実課金を発生させてtestしない | #318 |
| 公開・運用 | project、公開URL、IAM、secret、deploy/rollback、鮮度、監視を確認する | #318 |

Cloud Runのリクエスト課金では処理中にCPUを割り当てるため、HTTP ACK後の処理には実行方式の確認が必要。instance課金でもprocess停止への耐久性を保証しない。queue等を採用する場合は費用だけでなく保存対象・期限と既存非保存契約もレビューする。[Cloud Run課金とbackground処理](https://docs.cloud.google.com/run/docs/configuring/billing-settings)

Gatewayの[API key制限](https://docs.cloud.google.com/api-gateway/docs/authenticate-api-keys)と[割り当て](https://docs.cloud.google.com/api-gateway/docs/quotas)を基に検証する。独自の利用回数・頻度・IP制限をGateway単体で満たせるとはまだ断定しない。

## 設定・公開のゲート

- Status: Incomplete
- Missing evidence: 実project・請求先・URL・region、Gateway正式選定、予算メール受信設定、見積もり、非同期処理とIAMの検証、外部設定変更の承認。
- Required decision: #312の設計と#318のread-only環境評価を経て、ownerが実際の外部設定・課金・secret登録・deployを別途承認する。

このPRは方針を記録するだけで、budgetや通知を作成していない。旧Vercel向けscript・guideはCloud Runのセットアップに流用しない。公開状態は[service-status.md](service-status.md)を参照する。
