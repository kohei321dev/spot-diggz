#!/usr/bin/env bash
set -euo pipefail

# spot-diggz シークレット取得スクリプト
# 1Password CLI (op) を使って .env.tpl から .env を生成する

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# 色付き出力
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

info()  { echo -e "${GREEN}[INFO]${NC} $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }

# op CLI の存在チェック
check_op_cli() {
  if ! command -v op &> /dev/null; then
    error "1Password CLI (op) がインストールされていません"
    echo ""
    echo "インストール方法:"
    echo "  macOS:   brew install --cask 1password-cli"
    echo "  Linux:   https://developer.1password.com/docs/cli/get-started/"
    echo ""
    exit 1
  fi
}

# op CLI の認証チェック
check_op_auth() {
  if ! op account list &> /dev/null 2>&1; then
    error "1Password CLI が未認証です"
    echo ""
    echo "サインイン方法:"
    echo "  eval \$(op signin)"
    echo ""
    echo "1Password デスクトップアプリとの連携を推奨:"
    echo "  1Password > 設定 > 開発者 > 「CLI とデスクトップアプリを連携」を有効化"
    echo ""
    exit 1
  fi
}

# Vault の存在チェック
check_vault() {
  local vault_name="$1"
  if ! op vault get "${vault_name}" &> /dev/null 2>&1; then
    error "Vault '${vault_name}' が見つかりません"
    echo ""
    echo "Vault を作成してください:"
    echo "  op vault create '${vault_name}'"
    echo ""
    echo "必要なアイテムの作成方法は docs/secrets_management.md を参照してください"
    exit 1
  fi
}

# .env.tpl から .env を生成
inject_env() {
  local tpl_file="$1"
  local out_file="$2"
  local label="$3"

  if [[ ! -f "${tpl_file}" ]]; then
    error "テンプレートファイルが見つかりません: ${tpl_file}"
    return 1
  fi

  info "${label}: ${tpl_file} → ${out_file}"
  op inject -i "${tpl_file}" -o "${out_file}" --force
  info "${label}: 生成完了"
}

# 動的トークンの取得・追記
inject_dynamic_tokens() {
  local env_file="$1"

  if ! command -v gcloud &> /dev/null; then
    warn "gcloud CLI が見つかりません。SDZ_FIRESTORE_TOKEN / SDZ_STORAGE_SIGNING_TOKEN は手動で設定してください"
    return 0
  fi

  info "gcloud access token を取得中..."
  local token
  token=$(gcloud auth print-access-token 2>/dev/null) || {
    warn "gcloud auth print-access-token に失敗しました。gcloud auth login を実行してください"
    return 0
  }

  # sed で REPLACE_BY_SCRIPT を実際のトークンに置換
  if [[ "$(uname)" == "Darwin" ]]; then
    sed -i '' "s|SDZ_FIRESTORE_TOKEN=REPLACE_BY_SCRIPT|SDZ_FIRESTORE_TOKEN=${token}|" "${env_file}"
    sed -i '' "s|SDZ_STORAGE_SIGNING_TOKEN=REPLACE_BY_SCRIPT|SDZ_STORAGE_SIGNING_TOKEN=${token}|" "${env_file}"
  else
    sed -i "s|SDZ_FIRESTORE_TOKEN=REPLACE_BY_SCRIPT|SDZ_FIRESTORE_TOKEN=${token}|" "${env_file}"
    sed -i "s|SDZ_STORAGE_SIGNING_TOKEN=REPLACE_BY_SCRIPT|SDZ_STORAGE_SIGNING_TOKEN=${token}|" "${env_file}"
  fi
  info "動的トークンを .env に注入しました"
}

# メイン処理
main() {
  echo "=== spot-diggz シークレット取得 ==="
  echo ""

  check_op_cli
  check_op_auth
  check_vault "sdz-dev"

  echo ""

  # API 環境変数
  inject_env \
    "${PROJECT_ROOT}/web/api/.env.tpl" \
    "${PROJECT_ROOT}/web/api/.env" \
    "API"

  # 動的トークンの注入
  inject_dynamic_tokens "${PROJECT_ROOT}/web/api/.env"

  echo ""

  # UI 環境変数
  inject_env \
    "${PROJECT_ROOT}/web/ui/.env.tpl" \
    "${PROJECT_ROOT}/web/ui/.env.local" \
    "UI"

  echo ""
  info "全ての環境変数ファイルを生成しました"
  echo ""
  echo "次のステップ:"
  echo "  cd web/api && cargo run      # API サーバー起動"
  echo "  cd web/ui && npm run dev     # UI 開発サーバー起動"
}

main "$@"
