# ADR-001: SpotDiggz API を Go 製 Skate Spot Metadata API として再設計する

## Status

Accepted

## Context

[事実] SpotDiggz は過去にAPI、Web、mobile、deployment関連の成果物を同一repositoryで扱っていた。

[事実] 2026-05-30時点で、active repositoryはGo API、OpenAPI、PostgreSQL schema/migration、Browser UIを中心に整理する方針へ変更した。

[事実] 今回の設計入力では、SpotDiggzを地図アプリではなく、スケートスポットの位置情報と関連メタデータを保存・検索・返却するAPIとして扱うことが明示された。

[推測] API本体をGoへ寄せ、mobile clientとdeployment詳細をactive treeから外すことで、MVPの実装範囲と検証範囲を小さく保てる。

## Decision

SpotDiggz API本体はGoで実装する。APIの責務は Skate Spot Metadata API に限定する。

初期Go実装は `cmd/api` と `internal/...` に置き、単一バイナリを `go build -o bin/spotdiggz-api ./cmd/api` で生成する。

旧runtime、旧deployment証跡、mobile途中実装は active repository から外し、archive branchでのみ参照する。repository boundaryの詳細は [ADR-002](002-repository-boundary-api-browser-mobile.md) を優先する。

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
- legacy runtimeと旧deployment証跡はarchive branchに退避し、active CI/CDの対象から外す。
- storageはPostgreSQL前提へ寄せる。
- CI/CDはGo binary、vulnerability scan、container scanを主経路へ寄せる。
- 初期実装はin-memory storeでAPI境界を固定し、DB永続化は次フェーズで追加する。

## Migration Plan

1. Go API最小実装を追加し、OpenAPIとREADMEをGo前提へ更新する。
2. PostgreSQL schema、goose migration、sqlc queryを追加する。
3. in-memory storeをPostgreSQL storeへ差し替える。
4. Browser UIのAPI呼び出しを新契約へ合わせる。
5. legacy runtime、旧deployment証跡、mobile途中実装をarchive branchへ退避し、active repositoryとCIから外す。
6. iOS/Androidは別repositoryで新API契約を参照する。

## Open Questions

[未検証] dev / stg / prd のhosting境界。候補はBrowser UIをVercel、PostgreSQLをNeonへ寄せる構成。

[未検証] `private` / `unlisted` の認可モデル、owner identity、admin権限。

[未検証] PostgreSQLのホスティング先、PostGIS採用有無、backup/restore要件。

[未検証] Browser UIを `web/ui` のまま育てるか、`apps/browser` などへ再配置するか。
