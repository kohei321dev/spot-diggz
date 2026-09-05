# DR-0018: 地域を限定せずAPIを中心とするプロダクトとして定義する

- Status: Accepted
- Date: 2026-09-06
- Type: Product
- Related Issues: ownerからの直接の定義訂正依頼。対応Issueは未作成。
- Related Pull Requests: [#311](https://github.com/kohei321dev/spot-diggz/pull/311)
- Affected Docs: `product.md`, `requirements.md`, `architecture.md`, `README.md`, `specifications/facility-data.md`, `guides/how-to-use.md`
- Supersedes: [ADR-0011](0011-five-prefecture-mvp-scope.md)の利用者検証・Productの地理scopeを5府県に限定する判断のみ。既存catalogとschema・validatorの地域制約は実装変更まで維持する。
- Superseded By: None

## Context

ownerから、ターゲットは関西などの地域を決めず「スケボーをしたい人」であり、UIやSlack・Discordのbot・appなどが呼び出せるAPIとして提供できるように整備することがプロダクト定義だと訂正された。

既存Product文書は、5府県の初心者を主対象とし、既存Web UIやcatalogの範囲をProductそのものの範囲と混同していた。remote `main` `0a8cd3ba38cd67654c2bde0c4e4d2fc0d75302f9`にはWeb向けHTTP APIとchat adapterがあるが、後者は内部の共通推薦serviceを呼ぶ。API-firstの提供方針と現在の実装状態を分離する必要がある。

## Decision

1. ターゲットは「スケボーをしたい人」とし、地域や経験レベルでは限定しない。
2. 施設情報と条件に合う推薦を、UIやSlack・Discordのbot・appなどから利用できるAPIとして提供できるように整備する。
3. 特定の画面やchatサービスをプロダクトそのものとは定義しない。
4. 現行の収録地域、条件の許可値、private owner認証を、対象ユーザーの定義と区別する。全地域・全条件への対応済みや匿名公開を意味しない。
5. 今回はProduct方針と関連文書の訂正だけを行う。API契約、client向け認証・認可、実装の優先順序は別Issueと必要なDecision Recordで明確化する。コード、データ、デプロイ設定は変更しない。

## Rationale

現在のデータ収録範囲を固定的な市場の制限にせず、利用者の目的と、各入口から共有できる施設情報・推薦を中心に定義するため。既存の入口をそのまま将来のAPI利用形態と説明すると、整備済みの範囲を誤認させるため。

## Alternatives Considered

- 5府県・初心者中心のWebサービスという記述を維持する: ownerが示した対象と提供方針に合わないため採用しない。
- 全地域対応や各bot・appのAPI接続を実装済みと記載する: データとコードの証拠がないため採用しない。
- 方針と同時に地域制約・認証・構成を変更する: 安全境界と受入条件が未確定で、今回の文書訂正の範囲を超えるため採用しない。

## Consequences

- 現行文書は対象ユーザー、提供方針、既存実装、データ収録範囲を分ける。
- ADR-0011のID、日付、決定本文は保持し、利用者・Productの地理scopeだけの部分置換をmetadataとindexへ記録する。
- 収録範囲や入力条件の拡張にはデータ、schema、validator、testの整合が必要であり、今回の文書変更では解除しない。
- 過去の需要検証計画は達成済みとせず、地域を限定しない対象とAPI利用の入口に合わせて方法を再確認する。

## Security and Privacy

owner認証・platform署名とID認可・位置情報の非保存を維持する。API提供の方針は匿名利用、複数userへの開放、secretのclient配布を許可しない。client向けの安全境界は実装前に別途決定する。

## Migration and Rollback

文書だけを同じPull Requestで更新し、旧ADR本文を残す。編集ミスは対象commitのrevertで戻せるが、Product方針自体を戻す場合はownerの再判断と新しいDecision Recordが必要。データ移行・デプロイはない。

## Validation

- Product、requirements、architecture、guide、READMEで地域をターゲットの制限と書いていないことを確認する。
- 既存の地域許可値・owner認証・chat adapterの内部呼出しを、API-first構成の完成と混同していないことをコードと照合する。
- 文書構造・内部link・Decision Record検証、契約検証、`git diff --check`を行う。
- コード、catalog、OpenAPI、依存、CI・デプロイ設定が差分に含まれないことを確認する。

## Revisit Conditions

API利用client、認証・認可、対応条件、収録データ、公開環境の実装範囲を決めるとき。ターゲット自体を変更する場合は新しい判断として記録する。

## References

- ownerのProduct定義訂正（2026-09-06）。会話全文や個人情報は転載しない。
- [Product](../product.md)
- [Requirements](../requirements.md#product-direction-and-implementation-boundary)
- [Current architecture](../architecture.md)
- [ADR-0011](0011-five-prefecture-mvp-scope.md)
