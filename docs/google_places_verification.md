# Google Places API 統合 — 動作確認手順

PR #270 (`feat(web): Google Places API統合 + SdzSpotモデル全層整合`) の手動動作確認手順。

## 前提条件

- 1Password CLI (`op`) インストール済み・認証済み
- `gcloud` CLI 認証済み（`gcloud auth login`）
- Node.js 22 / npm インストール済み
- Rust toolchain インストール済み

---

## 1. 環境準備

```bash
# リポジトリルートに移動
cd /Users/kohei/workspace/uechikohei/spot-diggz

# シークレット取得（1Password + gcloud token）
make secrets

# admin UID の設定（web/api/.env に追記）
# ※ 自分の Firebase UID を設定する
echo 'SDZ_ADMIN_UIDS=zjJuiae1ymc6kqjU88yFsJvAuxG2' >> web/api/.env
```

## 2. サーバー起動

ターミナル1（API サーバー）:

```bash
cd /Users/kohei/workspace/uechikohei/spot-diggz/web/api
set -a && source ./.env && set +a
cargo run
# → localhost:8080 で起動
```

ターミナル2（UI 開発サーバー）:

```bash
cd /Users/kohei/workspace/uechikohei/spot-diggz/web/ui
npm install
npm run dev
# → localhost:3000 で起動
```

## 3. 管理画面 Google Places Autocomplete 動作確認

### 3-1. ログイン

1. ブラウザで `http://localhost:3000` を開く
2. Firebase Auth でログイン

### 3-2. スポット新規作成（Google Places 自動入力）

1. `/admin` ページへ移動
2. 「新規作成」ボタンをクリック
3. **場所検索欄** に「駒沢公園 スケートパーク」と入力
4. Autocomplete のドロップダウンから候補を選択

**確認項目:**

| #   | 確認内容                                      | 期待値                                                  |
| --- | --------------------------------------------- | ------------------------------------------------------- |
| A   | 地図上にマーカーが移動する                    | 選択した場所にピンが立つ                                |
| B   | 緯度・経度が自動入力される                    | 数値が入る（例: 35.6xxx, 139.6xxx）                     |
| C   | スポット名が自動入力される（空の場合のみ）    | Google Places の施設名が入る                            |
| D   | Google Places 情報セクションが表示される      | グレー背景のボックスが出現                              |
| E   | 住所が表示される                              | 日本語の住所が表示される                                |
| F   | Google Maps リンクが表示される                | 「開く」リンクが表示され、クリックで Google Maps が開く |
| G   | 電話番号が表示される（ある場合）              | 電話番号が表示される                                    |
| H   | 評価・レビュー数が表示される（ある場合）      | 「4.2 (120 件)」のような表示                            |
| I   | 施設タイプがタグ表示される                    | `park`, `point_of_interest` 等のバッジ                  |
| J   | Place ID が小さく表示される                   | `ChIJ...` 形式の文字列                                  |
| K   | 公式サイトURLが自動入力される（空の場合のみ） | Google Places の website 情報が入る                     |
| L   | 営業時間メモが自動入力される（ある場合）      | 曜日ごとの営業時間テキスト                              |

### 3-3. スポット保存

1. 上記の自動入力を確認後、必要に応じてタグ等を追加
2. 「作成」ボタンをクリック
3. 「スポットを作成しました」のメッセージが表示されることを確認

## 4. Firestore コンソールでの新フィールド確認

### 4-1. Firestore コンソールを開く

- URL: `https://console.firebase.google.com/project/sdz-dev/firestore/databases/-default-/data/~2Fspots`
- または: GCP Console → Firestore → `spots` コレクション

### 4-2. 作成したスポットのドキュメントを開く

Step 3 で作成したスポットのドキュメントを選択し、以下のフィールドを確認:

| #   | フィールド名        | 型        | 確認内容                                                                   |
| --- | ------------------- | --------- | -------------------------------------------------------------------------- |
| 1   | `name`              | string    | スポット名が保存されている                                                 |
| 2   | `approvalStatus`    | string    | `approved`（管理者作成のため自動承認）                                     |
| 3   | `parkAttributes`    | map       | `officialUrl`, `businessHours`（map 内にさらに `note`, `scheduleType` 等） |
| 4   | `googlePlaceId`     | string    | `ChIJ...` 形式の Place ID                                                  |
| 5   | `googleMapsUrl`     | string    | `https://maps.google.com/?cid=...` 形式の URL                              |
| 6   | `address`           | string    | 日本語の住所文字列                                                         |
| 7   | `phoneNumber`       | string    | 電話番号（存在する場合）                                                   |
| 8   | `googleRating`      | number    | 1.0〜5.0 の数値                                                            |
| 9   | `googleRatingCount` | number    | 整数値（レビュー数）                                                       |
| 10  | `googleTypes`       | array     | `["park", "point_of_interest", ...]` の文字列配列                          |
| 11  | `instagramTag`      | string    | 入力した場合のみ                                                           |
| 12  | `location`          | map       | `lat`, `lng` の数値が正しい                                                |
| 13  | `createdAt`         | timestamp | タイムスタンプが入っている                                                 |
| 14  | `updatedAt`         | timestamp | タイムスタンプが入っている                                                 |
| 15  | `userId`            | string    | ログインユーザーの Firebase UID                                            |

### 4-3. 旧フィールドが書き込まれていないことを確認

以下の **旧スキーマフィールドが存在しない** ことを確認:

