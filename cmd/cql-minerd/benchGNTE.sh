#! /usr/bin/env bash
set -euo pipefail

declare pkg="github.com/CovenantSQL/CovenantSQL/cmd/cql-minerd"
declare flags=(
    "-bench=^BenchmarkMinerGNTE$"
    "-benchtime=10s"
    "-run=^$"
)

fast() {
    echo "Fast benchmarking with flags: $@"
    go test        "${flags[@]}" "$pkg" "$@"                      | tee -a gnte.log
    go test        "${flags[@]}" "$pkg" "$@" -bench-miner-count=2 | tee -a gnte.log
    go test -cpu=1 "${flags[@]}" "$pkg" "$@" -bench-miner-count=2 | tee -a gnte.log
}

full() {
    echo "Full benchmarking with flags: $@"
    local cpus=("" 4 1) counts=(1 2 3 4 8)
    local cpu count caseflags
    for cpu in "${cpus[@]}"; do
        if [[ -z $cpu ]]; then
            caseflags=("${flags[@]}")
        else
            caseflags=("-cpu=$cpu" "${flags[@]}")
        fi
        for count in "${counts[@]}"; do
            go test "${caseflags[@]}" "$pkg" "$@" -bench-miner-count=$count | tee -a gnte.log
        done
        ips=(2 3 4 5 6 7 8 9)
        cur_sec=`date '+%s'`
        for ip in "${ips[@]}"; do
            go tool pprof -png -inuse_objects http://10.250.100.${ip}:6060/debug/pprof/heap \
                    > ${WORKSPACE}/${cur_sec}_minor_${ip}_objectinuse.png
            go tool pprof -png http://10.250.100.${ip}:6060/debug/pprof/heap \
                    > ${WORKSPACE}/${cur_sec}_minor_${ip}_heapinuse.png
        done
    done
}

main() {
    rm -f gnte.log
    touch gnte.log
    if [[ $# -gt 0 && $1 = "fast" ]]; then
        fast
    else
        full
        full -bench-eventual-consistency
    fi
}

main "$@"
