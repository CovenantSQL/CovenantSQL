#! /usr/bin/env bash
set -euo pipefail


main() {
    local wd="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.."; pwd)"
    make -C "$wd" clean
    make -C "$wd" use_all_cores

    local pkg="github.com/CovenantSQL/CovenantSQL/cmd/cql-minerd"
    #go test -bench=^BenchmarkSQLite$ -benchtime=10s -run=^$ "$pkg"

    local flags=(
        "-bench=^BenchmarkMiner$"
        "-benchtime=10s"
        "-run=^$"
    )
    local i
    for ((i=1; i<=3; i++)); do
        go test "${flags[@]}" "$pkg" -bench-miner-count=$i
        go test "${flags[@]}" "$pkg" -bench-miner-count=$i -bench-bypass-signature 
        go test "${flags[@]}" "$pkg" -bench-miner-count=$i -bench-eventual-consistency
        go test "${flags[@]}" "$pkg" -bench-miner-count=$i -bench-bypass-signature -bench-eventual-consistency
    done
}

main "$@"
