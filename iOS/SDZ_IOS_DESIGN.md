# spot-diggz iOS 設計書

## 0. 概要
- 目的: 既存Rust APIに接続するiOSアプリの設計を確定し、Xcode実装を継続可能にする。
- 対象: iOSネイティブアプリ（Swift/SwiftUI + Firebase Auth + MapKit + SwiftData + CloudKit）。
- スコープ: スポット一覧/詳細/Tier 2スポット登録/認証/プロフィール/お気に入り/ルート計画。

### データアーキテクチャ（2026-02-28 再設計）

- Tier 1（マスターデータ）: Rust API 経由で Firestore から読み取り。公開スポット。
- Tier 2（ユーザー個人データ）: SwiftData + CloudKit でローカル管理。サーバーには保存しない。
- マイリスト: SwiftData + CloudKit でローカル管理（Firestore API 経由から移行）。
- 詳細: `docs/designs/tier2-spot-data-architecture.md` を参照。

## 1. 参照
- `docs/spot_diggz_overview.md` (iOS想定/マルチチャネル方針)
- `docs/api_architecture.md` (API方針/認証方針)
- `docs/openapi.yaml` (API契約)
- `web/ui/src/App.tsx` (WebのUI/UXとAPI呼び出し)
- `web/ui/src/contexts/AuthProvider.tsx` (Firebase Authフロー/ユーザードキュメント作成)
- `web/ui/src/firebase.ts` (Firebase設定)
- `web/ui/src/types/spot.ts` (Spot型定義)
- `web/ui/.env` / `web/ui/.env.local` (API Base/Firebase設定値)
- `web/api/src/presentation/router.rs` (ルーティング)
- `web/api/src/presentation/middleware/auth.rs` (JWT検証)
- `web/api/src/presentation/middleware/client.rs` (X-SDZ-Client必須)
- `web/api/src/presentation/handlers/spot_handler.rs` (投稿/一覧/詳細)
- `web/api/src/presentation/handlers/user_handler.rs` (プロフィール取得)
- `web/api/src/presentation/error.rs` (エラーレスポンス)
- `web/api/src/application/use_cases/create_spot_use_case.rs` (入力検証)
- `web/api/src/application/use_cases/generate_upload_url_use_case.rs` (画像アップロード仕様)
- `web/api/src/domain/models.rs` (サーバー側モデル)
- `web/api/src/infrastructure/storage_signed_url_repository.rs` (Signed URLはPUT)
- `web/scripts/firestore_crud_smoke.sh` (ヘッダ/投稿リクエスト例)

## 2. 全体アーキテクチャとiOSの役割
### 2.1 位置付け
```
[iOS SwiftUI] -> [SdzApiClient] -> [Rust API] -> [Firestore/Cloud Storage]
                      |
                      +-> [Firebase Auth] (IDトークン発行)
```

### 2.2 役割
- iOSは「投稿/ナビ/現地利用」を強化したクライアント。
- Webは「閲覧中心」、iOSは「投稿・位置情報連携中心」を想定。
- APIは共通で、iOSはモバイル専用エンドポイントを利用可能。

## 3. iOSアプリの機能要件
### 3.1 コア機能

📋 Tier 1 マスターデータ（API経由・読み取り専用）:
- スポット一覧（GET /sdz/spots）
- スポット詳細（GET /sdz/spots/{spot_id}）
- スポット詳細拡張（営業時間/営業日/公式サイト/アクセス）※parkのみ表示
- プロフィール表示（GET /sdz/users/me）

📋 Tier 2 ユーザー個人データ（SwiftData + CloudKit・ローカル管理）:
- Tier 2 スポット登録（地図タップ → 位置選択 → 詳細入力）
- Tier 2 スポット編集・削除
- Tier 2 画像添付（ローカル保存、@Attribute(.externalStorage)）
- マイリスト管理（SwiftData + CloudKit）
- マイリスト詳細（お気に入りの一覧）

📋 休眠中（将来の Tier 2 → Tier 1 昇格で再利用の可能性）:
- ~~スポット投稿（POST /sdz/spots）~~ → Tier 2 はローカル保存
- ~~スポット編集（PATCH /sdz/spots/{spot_id}）~~ → Tier 2 はローカル保存
- ~~画像アップロード（POST /sdz/spots/upload-url）~~ → Tier 2 画像はローカル保存
- ~~承認リクエスト（スポットの公開申請）~~ → 将来の拡張ポイント

