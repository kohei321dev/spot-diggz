# SpotDiggz Go API 実装計画

## Evidence Map

[事実] 既存APIは `web/api/` のRust実装で、health、spots、upload-url、users、mylist、admin系handlerを持つ。

[事実] 既存OpenAPIはRust/Axumの内部契約として書かれている。

[事実] 既存CIはRust、React、Terraform、Docker、Trivyを実行する。

[推測] 既存APIは投稿、画像URL、mylist、Firebase Auth、Firestoreなどの責務を含んでおり、今回の「Skate Spot Metadata API」より広い。

## Target Architecture

```text
cmd/api
  Go API entrypoint

internal/httpapi
  HTTP routing, request/response, JSON/GeoJSON encoding

internal/spot
  spot model, validation, bbox/tag/visibility filter, repository interface

internal/postgres
  pgx pool, PostgreSQL-backed repository implementation

db/migrations
  goose migrations
```

## Phase 0: Minimal API

[事実] `DATABASE_URL` が未設定の場合はin-memory storeを使う。

- `GET /sdz/health`
- `POST /sdz/spots`
- `GET /sdz/spots`
- `GET /sdz/spots/{spotId}`
- `PATCH /sdz/spots/{spotId}`
- `DELETE /sdz/spots/{spotId}`
- `GET /sdz/spots.geojson`
- bbox query: `bbox=minLng,minLat,maxLng,maxLat`
- tag query: `tags=ledge,street`
- visibility query: default `public`

## Phase 1: PostgreSQL

[事実] ローカルPostgreSQLは `compose.yaml` の `postgres` serviceで起動する。

[事実] 初期schemaは `db/schema.sql`、goose migrationは `db/migrations/00001_create_sdz_spots.sql` に置く。

[事実] APIは `DATABASE_URL` が設定されている場合のみ `internal/postgres` の `pgxpool.Pool` を使う。

- `created_at`, `updated_at`, `deleted_at`
- bbox検索用index
- tag検索用GIN index
- PostGIS採用判断
- sqlc導入判断

## Phase 2: API Contract Hardening

- OpenAPI lint
- generated client検証
- pagination
- stable error code
- auth/owner model
- rate limit

## Phase 3: Delivery

- `go build -o bin/spotdiggz-api ./cmd/api`
- `govulncheck ./...`
- `govulncheck -mode=binary ./bin/spotdiggz-api`
- `trivy fs .`
- `trivy image`
- SBOM生成

## Risks

[推測] in-memory storeはAPI境界確認用であり、アプリを再起動するとデータは消える。

[未検証] iOS/Web clientが期待するfieldと新OpenAPIの差分。

[未検証] PostgreSQL運用先と費用。
