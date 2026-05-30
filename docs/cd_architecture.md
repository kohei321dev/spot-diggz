---
title: CD Architecture
---

# CDアーキテクチャ

spot-diggzのCD（継続的デプロイ）の確定仕様を記載する。

## 運用方針（ブランチと環境）

- `develop` へのマージで **dev** 環境へデプロイしている
- `release/*` へのマージで **stg** 環境へデプロイしている
- `master` へのマージで **prod** 環境へデプロイしている
- dev を主運用とし、stg/prod は同一設計で運用している

## デプロイ対象

- Rust API → Cloud Run
- React UI → Cloud Storage（静的ホスティング）

## デプロイ優先度

- dev 環境を優先して検証し、同一経路で stg/prod に適用している

## CDの実行経路

- GitHub Actions を起点に Cloud Build を呼び出している
- 認証方式は Workload Identity Federation（WIF）を使用している

## フロー概要

1. GitHub Actions が `develop` / `release/*` / `master` の push を検知している
2. WIF で GCP に認証している
3. Cloud Build で API をビルドし、Artifact Registry に push している
4. Cloud Run を更新している
5. UI をビルドし、GCS に配信している

## 命名規則（CDで使う名称）

CLAUDE.md「命名規約（4層ティア）> Tier 3」に準拠する。

- Cloud Run サービス名: `sdz-{stage}-api`
- Cloud Run イメージ: `{region}-docker.pkg.dev/sdz-{stage}/sdz-{stage}-api/sdz-api:latest`
- UI バケット名: `sdz-{stage}-ui-hosts`
- 画像バケット名: `sdz-{stage}-img-spots`

## 公開ポリシー

- 画像バケットは `allUsers` 公開で運用している

## 責務分担

- GitHub Actions: ブランチ検知と WIF 認証で Cloud Build を起動している
- Cloud Build: API/UI のビルドと Artifact Registry への push を担当している
- Cloud Run: API デプロイ先として稼働している
- Cloud Storage: UI の静的配信と画像バケット公開を担当している
- Terraform: 反映先の環境値とリソース定義の正を保持している

## ルール

- CDは `develop` / `release/*` / `master` のみに反応させている
- 反映先の環境変数は Terraform の値と一致させている
- UI 配信後は `index.html` を no-cache で配置している
- 変更があった場合は本ファイルを先に更新している

## 📋 アーキテクチャ変更（2026-02-28）

データアーキテクチャ再設計により、以下の変更が予定されている:

- Firestore: Cloud Run サービスアカウントの IAM 権限を読み取り専用（`roles/datastore.viewer`）に変更
- BigQuery: マスターデータの ETL パイプライン用データセットを追加
- Cloud Functions: BigQuery → Firestore データ同期関数を追加
- API: 書き込みエンドポイント（POST/PATCH spots, mylists）を休眠化
- マスターデータ投入: API 経由 → BigQuery パイプライン経由に変更

詳細: `docs/designs/tier2-spot-data-architecture.md`
