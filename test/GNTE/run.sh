#!/bin/bash -x
set -eo pipefail

param=$1
if [ "fast" == "$param" ]; then
    yaml=./scripts/gnte_0ms.yaml
else
    yaml=(
        ./scripts/gnte_{0,0.2,5,20,100,eventual}ms.yaml
    )
fi

export TEST_WD=$(cd $(dirname $0)/; pwd)
PROJECT_DIR=$(cd ${TEST_WD}/../../; pwd)

echo ${PROJECT_DIR}

# Build
#Notice!!!!: uncomment this when you run this manually.
#cd ${PROJECT_DIR} && make clean
#cd ${PROJECT_DIR} && make use_all_cores

BENCHRESULT_FILE=${PROJECT_DIR}/bench.txt
if [ -f ${BENCHRESULT_FILE} ];then
    rm -rf ${BENCHRESULT_FILE}
fi
tmp_file=${PROJECT_DIR}/tmp.log
if [ -f ${tmp_file} ];then
    rm -rf ${tmp_file}
fi

# Clean submodule
cd ${TEST_WD}/GNTE/ && sudo git clean -dfx

if [ -d ${TEST_WD}/GNTE/scripts/bin ];then
    rm -rf ${TEST_WD}/GNTE/scripts/bin
fi

# Prepare
cd ${PROJECT_DIR} && cp ./cleanupDB.sh ${TEST_WD}/GNTE/scripts
cd ${PROJECT_DIR} && cp -r ./bin ${TEST_WD}/GNTE/scripts
cd ${TEST_WD} && cp -r ./conf/* ./GNTE/scripts
cd ${TEST_WD}/GNTE && bash -x ./build.sh


for gnte_yaml in ${yaml[@]};
do
    export delay_file=${gnte_yaml}

    # Bench GNTE
    cd ${PROJECT_DIR}/cmd/cql-minerd/
    ips=(2 3 4 5 6 7 8 9)
    if ! bash -x ./benchGNTE.sh $param; then
        for ip in "${ips[@]}"; do
            docker logs miner10.250.100.${ip} 2> $WORKSPACE/miner10.250.100.${ip}.txt
        done
        exit 1
    fi
    echo "${delay_file}" >> ${tmp_file}
    grep BenchmarkMinerGNTE gnte.log | sed 's/BenchmarkMinerGNTE\///g' >> ${tmp_file}
    echo "" >> ${tmp_file}
done

# clean GNTE docker
cd ${TEST_WD} && sudo ./GNTE/scripts/cleanupDB.sh
cd ${TEST_WD} && bash ./GNTE/scripts/clean.sh

perl -lane 'print $F[0], "\t", $F[1], "\t", $F[2], "\t", 1000000000.0/$F[2] if $F[2]; print if /script/' ${tmp_file} > ${BENCHRESULT_FILE}
#cd ${TEST_WD}/GNTE && bash -x ./generate.sh stopall miner
#cd test/GNTE/GNTE/scripts/node_miner_10.250.100.3/
#cd data/randomid
#replace storage.db3* file
#cd ${TEST_WD}/GNTE && bash -x ./generate.sh startall

