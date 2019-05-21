#! /usr/bin/env bash
set -euo pipefail

declare pkg="github.com/CovenantSQL/CovenantSQL/cmd/cql-minerd"
declare flags=(
    "-test.bench=^BenchmarkMinerGNTE$"
    "-test.benchtime=10s"
    "-test.run=^$"
)

BENCH_BIN=${PROJECT_DIR}/bin/intergration.test

clean() {
    if [ -n "${TEST_WD}" ]; then
        # Clean
        sudo ${TEST_WD}/GNTE/scripts/cleanupDB.sh
        if [[ ${delay_file} =~ "eventual" ]]; then
            bash -x ${TEST_WD}/GNTE/generate.sh ./scripts/gnte_200ms.yaml
        else
            bash -x ${TEST_WD}/GNTE/generate.sh ${delay_file}
        fi
        sleep 5
    fi
}

fast() {
    echo "Fast benchmarking with flags: $@"
    clean
    ${BENCH_BIN} "$@" "${flags[@]}" "$pkg" | tee -a gnte.log
    clean
    ${BENCH_BIN} -bench-miner-count=2 "$@" "${flags[@]}" "$pkg" | tee -a gnte.log
    clean
    ${BENCH_BIN} -bench-miner-count=2 "$@" -test.cpu=1 "${flags[@]}" "$pkg" | tee -a gnte.log
}

full() {
    echo "Full benchmarking with flags: $@"
    local cpus=("" 4 1) counts=(1 2 4 8)
    local cpu count caseflags
    for count in "${counts[@]}"; do
        for cpu in "${cpus[@]}"; do
            if [[ -z $cpu ]]; then
                caseflags=("${flags[@]}")
            else
                caseflags=("-test.cpu=$cpu" "${flags[@]}")
            fi

            clean

            ${BENCH_BIN} "$@" -bench-miner-count=$count "${caseflags[@]}" "$pkg" | tee -a gnte.log

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
    if [[ $# -gt 0 && $1 == "fast" ]]; then
        fast
    else
        if [[ ${delay_file} =~ "eventual" ]]; then
            full -bench-eventual-consistency
        else
            full
        fi
    fi
}

main "$@"
