# CLAUDE.md

spot-diggz project instructions.

## Active Architecture

[事実] This repository is for the Go Skate Spot Metadata API and Browser UI.

[事実] Active API code lives under:

- `cmd/api`
- `internal/httpapi`
- `internal/postgres`
- `internal/spot`
- `db`

[事実] Browser UI code lives under `web/ui`.

[事実] Rust API, GCP deployment, Terraform, iOS, and Android implementation are not active source in this repository. Use archive branches only when historical reference is needed.

## Commands

Go:

```bash
go fmt ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
go test ./cmd/... ./internal/...
go build -o bin/spotdiggz-api ./cmd/api
```

Browser UI:

```bash
cd web/ui
npm ci
npm run lint
npm run type-check
npm test -- --watch=false
npm run build
```

Local PostgreSQL:

```bash
docker compose up -d postgres
DATABASE_URL='postgres://spotdiggz:spotdiggz@localhost:5432/spotdiggz?sslmode=disable' go run ./cmd/api
```

## Environment

- `DATABASE_URL`: optional. If unset, API uses the in-memory store.
- `ADDR`: optional API listen address. Default is `:8080`.
- `web/ui` may use Vite environment variables for API and browser auth integration.

Do not commit real secrets or local `.env` files.

## References

- `README.md`
- `docs/adr/001-go-skate-spot-metadata-api.md`
- `docs/adr/002-repository-boundary-api-browser-mobile.md`
- `docs/api_architecture.md`
- `docs/openapi.yaml`
