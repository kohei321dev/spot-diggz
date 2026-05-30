# シークレット管理ガイド（1Password CLI）

spot-diggz では 1Password CLI (`op`) を使ってシークレットを一元管理する。
`.env.tpl` テンプレートに `op://` 参照を記述し、`op inject` で実際の値を注入する。

## 構成

```
1Password Vault: sdz-dev
├── アイテム: api     ← web/api/.env.tpl が参照
│   ├── FIREBASE_WEB_API_KEY
│   ├── TEST_USER_ID
│   ├── TEST_USER_PASSWORD
│   ├── STORAGE_BUCKET
│   └── STORAGE_SERVICE_ACCOUNT_EMAIL
└── アイテム: ui      ← web/ui/.env.tpl が参照
    └── FIREBASE_API_KEY
```

### ファイル構成

| ファイル | Git管理 | 説明 |
|----------|---------|------|
| `web/api/.env.tpl` | ✅ コミット | `op://` 参照を含むテンプレート |
| `web/ui/.env.tpl` | ✅ コミット | `op://` 参照を含むテンプレート |
| `web/api/.env` | ❌ gitignore | `op inject` で生成される実ファイル |
| `web/ui/.env.local` | ❌ gitignore | `op inject` で生成される実ファイル |
| `web/api/.env.example` | ✅ コミット | プレースホルダ付きの参考ファイル（従来互換） |

## 初期セットアップ（新規開発者向け）

### 1. 1Password CLI のインストール

```bash
# macOS
brew install --cask 1password-cli

# バージョン確認
op --version
```

### 2. 1Password デスクトップアプリとの連携（推奨）

1Password デスクトップアプリ → 設定 → 開発者 → 「CLI とデスクトップアプリを連携」を有効化する。
これにより `eval $(op signin)` が不要になり、生体認証でCLI操作が可能になる。

### 3. Vault へのアクセス確認

```bash
# Vault 一覧を確認
op vault list

# sdz-dev Vault が表示されることを確認
op vault get sdz-dev
```

`sdz-dev` Vault が見えない場合は、チームの管理者に Vault 共有を依頼する。

### 4. シークレットの取得

```bash
# プロジェクトルートで実行
make secrets
```

これで以下が生成される:
- `web/api/.env` — API用の環境変数（gcloud token も自動取得）
- `web/ui/.env.local` — UI用の環境変数

## 日常の運用

### シークレットの取得・更新

```bash
# 全ファイルを再生成
make secrets

# API のみ再生成
make secrets-api

# UI のみ再生成
make secrets-ui
```

### gcloud トークンの期限切れ

`SDZ_FIRESTORE_TOKEN` と `SDZ_STORAGE_SIGNING_TOKEN` は gcloud access token であり、
デフォルトで 1 時間で失効する。失効したら以下のいずれかで更新:

```bash
# 方法 1: make secrets を再実行（推奨）
make secrets

# 方法 2: 手動で export
export SDZ_FIRESTORE_TOKEN=$(gcloud auth print-access-token)
export SDZ_STORAGE_SIGNING_TOKEN=$(gcloud auth print-access-token)
```

## 管理者向け: Vault・アイテムの初期構築

### 1. Vault の作成

```bash
op vault create sdz-dev
```

### 2. API 用アイテムの作成

```bash
op item create \
  --vault sdz-dev \
  --category login \
  --title api \
  'FIREBASE_WEB_API_KEY=YOUR_ACTUAL_API_KEY' \
  'TEST_USER_ID=YOUR_TEST_EMAIL' \
  'TEST_USER_PASSWORD=YOUR_TEST_PASSWORD' \
  'STORAGE_BUCKET=YOUR_BUCKET_NAME' \
  'STORAGE_SERVICE_ACCOUNT_EMAIL=YOUR_SA_EMAIL'
```

### 3. UI 用アイテムの作成

```bash
op item create \
  --vault sdz-dev \
  --category login \
  --title ui \
  'FIREBASE_API_KEY=YOUR_ACTUAL_API_KEY'
```

## シークレットの追加・変更手順

新しいシークレットを追加する場合:

### 1. 1Password にフィールドを追加

```bash
# 例: API アイテムに NEW_SECRET フィールドを追加
op item edit api --vault sdz-dev 'NEW_SECRET=the_secret_value'
```

### 2. `.env.tpl` に `op://` 参照を追加

```bash
# web/api/.env.tpl に追記
NEW_SECRET=op://sdz-dev/api/NEW_SECRET
```

### 3. コミット & チーム共有

```bash
git add web/api/.env.tpl
git commit -m "chore(api): NEW_SECRET を .env.tpl に追加"
```

チームメンバーは `git pull` 後に `make secrets` を実行すれば新しいシークレットが反映される。

## op CLI 未導入時のフォールバック

`op` CLI がインストールされていない、または 1Password アカウントがない場合:

1. `web/api/.env.example` を `web/api/.env` にコピーする
2. 各プレースホルダを手動で実際の値に置き換える
3. チームメンバーからシークレット値を安全な方法で受け取る

```bash
cp web/api/.env.example web/api/.env
# .env を手動編集
```

## セキュリティ上の注意

- `.env` / `.env.local` ファイルは絶対にコミットしない（`.gitignore` で除外済み）
- `.env.tpl` ファイルは `op://` 参照のみを含むため、コミットして問題ない
- シークレット値を Slack・メール・Issue 等の平文チャネルで共有しない
- 1Password の共有 Vault を通じてシークレットを共有する
