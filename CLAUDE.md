# CLAUDE.md

必ず日本語で回答すること。

## プロジェクト概要

spot-diggz: スケートスポット検索・シェアアプリ（旧SkateSpotSearchのリプレイス）

- バックエンド: Rust（スクラッチ実装、フレームワーク非使用） → `web/api/`
- フロントエンド: React + TypeScript → `web/ui/`
- インフラ: GCP (Cloud Run, Firestore, BigQuery, Cloud Functions) + Terraform → `web/resources/`
- モバイル: iOS (`iOS/`), Android (`android/`) ※予定
- CI/CD: GitHub Actions (`.github/workflows/ci.yml`)
- リポジトリ: `uechikohei/spot-diggz`（モノレポ構成）

## ビルド・テスト・Lintコマンド

### Rust API (`web/api/`)

```bash
cargo fmt -- --check          # フォーマットチェック
cargo clippy -- -D warnings   # Lintチェック
cargo test --verbose          # ユニットテスト
cargo build --release         # リリースビルド
cargo run                     # ローカル起動 (localhost:8080)
```

### React UI (`web/ui/`)

```bash
npm run lint                  # ESLint
npm run type-check            # TypeScript型チェック
npm test -- --coverage --watch=false  # ユニットテスト
npm run build                 # プロダクションビルド
npm run dev                   # ローカル起動 (localhost:3000)
```

### Terraform (`web/resources/`)

```bash
terraform fmt -check -recursive   # フォーマットチェック
terraform init -backend=false && terraform validate  # バリデーション
```

## コミット前の必須チェック

**pre-commitフック（Husky）が以下を自動実行する:**

- Prettier による自動フォーマット（JS/TS/JSON/CSS/MD）
- ESLint チェック（TS/TSX変更時）
- `cargo fmt --check` + `cargo clippy`（.rs変更時）
- `terraform fmt -check`（.tf変更時）

**テストはpre-commitに含まれない**ため、`/verify` で別途実行すること:

1. Rust: `cd web/api && cargo test`
2. React: `cd web/ui && npm run type-check && npm test -- --watch=false`

## コーディング規約

### 命名規約（4層ティア）

プロジェクト全体で以下の4層命名規約を適用する。

| Tier | 名称           | 形式                  | 用途                                              | 例                         |
| ---- | -------------- | --------------------- | ------------------------------------------------- | -------------------------- |
| 1    | Display Name   | `SpotDiggz`           | UI表示、ロゴ、ドキュメント見出し                  | `<title>SpotDiggz</title>` |
| 2    | Machine Name   | `spotdiggz`           | リポジトリ名、ディレクトリ名、ドメイン            | `uechikohei/spotdiggz`     |
| 3    | Infra Resource | `sdz-{env}-{purpose}` | GCPリソース名（全小文字・ハイフンのみ）           | `sdz-dev-img-spots`        |
| 4    | Source Code    | 言語規約準拠          | コード内識別子（`sdz`/`Sdz`/`SDZ`プレフィックス） | `sdzUserProfile`           |

#### Tier 1: Display Name（表示名）

- **`SpotDiggz`**（PascalCase、スペースなし）
- UI上のアプリ名、ロゴ、ドキュメント見出し、OGP等で使用

#### Tier 2: Machine Name（マシン名）

- **`spotdiggz`**（全小文字、区切りなし）
- リポジトリ名、ディレクトリ名、ドメイン、パッケージ名等で使用

#### Tier 3: Infrastructure Resource Name（インフラリソース名）

- **全小文字・ハイフンのみ**（アンダースコア禁止）
  - GCP/AWSタグの小文字正規化でキー衝突を防止
  - BigQuery/Athena連携時のアンダースコア制約を回避
- セグメント構成: `sdz-{env}-{purpose}`
- 冗長な型サフィックスは付けない（例: Cloud Storageに`bucket`は不要）

| リソース          | 命名パターン            | 例                    |
| ----------------- | ----------------------- | --------------------- |
| GCPプロジェクト   | `sdz-{env}`             | `sdz-dev`             |
| Cloud Run         | `sdz-{env}-api`         | `sdz-dev-api`         |
| 画像Storage       | `sdz-{env}-img-spots`   | `sdz-dev-img-spots`   |
| UIホスティング    | `sdz-{env}-ui-hosts`    | `sdz-dev-ui-hosts`    |
| Service Account   | `sdz-{env}-{role}-sa`   | `sdz-dev-api-sa`      |
| Artifact Registry | `sdz-{env}-api`         | `sdz-dev-api`         |
| WIF Pool          | `sdz-{env}-github-pool` | `sdz-dev-github-pool` |

#### Tier 4: Source Code Name（ソースコード名）

全識別子に `sdz`/`Sdz`/`SDZ` プレフィックスを付与する。

