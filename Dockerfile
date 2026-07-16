FROM golang:1.25.12-bookworm AS builder

WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
COPY data ./data
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/spotdiggz-api ./cmd/api

FROM scratch

COPY --from=builder /out/spotdiggz-api /spotdiggz-api
COPY --from=builder /src/data /data
EXPOSE 8080
USER 65532:65532

ENTRYPOINT ["/spotdiggz-api"]
