# spot-diggz 開発環境セットアップガイド

## 🚀 GitHub Codespaces + ローカルVS Code 開発環境

spot-diggzプロジェクトは、GitHub Codespacesとローカル VS Codeを連携した統合開発環境を提供しています。

## 📋 前提条件

- GitHub アカウント
- VS Code（ローカル開発用）
- GitHub Codespaces拡張機能

## 🌐 GitHub Codespaces セットアップ

### 1. Codespacesの作成

```bash
# GitHubリポジトリページから
1. 「Code」ボタンをクリック
2. 「Codespaces」タブを選択
3. 「Create codespace on develop」をクリック
```

### 2. 自動セットアップ

Codespacesが起動すると、`.devcontainer/setup.sh`が自動実行され、以下がセットアップされます：

- 🦀 Rust開発環境（cargo, rustc, clippy, rust-analyzer）
- 📦 Node.js/npm環境（React開発用）
- 🏗️ Terraform環境
- ☁️ Google Cloud CLI
- 🐳 Docker CLI
- ⚙️ VS Code拡張機能
- 📁 プロジェクト構造

### 3. 開発サービスの起動

```bash
# 個別サービス起動
cd web/api
set -a
source ./.env
set +a
export SDZ_FIRESTORE_TOKEN=$(gcloud auth print-access-token)
cargo run                     # Rust API サーバー (ポート8080)

cd ../ui
npm run dev                   # React 開発サーバー (ポート3000)
```

## 💻 ローカル VS Code 連携

### 1. GitHub Codespaces 拡張機能インストール

```bash
# VS Code 拡張機能
- GitHub Codespaces
- Remote - Containers
- Remote - SSH
```

### 2. Codespacesへの接続

1. VS Codeのコマンドパレット（`Ctrl+Shift+P`）を開く
2. `Codespaces: Connect to Codespace`を実行
3. 作成済みのCodespaceを選択

### 3. ワークスペースの開き方

```bash
# ワークスペースファイルを開く
File > Open Workspace from File > spot-diggz.code-workspace
```

## 🛠️ 開発ワークフロー

### プロジェクト構造

```
spot-diggz/
├── .devcontainer/          # Codespaces設定
│   ├── devcontainer.json   # VS Code + 拡張機能設定
│   ├── Dockerfile          # 開発環境イメージ
│   └── setup.sh           # 自動セットアップスクリプト
├── .github/               # GitHub Actionsなど
├── web/                   # Webアプリ（API/UI/IaC）
│   ├── api/               # 🦀 Rust APIサーバー
│   ├── ui/                # ⚛️ React UIアプリ
│   ├── resources/         # 🏗️ Terraform インフラ
│   ├── scripts/           # 🔧 開発用スクリプト
│   └── sample/            # 🧪 Seed用の画像サンプル
├── docs/                  # 📚 ドキュメント
├── iOS/                   # iOSアプリ（予定）
├── android/               # Androidアプリ（予定）
├── AGENTS.md              # Codex運用ルール
├── .gitignore             # 追跡対象外ファイル
├── README.md              # リポジトリ概要
└── spot-diggz.code-workspace  # VS Code ワークスペース設定
```

### 開発ルール

- Tier 1 マスターデータ: BigQuery → Cloud Functions → Firestore のパイプラインで投入。API は読み取り専用。
- Tier 2 ユーザーデータ: iOS SwiftData + CloudKit でローカル管理。API / Firestore は不使用。
- マイリスト: iOS SwiftData + CloudKit でローカル管理（Firestore API 経由から移行）。
- 詳細: `docs/designs/tier2-spot-data-architecture.md`

## 🧰 ローカル環境のTerraform（tfenv）

Codespacesとバージョンを揃えるため、`web/.terraform-version` を参照して固定する。

```bash
# tfenvのインストール（macOS想定）
brew install tfenv

# リポジトリのバージョンをインストール＆適用
cd web
tfenv install
tfenv use

# バージョン確認
terraform version
```

### VS Code タスク

`Ctrl+Shift+P` > `Tasks: Run Task` で以下のタスクを実行：

| タスク | 説明 | ショートカット |
|--------|------|-------------|
| 🦀 Build Rust API | Rust APIビルド | `Ctrl+Shift+B` |
| 🦀 Run Rust API | Rust API起動 | - |
| 🦀 Test Rust API | Rust APIテスト | `Ctrl+Shift+T` |
| ⚛️ Start React Dev Server | React開発サーバー起動 | - |
| ⚛️ Build React UI | React UIビルド | - |
| ⚛️ Test React UI | React UIテスト | - |
| 🏗️ Terraform Plan | インフラプラン表示 | - |
| 🚀 Start Development Environment | 開発環境一括起動 | - |