### 3.2 追加価値（iOS特化）
- MapKit表示、現在地連動、スポットピン表示
- 位置検索/逆ジオコーディングで投稿支援
- Apple Maps/Google Mapsへの外部ナビ連携
- ルート計画（詳細から追加/ルートタブで作成/移動手段の選択）

### 3.3 データティアと公開フロー

📋 現行設計（2026-02-28 再設計後）:
- Tier 1 マスターデータ: 管理者がキュレーションしたスポット。全ユーザーに公開。
- Tier 2 ユーザー個人データ: ユーザーが個人的にプロットしたスポット。作成者本人のみ閲覧可。
- Tier 2 → Tier 1 昇格機能は将来の拡張ポイント（現時点ではスコープ外）。

📋 旧設計（参考・現時点では未実装）:
- `approvalStatus` フィールドによる承認フローは、Tier 昇格機能の実装時に再検討する。
- 旧 `trustLevel` は使用しない。

### 3.4 スポット種別と表示
- `spotType=park` は施設詳細（営業時間/営業日/公式サイト/アクセス）を表示。
- `spotType=street` はストリート向け詳細（路面/障害物/難易度など）を任意入力で表示。
- 共通情報（名称/説明/位置/タグ/写真）は全種別で必須。
- 施設詳細/ストリート詳細は未入力なら非表示にする。

## 4. 画面設計とナビゲーション
### 4.1 画面一覧
- 起動画面
- 認証（ログイン/新規登録）
- ホーム（一覧 + 検索/タグ）
- スポット詳細
- マップ
- 投稿（フォーム + 画像 + 位置）
- 投稿編集（フォーム + 位置）
- プロフィール（自分の投稿/お気に入り）
- マイリスト詳細
- ルート作成（スポット選択/移動手段選択）
- ルート詳細（タイムライン/経路）
- 設定（アプリ情報/問い合わせ）

### 4.2 ナビゲーション構成（案）
- Tiqets寄せ（優先）
  - Spot: マップ中心の探索（検索バー/フィルタ/カード）
  - Favorite: お気に入り
  - Route: ルート一覧/作成/詳細
  - Setting: プロフィール/設定
  - 投稿はMap上のフローティングボタンから起動
- 現行Tab Bar（移行前の互換）
  - Home: 一覧/検索
  - Map: 地図とピン
  - MyList: お気に入り
  - Post: 投稿フロー
  - Profile: ユーザー/設定

### 4.3 主要画面の状態
- 一覧: loading / empty / error / results
- 詳細: loading / not found / error / results
- 投稿: draft / uploading / submitted / error
- 認証: unauthenticated / authenticated / error

### 4.4 Tiqets参考のUIパターン（方針）
- マップ上に検索バー＋フィルタチップをオーバーレイ。
- 画面下部はカードカルーセル（スポットカード/ルートカード）。
- カードには「ルートに追加」「お気に入り」など即時アクション。
- 設定/プロフィールはリスト形式（セクション分割）で整理。
- 空状態は主CTAボタン（例: 「ルートを検索」）を中央配置。

### 4.5 ルート計画フロー（MVP）
- 追加導線: スポット詳細/カードから「ルートに追加」。
- ルート作成: ルートタブで登録済みスポットを選択。
- 移動手段: 徒歩/電車/車の切替（`MKDirectionsTransportType`）。
- 順序調整: ドラッグで並び替え、所要時間は概算表示。
- 出発/到着: 現在地 or 任意地点を指定。

## 5. アプリケーションアーキテクチャ
### 5.1 レイヤ構成
- Presentation: SwiftUI + ViewModel (状態管理とUI)
- Domain: UseCase, Entity, Validation
- Data: Repository, DataSource (API/Firebase)
- Infra: URLSession, Firebase SDK, MapKit, CoreLocation

### 5.2 ディレクトリ構成（案）
```
iOS/
  SpotDiggz/
    App/
      SdzApp.swift
      SdzEnvironment.swift
    Presentation/
      Screens/
      Components/
      ViewModels/
    Domain/
      Entities/
      UseCases/
      Validators/
    Data/
      Repositories/
      DataSources/
    Infrastructure/
      API/
      Firebase/
      Location/
    Resources/
      Assets.xcassets
      Localizable.strings
```

