# Repository Structure

[事実] spot-diggz の active tree は Go API と Browser UI で構成する。

[事実] iOS / Android は別project / 別repositoryで扱う。過去の途中実装は archive branch に残す。

```text
spot-diggz/
  .github/workflows/
    ci.yml
  cmd/
    api/
  db/
    migrations/
    schema.sql
  docs/
    adr/
    api_architecture.md
    go-api-implementation-plan.md
    openapi.yaml
    repo_structure.md
  internal/
    httpapi/
    postgres/
    spot/
  web/
    ui/
  AGENTS.md
  CLAUDE.md
  Dockerfile
  README.md
  compose.yaml
  go.mod
  go.sum
```

## Directory Ownership

- `cmd/`, `internal/`, `db/`: Go API
- `docs/openapi.yaml`: API contract
- `docs/adr/`: accepted architecture decisions
- `web/ui/`: Browser UI
- `.github/workflows/ci.yml`: Go API / Browser UI / security checks

## Not Active In This Repository

- Rust API implementation
- GCP deployment configuration
- Terraform configuration
- iOS app implementation
- Android app implementation