### デバッグ設定

- **Rust API デバッグ**: `F5`キーで `🦀 Debug Rust API` 設定実行
- **ブレークポイント**: コード行番号左をクリック
- **変数確認**: デバッグ時に Variables パネルで確認

## 🔧 環境固有設定

### Google Cloud 認証

```bash
# Codespaces内で実行
gcloud auth login
gcloud config set project sdz-dev  # 開発環境プロジェクト
```

### 環境変数設定（1Password CLI 推奨）

```bash
# 1Password CLI のインストール（初回のみ）
brew install --cask 1password-cli

# プロジェクトルートで実行 → .env / .env.local が自動生成される
make secrets
```

`make secrets` は以下を行う:
- `web/api/.env.tpl` から `web/api/.env` を生成（1Password から秘匿値を注入）
- `web/ui/.env.tpl` から `web/ui/.env.local` を生成
- `gcloud auth print-access-token` で Firestore/Storage トークンを自動取得

詳細は [シークレット管理ガイド](secrets_management.md) を参照。

#### 1Password を使わない場合（フォールバック）

```bash
# Rust API用
cd web/api
cp .env.example .env
# .env を手動編集して実際の値に置き換える（コミットしない）
export SDZ_FIRESTORE_TOKEN=$(gcloud auth print-access-token)

# UI用
cd ../ui
cat > .env.local << 'EOF'
VITE_SDZ_API_URL=http://localhost:8080
VITE_FIREBASE_API_KEY=your_api_key
VITE_FIREBASE_AUTH_DOMAIN=your_auth_domain
VITE_FIREBASE_PROJECT_ID=your_project_id
EOF
```

## 📊 ポートフォワーディング

Codespacesで自動的に以下のポートがフォワードされます：

| ポート | サービス | アクセス |
|--------|----------|----------|
| 3000 | React UI | https://xxx-3000.githubpreview.dev |
| 8080 | Rust API | https://xxx-8080.githubpreview.dev |

## 🔍 トラブルシューティング

### よくある問題

#### 1. Rust コンパイルエラー

```bash
# Rust toolchain確認
rustc --version
cargo --version

# 依存関係更新
cd web/api
cargo update
```

#### 2. Node.js 依存関係エラー

```bash
# npm キャッシュクリア
cd web/ui
npm cache clean --force
rm -rf node_modules package-lock.json
npm install
```

#### 3. ポート競合エラー

```bash
# 使用中ポート確認
lsof -i :3000
lsof -i :8080

# プロセス終了
kill -9 <PID>
```

#### 4. Docker サービス起動失敗

```bash
# Docker状態確認
docker ps -a
```

### ログ確認

```bash
# Codespacesセットアップログ
cat /tmp/codespace-creation.log

# アプリケーションログ
cd web/api && cargo run       # Rust ログ
cd web/ui && npm run dev      # React ログ
```

## 📱 モバイル開発

GitHub Mobile アプリでもCodespacesにアクセス可能：

1. GitHub Mobile アプリインストール
2. リポジトリ > Codespaces から接続
3. ブラウザベース VS Code で編集

## 🔄 Codespaces ライフサイクル

### 一時停止・再開

```bash
# 自動一時停止：30分非アクティブ後
# 手動操作：GitHub > Codespaces > Stop codespace

# 再開時は状態が保持される（ファイル、環境変数、インストール済みツール）
```

### データ永続化

- **永続化される**：ホームディレクトリ（`/home/vscode`）
- **永続化される**：`/workspace`（プロジェクトファイル）
- **永続化されない**：Docker コンテナ内一時ファイル

### バックアップ推奨

```bash
# 重要な設定ファイルはGitで管理
git add .vscode/ .devcontainer/
git commit -m "update: 開発環境設定"
git push
```

## 🚀 高度な使用方法

### カスタム設定

```json
// .devcontainer/devcontainer.json カスタマイズ例
{
  "postCreateCommand": "bash .devcontainer/setup.sh && echo 'カスタム設定完了'",
  "containerEnv": {
    "CUSTOM_VAR": "custom_value"
  }
}
```

### 複数Codespaces運用

- **開発用**: develop ブランチ
- **実験用**: feature/experiment ブランチ  
- **本番確認用**: master ブランチ

## 📚 参考リンク

- [GitHub Codespaces Documentation](https://docs.github.com/en/codespaces)
- [VS Code Remote Development](https://code.visualstudio.com/docs/remote/remote-overview)
- [Dev Container specification](https://containers.dev/)

---

**🎉 Happy Coding with spot-diggz! 🛹**
