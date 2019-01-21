#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

test::package() {
  local package="${1:-notset}"

  if [[ "${package}" == "notset" ]]; then
    &>2 echo "empty package name"
    exit 1
  fi

  local coverage_file="${package//\//.}.cover.out"
  echo "[TEST] package=${package}, coverage=${coverage_file}"
  go test -race -failfast -parallel 16 -cpu 16 -coverpkg="github.com/CovenantSQL/CovenantSQL/..." -coverprofile "${coverage_file}" "${package}"
}

main() {
  make clean
  make use_all_cores

  # test package by package
  for package in $(go list ./... | grep -v "/vendor/"); do
    test::package "${package}"
  done

  # some benchmarks
  go test -bench=^BenchmarkPersistentCaller_Call$ -run ^$ ./rpc/
  bash cleanupDB.sh || true
  go test -bench=^BenchmarkMinerTwo$ -benchtime=5s -run ^$ ./cmd/cql-minerd/
  go test -bench=^BenchmarkTestnetMiner2$ -benchtime=5s -run ^$ ./cmd/cql-minerd/
  gocovmerge *.cover.out $(find cmd -name "*.cover.out") | grep -F -v '_gen.go' > coverage.txt && rm -f *.cover.out
  bash <(curl -s https://codecov.io/bash)
}

main "$@"

