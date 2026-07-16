# spot-diggz API Architecture

## Change History

- 2026-05-30: ADR-001 accepted. API本体をGo製 Skate Spot Metadata APIとして再設計した。
- 2026-05-30: ADR-002 accepted. Repository boundaryをAPI + Browser UIとMobile Clientsに分離した。
- 2026-05-30: active treeからlegacy runtime、deployment証跡、mobile sourceを削除し、archive branchで参照する方針へ整理した。

## Current Scope

[事実] SpotDiggz APIは、スケートスポットの緯度経度と関連メタデータを保存・検索・返却する。

[事実] APIは地図描画、地図タイル配信、ルート検索、ナビゲーション、geocoding、reverse geocodingを担当しない。

[推測] 地図表示はBrowser UIや将来のclientが MapLibre、Leaflet、MapKit、Google Maps などを必要に応じて選ぶ方が、APIの責務を小さく保てる。

## Components

```text
cmd/api
  API entrypoint

internal/httpapi
  chi router, request parsing, response encoding, JSON / GeoJSON endpoints

internal/spot
  domain model, validation, bbox / tags / visibility filtering, repository interface

internal/postgres
  pgx pool integration and PostgreSQL repository implementation

db
  schema and goose migrations

docs/openapi.yaml
  API contract

web/ui
  Browser UI
```

## API Endpoints

- `GET /sdz/health`
- `POST /sdz/spots`
- `GET /sdz/spots`
- `GET /sdz/spots/{spotId}`
- `PATCH /sdz/spots/{spotId}`
- `DELETE /sdz/spots/{spotId}`
- `GET /sdz/spots.geojson`

## Storage

[事実] `DATABASE_URL` が未設定の場合、APIはin-memory storeで動作する。

[事実] `DATABASE_URL` が設定されている場合、APIはPostgreSQL storeを使う。

[事実] local PostgreSQLは `compose.yaml` の `postgres` serviceで起動できる。

## Delivery

- Source checks: `go fmt`, `go vet`, `go test`
- Binary build: `go build -o bin/spotdiggz-api ./cmd/api`
- Vulnerability checks: `govulncheck`, Trivy fs scan
- Container build: `Dockerfile`

## References

- [ADR-001](adr/001-go-skate-spot-metadata-api.md)
- [ADR-002](adr/002-repository-boundary-api-browser-mobile.md)
- [OpenAPI](openapi.yaml)
- [Go API implementation plan](go-api-implementation-plan.md)