### 5.3 命名規約
- Swiftの型/機能名は `Sdz` プレフィックスを推奨（例: `SdzSpot`, `SdzApiClient`）。
- JSONキーはOpenAPIに準拠（camelCase）。

## 6. データモデル設計（Swift）
### 6.1 APIモデル
```swift
struct SdzSpotLocation: Codable {
    let lat: Double
    let lng: Double
}

enum SdzSpotApprovalStatus: String, Codable {
    case pending
    case approved
    case rejected
}

enum SdzSpotType: String, Codable {
    case park
    case street
}

struct SdzSpot: Codable, Identifiable {
    let spotId: String
    let name: String
    let description: String?
    let location: SdzSpotLocation?
    let spotType: SdzSpotType
    let tags: [String]
    let images: [String]
    let address: String?
    let accessInfo: String?
    let phoneNumber: String?
    let businessHours: String?
    let businessDays: [String]?
    let officialSiteUrl: String?
    let surface: String?
    let obstacles: [String]?
    let difficulty: String?
    let approvalStatus: SdzSpotApprovalStatus?
    let userId: String
    let createdAt: Date
    let updatedAt: Date

    var id: String { spotId }
}

struct SdzUser: Codable {
    let userId: String
    let displayName: String
    let email: String?
}

struct SdzCreateSpotInput: Codable {
    let name: String
    let description: String?
    let location: SdzSpotLocation?
    let spotType: SdzSpotType
    let tags: [String]?
    let images: [String]?
    let address: String?
    let accessInfo: String?
    let phoneNumber: String?
    let businessHours: String?
    let businessDays: [String]?
    let officialSiteUrl: String?
    let surface: String?
    let obstacles: [String]?
    let difficulty: String?
}

struct SdzUpdateSpotInput: Codable {
    let name: String
    let description: String?
    let location: SdzSpotLocation?
    let spotType: SdzSpotType?
    let tags: [String]?
    let images: [String]?
    let approvalStatus: SdzSpotApprovalStatus?
    let address: String?
    let accessInfo: String?
    let phoneNumber: String?
    let businessHours: String?
    let businessDays: [String]?
    let officialSiteUrl: String?
    let surface: String?
    let obstacles: [String]?
    let difficulty: String?
}

struct SdzUploadUrlRequest: Codable {
    let contentType: String
}

struct SdzUploadUrlResponse: Codable {
    let uploadUrl: String
    let objectUrl: String
    let objectName: String
    let expiresAt: Date
}

struct SdzErrorResponse: Codable, Error {
    let code: Int
    let errorCode: String
    let message: String
}
```

### 6.2 日付デコード方針
- APIの日時はRFC3339（JST +09:00）で返るため、`ISO8601DateFormatter` を使用。
- 例: `JSONDecoder.dateDecodingStrategy = .iso8601`。

### 6.3 ルート計画のローカル状態（案）
```swift
struct SdzRoutePlan: Codable, Identifiable {
    let id: String
    let spotIds: [String]
    let start: SdzSpotLocation?
    let goal: SdzSpotLocation?
    let transportType: String
}
```

## 7. API設計（iOS向け）
### 7.1 共通ヘッダ
- `Authorization: Bearer <Firebase ID Token>`（認証必須エンドポイントのみ）
- `X-SDZ-Client: ios`（モバイル専用エンドポイント）

### 7.2 エンドポイント一覧
| Method | Path | Auth | Client Header | 用途 |
| --- | --- | --- | --- | --- |
| GET | `/sdz/health` | No | No | ヘルスチェック |
| GET | `/sdz/spots` | No | No | 一覧取得 |
| GET | `/sdz/spots/{spot_id}` | No | No | 詳細取得 |
| POST | `/sdz/spots` | Yes | Yes | スポット投稿 |
| PATCH | `/sdz/spots/{spot_id}` | Yes | Yes | スポット編集 |
| POST | `/sdz/spots/upload-url` | Yes | Yes | 画像アップロードURL取得 |
| GET | `/sdz/users/me` | Yes | No | プロフィール取得 |

### 7.3 バリデーション制約（サーバー側）
- `name` 必須
- `location.lat` は -90〜90
- `location.lng` は -180〜180
- `tags` は最大10件
- `images` は最大10件

## 8. 認証設計（Firebase Auth）
### 8.1 サインイン手段
- Email/Password
- Google（Firebase Auth + Google Sign-In）
- Apple（Firebase Auth + Sign in with Apple）

