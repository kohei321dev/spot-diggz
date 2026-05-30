---
name: scan-infra
description: インフラレイヤーのセキュリティスキャン（Terraform, GCP Provider, GCPサービス, CI Actions）
allowed-tools: Bash(gh issue:*), Bash(gh project:*)
---

インフラレイヤーのセキュリティ・バージョンスキャンを実行してください。

現在のインフラ設定:

- Terraform バージョン: !`cat web/.terraform-version`
- Google provider: !`grep -A2 'google' web/resources/versions.tf 2>/dev/null || grep -r 'required_providers' web/resources/ --include='*.tf' -A5 2>/dev/null | head -20`

手順:

## 1. Terraform バージョン・CVE確認

1. 上記で取得した現在のTerraformバージョンを確認
2. WebSearch で「Terraform latest stable version 2026」を検索し最新安定版を取得
3. WebSearch で「Terraform CVE 2025 2026」を検索し直近の重大CVEを確認
4. 現在バージョンとの差分を算出（パッチ差分 / マイナー差分 / メジャー差分）

## 2. Google Provider バージョン・CVE確認

1. 上記で取得した現在のGoogle providerバージョンを確認
2. WebSearch で「terraform-provider-google latest version 2026」を検索し最新版を取得
3. WebSearch で「terraform-provider-google CVE vulnerability 2025 2026」を検索
4. 現在バージョンとの差分を算出

## 3. GCPサービス脆弱性確認

1. WebSearch で「Google Cloud Run vulnerability CVE 2025 2026」を検索
2. WebSearch で「Google Firestore vulnerability CVE 2025 2026」を検索
3. WebSearch で「Google Cloud Storage vulnerability CVE 2025 2026」を検索
4. 本プロジェクトに影響する重大脆弱性がないか確認

## 4. GitHub Actions 使用アクション脆弱性確認

`.github/workflows/ci.yml` で使用している以下のアクションについて確認:

| アクション                              | 用途                           |
| --------------------------------------- | ------------------------------ |
| `actions/checkout@v4`                   | リポジトリチェックアウト       |
| `dtolnay/rust-toolchain@stable`         | Rust セットアップ              |
| `actions/cache@v3`                      | キャッシュ                     |
| `actions/setup-node@v4`                 | Node.js セットアップ           |
| `hashicorp/setup-terraform@v3`          | Terraform セットアップ         |
| `aquasecurity/tfsec-action@v1.0.3`      | Terraform セキュリティスキャン |
| `aquasecurity/trivy-action@master`      | 脆弱性スキャン                 |
| `codecov/codecov-action@v3`             | カバレッジレポート             |
| `docker/setup-buildx-action@v3`         | Docker Buildx                  |
| `docker/login-action@v3`                | Docker ログイン                |
| `docker/metadata-action@v5`             | Docker メタデータ              |
| `docker/build-push-action@v5`           | Docker ビルド＆プッシュ        |
| `github/codeql-action/upload-sarif@v3`  | SARIF アップロード             |
| `actions/dependency-review-action@v3`   | 依存関係レビュー               |
| `google-github-actions/auth@v2`         | GCP認証                        |
| `google-github-actions/setup-gcloud@v2` | gcloud セットアップ            |

1. WebSearch で「GitHub Actions supply chain attack 2025 2026」を検索
2. 上記アクションのうち、`@master` や古いメジャーバージョンを使用しているものを警告
3. 特に `aquasecurity/trivy-action@master` はタグ固定を推奨

## 5. tfsec ルール更新確認

1. WebSearch で「tfsec latest version rules update 2026」を検索
2. 現在使用中の `tfsec-action@v1.0.3` が最新か確認

## 6. サマリ表示

結果を以下のフォーマットで表示:

```
## scan-infra 結果

### Terraform
- 現在: X.X.X → 最新: X.X.X (差分: Xマイナー)
- 重大CVE: あり/なし
- 詳細: (該当CVEがあれば列挙)

### Google Provider
- 現在: ~> X.X → 最新: X.X.X
- 重大CVE: あり/なし

### GCPサービス
- Cloud Run: 問題なし / 要確認
- Firestore: 問題なし / 要確認
- Cloud Storage: 問題なし / 要確認

### GitHub Actions
- 要更新アクション: (あれば列挙)
- サプライチェーンリスク: あり/なし

### tfsec
- 現在: v1.0.3 → 最新: X.X.X

### 要対応事項
- (対応が必要な項目を優先度順に列挙)
- なければ「scan-infra: All Clear」
```

## 7. Issue起票判定

以下のルールに従い **ユーザー確認なしに自動起票** する。起票されたIssueが人間への最初の連絡となる。

| 条件                        | Priority | ラベル            | フォーマット |
| --------------------------- | -------- | ----------------- | ------------ |
| Critical脆弱性（CVSS 9.0+） | P0       | `troubleshooting` | 4F           |
| High脆弱性（CVSS 7.0-8.9）  | P1       | `troubleshooting` | 4F           |
| メジャーバージョン遅延      | P2       | `planning`        | STAR         |
| Actions `@master` 参照      | P1       | `troubleshooting` | 4F           |

自動起票フロー:

1. `gh issue list -R uechikohei/spot-diggz --search "KEYWORD" --state all --limit 10` で重複チェック
2. 重複がなければ `gh issue create` で起票 → `gh project item-add 2 --owner uechikohei --url ISSUE_URL` でProject追加
3. 重複がある場合は起票スキップし、サマリで既存Issue番号を報告
