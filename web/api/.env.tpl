# spot-diggz API 環境変数テンプレート
# op inject で 1Password から秘匿値を注入する
# 使い方: op inject -i web/api/.env.tpl -o web/api/.env

# --- 非秘匿設定（直接記載） ---
RUST_LOG=debug
SDZ_AUTH_PROJECT_ID=sdz-dev
SDZ_USE_FIRESTORE=1
SDZ_FIRESTORE_PROJECT_ID=sdz-dev
SDZ_CORS_ALLOWED_ORIGINS=http://localhost:3000
SDZ_STORAGE_SIGNED_URL_EXPIRES_SECS=900

# --- 動的取得（スクリプトで後から上書き） ---
SDZ_FIRESTORE_TOKEN=REPLACE_BY_SCRIPT
SDZ_STORAGE_SIGNING_TOKEN=REPLACE_BY_SCRIPT

# --- 秘匿値（1Password から注入） ---
YOUR_FIREBASE_WEB_API_KEY=op://sdz-dev/api/FIREBASE_WEB_API_KEY
TEST_USER_ID=op://sdz-dev/api/TEST_USER_ID
TEST_USER_PASSWORD=op://sdz-dev/api/TEST_USER_PASSWORD
SDZ_STORAGE_BUCKET=op://sdz-dev/api/STORAGE_BUCKET
SDZ_STORAGE_SERVICE_ACCOUNT_EMAIL=op://sdz-dev/api/STORAGE_SERVICE_ACCOUNT_EMAIL
