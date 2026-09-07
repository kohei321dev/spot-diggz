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
| [ADR-0009](0009-session-recommendation-ui.md) | Accepted | 2026-07-16 | Specification | 選択式session検索とWeb UI | Incomplete | #291 | ADR-0010・0011、DR-0019、DR-0020（各一部） |
| [ADR-0010](0010-google-maps-provider-and-fallback.md) | Accepted | 2026-07-20 | Architecture | Optional Google Mapsとfallback | Incomplete | #291 | — |
| [ADR-0011](0011-five-prefecture-mvp-scope.md) | Accepted | 2026-07-20 | Product | MVPの地理scopeを5府県へ拡大 | Incomplete | #291 | DR-0018（利用者・Product）、DR-0022（schema・validator） |
| [ADR-0012](0012-vercel-neon-deployment.md) | Accepted | 2026-07-20 | Architecture | Vercel ContainerとNeonによるMVP公開 | Incomplete | #291 | DR-0019（新MVPの提供境界のみ） |
| [ADR-0013](0013-curated-external-media.md) | Accepted | 2026-07-21 | Specification | 手動選定YouTubeと公式SNS導線 | Incomplete | #292, #293, #304 | ADR-0014（一部）、DR-0019（新MVPの提供境界のみ） |
| [ADR-0014](0014-progressive-facility-details.md) | Accepted | 2026-07-22 | Specification | 施設補助情報とYouTubeを明示操作後に表示 | Incomplete | #304 | DR-0019（新MVPの提供境界のみ） |
| [ADR-0015](0015-owner-auth-and-chat-entrypoints.md) | Accepted | 2026-08-01 | Security | GitHub owner認証とSlack・Discord入口 | Incomplete | #304 | ADR-0016（Slack flowのみ）、DR-0019（新MVPの提供境界のみ） |
| [ADR-0016](0016-slack-guided-recommendation.md) | Accepted | 2026-08-03 | Specification | Slack条件入力と推薦応答 | Incomplete | #304 | DR-0019（提供境界）、DR-0020（入力/候補数） |
| [DR-0017](0017-minimal-development-ci.md) | Accepted | 2026-09-05 | Operation | 通常CIとリリース前検証を分ける | #307 | #308 | — |
| [DR-0018](0018-api-first-product-definition.md) | Accepted | 2026-09-06 | Product | 地域を限定せずAPIを中心とするプロダクトとして定義する | owner直接依頼 | [#311](https://github.com/kohei321dev/spot-diggz/pull/311) | — |
| [DR-0019](0019-api-client-boundary.md) | Accepted | 2026-09-06 | Architecture | API利用clientと認証境界の初期構成 | #312 | [#319](https://github.com/kohei321dev/spot-diggz/pull/319)（Merged） | DR-0020（入力/検索）、DR-0021（API認証の具体化） |
| [DR-0020](0020-mention-nearby-search.md) | Accepted | 2026-09-07 | Specification | メンションと場所名による周辺スポット検索をMVPにする | #312 | [#320](https://github.com/kohei321dev/spot-diggz/pull/320) | DR-0021（API数値/応答等のみ） |
| [DR-0021](0021-read-api-contract.md) | Accepted | 2026-09-07 | Architecture | 読み取りAPIのBearer認証と周辺検索の最小契約 | #313、#312、#314 | [#321](https://github.com/kohei321dev/spot-diggz/pull/321) | DR-0022（地域・営業時間状態の具体化） |
| [DR-0022](0022-domestic-catalog-quality.md) | Accepted | 2026-09-07 | Specification | 国内の地域表現と営業時間の確認状態を分離する | #314、#312、#313、#318 | [#322](https://github.com/kohei321dev/spot-diggz/pull/322) | — |

## Supersession map

- ADR-0006はADR-0001の「旧実装を初期段階では削除しない」判断を置換しました。
- ADR-0010はADR-0009のorigin/provider判断を置換しました。
- ADR-0011はADR-0002とADR-0009の地理scopeだけを置換しました。
- ADR-0014はADR-0013のDecision 4と初期iframe load判断だけを置換しました。
- ADR-0016はADR-0015のSlack入力・response flowだけを置換し、Discordの既定値設計は維持します。
- DR-0017はADR-0007のcontainer scanを通常CIで必須とする運用だけを置換し、Go runtimeと基本検証を維持します。
- DR-0018はADR-0011の利用者検証・Productの地理scopeだけを置換しました。残したschema・validatorの制約はDR-0022で変更し、既存catalogの実データは維持します。

- DR-0019は独立API、Bot入口、Cloud Run、独自Web UI不要、検索中→結果/エラー、月額目安USD 3・メール通知のみ・予算超過時停止なしを採用します。ADR-0009/0013/0014の独自Web UI提供、ADR-0012の今後のAPIホスト、ADR-0015の新APIのWeb session必須とDiscord固定条件、ADR-0016の呼出・返信境界だけを部分置換します。owner限定、非保存、施設品質、旧実装の安全性は維持し、具体的な認証・保存・非同期方式はIncompleteです。

DR-0020はADR-0009の新MVPの6条件必須・即時滑走推薦/固定3件、ADR-0016のslash/modal入口・mention非対応と固定候補数、DR-0019の入力/検索の範囲を部分置換します。旧record本文とIDは維持し、認証・非保存・施設品質・Cloud Run・費用方針は変更しません。検索範囲の指定可能方針と、その構文/既定値/上限の未確定を区別します。

DR-0021はDR-0019の読み取りAPI認証方式、DR-0020のAPI数値/応答/場所解決/品質filterを具体化します。旧record本文は維持し、Bot/公開/旧コード退役を実装完了としません。

DR-0022はADR-0011の5府県validator制約と、DR-0021の後続事項である地域表現・streetの営業時間未定義表現を具体化します。47都道府県の受理と実収録範囲、営業時間非該当と不明、周辺検索と即時滑走判定を分離し、確認責任と公開前gateを定めます。実施設の追加・確認日時の変更・公開は含めません。

## When to write

複数の妥当な選択肢とtrade-offがある、後からの変更costが高い、Product scope・要求・外部仕様・責務境界・data・security・運用が変わる、または複数Issueやreleaseへ影響する場合にDecision Recordを作成します。

新規記録は[`TEMPLATE.md`](TEMPLATE.md)を使い、`DR-NNNN: Title`とします。既存判断を上書きせず、変更時は新しいrecordを作って旧recordの`Superseded By`を更新します。
