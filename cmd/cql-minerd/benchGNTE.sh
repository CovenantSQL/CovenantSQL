#! /usr/bin/env bash
set -euo pipefail

declare pkg="github.com/CovenantSQL/CovenantSQL/cmd/cql-minerd"
declare flags=(
    "-bench=^BenchmarkMinerGNTE$"
    "-benchtime=10s"
    "-run=^$"
)

clean() {
    if [ -n "${TEST_WD}" ]; then
        # Clean
        sudo ${TEST_WD}/GNTE/scripts/cleanupDB.sh
        bash -x ${TEST_WD}/GNTE/generate.sh ${delay_file}
        sleep 5
    fi
}

fast() {
    echo "Fast benchmarking with flags: $@"
    clean
    go test        "${flags[@]}" "$pkg" "$@"                      | tee -a gnte.log
    clean
    go test        "${flags[@]}" "$pkg" "$@" -bench-miner-count=2 | tee -a gnte.log
    clean
    go test -cpu=1 "${flags[@]}" "$pkg" "$@" -bench-miner-count=2 | tee -a gnte.log
}

full() {
    echo "Full benchmarking with flags: $@"
    local cpus=("" 4 1) counts=(1 2 4 8)
    local cpu count caseflags
    for cpu in "${cpus[@]}"; do
        if [[ -z $cpu ]]; then
            caseflags=("${flags[@]}")
        else
            caseflags=("-cpu=$cpu" "${flags[@]}")
        fi
        for count in "${counts[@]}"; do
            clean

            go test "${caseflags[@]}" "$pkg" "$@" -bench-miner-count=$count | tee -a gnte.log

            ips=(2 3 4 5 6 7 8 9)
            cur_sec=`date '+%s'`
            for ip in "${ips[@]}"; do
                go tool pprof -png -inuse_objects http://10.250.100.${ip}:6060/debug/pprof/heap \
                        > ${WORKSPACE}/${cur_sec}_minor_${ip}_objectinuse.png
                go tool pprof -png http://10.250.100.${ip}:6060/debug/pprof/heap \
                        > ${WORKSPACE}/${cur_sec}_minor_${ip}_heapinuse.png
            done
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
