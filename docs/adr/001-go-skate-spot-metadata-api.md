# ADR-001: SpotDiggz API を Go 製 Skate Spot Metadata API として再設計する

## Status

Accepted

## Context

[事実] 既存repoは `web/api/` に Rust/Axum API、`docs/openapi.yaml` にRust前提のAPI契約、`.github/workflows/ci.yml` にRust CIを持つ。

[事実] 既存docsには、2026-02-28時点でAPIの責務をTier 1マスターデータ配信へ縮小した履歴がある。

[事実] 今回の設計入力では、SpotDiggzを地図アプリではなく、スケートスポットの位置情報と関連メタデータを保存・検索・返却するAPIとして扱うことが明示された。

[推測] Rust実装を即削除すると、既存iOS/Web/Firestore/Cloud Run連携の参照点を失い、移行差分が大きくなる。

## Decision

SpotDiggz API本体はGoで実装する。APIの責務は Skate Spot Metadata API に限定する。

初期Go実装は `cmd/api` と `internal/...` に置き、単一バイナリを `go build -o bin/spotdiggz-api ./cmd/api` で生成する。

既存Rust実装は、Go APIの設計・移行が安定するまで参照点として扱う。

2026-05-30追補: active repository から Rust/GCP/mobile legacy を外し、archive branchに退避する方針へ更新した。repository boundaryの詳細は [ADR-002](002-repository-boundary-api-browser-mobile.md) を優先する。

## In Scope

- スケートボードスポットの登録、一覧取得、詳細取得、更新、論理削除
- 緯度経度保存
- bbox検索
- tag filter
- `public` / `private` / `unlisted` のvisibility属性
- JSON response
- GeoJSON response
- OpenAPI管理
- `go fmt` / `go vet` / `go test` / `go build` / `govulncheck`
- 将来のPostgreSQL + pgx + sqlc + goose移行

## Out of Scope

- 地図描画
- 地図タイル配信
- Google Maps APIの完全代替
- ルート検索
- ナビゲーション
- geocoding / reverse geocoding
- SNS機能
- 写真投稿
- 複雑な権限管理
- 初期段階からのEKS本番運用
- 過剰なClean Architecture / DDD / microservice分割

## Rationale

[事実] Goは標準toolchainでformat、vet、test、buildを提供し、単一バイナリ生成が容易。

[推測] 個人利用MVPから公開運用へ進めるAPIとしては、Rustスクラッチ実装よりGo + chi + pgx/sqlc/gooseの方が学習対象と保守対象のバランスが取りやすい。

[推測] APIを独自スポットメタデータ基盤に限定すると、Google Maps / MapKit / MapLibre / Leaflet はクライアント責務として使い分けられる。

## Consequences

- Go APIを新しい本体として扱う。
- Rust APIはlegacy参照としてarchive branchに退避し、active CI/CDの対象から外す。
- Firestore前提はPostgreSQL前提へ移行する。
- CI/CDはGo binary、vulnerability scan、container scanを主経路へ寄せる。
- 初期実装はin-memory storeでAPI境界を固定し、DB永続化は次フェーズで追加する。

## Migration Plan

1. Go API最小実装を追加し、OpenAPIとREADMEをGo前提へ更新する。
2. PostgreSQL schema、goose migration、sqlc queryを追加する。
3. in-memory storeをPostgreSQL storeへ差し替える。
4. Browser UIのAPI呼び出しを新契約へ合わせる。
5. Rust/GCP/mobile legacyをarchive branchへ退避し、active repositoryとCIから外す。
6. iOS/Androidは別repositoryで新API契約を参照する。

## Open Questions

[未検証] 本番runtimeをGCP Cloud Run継続にするか、AWS ECS/Lambda/EKS学習経路へ寄せるか。

[未検証] `private` / `unlisted` の認可モデル、owner identity、admin権限。

[未検証] PostgreSQLのホスティング先、PostGIS採用有無、backup/restore要件。

[未検証] Browser UIを `web/ui` のまま育てるか、`apps/browser` などへ再配置するか。
