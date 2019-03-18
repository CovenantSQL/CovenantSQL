#! /usr/bin/env bash
set -euo pipefail

main() {
    local wd="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.."; pwd)"
    make -C "$wd" clean
    make -C "$wd" use_all_cores

    rm -f testnet.log
    touch testnet.log

    local flags=(
        "-bench=^BenchmarkTestnetMiner$"
        "-benchtime=10s"
        "-run=^$"
    )
    local pkg="github.com/CovenantSQL/CovenantSQL/cmd/cql-minerd"
    local cpus=("" 1) counts=(1 2 3)
    local cpu count caseflags
    for cpu in "${cpus[@]}"; do
        if [[ -z $cpu ]]; then
            caseflags=("${flags[@]}")
        else
            caseflags=("-cpu=$cpu" "${flags[@]}")
        fi
        for count in "${counts[@]}"; do
            go test "${caseflags[@]}" "$pkg" -bench-miner-count=$count | tee -a testnet.log
        done
    done
}

main "$@"
