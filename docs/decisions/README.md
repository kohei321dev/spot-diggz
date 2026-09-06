# SpotDiggz Decision Records

- Status: Current index

このディレクトリは、Product、Requirement、Specification、Architecture、Security、Process、Operationに関する恒久的な判断理由を管理します。既存ADRはID、日付、status、本文を維持し、追跡metadataと正規文書へのlinkだけを補っています。

## Index

| ID | Status | Date | Type | Title | Issue | Pull Request | Superseded by |
| --- | --- | --- | --- | --- | --- | --- | --- |
| [ADR-0001](0001-repository-strategy.md) | Superseded | 2026-07-12 | Architecture | リポジトリと旧実装の扱い | Incomplete | Incomplete | ADR-0006 |
| [ADR-0002](0002-facility-data-source-and-freshness.md) | Accepted | 2026-07-14 | Requirement | 施設データの出典と鮮度管理 | #278 | #286, #291 | ADR-0011（地理scopeのみ） |
| [ADR-0003](0003-recommendation-engine-before-ai.md) | Accepted | 2026-07-12 | Architecture | AI導入前にルールベース推薦を作る | Incomplete | #291 | — |
| [ADR-0004](0004-localization-strategy.md) | Accepted | 2026-07-12 | Product | 多言語対応の開始範囲 | Incomplete | #291 | — |
| [ADR-0005](0005-main-branch-migration.md) | Accepted | 2026-07-12 | Process | mainブランチへの移行 | Incomplete | Incomplete | — |
| [ADR-0006](0006-remove-legacy-implementation.md) | Accepted | 2026-07-12 | Architecture | 旧実装を現行ツリーから削除する | Incomplete | Incomplete | — |
| [ADR-0007](0007-go-modular-monolith-runtime.md) | Accepted | 2026-07-16 | Architecture | Go製モジュラーモノリスを初期runtimeに採用 | Incomplete | #288, #291 | DR-0017（container scanの通常CI運用のみ） |
| [ADR-0008](0008-facility-catalog-api-and-storage.md) | Accepted | 2026-07-16 | Architecture | 施設カタログの初期APIと保存方式 | Incomplete | #291 | — |
| [ADR-0009](0009-session-recommendation-ui.md) | Accepted | 2026-07-16 | Specification | 選択式session検索とWeb UI | Incomplete | #291 | ADR-0010・0011（各一部） |
| [ADR-0010](0010-google-maps-provider-and-fallback.md) | Accepted | 2026-07-20 | Architecture | Optional Google Mapsとfallback | Incomplete | #291 | — |
| [ADR-0011](0011-five-prefecture-mvp-scope.md) | Accepted | 2026-07-20 | Product | MVPの地理scopeを5府県へ拡大 | Incomplete | #291 | DR-0018（利用者・Productの地理scopeのみ） |
| [ADR-0012](0012-vercel-neon-deployment.md) | Accepted | 2026-07-20 | Architecture | Vercel ContainerとNeonによるMVP公開 | Incomplete | #291 | — |
| [ADR-0013](0013-curated-external-media.md) | Accepted | 2026-07-21 | Specification | 手動選定YouTubeと公式SNS導線 | Incomplete | #292, #293, #304 | ADR-0014（一部） |
| [ADR-0014](0014-progressive-facility-details.md) | Accepted | 2026-07-22 | Specification | 施設補助情報とYouTubeを明示操作後に表示 | Incomplete | #304 | — |
| [ADR-0015](0015-owner-auth-and-chat-entrypoints.md) | Accepted | 2026-08-01 | Security | GitHub owner認証とSlack・Discord入口 | Incomplete | #304 | ADR-0016（Slack flowのみ） |
| [ADR-0016](0016-slack-guided-recommendation.md) | Accepted | 2026-08-03 | Specification | Slack条件入力と推薦応答 | Incomplete | #304 | — |
| [DR-0017](0017-minimal-development-ci.md) | Accepted | 2026-09-05 | Operation | 通常CIとリリース前検証を分ける | #307 | #308 | — |
| [DR-0018](0018-api-first-product-definition.md) | Accepted | 2026-09-06 | Product | 地域を限定せずAPIを中心とするプロダクトとして定義する | owner直接依頼 | [#311](https://github.com/kohei321dev/spot-diggz/pull/311) | — |
| [DR-0019](0019-api-client-boundary.md) | Proposed | 2026-09-06 | Architecture | API利用clientと認証境界の初期構成 | #312 | [#319](https://github.com/kohei321dev/spot-diggz/pull/319)（Draft） | — |

## Supersession map

- ADR-0006はADR-0001の「旧実装を初期段階では削除しない」判断を置換しました。
- ADR-0010はADR-0009のorigin/provider判断を置換しました。
- ADR-0011はADR-0002とADR-0009の地理scopeだけを置換しました。
- ADR-0014はADR-0013のDecision 4と初期iframe load判断だけを置換しました。
- ADR-0016はADR-0015のSlack入力・response flowだけを置換し、Discordの既定値設計は維持します。
- DR-0017はADR-0007のcontainer scanを通常CIで必須とする運用だけを置換し、Go runtimeと基本検証を維持します。
- DR-0018はADR-0011の利用者検証・Productの地理scopeだけを置換します。既存catalogとschema・validatorの地域制約は別の実装変更まで維持します。

## When to write

複数の妥当な選択肢とtrade-offがある、後からの変更costが高い、Product scope・要求・外部仕様・責務境界・data・security・運用が変わる、または複数Issueやreleaseへ影響する場合にDecision Recordを作成します。

新規記録は[`TEMPLATE.md`](TEMPLATE.md)を使い、`DR-NNNN: Title`とします。既存判断を上書きせず、変更時は新しいrecordを作って旧recordの`Superseded By`を更新します。