| 言語       | 変数・関数                       | 型・構造体                  | 定数                                  | ファイル名            |
| ---------- | -------------------------------- | --------------------------- | ------------------------------------- | --------------------- |
| TypeScript | `camelCase`: `sdzUserProfile`    | `PascalCase`: `SdzSpotData` | `SCREAMING_SNAKE`: `SDZ_API_BASE_URL` | `SdzSpotCard.tsx`     |
| Rust       | `snake_case`: `sdz_user_profile` | `PascalCase`: `SdzSpotData` | `SCREAMING_SNAKE`: `SDZ_MAX_SPOTS`    | `sdz_spot_service.rs` |
| Terraform  | `snake_case`: `sdz_api`          | -                           | -                                     | `sdz_cloud_run.tf`    |

### Gitコミットメッセージ

- プレフィックスは英語（Conventional Commits準拠）: `feat` / `fix` / `docs` / `refactor` / `test` / `chore` / `style` / `perf` / `ci`
- スコープ: `(ios)` / `(api)` / `(web)` / `(infra)` / `(repo)`
- タイトル・詳細メッセージは日本語で記載
- 詳細（-m body）は箇条書きで変更内容を簡潔に記載
- Co-Authored-By行は付けない（著者はuechikoheiのみとする）

```
feat(ios): スポット投稿画面のバリデーションを追加

- 必須項目の未入力チェックを実装
- エラーメッセージを日本語で表示
```

### ブランチ戦略

GitHub Flow を使用する

## 課題管理（GitHub Issues/Projects）

### 起票フォーマット（自動判定）

`/issue` コマンドで起票時、内容に応じてフォーマットを自動選択する:

| 判定条件                           | フォーマット                                 | ラベル                |
| ---------------------------------- | -------------------------------------------- | --------------------- |
| 既に発生した障害・バグ・技術調査   | **4F形式**（Fact/Find/Fix/Future）           | `troubleshooting`     |
| 未来の検討・新機能・設計・要件整理 | **STAR形式**（Situation/Task/Action/Result） | `planning` / `design` |
| 上記に当てはまらないナレッジ・メモ | **フリーフォーマット**                       | `knowledge`           |

### 起票時の必須設定

- label: `troubleshooting` / `planning` / `design` / `knowledge`
- Priority: `P0`（本番障害）/ `P1`（開発に支障）/ `P2`（改善・バックログ）

### トラブルシューティング自動起票

Claude Codeの作業中にトラブルシューティングが発生し解消した場合、自動的に `/issue` を実行して4F形式で知見を記録する（起票前にユーザー承認を取る）。

### 4F参照の優先

動作テスト異常・エラーログ・障害調査の際は、`troubleshooting` / `knowledge` ラベル付きIssueを優先的に参照して過去事象を確認する。

## よくある落とし穴と回避策

- FirebaseのAPIキーをローテーションした場合、`GoogleService-Info.plist`の`API_KEY`は自動更新されないため手動で差し替え（ファイルはGit追跡しない）
- iOSで期待動作しない場合「設定 > 一般 > VPNとデバイス管理」で開発者アプリの信頼確認（Issue #188）
- Terraformバージョンは `web/.terraform-version` で固定（tfenv想定）
- `.env`ファイル・シークレット情報は絶対にコミットしない（`.env.tpl` は `op://` 参照のみなのでコミットOK）
- シークレット値の追加・変更時は 1Password アイテムと `.env.tpl` の両方を更新する（手順: `docs/secrets_management.md`）

## ローカル起動手順

### 前提条件

- 1Password CLI (`op`): `brew install --cask 1password-cli`
- 1Password デスクトップアプリとの CLI 連携を有効化（推奨）
- `gcloud` CLI 認証済み: `gcloud auth login`

### シークレット取得（初回 or 更新時）

```bash
make secrets    # 1Password から .env / .env.local を生成（gcloud token も自動取得）
```

### サーバー起動

```bash
# 1) Rust API
cd web/api
set -a && source ./.env && set +a
cargo run

# 2) React UI（別ターミナル）
cd web/ui && npm install && npm run dev

# 3) 確認
# UI: http://localhost:3000
# API: http://localhost:8080/sdz/health
```

📋 `SDZ_FIRESTORE_TOKEN` は `make secrets` 実行時に `gcloud auth print-access-token` で自動取得される。
トークン期限切れ時は `make secrets` を再実行するか、手動で `export SDZ_FIRESTORE_TOKEN=$(gcloud auth print-access-token)` を実行する。

## スラッシュコマンド

