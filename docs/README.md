# SpotDiggz Documentation

- Standard-Version: 1
- Status: Current
- Product: SpotDiggz

このディレクトリは、SpotDiggzの現在の契約と、恒久的な判断理由を管理するSource of Truthです。調査資料とUI案は`research/`、現行ではない計画は`archive/`に分離しています。

## Reading order

1. [`product.md`](product.md): なぜ、誰のために、何を作るか
2. [`requirements.md`](requirements.md): 何を満たす必要があるか
3. [`specifications/`](specifications/README.md): 外部から観測できる振る舞い
4. [`architecture.md`](architecture.md): どのように構成するか
5. [`security.md`](security.md): 守る対象と安全境界
6. [`decisions/`](decisions/README.md): なぜその判断を選んだか
7. [`process/`](process/development.md): 変更をどう進めるか
8. [`operations/`](operations/README.md): どう運用・復旧するか
9. [`guides/`](guides/README.md): どう利用・設定するか

## Document map

| Path | Responsibility | Status | Update trigger |
| --- | --- | --- | --- |
| [`product.md`](product.md) | 利用者、課題、価値、scope、non-goal、現在の状態 | Current | Product目的またはscopeの変更 |
| [`requirements.md`](requirements.md) | 機能・非機能要件、制約、release gate | Current | 実現必須事項または受入条件の変更 |
| [`specifications/`](specifications/README.md) | UI、API、data、chat連携の外部仕様 | Current index | 観測可能な振る舞いの変更 |
| [`architecture.md`](architecture.md) | component、data flow、provider、deployment境界 | Current | 内部責務または技術構成の変更 |
| [`security.md`](security.md) | 認証、認可、secret、privacy、脅威、保持 | Current | 安全境界またはdata取扱いの変更 |
| [`decisions/`](decisions/README.md) | 採用判断、代替案、結果、置換関係 | Current index | 長期間残す判断または既存判断の置換 |
| [`process/`](process/development.md) | Issue、Pull Request、文書、releaseの進め方 | Current | 開発・承認processの変更 |
| [`operations/`](operations/README.md) | deploy、monitor、incident、rollback、data保守 | Current index | 運用手順または運用条件の変更 |
| [`guides/`](guides/README.md) | 利用者・管理者向けの操作と設定 | Current index | 操作・設定手順の変更 |
| [`research/`](research/README.md) | 調査、候補、未承認UI案 | Non-normative | 調査または実験の追加 |
| [`archive/`](archive/README.md) | 現行ではないが文脈保持が必要な資料 | Historical | 現行文書からの退役 |

## Source and status rules

- 現在の正しい状態は、日付やversionをファイル名へ付けず、上表の正規pathで更新します。
- Issue、Pull Request、実装、Accepted Decision Record、現行文書を照合し、一つの情報源だけで仕様を断定しません。
- 実装から確認した内容は「観測した実装」、本番確認済みの内容は「確認済み運用」、証拠不足は`Incomplete`と区別します。
- 対象外を記録する場合は、理由と再検討条件を併記します。
- 変更理由はIssue、Pull Request、Decision Recordを相互にlinkして残します。
- 過去文書はGit履歴を優先し、特別な文脈保持が必要な場合だけ`archive/`へ置きます。
- secret、private URL、個人情報、正確な現在地、raw provider response、raw logは文書へ記載しません。

## Naming

- 新しいMarkdown pathは英小文字のkebab-caseを使用します。
- 本文は日本語で記載します。
- 新しいDecision Recordは`decisions/NNNN-short-title.md`、タイトルは`DR-NNNN: Title`とします。
- 移行済みADRは参照互換性のため既存の`ADR-NNNN`タイトルとIDを維持します。
