FROM golang:1.25.13-bookworm@sha256:e401dae1bf814e29204a8cb7915682e1780951e609ca0dd8865ee1937f510c48 AS builder

WORKDIR /src
COPY go.mod ./
COPY go.sum ./
COPY cmd ./cmd
COPY internal ./internal
COPY data ./data
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/spotdiggz-api ./cmd/api \
    && install -d -m 0700 /out/state

FROM scratch

COPY --from=builder /out/spotdiggz-api /spotdiggz-api
COPY --from=builder /src/data /data
COPY --from=builder --chown=65532:65532 /out/state /var/lib/spotdiggz
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
ENV FACILITY_CATALOG_PATH=/data/facilities.json
ENV CORRECTION_STORE_PATH=/var/lib/spotdiggz/corrections.jsonl
ENV APP_ENV=production
EXPOSE 8080
USER 65532:65532

ENTRYPOINT ["/spotdiggz-api"]
