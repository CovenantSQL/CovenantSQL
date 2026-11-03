#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

main() {
  go test -tags "${UNITTESTTAGS:-}" -race -failfast -parallel 16 -cpu 16 $(go list ./... | grep -v CovenantSQL/api)
  go test -tags "${UNITTESTTAGS:-}" -race -failfast -parallel 16 -cpu 16 ./api/...

  set -x
  # some benchmarks
  go test -tags "${UNITTESTTAGS:-}" -bench=^BenchmarkPersistentCaller_Call$ -run ^$ ./rpc/
  bash cleanupDB.sh || true
  go test -tags "${UNITTESTTAGS:-}" -bench=^BenchmarkMiner$ -benchtime=5s -run ^$ ./cmd/cql-minerd/ -bench-miner-count=2
  bash cleanupDB.sh || true
}

main "$@"
