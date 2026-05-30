# Cloud Run デプロイとAPI構成ガイド

## 1. 全体像 (What)

- `api`で実装したRustコンテナをCloud Runへデプロイし、HTTPSエンドポイントとして公開する。
- クライアント（Web/モバイル）はCloud RunのURLまたはカスタムドメイン経由でAPIへリクエストする。
- FirestoreやCloud StorageといったGCPマネージドサービスへはRustアプリから直接アクセスする。

## 命名規則（環境別）

CLAUDE.md「命名規約（4層ティア）> Tier 3」に準拠する。

- Cloud Run サービス名: `sdz-{stage}-api`
- Cloud Run イメージ: `{region}-docker.pkg.dev/sdz-{stage}/sdz-{stage}-api/sdz-api:latest`
- UIホスティングバケット: `sdz-{stage}-ui-hosts`
- 画像保存バケット: `sdz-{stage}-img-spots`

## 2. なぜAPI Gatewayが不要なのか / 使う場合はいつか (Why)

- Cloud RunはHTTP(S)エンドポイントを自動で提供し、スケーリング・TLS終端・コンテナ実行管理を担うため、必須でAPI Gatewayを挟む必要はない。
- 追加でAPI Gateway/Cloud Endpointsを利用することで以下が実現できる:
  - 細かいレート制限、APIキー管理、リクエスト検証などをGCPマネージド側で実装したい場合。
  - 一つのドメインで複数バックエンド（Cloud Run複数サービスやCloud Functionsなど）へルーティングしたい場合。
- 最小構成では Cloud Load Balancing + Cloud Armor + Identity Platform/IAP で十分にセキュリティを確保できるため、本プロジェクトではまずCloud Run単体 + カスタムドメインを想定する。

## 3. デプロイフロー (How)

1. `Dockerfile`でRust APIをコンテナ化（マルチステージビルドで軽量化）。
2. Cloud BuildまたはGitHub Actionsでビルドし、Artifact Registryへプッシュ。
3. `gcloud run deploy`（またはTerraform）でCloud Runサービス `sdz-{env}-api` を作成。
4. 必要ならCloud Runのインバウンドアクセスを`--ingress internal-and-cloud-load-balancing`にし、外部公開はHTTPS Load Balancer経由に限定する。
5. 認証は以下いずれかで実装:
   - Cloud Run IAM + IDトークン（サービス間通信）
   - Identity Platform / Firebase AuthでJWTを検証（今回の想定）
   - Cloud IAPでGoogleアカウント保護
6. Cloud ArmorでWAFルールを定義し、異常トラフィックを遮断。

## 4. データアクセスの流れ (Flow)

```
Client (Web/iOS)
   |
   | HTTPS
   v
Cloud Storage + Cloud CDN (React SPA)  ※UIホスティング
   |
   | axios/fetch (同一カスタムドメイン or api.* サブドメイン)
   v
Cloud Run (Rust APIコンテナ)
   |
   +--> Firestore (スポット・ユーザーデータ)
   |
   +--> Cloud Storage (画像アップロード)
   |
   +--> 外部API (MapKit/Google Maps) ※infrastructure層で呼び出し
```

## 5. 5W1H: Cloud Run経路での注意点

- **Who**: API利用者はWebフロント、iOSクライアント、将来の運用ツール。クラウド側はCloud Run、Cloud Armor、Identity Platformが担当。
- **What**: Cloud RunのURL（`https://{service}-{hash}-a.run.app`）またはHTTPS LBのカスタムドメインをAPIエンドポイントとして提供する。
- **When**: デプロイごとに新しいリビジョンをローリング更新。CI/CDで自動化し、Published URLは一定。
- **Where**: トラフィックは外部→Cloud Load Balancer→Cloud Run→Firestore/Storage。VPC接続が必要ならServerless VPC Access Connectorを介する。
- **Why**: API Gateway/ Lambda構成と違い、Rustアプリをそのまま長時間稼働させられるため、低レイテンシと柔軟なライブラリ利用が可能。
- **How**: TerraformでCloud Runサービス、IAM、Cloud Armorポリシーをコード化し、Identity PlatformまたはJWT検証ミドルウェアをpresentation層へ実装する。

## 6. 参考設定 (Terraformスニペット)

```hcl
resource "google_cloud_run_service" "sdz_api" {
  name     = "sdz-${var.sdz_environment}-api"
  location = var.sdz_region

  template {
    spec {
      containers {
        image = var.sdz_image
        env {
          name  = "PORT"
          value = "8080"
        }
      }
      container_concurrency = 80
    }
  }

  traffic {
    percent         = 100
    latest_revision = true
  }
}
```

## 7. 今後の発展

- Cloud Endpointsを導入してOpenAPI定義ベースのAPIキー管理やログ拡張を行う。
- サービスメッシュ（Anthos Service Mesh）を利用してmTLSやゼロトラスト構成を学習。
- サードパーティAPIを呼ぶGatewayを別サービスに切り出し、バックエンド同士のサービス分割を検討。
- UIカスタムドメインを`spot-diggz.321dev.org`、APIを`api.spot-diggz.321dev.org`のように分離し、CORS/認証ヘッダーの制御を整理する。
- Cloud CDNは開発では使わず、staging/prodで必要に応じて後付けする。
