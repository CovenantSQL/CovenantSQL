#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

main() {
  # These packages start BP and miner processes from the binaries built by make, whose ports
  # and log files clash when the packages run in parallel. go test cannot tell when those
  # binaries change, so never reuse cached test results either.
  local nodes='CovenantSQL/(cmd/cql-fuse|cmd/cql-minerd|cmd/cqld|rpc/mux|sqlchain/mirror|sqlchain/observer)$'
  go test -tags "${UNITTESTTAGS:-}" -race -failfast -parallel 16 -cpu 16 $(go list ./... | grep -v CovenantSQL/api | grep -vE "$nodes")
  go test -tags "${UNITTESTTAGS:-}" -race -failfast -count=1 -parallel 16 -cpu 16 -p 1 $(go list ./... | grep -E "$nodes")
  go test -tags "${UNITTESTTAGS:-}" -race -failfast -parallel 16 -cpu 16 ./api/...

  set -x
  # some benchmarks
  go test -tags "${UNITTESTTAGS:-}" -bench=^BenchmarkPersistentCaller_Call$ -run ^$ ./rpc/
  bash cleanupDB.sh || true
  go test -tags "${UNITTESTTAGS:-}" -bench=^BenchmarkMiner$ -benchtime=5s -run ^$ ./cmd/cql-minerd/ -bench-miner-count=2
  bash cleanupDB.sh || true
}

main "$@"
