FROM rust:1.90-bookworm AS builder

WORKDIR /app
COPY Cargo.toml Cargo.lock ./
COPY src ./src

ENV CARGO_REGISTRIES_CRATES_IO_PROTOCOL=sparse
ENV RUST_MIN_STACK=16777216
ENV CARGO_BUILD_JOBS=1
RUN cargo build --release

FROM debian:bookworm-slim

RUN apt-get update \
  && apt-get upgrade -y \
  && apt-get install -y --no-install-recommends ca-certificates libssl3 \
  && rm -rf /var/lib/apt/lists/*

RUN useradd -r -u 10001 sdz

WORKDIR /app
COPY --from=builder /app/target/release/sdz_api /usr/local/bin/sdz_api

ENV RUST_LOG=info,sdz_api=debug
EXPOSE 8080
USER sdz

CMD ["sdz_api"]
