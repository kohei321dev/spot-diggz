.PHONY: fmt test vet build run run-dev

fmt:
	gofmt -w cmd internal

test:
	go test ./cmd/... ./internal/...

vet:
	go vet ./cmd/... ./internal/...

build:
	go build -trimpath -o bin/spotdiggz-api ./cmd/api

run:
	go run ./cmd/api

run-dev:
	FACILITY_CATALOG_PATH=testdata/facilities.dev.json go run ./cmd/api
