# spot-diggz API アーキテクチャ計画

## 📋 アーキテクチャ変更履歴

- 2026-02-28: データアーキテクチャ再設計。APIの役割を「読み取り専用の Tier 1 マスターデータ配信」に縮小。
  - Firestore: 読み取り専用（マスターデータ配信専用）
  - ユーザーデータ（Tier 2）: iOS SwiftData + CloudKit に移行（API不使用）
  - マイリスト: iOS SwiftData + CloudKit に移行（API不使用）
  - 書き込みエンドポイント（POST/PATCH spots, mylists）: 休眠化
  - 詳細: `docs/designs/tier2-spot-data-architecture.md`

## OpenAPI 定義
- `docs/openapi.yaml`

## STAR: API開発の目的と狙い
### Situation
- Rustスクラッチ実装でCloud Run上にデプロイするマイクロサービスを構築する計画がある。
- 既存Rails版のAPIは機能が肥大化しメンテナンスが困難だった。
### Task
- 高可用性かつセキュアなREST APIを設計し、Tier 1 マスターデータの読み取り配信基盤を整える。
- 認証・プロファイル取得の共通エンドポイントを提供する。
### Action
- 非同期ランタイム（Tokio）とAxumルータを採用したレイヤードアーキテクチャを定義する。
- 認証／認可、スポット読み取り、ユーザー管理の各コンポーネントを明確に分離し、Firestoreと疎結合に連携する。
### Result
- iOSアプリやWebフロントが同一APIを利用でき、Tier 1 マスターデータを低レイテンシで配信できる基盤が整う。

## STAR: Rust APIモジュール構成計画
### Situation
- モノレポ内でRust APIを`api`配下に配置するが、ディレクトリ構成やモジュール命名が未定。
- 認証やデータアクセスなど横断的関心事を分離しないと複雑化が懸念される。
### Task
- クリーンなレイヤー分割（presentation / application / domain / infrastructure）をRustらしく実現する。
- sdzプレフィックス命名規約に従い、一貫性のあるモジュール名・構造体名を定義する。
### Action
- ルータ層: `sdz_router.rs`でエンドポイントを定義し、モジュールごとにハンドラを分割。
- アプリ層: `sdz_usecases`ディレクトリにユースケース単位のサービス（例: `sdz_spot_service.rs`）を配置。
- ドメイン層: エンティティや値オブジェクトを`sdz_models.rs`等にまとめ、バリューの検証を担う。
- インフラ層: Firestoreクライアント、Cloud Storageアップローダ、地図API連携クライアントなどを`infrastructure/`配下に実装。
### Result
- モジュールの責務が明確になり、テスト・Mockの分離が容易になる。CI/CDでのコンパイル時間や可観測性も改善が見込まれる。

## 5W1H: Rust APIプロジェクト初期構成
- **Who**: バックエンド担当（Rust習熟を目指す開発者）が中心、iOS/フロントチームもAPI契約を参照。
- **What**: `api`配下に以下の主要構成を作成。`Cargo.toml`でTokio・Axum・serdeなどを管理。
  - `src/main.rs`（エントリポイント、PORT解決とAxum Router組み立て・serve）
  - `presentation/sdz_router.rs`（ルーティング定義）
  - `presentation/handlers/`（`sdz_auth_handler.rs`, `sdz_spot_handler.rs`, `sdz_user_handler.rs`）
  - `application/sdz_usecases/`（ユースケースロジック）
  - `domain/`（`sdz_models.rs`, `sdz_value_objects.rs`）
  - `infrastructure/`（`sdz_firestore_repository.rs`, `sdz_storage_client.rs`, `sdz_mapkit_gateway.rs` など）
