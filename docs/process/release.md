# SpotDiggz Release Process

- Status: Current
- Operational details: [`../operations/continuous-delivery.md`](../operations/continuous-delivery.md)

## Preconditions

- 対象IssueとPull Requestが一致し、人間の実装・release承認範囲を超えていない。
- format、vet、test、contract、catalog、E2E、build、scan等の必須checkが成功している。
- Product、requirements、specifications、architecture、security、guide、operationsが実装と一致している。
- migration、data compatibility、secret、external provider、rollbackを確認している。
- 未確認のProduction条件を成功扱いしていない。

## Environment model

- Local: 開発、unit/component test、manual UI確認。
- CI: deterministic test、contract、catalog freshness、E2E、binary/image、security scan。
- Vercel Preview: 外部連携、data migration、infrastructure等で必要な期間だけ明示的に作成。
- Production: `main`反映後の正式環境。

### Permanent staging

- Status: Not applicable
- Reason: 単一ownerの小規模private MVPであり、local/CIと必要時だけのPreview、Production smokeで現在のriskを管理するため。
- Revisit when: 複数人による継続的受入、長期external test、独立data/credential境界が必要になったとき。

## Release steps

1. remote `main`基準と対象commitを記録します。
2. 同じcommitのverification結果とartifactを確認します。
3. 必要なschema migrationとenvironment設定を承認済み手順で行います。
4. 対象artifactをdeployまたは昇格します。環境ごとに別sourceをbuildしません。
5. `/healthz`、`/readyz`、owner login、主要API/UI、変更対象の外部連携をsmokeします。
6. metrics/logでerror、latency、retention、provider fallbackを観察します。
7. 結果、時刻、commit/artifact、未解決事項をsecretなしで記録します。

## Rollback

- health/readiness failure、継続する5xx/latency悪化、store failure、秘密情報・位置・contactのlog混入、重大catalog誤りをrollback条件とします。
- 直前の検証済みartifactを同じNeon接続設定で起動し、reportやschemaをtruncate・downgradeしません。
- Googleだけを無効化する場合はkeyを削除して再起動し、straight-line推薦とlocation search 503を確認します。
- 詳細手順は[`../operations/mvp-runbook.md`](../operations/mvp-runbook.md)を正とします。

## Post-release evidence status

- [確認済み運用] Vercel/Neon、migration、health/readiness、facility/correction API、UI、GitHub owner login、Slack modal・candidate responseの記録があります。
- Status: Incomplete
- Missing evidence: Discord Production E2E、Google実通信・quota・billing・key restriction、metrics network restriction・dashboard・alert、custom domain/DNS、`main`自動deploy、rollback exercise。
- Required decision: ownerが該当機能を有効化またはreleaseする際に検証し、operationsとrelease記録を更新する。