| カテゴリ         | コマンド         | 用途                                                                           |
| ---------------- | ---------------- | ------------------------------------------------------------------------------ |
| **日次スキャン** | `/daily-scan`    | 全レイヤー統合スキャン（Phase 0〜5）→ 自動Issue起票                            |
|                  | `/scan-infra`    | インフラ: Terraform, GCP Provider, GCPサービス, CI Actions, tfsec              |
|                  | `/scan-backend`  | バックエンド: Rust依存関係, Docker base image, OpenSSL, Rust toolchain         |
|                  | `/scan-frontend` | フロントエンド: npm依存関係, Node.js, React, Firebase JS SDK, サプライチェーン |
|                  | `/scan-ios`      | iOS: Swift Package, Firebase iOS SDK, GoogleSignIn, gRPC, Xcode/Swift          |
| **品質チェック** | `/verify`        | ローカルCI実行（fmt/clippy/test/lint/type-check/build）                        |
| **レビュー**     | `/review`              | 差分のセキュリティ・バグ・命名規則・シークレット漏洩レビュー                   |
|                  | `/simplify`            | 変更差分のコード簡素化                                                         |
|                  | `/review-appstore`     | App Store審査ガイドライン準拠チェック（iOS実装のストア申請レビュー）            |
| **AIDDレビュー** | `/design-review`       | 設計レビュー（直斗＋八幡＋臨也）                                               |
|                  | `/code-review`         | コードレビュー（八幡＋雪乃＋ヒュルケン）                                       |
|                  | `/ux-review`           | UI/UXレビュー（玉縄＋雪乃）                                                    |
|                  | `/cost-optimization`   | コスト最適化相談（雪乃＋臨也）                                                 |
|                  | `/security-review`     | セキュリティレビュー（ヒュルケン＋臨也＋直斗）                                 |
|                  | `/team-review`         | 全員参加フルレビュー会議（6名全員）                                            |
| **課題管理**     | `/issue [内容]`        | STAR/4F形式を自動選択しIssue起票 → Project追加                                 |

### `/daily-scan` のフェーズ構成

```
Phase 0: リポジトリ状態（daily-scan内蔵）
Phase 1: /scan-infra 相当
Phase 2: /scan-backend 相当
Phase 3: /scan-frontend 相当
Phase 4: /scan-ios 相当
Phase 5: 統合サマリ + Issue自動起票（daily-scan内蔵）
```

### 推奨ワークフロー

```
日次:      /daily-scan → 全レイヤーの脆弱性・バージョンチェック（自動Issue起票）
開発中:    /review     → セルフレビュー
          /simplify   → 必要に応じて簡素化
コミット前: /verify     → ローカルCI確認
随時:      /issue      → 課題・障害の起票
```

### 前提条件

```bash
cargo install cargo-audit cargo-outdated   # Rust監査ツール
gh auth login                              # GitHub CLI認証
```

## Plan Mode運用ルール

- 新機能実装は必ずPlan Mode（Shift+Tab x2）で設計を合意してから実装に入る
- 「コードを1行も書かせずに設計承認」の原則を守る
- 合意後にAuto-accept editsモードへ移行し、一撃実装する

## 実装完了後の品質チェック（必須）

コード実装が一段落したら、コミット前に必ず `/verify` を実行すること。
ユーザーから指示がなくても、以下の条件を満たしたら自発的に `/verify` を実行する:

- Plan Mode承認後の実装が完了したとき
- バグ修正やリファクタリングの作業が完了したとき
- ユーザーが「できた」「完了」「実装して」等の完了を示す発言をしたとき

`/verify` で失敗が見つかった場合は、修正してから再度 `/verify` を実行し、全PASSを確認してからユーザーに報告する。

## 環境変数

### API (`web/api/.env`)

`web/api/.env.example` をコピーして `web/api/.env` を作成する（秘匿情報はコミットしない）。

| 変数名                                | 説明                                                                               |
| ------------------------------------- | ---------------------------------------------------------------------------------- |
| `SDZ_AUTH_PROJECT_ID`                 | Firebase/Identity PlatformのプロジェクトID（例: sdz-dev）                          |
| `SDZ_USE_FIRESTORE`                   | `1` でFirestore利用、未設定ならインメモリ                                          |
| `SDZ_FIRESTORE_PROJECT_ID`            | FirestoreのプロジェクトID（省略時はSDZ_AUTH_PROJECT_IDを使用）                     |
| `SDZ_FIRESTORE_TOKEN`                 | Firestore RESTに使うBearerトークン（`gcloud auth print-access-token`）             |
| `SDZ_CORS_ALLOWED_ORIGINS`            | カンマ区切りの許可オリジン（未設定時はlocalhost:3000のみ）                         |
| `SDZ_STORAGE_BUCKET`                  | 画像アップロード先のCloud Storageバケット名                                        |
| `SDZ_STORAGE_SERVICE_ACCOUNT_EMAIL`   | 署名URL生成に使うサービスアカウントのメール                                        |
| `SDZ_STORAGE_SIGNED_URL_EXPIRES_SECS` | 署名URLの有効期限（秒、デフォルト900）                                             |
| `SDZ_STORAGE_SIGNING_TOKEN`           | 署名URL生成に使うアクセストークン（未設定時はSDZ_FIRESTORE_TOKENやメタデータ経由） |

