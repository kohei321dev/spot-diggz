# ADR-0012 Vercel ContainerとNeonによるMVP公開

- Status: Accepted
- Date: 2026-07-20
- Related: [ADR-0007](0007-go-modular-monolith-runtime.md)
- Related: [ADR-0008](0008-facility-catalog-api-and-storage.md)

## Context

MVPはUI、API、施設カタログを1つのGo HTTP applicationとして実装している。Vercelで公開する場合、Go Functions向けの単一handlerへ分割するより、既存のHTTP serverとDocker成果物を維持できる方式が適している。VercelのContainer runtimeを有効にするため、Services設定上は`app`という単一serviceとして公開する。

Vercelの実行環境はインスタンス交換を前提とするため、訂正報告をコンテナのローカルファイルだけへ保存すると、デプロイ後に報告が失われる。施設カタログは運用者が出典と検証時刻を管理するGit管理JSONであり、MVPではDBへ移す必要がない。

## Decision

1. Vercel Servicesの単一`app` serviceで、`Dockerfile.vercel`からGo applicationをVercel Containerとして公開する。catch-all rewriteでUI/APIを同じserviceへ転送する。
2. `main` branchだけをGit自動デプロイ対象とし、その他branchの自動Previewは無効にする。必要なPreviewはCLIまたは明示的なGit設定で作成する。
3. 施設カタログは`data/facilities.json`を正本としてimageへ同梱する。
4. `DATABASE_URL`がある場合、訂正報告をNeon/PostgreSQLの`correction_reports`へ保存する。未設定のローカル環境では既存file storeへフォールバックする。
5. Neonのschemaは`db/migrations/0001-correction-reports.sql`で管理し、migrationはアプリ起動時に自動適用しない。
6. Neonの接続失敗、schema未適用、期限削除失敗は起動または運用エラーとして扱い、別の一時保存先へ暗黙に縮退しない。

## Alternatives

### Vercel Go Functionsへ分割する

Vercelの標準runtimeに合うが、既存の全体HTTP handlerを単一handlerへ組み替え、埋め込みUIとAPIのrouteを再構成する必要がある。MVP公開の変更範囲が大きいため採用しない。

### Vercel Servicesでfrontendとbackendを分ける

複数アプリを同一projectで扱えるが、現在は単一Go applicationであり、frontendとbackendのサービス境界とroute管理を増やす効果がないため採用しない。Vercel Containerを選択するための単一`app` service設定だけを使う。

### 施設カタログをNeonへ移す

運用者編集、履歴、検索に適する一方、migration、DB障害、catalogの出典検証経路を増やす。更新頻度と履歴要件がJSON運用を超えた時点で別ADRとして再評価する。

## Consequences

- 既存のUI/API、Go単一binary、catalog validationを維持したまま公開できる。
- 訂正報告はVercelのインスタンス交換で失われにくくなる。
- Neonが未設定のローカル開発とCIは、従来どおりfile storeで実行できる。
- Neonのschema適用、接続情報、バックアップ、保持期限削除の運用が追加される。
- Vercel ContainerとNeonの実通信、migration、production smokeを確認する必要がある。Google API quota、metricsのprivate network制限、カスタムドメインは別の運用課題として残る。

## Verification

- `go test ./...`、`go vet ./cmd/... ./internal/...`、race、govulncheckを実行する。
- migrationを適用したNeonで起動し、訂正APIがPostgreSQLへ保存できることを確認する。
- Vercel Productionで`/healthz`、`/readyz`、施設一覧、推薦、訂正APIを確認する。UIのdesktop/mobile E2Eも実行する。
