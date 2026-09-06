# DR-0019: API利用clientと認証境界の初期構成

- Status: Proposed
- Date: 2026-09-06
- Type: Architecture
- Related Issues: [#312](https://github.com/kohei321dev/spot-diggz/issues/312)、後続#313〜#318
- Related Pull Requests: [#319](https://github.com/kohei321dev/spot-diggz/pull/319)（Draft）
- Affected Docs: `requirements.md`, `architecture.md`, `security.md`, `specifications/facility-catalog.openapi.yaml`, `specifications/chat-integrations.md`, `guides/`, `operations/`
- Supersedes: None — 未承認。採用時にADR-0015のAPI認証境界、ADR-0016のchat呼出境界への影響を特定する。
- Superseded By: None

## Context

DR-0018で地域を限定しない対象とAPI提供方針を採用した。ownerは新計画#312〜#318の実装着手と関連docs更新を依頼した。一方、同一Goアプリのplatform HTTP入口を整備するか、独立bot/appへJSON API利用資格を発行するかは、まだ明示された採用判断がない。

remote main `1b4797ce488961bf02572573999f71d0b1d7e9d1`ではWebはowner sessionでJSON APIを呼び、Slack/Discordは署名とowner ID認可後に内部serviceを呼ぶ。独立clientのAPI認証が完成しているとは言えない。

## Decision

未決定。詳細な対応表、API契約案、テスト表は[実装前評価](../research/api-client-contract-plan.md)を参照する。

- 案A: 同一Goアプリ内でWeb向けJSON API、platformの署名付きHTTP入口、共通の型付き処理を整備する。既存認証を保ち、新しいclient資格情報を発行しない。
- 案B: 独立bot/appがJSON APIを呼ぶ構成を初回から提供し、client認証・owner対応付け・権限・期限・失効・更新の仕組みを別途設計する。

小規模owner利用と現行単一deploy構成を優先するならAを提案する。ただしAで独立clientのAPI利用要求を満たしたとしない。Bが必須ならその安全境界を決めてから実装する。

## Rationale

新しいclientへどの権限を委任するかは、単なるコード配置ではなく信頼境界の判断である。全体の実装着手依頼を、まだ提示していなかった資格情報管理方式の承認へ読み替えない。

## Alternatives Considered

- A: 既存GitHub owner認証とplatform署名を再利用できる。独立clientへの汎用API利用資格を提供できない制約がある。
- B: 独立bot/appによるHTTP API利用に対応できる。credential管理、なりすまし防止、失効・監査、互換性の責任が増える。
- 自己HTTP呼出しと特別bypass header: 通信だけを増やす、または認可迂回を生むため採用しない。
- 共通API keyだけで全利用者を認可: 既存のowner限定・platform本人確認を代替できないため採用しない。

## Consequences

- #312は設計案まで作成できるが、採用承認前に完了とは扱わない。
- #313〜#317は採用した境界に従い、既存engine・施設情報・必要なUIを再利用する。Aを採用する場合は独立HTTP clientを前提にしたIssue範囲を明示的に訂正する。
- #318の公開先・secret登録・deployは、この判断と別の外部変更承認を必要とする。
- 採用後、正規のrequirements・specifications・architecture・securityを同じPRで更新し、未承認のresearchを現行仕様として扱わない。

## Security and Privacy

どちらの案でもowner限定、署名・timestamp・ID検証、位置・本文非保存を維持する。資格情報をbrowser、chat本文、log、Gitへ配布しない。Bの資格情報・失効方式はIncompleteのままであり、未承認の方式を実装しない。

## Migration and Rollback

現時点は文書だけの提案。runtime、DB、catalog、deploy設定に移行はない。本提案を取り下げる場合はRejectedと理由を記録し、実装前評価は履歴として残せる。採用済みの旧ADR本文は変更していない。

## Validation

現行path・コード・認証・入力制約との対応表、境界・エラー・互換性のテスト表をレビューする。文書構造・内部link・Decision Record、JSON/OpenAPI契約、git diffを検証する。提案の文書検証と、後続runtimeの動作検証を混同しない。

## Revisit Conditions

ownerがA/Bを選択するとき。独立client、公開user、外部hosting、認証provider等を追加する場合は、新しい採用判断としてレビューする。

## References

- [実装前評価と契約案](../research/api-client-contract-plan.md)
- [DR-0018](0018-api-first-product-definition.md)
- [ADR-0015](0015-owner-auth-and-chat-entrypoints.md)
- [ADR-0016](0016-slack-guided-recommendation.md)
- [Security](../security.md)