### UI (`web/ui/.env.local`)

`VITE_*` プレフィックスの変数を設定する。詳細は `docs/DEVELOPMENT_SETUP.md` を参照。

## APIエンドポイント（現在の実装状況）

📋 2026-02-28 データアーキテクチャ再設計により、APIは Tier 1 マスターデータの読み取り配信が主な役割。
書き込みエンドポイントは休眠中（将来の Tier 2 → Tier 1 昇格で再利用の可能性）。
詳細: `docs/designs/tier2-spot-data-architecture.md`

| メソッド | パス                    | 認証 | 説明                                              | ステータス |
| -------- | ----------------------- | ---- | ------------------------------------------------- | ---------- |
| GET      | `/sdz/health`           | 不要 | ヘルスチェック                                    | ✅ 稼働中  |
| GET      | `/sdz/users/me`         | 必須 | Firebase IDトークンを検証し、ユーザー情報を返却   | ✅ 稼働中  |
| GET      | `/sdz/spots`            | 不要 | Tier 1 スポット一覧取得                           | ✅ 稼働中  |
| GET      | `/sdz/spots/{id}`       | 不要 | Tier 1 スポット詳細取得（存在しなければ404）      | ✅ 稼働中  |
| POST     | `/sdz/spots`            | 必須 | スポット作成                                      | 💤 休眠中  |
| PATCH    | `/sdz/spots/{id}`       | 必須 | スポット更新                                      | 💤 休眠中  |
| POST     | `/sdz/spots/upload-url` | 必須 | 画像アップロード用の署名URL発行                   | 💤 休眠中  |
| POST     | `/sdz/mylists/{id}`     | 必須 | マイリストにスポット追加                          | 💤 休眠中  |
| DELETE   | `/sdz/mylists/{id}`     | 必須 | マイリストからスポット削除                        | 💤 休眠中  |
| GET      | `/sdz/mylists`          | 必須 | マイリスト一覧取得                                | 💤 休眠中  |

## 動作確認手順（IDトークン取得→CRUDスモーク）

```bash
# 1) 環境変数を用意（秘匿情報は.env.localなど非追跡ファイルに保存）
export SDZ_FIREBASE_WEB_API_KEY="YOUR_FIREBASE_WEB_API_KEY"
export SDZ_TEST_USER_EMAIL="YOUR_TEST_EMAIL"
export SDZ_TEST_USER_PASSWORD="YOUR_TEST_PASSWORD"

# 2) Firebase Auth REST APIでIDトークン取得
payload=$(jq -n --arg email "${SDZ_TEST_USER_EMAIL}" \
  --arg password "${SDZ_TEST_USER_PASSWORD}" \
  '{email:$email,password:$password,returnSecureToken:true}')
SDZ_ID_TOKEN=$(curl -sS "https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=${SDZ_FIREBASE_WEB_API_KEY}" \
  -H "Content-Type: application/json" \
  -d "${payload}" | jq -r '.idToken')

# 3) CRUDスモークテスト実行
SDZ_API_URL=http://localhost:8080 SDZ_ID_TOKEN="${SDZ_ID_TOKEN}" ./web/scripts/firestore_crud_smoke.sh
```

## ドキュメント参照

- [開発環境セットアップ](docs/DEVELOPMENT_SETUP.md)
- [シークレット管理（1Password CLI）](docs/secrets_management.md)
- [CD設計](docs/cd_architecture.md)
- [dev seed運用ルール](docs/seed_runbook.md)
- [ユーザー識別ポリシー](docs/user_identity_policy.md)

## 自律動作の方針

### 許可を求めずに進めてよい操作

- ファイルの読み取り・編集・新規作成
- ビルド・テスト・Lint・フォーマッターの実行
- git add / git commit / git status / git diff / git log
- Plan承認後の実装作業全般

### 必ずユーザーに確認が必要な操作

- ファイルやディレクトリの削除（特に `rm -rf` 等の再帰削除）
- `git push`（リモートへの反映）
- `git reset --hard` / `git push --force` / `git rebase` 等の履歴改変操作
- バックアップなしでの不可逆な変更
- 本番環境・共有リソースへの影響がある操作

## AIDDレビュワーチーム

このプロジェクトではAIDDペルソナレビュワーを使用する。
新機能の相談・実装相談・「〇〇を作りたい」という要望が来たら、
`.claude/skills/initial-hearing/SKILL.md` を読み込み、白鐘直斗としてヒアリングを行うこと。
ペルソナ定義は `.claude/skills/personas/naoto.md` を参照すること。
