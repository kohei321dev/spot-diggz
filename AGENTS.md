# AGENTS.md

この repository で作業する agent 向けの指示です。

## Scope

[事実] spot-diggz の active tree は Go 製 Skate Spot Metadata API と Browser UI を対象にする。

[事実] Rust API、GCP deploy / Terraform、iOS、Android の active source はこの repository から外している。

[事実] legacy code は `archive/rust-gcp-legacy-20260530` と `archive/rails-original` で参照する。

## Working Rules

- 日本語で簡潔に報告する。
- 技術判断、原因分析、不確実な事項では `[事実]` `[推測]` `[未検証]` を分ける。
- `develop`、`main`、`master` へ直接 commit しない。作業用 branch を使う。
- secret、token、key、account ID、private URL、raw log を commit しない。
- 外部 write、GitHub settings、cloud、secret、DNS の変更前には人間承認を取る。
- 変更前に対象ファイルを読み、既存の Go / Browser UI 構成に合わせる。
- active tree に Rust / GCP deploy / Terraform / iOS / Android 実装を戻さない。

## Expected Checks

Go API:

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

Security checks, when available:

```bash
govulncheck ./cmd/... ./internal/...
govulncheck -mode=binary ./bin/spotdiggz-api
trivy fs .
```

## Architecture References

- [docs/adr/001-go-skate-spot-metadata-api.md](docs/adr/001-go-skate-spot-metadata-api.md)
- [docs/adr/002-repository-boundary-api-browser-mobile.md](docs/adr/002-repository-boundary-api-browser-mobile.md)
- [docs/api_architecture.md](docs/api_architecture.md)
- [docs/openapi.yaml](docs/openapi.yaml)