- **When**: Phase 1の基盤構築でディレクトリと雛形を整備し、Phase 2で各ユースケースを実装。
- **Where**: Gitモノレポ内の`api`ディレクトリ、およびCloud Runデプロイパイプライン（Cloud Build）。
- **Why**: レイヤー分離により認証、スポット管理、外部API連携を疎結合化し、テスト容易性とセキュリティ制御を高めるため。
- **How**: Configは`sdz_config`モジュールで環境変数（Secret Manager連携）を読み込み、依存注入は構造体パターンで実施。FireStoreアクセスは`google-cloud-firestore`互換クライアント、MapKit/Google Maps連携はREST/gRPCクライアントを用意する。

## ランタイム環境変数（API）
最新の一覧は `README.md` の「環境変数（API）」と `docs/DEVELOPMENT_SETUP.md` を参照。

## Firestore 命名・ID方針

📋 Firestore は Tier 1 マスターデータの読み取り配信専用ストアとして使用。
ユーザーデータ（Tier 2 スポット、マイリスト）は Firestore に保存しない。
マスターデータの投入は BigQuery → Cloud Functions → Firestore のパイプラインで行う。

- データベース: `(default)`（環境はプロジェクト分離で管理: sdz-dev/stg/prod）
- コレクション: `users`, `spots`（プレフィックス不要）
- ドキュメントID:
  - users … Firebase UID
  - spots … サーバー生成UUID（プレフィックスなし）
- タイムスタンプ: `createdAt` / `updatedAt` をJST(UTC+9)で付与
- 位置情報: `location { lat, lng }`（必要なら将来 geohash 追加）

## 認証実装メモ（Web/iOS/バックエンド）
- Auth基盤: Firebase Authで統一し、JWT検証はバックエンドで実施（既存のJWT検証ロジックを流用）。
- ログイン手段（最小実装方針）
  - Email/Password
  - Google認証
  - Apple認証（Firebase AuthのAppleプロバイダを利用。Apple側のサービスID/Key ID/Team ID/リダイレクトURL設定が必要）
- Web側: AuthContextでIDトークンを保持し、`Authorization: Bearer <idToken>`でAPIを呼ぶ。Appleログインは`OAuthProvider('apple.com')` + `signInWithPopup/Redirect`を追加する。
- iOS側: Apple MapKitで検索/ジオコーディング→取得した座標をFirestoreへ送信（Google Maps API不要）。AuthはFirebaseでサインイン、IDトークンをバックエンドへ付与。
- 機能出し分け（Web最小化）：閲覧/マイページ/自分の投稿更新のみ。新規投稿はiOS優先。未認証は一覧参照のみ、認証済は自分の投稿にだけ編集ボタンを表示。

## 投稿制限
- 画像付きスポットは1ユーザーあたり最大10件（API側でバリデーション）。
- 1スポットあたりの画像は最大3枚。

## 5W1H: 投稿APIと位置情報連携のトラブル観点

⚠️ 2026-02-28 アーキテクチャ変更により、以下の投稿API設計は【休眠中】です。
ユーザーによるスポット投稿は iOS SwiftData（Tier 2 ローカル保存）に移行しました。
将来の Tier 2 → Tier 1 昇格機能で、投稿APIを再利用する可能性があります。

<details>
<summary>旧設計（参考）</summary>

- **Who**: 投稿するのは認証済み会員ユーザー。iOSアプリからの投稿が主だがWeb管理画面も想定。
- **What**: `POST /v1/sdz-spots`で座標・画像・タグを受け取り、Firestoreに保存しCloud Storageへ画像を転送。MapKit経由の経路提案情報はメタデータとして保持。
- **When**: 投稿時に端末位置情報と撮影タイミングを同時取得。エラーは即時フィードバックし、再試行ポリシーを定める。
- **Where**: クライアントはiOSアプリ（オフライン対応あり）とブラウザ。サーバー側はCloud Run内のRust API、データはFirestore/Storage。
- **Why**: コミュニティ主導でスポット情報を充実させ、ナビ機能と連携した価値を提供するため。
- **How**: JWT（Firebase Auth予定）で認証。リクエスト検証→ユースケース→リポジトリ→Firestore書込み→イベント発行（Pub/Subでレコメンド更新）。失敗時はIdempotency-Keyを利用し重複投稿を防ぐ。

</details>
