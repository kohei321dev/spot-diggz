# DR-0019: API利用clientと認証境界の初期構成

- Status: Accepted
- Date: 2026-09-06
- Type: Architecture
- Related Issues: [#312](https://github.com/kohei321dev/spot-diggz/issues/312)、後続#313〜#318
- Related Pull Requests: [#319](https://github.com/kohei321dev/spot-diggz/pull/319)（Merged）
- Affected Docs: `product.md`, `requirements.md`, `architecture.md`, `security.md`, `specifications/api-mvp.md`, `specifications/chat-integrations.md`, `operations/cloud-run-plan.md`, `guides/`, `operations/`（OpenAPIは後続実装PRで更新）
- Supersedes: ADR-0009・0013・0014の独自Web UI提供要件、ADR-0012のVercelを今後の公開先とする判断、ADR-0015のWeb session必須API境界・Discord固定条件、ADR-0016の内部推薦service呼出境界のみ。旧本文は保持し、実装移行は後続Issueで行う。
- Superseded By: [DR-0020](0020-mention-nearby-search.md)（新MVPの入力/検索の範囲のみ。API/Bot境界、認証、非保存、品質、ホスト、予算方針は維持）

## Context

DR-0018で地域を限定しない対象とAPI提供方針を採用した。2026-09-06のownerとの検討で、独立clientから呼べるAPI、Slack/Discordを入口とするMVP、独自Web UI不要、Cloud Runへの配置、予算通知と検索返信の方針が承認された。本記録のAcceptedはこれらの方針の採用を示し、認証方式など未確定の技術詳細や実装完了を示さない。

remote main `1b4797ce488961bf02572573999f71d0b1d7e9d1`ではWebはowner sessionでJSON APIを呼び、Slack/Discordは署名とowner ID認可後に内部serviceを呼ぶ。独立clientのAPI認証が完成しているとは言えない。

## Decision

1. 初期から独立bot/appが認証付きHTTP APIを呼べる構成（旧案B）を採用する。MVPの経路はSlack/Discord → 自分のBotサーバー → SpotDiggz API。BotとAPIの責務・認証境界を分け、物理的な配置数・リポジトリ分割は別途評価する。
2. 独自Web UIはMVPで提供しない。#317のWeb UI整備は対象外とする。既存asset・OAuth・UI専用APIの廃止範囲と互換性は#312/#313で確認し、今回コードや保存済みデータを削除しない。
3. API認証と許可済みownerのみの認可を維持する。Bot側のplatform署名・ID認可とAPI側のclient認証を分ける。具体的なcredential方式は未確定であり、単一API keyを本人確認の代わりにしない。
4. Slackは無料プランの通常AppをHTTPトリガーとして使う。処理は外部サーバーで行い、Slack-hosted workflow、Socket Mode、履歴監視を追加しない。Discordも明示的なinteractionを入口とする。
5. 位置・条件を入力して推薦を求める。対応スポットデータの追加は後続作業とし、今回の設計完了条件に実施設の追加を含めない。未検証施設を推薦しない原則を維持する。
6. 条件送信後は先に「検索中」を返し、処理完了後に結果、失敗時には利用者向けエラー通知を返す。HTTP ACKだけで検索中表示を満たしたとは扱わない。
7. SpotDiggz APIのホスト先はCloud Runとする。API管理はApigeeではなくownerが指定した「APIG」を利用したい方針。Google Cloud API Gatewayとして調査しているが、正式サービスの確認と回数・頻度・送信元IP制限の適合検証は未完了とする。
8. SpotDiggz全体の月額目安はUSD 3。Google Cloudの予算アラートをメール通知する。超過後も稼働を続け、予算超過を理由とした自動停止・課金無効化・推薦拒否を実装しない。追加料金が発生し得る。濫用対策や外部サービス固有の上限とは区別する。

採用済みの要求と未確定事項の正本は[MVP API提供契約](../specifications/api-mvp.md)、運用方針は[Cloud Run運用計画](../operations/cloud-run-plan.md)。[実装前評価](../research/api-client-contract-plan.md)は未承認の詳細案を分離して管理する。

## Rationale

APIを独立して呼べることがownerの要求であり、内部service共有だけでは満たせない。独自Web UIを省き、既存推薦engineと検証済み施設情報を再利用する。小規模利用の費用を観測しながら、予算超過による利用中断は避ける。全体の方針承認を、まだ提示していないcredential・queue・保存方式の承認へ読み替えない。

## Alternatives Considered

- A（不採用）: 内部service共有のみでは独立clientのAPI利用要求を満たさない。
- B（方針を採用）: 独立bot/appによるHTTP API利用に対応する。credential管理、失効・監査、互換性の責任が増えるため詳細は別途確定する。
- 独自Web UIの継続整備（不採用）: ownerが不要と判断した。API向け情報・安全性まで削除する理由にはしない。
- Apigee採用、予算超過による自動停止（不採用）: ownerのサービス訂正と通知のみの運用判断に従う。
- 自己HTTP呼出しと特別bypass header: 通信だけを増やす、または認可迂回を生むため採用しない。
- 共通API keyだけで全利用者を認可: 既存のowner限定・platform本人確認を代替できないため採用しない。

## Consequences

- #312は方針承認済みだが、認証・非同期実行・互換性等の詳細が未確定であり未完了とする。
- #313はAPIと旧Web境界の移行、#314は地域制約・品質の整合（実スポット追加は後続）、#315/#316はBotのAPI接続、#318はCloud Runと予算メール・公開条件の検証を担う。
- #317のWeb UI整備はNot plannedとし、完了扱いにしない。旧Issue本文は履歴を残す。
- #318の公開先・secret登録・deployは、この判断と別の外部変更承認を必要とする。
- 正規文書に承認済み方針を反映する。既存Web/Vercel仕様は移行前実装の説明と明示し、Cloud Run公開済みと扱わない。

## Security and Privacy

owner限定、署名・timestamp・ID検証、位置・本文非保存を維持する。資格情報をbrowser、chat本文、log、Gitへ配布しない。APIの資格情報・失効方式はIncompleteのままであり、未承認の方式を実装しない。受付後処理の耐久性のために位置や返信tokenをqueueへ保存する変更も自動的には許可しない。予算通知に本文・位置・secretを含めない。

## Migration and Rollback

現時点は文書とIssueの整合更新だけで、runtime、DB、catalog、deploy設定に移行はない。旧ADRの本文・ID・日付を保持し、部分置換のmetadataを追記する。編集ミスは文書commitをrevertし、Issueの旧本文は更新前控えと履歴から戻せる。採用した方針自体を変更する場合は新しいDecision Recordでownerの判断を残す。

## Validation

現行path・コード・認証・入力制約との対応表、境界・エラー・互換性のテスト表をレビューする。文書構造・内部link・Decision Record、JSON/OpenAPI契約、git diffを検証する。提案の文書検証と、後続runtimeの動作検証を混同しない。

## Revisit Conditions

具体的なAPI認証、Bot配置、非同期処理と短期状態保存、API Gatewayの正式選定・制限方式を確定するとき。対象user、独自Web UI、予算超過時の停止方針を変更するときは新しい採用判断としてレビューする。

## References

- [実装前評価と契約案](../research/api-client-contract-plan.md)
- [DR-0018](0018-api-first-product-definition.md)
- [ADR-0015](0015-owner-auth-and-chat-entrypoints.md)
- [ADR-0016](0016-slack-guided-recommendation.md)
- [Security](../security.md)