### 8.2 トークン取得
- `Auth.auth().currentUser?.getIDToken()` で毎リクエスト取得。
- 401時はトークンリフレッシュ→再試行を1回のみ許可。

### 8.3 Firestoreユーザードキュメント作成
- Webと同様、ログイン直後に `users/{uid}` を作成/更新。
- 必須フィールド: `email`（Firestore側で`displayName`が未設定だとAPIは`unknown`を返す）。
- 推奨: `displayName`, `createdAt`, `updatedAt` をセット。

## 9. 画像アップロード設計
### 9.1 フロー
1. `POST /sdz/spots/upload-url` に `contentType` を送信。
2. レスポンスの `uploadUrl` へ `PUT` で画像バイナリを送信。
3. `objectUrl` を `CreateSpotInput.images` に追加し、`POST /sdz/spots` 実行。

### 9.2 Content-Type対応
- `image/jpeg`, `image/jpg`, `image/png`, `image/webp`, `image/gif`, `image/heic`, `image/heif`

### 9.3 アップロード時の注意点
- `PUT` の `Content-Type` を必ず一致させる。
- 失敗時はその画像だけ再試行可能にする。

## 10. 位置情報/MapKit設計
### 10.1 位置情報取得
- `CLLocationManager` を使用（`when-in-use`）。
- 位置情報の許可がない場合はMapのみ閲覧可。

### 10.2 Map表示
- `Map` (SwiftUI) / `MKMapView` を利用。
- スポットはピン表示、タップで詳細へ遷移。

### 10.3 投稿時の位置入力
- 現在地 or 地図タップ/検索による位置指定。
- `CLGeocoder` で住所取得し、UIに補助表示。

### 10.4 ルート計画（MapKit）
- `MKDirections` で経路を取得し、`Map`にポリライン表示。
- 徒歩/電車/車の移動手段を切り替える。
- MVPは「現在地→スポット→ゴール」の単純経路。
- 将来的に最適化（所要時間最短/人気順）を検討。

## 11. ローカル保持/キャッシュ

📋 SwiftData + CloudKit（主データストア）:
- Tier 2 スポット: SwiftData で永続化。iCloud オン時は CloudKit で同期。
- マイリスト（お気に入り）: SwiftData で永続化。iCloud オン時は CloudKit で同期。
- ルート計画: SwiftData で永続化（将来実装）。

📋 iCloud オフ時の注意:
- アプリ削除で Tier 2 データ・マイリストが失われることをユーザーに通知。
- 「iCloudを有効にするとバックアップできます」の表示。

📋 キャッシュ:
- Tier 1 一覧キャッシュ: メモリキャッシュ（API レスポンス）。
- 画像キャッシュ: `URLCache` / `AsyncImage` を利用（Tier 1 画像）。
- Tier 2 画像: `@Attribute(.externalStorage)` でローカル保存。

## 12. エラーハンドリング
- APIエラーは `SdzErrorResponse` に変換。
- 401: 再ログイン誘導
- 403: 「モバイル専用」メッセージ
- 404: 「存在しないスポット」表示
- 500: 汎用エラー

## 13. 環境設定（Xcode）
### 13.1 Build Config
- Debug / Release に加え `Dev/Stg/Prod` を推奨。
- `SdzEnvironment` でAPI Baseを切替。

### 13.2 主要設定値
- API Base: `http://localhost:8080` (ローカル), `https://sdz-dev-api-...run.app` (dev)
- Firebase: `GoogleService-Info.plist` を環境別に切替。

### 13.3 Info.plist 必須項目
- `NSLocationWhenInUseUsageDescription`
- `NSCameraUsageDescription`
- `NSPhotoLibraryUsageDescription`
- ローカルHTTP用のATS例外（必要時のみ）

## 14. テスト方針
- APIクライアント: `URLProtocol` モックでユニットテスト。
- ViewModel: 状態遷移のテスト。
- UIテスト: ログイン/投稿/一覧表示のスモーク。

## 15. 未決定/今後の検討
- ~~お気に入りのサーバー同期有無~~ → SwiftData + CloudKit に決定（2026-02-28）
- ~~投稿時のオフラインキュー~~ → Tier 2 はローカル保存のためオフライン対応は不要
- 画像の圧縮/リサイズ方針（Tier 2 ローカル画像の最適化）
- Tier 2 → Tier 1 昇格フロー（将来の拡張ポイント）
- アカウント削除時の SwiftData + CloudKit データクリア処理