- `trustLevel` → 代わりに `approvalStatus` が使われている
- `trustSources` → 存在しない
- `spotType` → 存在しない（`parkAttributes` / `streetAttributes` で暗黙判別）
- `instagramUrl` → 代わりに `instagramTag` が使われている
- フラットな `officialUrl` → `parkAttributes.officialUrl` に移動
- フラットな `businessHours`（文字列） → `parkAttributes.businessHours`（map）に移動
- フラットな `sections`（文字列配列） → `streetAttributes.sections`（map配列）に移動

## 5. 旧スキーマ既存データの後方互換読み取り確認

### 目的

Firestore に旧スキーマ（`trustLevel`, `spotType`, フラットな `instagramUrl` 等）で保存されている既存データが、新しいコードで正しく読み取れることを確認する。

### 5-1. 既存データの確認

Firestore コンソールで `spots` コレクション内に **旧スキーマのドキュメント** が存在するか確認する。
旧スキーマのドキュメントは以下の特徴を持つ:

- `trustLevel` フィールドがある（`verified` or `unverified`）
- `spotType` フィールドがある（`park` or `street`）
- `instagramUrl` フィールドがある（フラットな文字列）
- `officialUrl` がトップレベルにある（`parkAttributes` 内ではない）
- `businessHours` がトップレベルの文字列

### 5-2. API経由での読み取り確認

```bash
# API サーバーが起動している状態で実行

# 旧スキーマのスポットIDを指定（Firestore コンソールで確認したもの）
SPOT_ID="<旧スキーマのスポットID>"

# スポット詳細を取得
curl -s http://localhost:8080/sdz/spots/${SPOT_ID} | jq .
```

**確認項目:**

| #   | 確認内容                                    | 期待値                                                              |
| --- | ------------------------------------------- | ------------------------------------------------------------------- |
| 1   | レスポンスが 200 で返る                     | エラーにならない                                                    |
| 2   | `approvalStatus`                            | 旧 `trustLevel: "verified"` → `"approved"` に変換されている         |
| 3   | `approvalStatus`                            | 旧 `trustLevel: "unverified"` → `"pending"` に変換されている        |
| 4   | `parkAttributes.officialUrl`                | 旧トップレベル `officialUrl` から読み取られている                   |
| 5   | `parkAttributes.businessHours.note`         | 旧トップレベル `businessHours`（文字列）が `note` に格納されている  |
| 6   | `parkAttributes.businessHours.scheduleType` | `"manual"` になっている                                             |
| 7   | `instagramTag`                              | 旧 `instagramUrl` の値が入っている                                  |
| 8   | `streetAttributes.sections[].type`          | 旧 `sections`（文字列配列）の各要素が `type` フィールドに入っている |
| 9   | Google Places 系フィールドが null/省略      | `googlePlaceId` 等が省略されている（旧データには存在しない）        |

### 5-3. 一覧取得での確認

```bash
# スポット一覧を取得（旧・新データが混在しても正常に返る）
curl -s http://localhost:8080/sdz/spots | jq '.[0:3]'
```

**確認項目:**

| #   | 確認内容                            | 期待値                            |
| --- | ----------------------------------- | --------------------------------- |
| 1   | レスポンスが 200 で JSON 配列が返る | エラーにならない                  |
| 2   | 旧データと新データが混在して返る    | 両方とも同じ `SdzSpot` 構造で返る |

### 5-4. 旧データがない場合のテスト方法

既存の旧スキーマデータが Firestore にない場合は、Firestore コンソールから手動でテストドキュメントを作成する:

1. Firestore コンソール → `spots` コレクション → 「ドキュメントを追加」
2. ドキュメント ID: `test-legacy-001`
3. 以下のフィールドを追加:

| フィールド      | 型        | 値                           |
| --------------- | --------- | ---------------------------- |
| `name`          | string    | `レガシーテスト公園`         |
| `trustLevel`    | string    | `verified`                   |
| `spotType`      | string    | `park`                       |
| `instagramUrl`  | string    | `https://instagram.com/test` |
| `officialUrl`   | string    | `https://example.com`        |
| `businessHours` | string    | `平日 9:00-17:00`            |
| `tags`          | array     | `["パーク", "テスト"]`       |
| `images`        | array     | `[]`                         |
| `userId`        | string    | `test-user`                  |
| `createdAt`     | timestamp | 現在時刻                     |
| `updatedAt`     | timestamp | 現在時刻                     |

4. API から取得して変換結果を確認:

```bash
curl -s http://localhost:8080/sdz/spots/test-legacy-001 | jq .
```

5. 期待されるレスポンス:

```json
{
  "spotId": "test-legacy-001",
  "name": "レガシーテスト公園",
  "approvalStatus": "approved",
  "parkAttributes": {
    "officialUrl": "https://example.com",
    "businessHours": {
      "scheduleType": "manual",
      "is24Hours": false,
      "sameAsWeekday": false,
      "note": "平日 9:00-17:00"
    }
  },
  "instagramTag": "https://instagram.com/test",
  "tags": ["パーク", "テスト"],
  "images": [],
  "userId": "test-user",
  ...
}
```

6. 確認後、テストドキュメントを削除:
   - Firestore コンソールで `test-legacy-001` を選択 → 「ドキュメントを削除」

---

## 確認完了チェックリスト

- [ ] Step 3: Google Places Autocomplete でスポット名・住所・電話番号等が自動入力される
- [ ] Step 3: スポットが正常に作成できる
- [ ] Step 4: Firestore に新フィールド（googlePlaceId 等）が保存されている
- [ ] Step 4: 旧フィールド（trustLevel 等）が書き込まれていない
- [ ] Step 5: 旧スキーマの既存データが API 経由で正しく読み取れる（フォールバック変換）
- [ ] Step 5: 一覧取得で旧・新データが混在してもエラーにならない
