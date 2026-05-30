# 環境変数インベントリ

spot-diggz プロジェクトで使用される全環境変数の一覧。
1Password Vault `sdz-dev` への移行時の参照用。

---

## API（Rust — `web/api/.env`）

### 秘匿情報（1Password に格納すべき値）

| #   | 変数名                              | 用途                                              | データ型       | 値の例・形式                         | 参照箇所                 |
| --- | ----------------------------------- | ------------------------------------------------- | -------------- | ------------------------------------ | ------------------------ |
| 1   | `YOUR_FIREBASE_WEB_API_KEY`         | Firebase Auth REST API のキー（IDトークン取得用） | 文字列         | `AIzaSy...` (39文字)                 | スモークテストスクリプト |
| 2   | `TEST_USER_ID`                      | テスト用 Firebase ユーザーのメールアドレス        | メールアドレス | `user@example.com`                   | スモークテストスクリプト |
| 3   | `TEST_USER_PASSWORD`                | テスト用 Firebase ユーザーのパスワード            | 文字列         | パスワード文字列                     | スモークテストスクリプト |
| 4   | `SDZ_STORAGE_BUCKET`                | 画像アップロード先の Cloud Storage バケット名     | 文字列         | `sdz-dev-img-spots`                  | `router.rs`              |
| 5   | `SDZ_STORAGE_SERVICE_ACCOUNT_EMAIL` | 署名URL生成用サービスアカウントのメール           | メールアドレス | `sa@sdz-dev.iam.gserviceaccount.com` | `router.rs`              |
| 6   | `SDZ_ADMIN_UIDS`                    | 管理者の Firebase UID（カンマ区切り）             | CSV文字列      | `uid1,uid2`                          | `admin.rs`               |

### 動的トークン（`gcloud auth print-access-token` で取得、1時間で失効）

| #   | 変数名                      | 用途                                                                       | データ型        | 値の例・形式              | 参照箇所                                                                     |
| --- | --------------------------- | -------------------------------------------------------------------------- | --------------- | ------------------------- | ---------------------------------------------------------------------------- |
| 7   | `SDZ_FIRESTORE_TOKEN`       | Firestore REST API / Cloud Run メタデータサーバーのトークン                | Bearer トークン | `ya29.a0...` (長い文字列) | `router.rs`, `firestore_*_repository.rs`, `storage_signed_url_repository.rs` |
| 8   | `SDZ_STORAGE_SIGNING_TOKEN` | 署名URL生成用トークン（未設定時は `SDZ_FIRESTORE_TOKEN` にフォールバック） | Bearer トークン | `ya29.a0...`              | `storage_signed_url_repository.rs`                                           |

### 非秘匿設定（直接記載 OK）

| #   | 変数名                                | 用途                                                         | データ型      | デフォルト値                                      | 参照箇所               |
| --- | ------------------------------------- | ------------------------------------------------------------ | ------------- | ------------------------------------------------- | ---------------------- |
| 9   | `RUST_LOG`                            | ログレベル制御                                               | 文字列        | `debug`（ローカル）/ `info,sdz_api=debug`（本番） | `main.rs`              |
| 10  | `SDZ_LOG_FORMAT`                      | ログ出力形式（`json` で JSON 形式）                          | 文字列        | 未設定（compact 形式）                            | `main.rs`              |
| 11  | `SDZ_AUTH_PROJECT_ID`                 | Firebase/Identity Platform のプロジェクトID                  | 文字列        | `sdz-dev`                                         | `router.rs`, `auth.rs` |
| 12  | `SDZ_USE_FIRESTORE`                   | Firestore 利用フラグ（`1` で有効）                           | `1` or 未設定 | 未設定（インメモリ）                              | `router.rs`            |
| 13  | `SDZ_FIRESTORE_PROJECT_ID`            | Firestore のプロジェクトID（省略時は `SDZ_AUTH_PROJECT_ID`） | 文字列        | `sdz-dev`                                         | `router.rs`            |
| 14  | `SDZ_CORS_ALLOWED_ORIGINS`            | CORS 許可オリジン（カンマ区切り）                            | CSV文字列     | `http://localhost:3000`                           | `router.rs`            |
| 15  | `SDZ_STORAGE_SIGNED_URL_EXPIRES_SECS` | 署名URL有効期限（秒）                                        | 整数          | `900`                                             | `router.rs`            |
| 16  | `PORT`                                | サーバーリッスンポート（Cloud Run 自動設定）                 | 整数          | `8080`                                            | `main.rs`              |
| 17  | `K_SERVICE`                           | Cloud Run 実行検知（存在するかのみ確認）                     | 文字列        | Cloud Run が自動設定                              | `router.rs`            |

---

## UI（React/Vite — `web/ui/.env` + `web/ui/.env.local`）

### 秘匿情報（1Password に格納すべき値）

