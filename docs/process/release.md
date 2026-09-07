# SpotDiggz Release Process

> 移行中: [DR-0019](../decisions/0019-api-client-boundary.md)で独立API・Slack/Discord・Cloud Run・独自Web UI不要を採用しました。本書のWeb/OAuth/Vercelに関する記載は移行前実装の参照であり、新構成の提供・設定済みを示しません。新方針は[API契約](../specifications/api-mvp.md)と[Cloud Run運用計画](../operations/cloud-run-plan.md)を参照し、旧設定手順を新環境へ流用しないでください。

- Status: Current
- Operational details: [`../operations/continuous-delivery.md`](../operations/continuous-delivery.md)

## Preconditions

- 対象IssueとPull Requestが一致し、人間の実装・release承認範囲を超えていない。
- format、vet、test、contract、catalog、E2E、build、scan等の必須checkが成功している。
- Product、requirements、specifications、architecture、security、guide、operationsが実装と一致している。
- migration、data compatibility、secret、external provider、rollbackを確認している。
- 未確認のProduction条件を成功扱いしていない。

### API modeのcatalog gate

[DR-0022](../decisions/0022-domestic-catalog-quality.md)に従い、配備するcatalogに`go run ./cmd/catalogcheck -path data/facilities.json -require-searchable`を実行する。既存の全recordが既定168時間後も鮮度内である条件を維持し、同時点の検索掲載候補を1件以上要求する。#314では実catalogの再調査・分類を行っていないため、schema対応や通常CIを公開gateの完了証拠にしない。出典の確認を伴わない日時更新やfixture配備は禁止し、[catalog保守手順](../guides/catalog-maintenance.md)に従う。

APIの`/readyz`では認証構成・地点provider・現時点の掲載候補を確認する。freshだけで未分類/営業時間不明/日付確認必須のrecordは候補にならない。readinessは実Google疎通や168時間先の鮮度を保証しないので、公開前gateと認証済み検索smokeを別に実施する。

## Environment model

- Local: 開発、unit/component test、manual UI確認。
- CI: deterministic Go test、format/vet/build、contract、docs、Go source/binary・secret・dependency scan。
- Release前の手動検証: 本番catalog freshness、desktop/mobile E2E、container build・scan・smoke。通常CIはこれらの成功証拠を提供しない。詳細は[DR-0017](../decisions/0017-minimal-development-ci.md)と[Continuous delivery](../operations/continuous-delivery.md)を参照する。
- 一時検証環境: Cloud Runの外部連携・data migration等で必要な範囲を#318で決め、明示承認後に作成する。旧Vercel Previewを新構成の既定にしない。
- Production: `main`反映後の正式環境。

### Permanent staging

- Status: Not applicable
- Reason: 単一ownerの小規模private MVPであり、local/CIと必要時だけのPreview、Production smokeで現在のriskを管理するため。
- Revisit when: 複数人による継続的受入、長期external test、独立data/credential境界が必要になったとき。

## Release steps

1. remote `main`基準と対象commitを記録します。
2. 同じcommitのCI結果、手動のcatalog・E2E・container検証結果、deploy先のartifactと直前版へのrollback手段を確認します。CIからDocker archiveを取得する運用は終了しています。
3. 必要なschema migrationとenvironment設定を承認済み手順で行います。
4. 対象artifactをdeployまたは昇格します。環境ごとに別sourceをbuildしません。
5. `/healthz`、`/readyz`、owner login、主要API/UI、変更対象の外部連携をsmokeします。
6. metrics/logでerror、latency、retention、provider fallbackを観察します。
7. 結果、時刻、commit/artifact、未解決事項をsecretなしで記録します。

上記のUI、OAuth login、retention、Routes fallbackはlegacy modeの確認対象である。API modeは[読み取りAPIの公開前確認項目](../guides/read-api-setup.md)でBearer認証・検索/詳細・旧route非公開を確認し、Cloud Runの具体的な配備/運用検証は#318の承認範囲で行う。

## Rollback

API modeでは、認証を維持する設定と互換なschema・binary・catalog snapshotの組へ戻す。新fieldや旧5府県外recordは旧binaryで読み込めない可能性があるため、binaryだけを戻さない。現在時刻でcatalog gateとAPI smokeを再実行し、失効済みtokenを復活させず、catalogや訂正DBを削除しない。API modeのprovider障害は503であり、以下の旧straight-line縮退手順を流用しない。

移行前legacy modeのrollback記録:

- health/readiness failure、継続する5xx/latency悪化、store failure、秘密情報・位置・contactのlog混入、重大catalog誤りをrollback条件とします。
- 直前の検証済みartifactを同じNeon接続設定で起動し、reportやschemaをtruncate・downgradeしません。
- Googleだけを無効化する場合はkeyを削除して再起動し、straight-line推薦とlocation search 503を確認します。
- 詳細手順は[`../operations/mvp-runbook.md`](../operations/mvp-runbook.md)を正とします。

## Post-release evidence status

- [確認済み運用] Vercel/Neon、migration、health/readiness、facility/correction API、UI、GitHub owner login、Slack modal・candidate responseの記録があります。
- Status: Incomplete
- Missing evidence: Discord Production E2E、Google実通信・quota・billing・key restriction、metrics network restriction・dashboard・alert、custom domain/DNS、`main`自動deploy、rollback exercise。
- Required decision: ownerが該当機能を有効化またはreleaseする際に検証し、operationsとrelease記録を更新する。
