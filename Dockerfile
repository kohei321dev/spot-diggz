FROM golang:1.25.10-bookworm AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/spotdiggz-api ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/spotdiggz-api /spotdiggz-api
EXPOSE 8080
USER nonroot:nonroot

ENTRYPOINT ["/spotdiggz-api"]
