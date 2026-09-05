# SpotDiggz Operations

- Status: Current index

Production、local、CIのdeploy、monitoring、incident、rollback、data maintenanceを管理します。Secret値、private URL、個人情報、正確な現在地、raw logは記載しません。

## Index

現在の本番提供状況は[公開先の確認状況](service-status.md)を参照してください。過去のsmoke成功は現在の稼働証拠ではありません。

| Runbook | Trigger | Owner | Status / last evidence |
| --- | --- | --- | --- |
| [`mvp-runbook.md`](mvp-runbook.md) | 起動、smoke、incident、fallback、rollback | release owner / on-call operator | Current。Vercel/Neon smoke 2026-07-20 |
| [`continuous-delivery.md`](continuous-delivery.md) | CI、artifact、deploy、rollback設計 | release owner | Current。CI implemented、Production smoke記録あり |
| [`observability.md`](observability.md) | log、metrics、SLI/SLO、privacy | operator | Application実装済み。Production wiringはIncomplete |
| [`vercel-neon-deployment.md`](vercel-neon-deployment.md) | 再提供前の配置・設定確認 | release owner | 公開先未確認。実行手順は再確認が必要 |

管理画面を使う初回認証・chat設定は[`../guides/`](../guides/README.md)、release承認flowは[`../process/release.md`](../process/release.md)を参照します。

## Minimum operational contract

- `/healthz`と`/readyz`でlivenessとcatalog freshnessを分離する。
- request、recommendation、external provider、catalog、retentionを秘密情報なしで観測する。
- Production correction reportとSlack短期状態はNeon/PostgreSQLで保持する。
- Google障害時はstraight-lineへ縮退し、key compromise時は失効・環境削除・再起動する。
- Application rollbackでは直前artifactと同じdata接続を使い、reportやschemaを破壊しない。
- Catalogのweekly freshness failureは自動延命せず、公式sourceを再確認する。

## Incomplete operations

- Status: Incomplete
- Missing evidence: Discord Production E2E、Google実通信・restriction・quota・billing、metrics network restriction・collector・dashboard・alert・retention、custom domain/DNS、`main`自動deploy、rollback/incident exercise、Neon backup/retentionの継続確認。
- Required decision: ownerが有効化・運用開始する項目ごとにprovider設定と実測結果をrunbookへ記録する。
