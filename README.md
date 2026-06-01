# spot-diggz

SpotDiggz is a Go-based Skate Spot Metadata API with a browser UI.

## Repository Scope

[事実] This repository is the active home for:

- Go API implementation
- OpenAPI contract
- PostgreSQL schema and migrations
- Browser UI
- API / Browser CI

[事実] The following are not active source trees in this repository:

- Rust API implementation
- GCP deployment / Terraform configuration
- iOS app implementation
- Android app implementation

[事実] Legacy code is kept on archive branches instead of the active tree.

- Rails archive: `archive/rails-original`
- Rust / GCP / mobile archive: `archive/rust-gcp-legacy-20260530`

## Tech Stack

| Layer | Technology |
| --- | --- |
| API | Go, chi, pgx |
| Database | PostgreSQL |
| Browser UI | React, TypeScript, Vite |
| API Spec | OpenAPI |
| Local runtime | Docker Compose for PostgreSQL |
| CI | GitHub Actions |

## Project Structure

```text
spot-diggz/
  cmd/api/                 Go API entrypoint
  internal/httpapi/        HTTP routing and JSON / GeoJSON responses
  internal/postgres/       PostgreSQL-backed repository
  internal/spot/           spot domain model, validation, repository interface
  db/                      PostgreSQL schema and goose migrations
  docs/                    ADR, OpenAPI, architecture notes
  web/ui/                  Browser UI
  .github/workflows/       CI
  Dockerfile               Go API container image
  compose.yaml             local PostgreSQL
```

## Local API

Without `DATABASE_URL`, the API uses an in-memory store.

```bash
go run ./cmd/api
```

For local PostgreSQL:

```bash
docker compose up -d postgres
DATABASE_URL='postgres://spotdiggz:spotdiggz@localhost:5432/spotdiggz?sslmode=disable' go run ./cmd/api
```

Build the API binary:

```bash
go build -o bin/spotdiggz-api ./cmd/api
```

## Browser UI

```bash
cd web/ui
npm ci
npm run dev
```

## Verification

```bash
go fmt ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
go test ./cmd/... ./internal/...
go build -o bin/spotdiggz-api ./cmd/api

cd web/ui
npm ci
npm run lint
npm run type-check
npm test -- --watch=false
npm run build
```

## Documentation

- ADR-001: [docs/adr/001-go-skate-spot-metadata-api.md](docs/adr/001-go-skate-spot-metadata-api.md)
- ADR-002: [docs/adr/002-repository-boundary-api-browser-mobile.md](docs/adr/002-repository-boundary-api-browser-mobile.md)
- API architecture: [docs/api_architecture.md](docs/api_architecture.md)
- API implementation plan: [docs/go-api-implementation-plan.md](docs/go-api-implementation-plan.md)
- OpenAPI: [docs/openapi.yaml](docs/openapi.yaml)
