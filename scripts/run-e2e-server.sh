#!/usr/bin/env bash
set -Eeuo pipefail

repository_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
temporary_store_directory="$(mktemp -d "${TMPDIR:-/tmp}/spot-diggz-e2e.XXXXXX")"
server_pid=""

cleanup() {
  exit_code=$?
  trap - EXIT HUP INT TERM

  if [[ -n "${server_pid}" ]] && kill -0 "${server_pid}" 2>/dev/null; then
    kill -TERM "${server_pid}" 2>/dev/null || true
    wait "${server_pid}" 2>/dev/null || true
  fi

  rm -f -- \
    "${temporary_store_directory}/spotdiggz-api" \
    "${temporary_store_directory}/facilities.json" \
    "${temporary_store_directory}/corrections.jsonl" \
    "${temporary_store_directory}/corrections.jsonl.tmp"
  rmdir -- "${temporary_store_directory}" 2>/dev/null || true
  exit "${exit_code}"
}

trap cleanup EXIT HUP INT TERM
cd "${repository_root}"
unset GOOGLE_MAPS_API_KEY

node scripts/prepare-e2e-catalog.mjs \
  testdata/facilities.dev.json \
  "${temporary_store_directory}/facilities.json"
CGO_ENABLED=0 go build -trimpath -o "${temporary_store_directory}/spotdiggz-api" ./cmd/api

FACILITY_CATALOG_PATH="${temporary_store_directory}/facilities.json" \
CORRECTION_STORE_PATH="${temporary_store_directory}/corrections.jsonl" \
PORT="${PORT:-18080}" \
  "${temporary_store_directory}/spotdiggz-api" &
server_pid=$!

wait "${server_pid}"
