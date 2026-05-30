# spot-diggz Terraform（dev）

## 概要

`web/resources/` は dev 環境向けの IaC を管理する。既存の GCP リソースがある場合は **先に import** してから apply を行う。

## 初期セットアップ

```bash
cd web/resources
terraform init
```

## 変数ファイル

```bash
cp environments/dev/terraform.tfvars.example environments/dev/terraform.tfvars
```

## 既存サービスアカウントの import

```bash
terraform import \
  module.sdz_dev.google_service_account.sdz_dev_api_sa \
  projects/sdz-dev/serviceAccounts/sdz-dev-api-sa@sdz-dev.iam.gserviceaccount.com

terraform import \
  module.sdz_dev.google_service_account.sdz_dev_deploy_sa \
  projects/sdz-dev/serviceAccounts/sdz-dev-deploy-sa@sdz-dev.iam.gserviceaccount.com

terraform import \
  module.sdz_dev.google_service_account.sdz_dev_terraform_sa \
  projects/sdz-dev/serviceAccounts/sdz-dev-terraform-sa@sdz-dev.iam.gserviceaccount.com

terraform import \
  module.sdz_dev.google_service_account.sdz_firebase_adminsdk_sa \
  projects/sdz-dev/serviceAccounts/firebase-adminsdk-fbsvc@sdz-dev.iam.gserviceaccount.com
```

## Plan / Apply

```bash
terraform plan -var-file=environments/dev/terraform.tfvars
terraform apply -var-file=environments/dev/terraform.tfvars
```

## 補足

命名規約は CLAUDE.md「命名規約（4層ティア）> Tier 3」に準拠する。

- `sdz-{env}-img-spots` を画像Storage命名の標準とする。
- `sdz-{env}-ui-hosts` を UIホスティング用Storage命名の標準とする。
- Firebase Web App の displayName は `SpotDiggz {Env}` を標準とする（Tier 1準拠）。
- Firebase Web App は常に作成し、iOS/Android は bundle/package が設定された場合のみ作成する。
- Cloud Run / Artifact Registry も IaC に含める（アプリのデプロイはCI/CDで行う）。
- `sdz_ui_public_members` で UI バケットの公開権限を付与する。組織ポリシーで `allUsers` が禁止の場合は `domain:321dev.org` などに切り替える。
- `sdz_enable_cloud_run=false` の間は Cloud Run の作成をスキップする。イメージが Artifact Registry に push 済みになったら `true` に切り替える。