| #   | 変数名                     | 用途                                                          | データ型 | 値の例・形式         | 参照箇所                |
| --- | -------------------------- | ------------------------------------------------------------- | -------- | -------------------- | ----------------------- |
| 1   | `VITE_FIREBASE_API_KEY`    | Firebase Web SDK の API キー                                  | 文字列   | `AIzaSy...` (39文字) | `firebase.ts`           |
| 2   | `VITE_GOOGLE_MAPS_API_KEY` | Google Maps JavaScript API キー（管理画面の地図・Places検索） | 文字列   | `AIzaSy...` (39文字) | `SdzAdminMapPicker.tsx` |

### 非秘匿設定（直接記載 OK）

| #   | 変数名                      | 用途                          | データ型  | デフォルト値              | 参照箇所                                                                    |
| --- | --------------------------- | ----------------------------- | --------- | ------------------------- | --------------------------------------------------------------------------- |
| 3   | `VITE_SDZ_API_URL`          | バックエンド API のベース URL | URL文字列 | `http://localhost:8080`   | `SdzAdminApi.ts`, `SdzAdminSpotForm.tsx`, `SdzAdminSpotList.tsx`, `App.tsx` |
| 4   | `VITE_FIREBASE_AUTH_DOMAIN` | Firebase Auth ドメイン        | ドメイン  | `sdz-dev.firebaseapp.com` | `firebase.ts`                                                               |
| 5   | `VITE_FIREBASE_PROJECT_ID`  | Firebase プロジェクトID       | 文字列    | `sdz-dev`                 | `firebase.ts`                                                               |

### 未使用（削除候補）

| #   | 変数名                | 状況                                                                                            |
| --- | --------------------- | ----------------------------------------------------------------------------------------------- |
| 6   | `VITE_SDZ_ADMIN_UIDS` | `.env` に定義あるが、UI コード内で参照なし。管理者チェックは API 側 `SDZ_ADMIN_UIDS` のみで実施 |

---

## 1Password Vault `sdz-dev` 登録対象まとめ

### アイテム: `api`（API用シークレット）

| フィールド名                    | 対応する環境変数                    | 備考                                                     |
| ------------------------------- | ----------------------------------- | -------------------------------------------------------- |
| `FIREBASE_WEB_API_KEY`          | `YOUR_FIREBASE_WEB_API_KEY`         | Firebase コンソール → プロジェクト設定 → ウェブ API キー |
| `TEST_USER_ID`                  | `TEST_USER_ID`                      | スモークテスト用メールアドレス                           |
| `TEST_USER_PASSWORD`            | `TEST_USER_PASSWORD`                | スモークテスト用パスワード                               |
| `STORAGE_BUCKET`                | `SDZ_STORAGE_BUCKET`                | GCP Console → Cloud Storage → バケット名                 |
| `STORAGE_SERVICE_ACCOUNT_EMAIL` | `SDZ_STORAGE_SERVICE_ACCOUNT_EMAIL` | GCP Console → IAM → サービスアカウント                   |
| `ADMIN_UIDS`                    | `SDZ_ADMIN_UIDS`                    | Firebase Console → Authentication → UID                  |

### アイテム: `ui`（UI用シークレット）

| フィールド名          | 対応する環境変数           | 備考                                                     |
| --------------------- | -------------------------- | -------------------------------------------------------- |
| `FIREBASE_API_KEY`    | `VITE_FIREBASE_API_KEY`    | Firebase コンソール → プロジェクト設定 → ウェブ API キー |
| `GOOGLE_MAPS_API_KEY` | `VITE_GOOGLE_MAPS_API_KEY` | GCP Console → APIs & Services → 認証情報 → API キー      |

### 1Password に入れないもの

| 変数                                                | 理由                                                                       |
| --------------------------------------------------- | -------------------------------------------------------------------------- |
| `SDZ_FIRESTORE_TOKEN` / `SDZ_STORAGE_SIGNING_TOKEN` | `gcloud auth print-access-token` で動的取得（1時間で失効するため保管不適） |
| `RUST_LOG`, `PORT`, `K_SERVICE` 等の非秘匿設定      | `.env.tpl` に直接記載                                                      |
| `VITE_SDZ_API_URL`, `VITE_FIREBASE_AUTH_DOMAIN` 等  | 公開情報、`.env.tpl` に直接記載                                            |

---

## 現在の .env.tpl との差分

### `web/api/.env.tpl` に不足しているフィールド

| 変数名           | 現状                                 | 対応                                                                  |
| ---------------- | ------------------------------------ | --------------------------------------------------------------------- |
| `SDZ_ADMIN_UIDS` | `.env` にのみ存在、`.env.tpl` になし | `.env.tpl` に `SDZ_ADMIN_UIDS=op://sdz-dev/api/ADMIN_UIDS` を追加     |
| `SDZ_LOG_FORMAT` | `.env.tpl` になし                    | 非秘匿のため `.env.tpl` に `SDZ_LOG_FORMAT=` を直接記載（空=compact） |

### `web/ui/.env.tpl` に不足しているフィールド

なし（`VITE_GOOGLE_MAPS_API_KEY` は既に `.env.tpl` に記載済み）

### `web/ui/.env` の `VITE_SDZ_ADMIN_UIDS` について

UI コードで参照されていないため、削除して問題なし。
