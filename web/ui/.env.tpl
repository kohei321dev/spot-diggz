# spot-diggz UI 環境変数テンプレート
# op inject で 1Password から秘匿値を注入する
# 使い方: op inject -i web/ui/.env.tpl -o web/ui/.env.local

# --- 非秘匿設定（直接記載） ---
VITE_SDZ_API_URL=http://localhost:8080
VITE_FIREBASE_AUTH_DOMAIN=sdz-dev.firebaseapp.com
VITE_FIREBASE_PROJECT_ID=sdz-dev

# --- 秘匿値（1Password から注入） ---
VITE_FIREBASE_API_KEY=op://sdz-dev/ui/FIREBASE_API_KEY
VITE_GOOGLE_MAPS_API_KEY=op://sdz-dev/ui/GOOGLE_MAPS_API_KEY
