# SpotDiggz Documentation Process

- Status: Current

## Required review

変更前に[`../README.md`](../README.md)と、変更対象に関係するProduct、requirements、specifications、architecture、security、operations、guides、Accepted Decision Recordを確認します。

すべてのIssueとPull Requestで文書影響を次のいずれかに分類します。

- `DOC_UPDATE_REQUIRED`: 現在の要求、仕様、設計、security、運用、操作方法のいずれかが変わる。
- `NO_DOC_CHANGE`: 現行契約内のbug修正または実装不足。理由をPull Requestへ記録する。

Decision Recordの要否も別に分類します。

- `DR_REQUIRED`: 長期間残る判断、scope、外部仕様、責務境界、data、security、運用が変わる。
- `DR_NOT_REQUIRED`: 現行判断内の局所変更。理由を記録する。

## Update matrix

| Change | Required documents |
| --- | --- |
| 利用者、課題、価値、scope、non-goal | `product.md`、`requirements.md`、必要なDecision Record |
| 機能・非機能要件、release gate | `requirements.md`、関連仕様、test、必要なDecision Record |
| UI、API、schema、入出力、error | `specifications/`、`requirements.md`、必要に応じてguides |
| component、data、provider、deployment境界 | `architecture.md`、`security.md`、operations、Decision Record |
| 認証、認可、secret、privacy、retention | `security.md`、requirements、architecture、operations、Decision Record |
| deploy、monitor、incident、rollback | `operations/`、`process/release.md`、必要なDecision Record |
| 利用者・管理者の操作 | `guides/`、関連仕様 |
| Issue・PR・承認flow | `process/`、必要なDecision Record |

## Timing

1. Issueで文書影響とDecision Record要否を予測します。
2. 恒久契約を変更する場合は、必要に応じて文書だけの計画Pull Requestを先に承認します。
3. 実装Pull Requestで実際の差分と文書を再照合します。
4. merge前にrequirements、specifications、architecture、実装、guide、operationsの整合を確認します。
5. 実装で採用が確定するDecision Recordは同じPull Requestで`Accepted`へ更新します。

## Evidence and status

- 根拠不足は`Incomplete`とし、`Missing evidence`と`Required decision`を記載します。
- 対象外を根拠付きで確認できた場合だけ`Not applicable`とし、`Reason`と`Revisit when`を記載します。
- Codeから確認した事実は「観測した実装」であり、承認済み要求として扱いません。
- Production確認は日付、環境、対象flow、結果をsecretなしで記録します。
- 未承認のresearch、UI案、会話logを現行仕様へ昇格させません。

## Migration and compatibility

- [PR #309](https://github.com/kohei321dev/spot-diggz/pull/309)でAIエージェントの共通指示をリポジトリ外へ集約したため、標準テンプレートの`AGENTS.md`はこのリポジトリへ配置せず、文書検証の必須pathにも含めません。文書の参照・更新・レビュー手順は本書で管理します。この例外は、所有者がリポジトリ内での指示管理を再採用すると判断した場合に見直します。
- 単純移動は`git mv`を使い、旧path参照を更新します。
- Open Issue等が旧pathを参照している場合、必要最小限の非正規compatibility stubを期限または再検討条件付きで残せます。
- Decision RecordのID、日付、status、本文、置換関係を保持します。番号を振り直しません。
- 対応先のない削除をしません。移行台帳へMOVE、MERGE、SPLIT、KEEP、ARCHIVE等を記録します。

## Validation

- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\validate-docs.ps1`
- `npm run test:contracts`
- `git diff --check`
- Pull Requestの`docs-check`と既存CI

Validatorは必須path、kebab-case、compatibility stub、内部link、Decision Record metadata・ID・status・index掲載を確認します。

## Prohibited patterns

- `latest`、`final`、`new`、`v2`を付けた現行文書の複製
- 同じ契約をREADME、Issue、複数docsへ不必要に重複記載すること
- raw chat logを判断記録の代わりに保存すること
- secret、private URL、個人情報、正確な現在地、raw response/logを証拠として貼ること
- 古いcompatibility stubを正規文書として更新すること
