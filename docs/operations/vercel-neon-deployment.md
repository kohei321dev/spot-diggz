# Vercel・Neonデプロイ手順

## 方針

spot-diggzはVercel Servicesの単一`container` serviceとして、`Dockerfile.vercel`からGo applicationを公開する。施設カタログは`data/facilities.json`、訂正報告は`DATABASE_URL`設定時にNeonへ保存する。

## 初回セットアップ

1. Vercel CLIへログインする。

```bash
npx vercel login
npx vercel whoami
```

2. 既存Team `uechikoheis-projects` にVercel Projectを作成してlinkする。

```bash
npx vercel project add spotdiggz --scope uechikoheis-projects
npx vercel link --yes --project spotdiggz --scope uechikoheis-projects
npx vercel project update spotdiggz --framework services --scope uechikoheis-projects
npx vercel git connect git@github.com:kohei321dev/spot-diggz.git --scope uechikoheis-projects
```

`vercel.json`の`services.app`が`Dockerfile.vercel`を指し、catch-all rewriteが`app` serviceへ転送する。ProjectのFramework Presetは`Services`でなければContainer buildにならない。

3. 既存Neon OrganizationへProjectを作成する。Vercel Marketplaceの自動作成は使わず、Organizationを指定してNeon CLIから作成する。

```bash
npx neonctl auth
npx neonctl projects create \
  --org-id <existing-neon-org-id> \
  --name spotdiggz \
  --region-id aws-us-east-1
npx neonctl env pull --project-id <spotdiggz-project-id>
```

`neonctl env pull`はlocalのignored `.env.local`へ接続情報を書き込む。接続情報を画面、shell history、Gitへ出力しない。

4. Neonの接続文字列を使ってmigrationを適用する。

```bash
set -a
. ./.env.local
set +a
go run ./cmd/dbmigrate
```

接続文字列はshell履歴、ログ、Gitへ保存しない。

5. `DATABASE_URL`をVercel Productionへsecretとして設定する。

```bash
set -a
. ./.env.local
set +a
printf '%s' "$DATABASE_URL" | npx vercel env add DATABASE_URL production --sensitive --yes --scope uechikoheis-projects --project spotdiggz
```

6. Productionへdeployする。

```bash
npx vercel --prod
```

## 確認

`spotdiggz.vercel.app`のようなProduction aliasを取得したら、次を確認する。

```bash
export SPOTDIGGZ_URL='https://spotdiggz.vercel.app'
curl --fail --silent "$SPOTDIGGZ_URL/healthz"
curl --fail --silent "$SPOTDIGGZ_URL/readyz"
curl --fail --silent "$SPOTDIGGZ_URL/api/facilities?activity=skateboard"
curl --fail --silent "$SPOTDIGGZ_URL/metrics"
```

UIからワンタップ推薦、条件変更、公式情報、外部ナビ、訂正報告を確認する。Google Maps連携を有効化する場合は`GOOGLE_MAPS_API_KEY`をVercel Productionへ追加し、API keyにserver-side API制限を設定してから再deployする。

## GitHub自動デプロイ

`vercel.json`で`main`だけを自動デプロイ対象にしている。GitHub連携をCLIまたはDashboardで設定した後、`main`へのmergeをProduction deployのトリガーとする。Production secretをPreviewへコピーしない。

## rollback

```bash
npx vercel ls
npx vercel rollback
```

rollback時もNeonの`correction_reports`を削除、truncate、旧schemaへ戻す操作は行わない。アプリの前方互換を保ったまま、同じDBを使って直前のデプロイへ戻す。

## 検証済み・残作業

- [確認済み 2026-07-20] Vercel Project `spotdiggz`を作成し、GitHub repositoryへ接続した。
- [確認済み 2026-07-20] 既存Neon Organizationへ`spotdiggz` Projectを作成し、local `.env.local`へ接続情報を設定した。
- [確認済み 2026-07-20] `correction_reports` migrationを適用し、Vercel Productionへ`DATABASE_URL`をsecretとして設定した。
- [確認済み 2026-07-20] Productionの`/healthz`、`/readyz`、施設検索、UI、訂正APIを確認した。
- [残作業] Google Maps API keyを設定していないため、経路推定はstraight-line fallback、地点検索は利用不可のままである。
- [残作業] `/metrics`の公開範囲制限、Google API quota監視、カスタムドメインのDNS設定は別途実施する。
