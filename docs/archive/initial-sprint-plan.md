# Initial Sprint Plan

- Status: Archived
- Archived: 2026-09-05
- Successor: [`../process/development.md`](../process/development.md)
- Reference condition: 初期計画の文脈確認だけに使用し、現在のscope、要求、作業順序の根拠には使用しない。

この計画は移行前の`docs/development_workflow.md`に記載されていた初期案です。現在のProduct、requirements、Issue、Accepted Decision Recordを優先します。

## Discovery Sprint 0

- 大阪市・堺市を利用者の起点とする大阪都市圏の確定
- 一次persona、Job、対象外利用者の確定
- 20〜30施設の候補と検証方法の確認
- 比較対象service、質問票、成功条件、停止条件の決定

## Sprint 1: Foundation and Facility Catalog

- Web、data保存、CI/CD、observabilityの最小構成をDecision Recordで決定
- 施設data model
- 公式情報の収集
- 候補20〜30施設の検証と初期登録
- 施設詳細表示

## Sprint 2: Recommendation

- 条件入力
- 利用不可施設の除外
- rule-based scoring
- 最大3件の推薦と理由表示

## Sprint 3: Action, Localization and Feedback

- 外部navigation連携
- 日本語・英語対応
- 訂正報告
- 主要導線のE2E

## Sprint 4: Validation

- 利用者test
- 実際の訪問・再利用の計測
- SLI/SLO、alert、security、運用時間の確認
- Go、仮説修正、No-Goの判断

AIによる自然文入力や説明の最適化は初期計画の対象外でした。現在もMVP推薦へAIを使用しません。
