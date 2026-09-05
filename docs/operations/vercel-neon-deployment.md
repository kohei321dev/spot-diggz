# Vercel・Neon構成の再確認

- Status: Incomplete
- Missing evidence: 現在の公開先、Vercel ProjectとGitHub連携の有効性、Neon接続先・保持状態。
- Required decision: ownerが提供継続・移転・廃止を確認し、再提供する場合は配置先と操作範囲を承認する。

[公開先の確認状況](service-status.md)のとおり、従来hostを現在の本番URLとして使用することはできません。過去のProject作成・GitHub自動deploy接続コマンドは、現在の外部状態を確認せず再実行しないよう本書から除去しました。これはVercelやNeon自体の廃止決定ではありません。

## リポジトリに残る配置設計

- `vercel.json`は単一container serviceと`Dockerfile.vercel`を参照します。
- Go applicationはGit管理の`data/facilities.json`を読みます。
- `DATABASE_URL`設定時の訂正reportと、Production Slackの重複処理防止状態はPostgreSQLへ保存します。
- `vercel.json`のmain deployment設定だけでは、GitHub連携や本番deployが現在動作している証拠にはなりません。

設計判断の履歴は[ADR-0012](../decisions/0012-vercel-neon-deployment.md)に保持します。

## 再提供を選択した場合の確認事項

1. ownerが既存Project・DBの有無と用途を確認する。同名ProjectやDBを自動作成しない。
2. 承認済みのHTTPS originを決め、GitHub OAuth callback、Slack/Discord endpoint、manifest、scriptの固定値を一致させる。
3. [GitHub認証](../guides/github-oauth-setup.md)、[Slack](../guides/slack-setup.md)、[Discord](../guides/discord-setup.md)の設定手順を新しい公開先と照合する。外部設定変更は承認後だけ行う。
4. [release手順](../process/release.md)でartifact、認証設定、migration、secret、DB backup、rollback先を確認する。
5. deploy後に[MVP runbook](mvp-runbook.md)のhealth/readinessとowner認証済みUI/APIのsmokeを行う。未認証APIの401を正常な認証境界として扱う。
6. 確認日・環境・結果を本書と公開状況へ記録してから、利用者向けアクセスリンクを復活させる。

Google設定の有無、quota、metrics公開制限は現環境で再確認する。過去の「未設定」「確認済み」を現在の事実として引き継がない。

## Rollbackとデータ保護

rollback先のdeploymentは操作前に特定し、Neonのreportやschemaを削除・truncateしない。詳細は[continuous delivery](continuous-delivery.md)と[MVP runbook](mvp-runbook.md)に従う。現在の外部状態が不明なまま古いdeploy・rollbackコマンドを実行しない。
