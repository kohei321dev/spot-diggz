# SpotDiggz Development Process

- Status: Current
- Historical source: `docs/development_workflow.md` before Issue #305

## Principles

- Lean Product Discoveryと短いdelivery cycleを組み合わせ、機能数ではなく検証可能な成果を優先します。
- 1 Issueは1つの独立して検証可能な成果、1 Pull Requestは原則1 Issueとします。
- 人間の明示承認なしに実装、merge、deployへ進みません。
- remote `main`を現在の基準とし、作業branchやlocal HEADと区別します。
- 変更前に[`../README.md`](../README.md)と関係文書、Accepted Decision Recordを確認します。

## Flow

```text
相談・アイデア
  -> read-only review
  -> 人間がIssue化を判断
  -> scope・受入条件・検証・文書影響をIssueへ記録
  -> read-only implementation assessment
  -> 人間が対象Issueの実装開始を承認
  -> remote mainから短命branchを作成
  -> Implement / verify / document
  -> 1 Issue / 1 Pull Request
  -> 人間がreview・merge
  -> release process
```

Issue #305のような文書移行は、PLANをread-onlyで行い、全移行台帳と停止条件を人間が承認してからIMPLEMENTへ進みます。Issue作成は実装承認を意味しません。

## Discovery and sprint

個人開発では1週間を基本とし、各sprintへ作業量ではなく1つの検証可能なgoalを設定します。

```text
Sprint Goal: 何を検証・実現するか
Deliverable: 実際に触れる成果物
Evidence: 利用者反応、test結果、計測値
```

- Planning: Product、requirements、Issueを確認し、goal、完了条件、必要なDecision Recordを決める。
- Review: 実際に動くUI/APIと利用者flowを確認し、期待した証拠を記録する。
- Retrospective: 継続すること、問題、次に試すことを短く残す。

初期Sprint案は現行計画ではないため[`../archive/initial-sprint-plan.md`](../archive/initial-sprint-plan.md)へ移しました。

## Definition of Ready

- 対象user、解決する問題、期待する成果がIssueまたはProductにある。
- WHAT、WHY、対象、対象外、受入条件、検証方法が明確である。
- remote `main`、重複Issue/PR、未commit変更、関連Decision Recordを確認している。
- 必要な設計判断がDecision Recordにあるか、不要である理由を記録している。
- 外部data、位置情報、AI、secret、破壊的migration、外部公開のriskを確認している。
- 人間が対象Issueの実装開始を明示的に承認している。

## Implementation

- `main`最新から短命な`feature/*`、`fix/*`、`docs/*`branchを作ります。
- 承認済みIssueの範囲だけを変更し、無関係なuser差分を変更しません。
- code/data変更と同じPull Requestで、必要な仕様、security、guide、operations、Decision Recordを更新します。
- 変更範囲に応じたtest、format、lint/vet、build、security scan、文書検証を実行します。
- Pull RequestへWHAT、WHY、受入条件、risk、検証結果、未確認事項、rollback、Issue linkを記載します。
- 自動merge・deployを行わず、人間のreview後に停止します。

## Definition of Done

- Issueの要求と受入条件を満たしている。
- 変更に対応するtestと既定checkがPASSしている。
- 主要利用scenarioを自動または手動で確認している。
- 施設情報変更にはsource URLと確認日がある。
- AIが未確認事実を生成していない。
- 文書影響とDecision Record要否を判定し、必要な更新が同じPull Requestにある。
- secret、個人情報、正確な位置の混入がない。
- rollbackまたは安全な無効化方法がある。
- 未解決事項と未実行検証をPull Requestへ記録している。

## Branch policy

| Branch | Purpose |
| --- | --- |
| `main` | release可能な統合branch |
| `feature/*` | 短期間の機能変更 |
| `fix/*` | 短期間の不具合修正 |
| `docs/*` | 文書だけの変更 |
| `archive/*` | 特別な理由がある履歴退避。通常はGit履歴を使う |

`main`へ直接commit・force pushせず、required checkを無効化しません。

## Related process

- Documentation: [`documentation.md`](documentation.md)
- Release: [`release.md`](release.md)
- Engineering sources: [`engineering-principles-and-sources.md`](engineering-principles-and-sources.md)
