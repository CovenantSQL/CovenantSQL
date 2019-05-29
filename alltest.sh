#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

main() {
  go test -tags "${UNITTESTTAGS:-}" -race -failfast -parallel 16 -cpu 16 -coverprofile main.cover.out $(go list ./... | grep -v CovenantSQL/api)
  go test -tags "${UNITTESTTAGS:-}" -race -failfast -parallel 16 -cpu 16 -coverpkg ./api/...,./rpc/jsonrpc -coverprofile api.cover.out ./api/...

  set -x
  gocovmerge main.cover.out api.cover.out $(find cmd -name "*.cover.out") | grep -F -v '_gen.go' > coverage.txt && rm -f *.cover.out
  bash <(curl -s https://codecov.io/bash)

  # some benchmarks
  go test -tags "${UNITTESTTAGS:-}" -bench=^BenchmarkPersistentCaller_Call$ -run ^$ ./rpc/
  bash cleanupDB.sh || true
  go test -tags "${UNITTESTTAGS:-}" -bench=^BenchmarkMiner$ -benchtime=5s -run ^$ ./cmd/cql-minerd/ -bench-miner-count=2
  bash cleanupDB.sh || true
}

main "$@"

