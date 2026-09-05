# DR-0017: 通常CIとリリース前検証を分ける

- Status: Accepted
- Date: 2026-09-05
- Type: Operation
- Related Issues: [#307](https://github.com/kohei321dev/spot-diggz/issues/307)
- Related Pull Requests: [#308](https://github.com/kohei321dev/spot-diggz/pull/308)
- Affected Docs: `operations/continuous-delivery.md`, `operations/mvp-runbook.md`, `process/release.md`, `requirements.md`, `security.md`
- Supersedes: [ADR-0007](0007-go-modular-monolith-runtime.md)（container scanを通常CIで必須とする運用のみ）
- Superseded By: None

## Context

単一ownerの通常開発で、source検証と、Docker成果物の作成・保存、ブラウザE2E、本番catalogの実時間鮮度検査を同じCIで行っていた。catalogの確認日が古くなるだけで無関係な文書変更もfailし、Docker smokeは現在必須のProduction認証設定を与えずに起動していた。GitHub ActionsにVercelへの接続・deploy処理はなく、外部連携の解除が直接の失敗原因とは確認できない。

ユーザーはIssue #307で、不要な処理を削除して現在の開発に必要な最小CIへ整理するよう指示した。実時間に依存するHTTPテストと既存依存脆弱性は、通常のsource検証に必要な修正として扱う。

## Decision

1. 通常CIはPR、main push、手動実行を対象にする。短命branch pushとの重複実行と週次scheduleを削除する。同じrefへの更新は古いCIをcancelする。
2. Go format、vet、race test、静的binary build、govulncheckのsource/binary scan、JSON/OpenAPI契約、文書構造・内部リンク、Gitleaks、PR dependency reviewを残す。Go versionはgo.modを参照する。
3. Docker smoke、image scan、SBOM、Docker archive保存、Playwright job、Trivy filesystem scanを通常CIから削除する。Go脆弱性検査とGitleaksは継続し、scanの失敗を無視する設定は追加しない。
4. 本番catalogの168時間先の鮮度確認はrelease前と運用者の定期保守で行う。通常CIではschemaと固定時刻の鮮度ルールを検証する。公式再確認なしにproductionの検証時刻を更新しない。
5. E2EはUI変更時とrelease前、container build・scan・smokeはrelease前に手動で行い、結果を記録する。既存のtest、Dockerfile、検証commandは保持する。
6. API、認証、推薦rule、Vercel設定、DBの意味は変更しない。HTTPテストには既存の固定clockを注入する。
7. 基本検証で検出した既存脆弱性は、Go 1.25.13、x/text v0.39.0とその要求するx/sync v0.21.0へ更新して修正する。Go builderのパッチ版も合わせる。新しいlibraryやGoのminor系列は導入しない。

## Rationale

通常CIはsourceの変更に対して再現可能な結果を返す。実データの再調査、外部環境、release成果物の検証は、運用者が実施時点と結果を記録する。Go、contract、secret、dependencyの基本検証を維持するため、CIを空にして成功を作る構成にはしない。

## Alternatives Considered

- 現状維持: 全変更でrelease向けjobが走り、実時間と外部artifactに依存する負荷が残る。
- 失敗jobをすべて削除: 有効なGoテストと脆弱性検査を失うため不採用。
- 新しいrelease用workflowを追加: 現段階では自動実行経路を増やさず、手動手順を利用する。実施漏れが観測された場合に再検討する。

## Consequences

- PRで必要な検証を実行し、不要なDocker/browser download、成果物保存、重複scanを減らせる。
- 通常CI成功は本番catalogの鮮度、ブラウザE2E、container起動・脆弱性、deploy成功を保証しない。
- 週次の自動鮮度通知とCI image archiveによるrollback材料を失う。ownerは保守時の鮮度確認と、deploy先の直前成果物・rollback手段を管理する。
- Trivyのfilesystem misconfiguration検査は通常CIでは行わない。Docker・infra変更時はrelease前の手動検証で補う。

## Security and Privacy

workflow権限はcontents: read、ActionsはSHA固定を維持する。Go脆弱性とsecret検査は残す。Production credentialをCIへ追加せず、catalog事実の未確認更新を行わない。

## Migration and Rollback

PR #306の文書移行branchをbaseに後続PRを作る。後続PRを文書移行branchへ統合してからmain向けPRのCIを確認する。rollbackは本Issueのcommitを逆順にrevertする。以前の重いjob、fixtureの時刻依存、旧依存脆弱性が復活する点を確認する。DB変更・deployは伴わない。

## Validation

- 既存CIで失敗していたHTTPテスト3件が修正後にPASSする。
- 最新headのGo、contract、docs、secret、dependency checkがPASSする。
- workflowにDocker、Playwright、artifact upload、schedule、実時間catalog checkが残っていない。
- 文書リンクとgit diff --checkがPASSする。

## Revisit Conditions

release前の手動検証漏れ、browser/containerだけの回帰、catalog更新漏れが観測された場合、対象に限定した自動検証を再導入する。CI由来のimageを配布・昇格する運用を採用した場合、artifact・SBOM・scanを再設計する。

## References

- [Continuous delivery](../operations/continuous-delivery.md)
- [MVP runbook](../operations/mvp-runbook.md)
- [Release process](../process/release.md)
