#! /usr/bin/env bash
set -euo pipefail

main() {
    local wd="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.."; pwd)"
    make -C "$wd" clean
    make -C "$wd" use_all_cores

    rm -f custom_miner.log
    touch custom_miner.log

    local flags=(
        "-bench=^BenchmarkCustomMiner$"
        "-benchtime=10s"
        "-run=^$"
    )
    local pkg="github.com/CovenantSQL/CovenantSQL/cmd/cql-minerd"
    local i subflags
    for ((i=1; i<=3; i++)); do
        subflags=(
            "-bench-miner-config-dir=$wd/test/bench_testnet/node_c"
            "-bench-miner-count=$i"
        )
        go test        "${flags[@]}" "$pkg" "${subflags[@]}" | tee -a custom_miner.log
        go test -cpu=1 "${flags[@]}" "$pkg" "${subflags[@]}" | tee -a custom_miner.log
    done
}

main "$@"
