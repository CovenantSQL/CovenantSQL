#!/bin/bash -x

yaml[0]=./scripts/gnte_0ms.yaml
yaml[1]=./scripts/gnte_5ms.yaml
yaml[2]=./scripts/gnte_20ms.yaml
yaml[3]=./scripts/gnte_100ms.yaml
yaml[4]=./scripts/gnte_200ms.yaml


TEST_WD=$(cd $(dirname $0)/; pwd)
PROJECT_DIR=$(cd ${TEST_WD}/../../; pwd)

echo ${PROJECT_DIR}

# Build
#Notice!!!!: uncomment this when you run this manually.
#cd ${PROJECT_DIR} && ./build.sh

BENCHRESULT_FILE=${PROJECT_DIR}/bench.txt
if [ -f ${BENCHRESULT_FILE} ];then
    rm -rf ${BENCHRESULT_FILE}
fi
tmp_file=${PROJECT_DIR}/tmp.log
if [ -f ${tmp_file} ];then
    rm -rf ${tmp_file}
fi

# Clean submodule
cd ${TEST_WD}/GNTE/ && git clean -dfx

for gnte_yaml in ${yaml[@]};
do
    if [ -d ${TEST_WD}/GNTE/scripts/bin ];then
        rm -rf ${TEST_WD}/GNTE/scripts/bin
    fi

    # Prepare
    cd ${PROJECT_DIR} && cp ./cleanupDB.sh ${TEST_WD}/GNTE/scripts
    cd ${PROJECT_DIR} && cp -r ./bin ${TEST_WD}/GNTE/scripts
    cd ${TEST_WD} && cp -r ./conf/* ./GNTE/scripts
    cd ${TEST_WD}/GNTE && bash -x ./build.sh

    # Clean
    cd ${TEST_WD} && bash ./GNTE/scripts/cleanupDB.sh
    cd ${TEST_WD}/GNTE && bash -x ./generate.sh ${gnte_yaml}

    # Bench GNTE
    cd ${PROJECT_DIR}/cmd/cql-minerd/
    bash -x ./benchGNTE.sh
    echo "${gnte_yaml}" >> ${tmp_file}
    grep BenchmarkMinerGNTE gnte.log >> ${tmp_file}
    echo "" >> ${tmp_file}
done

# clean GNTE docker
cd ${TEST_WD} && bash ./GNTE/scripts/cleanupDB.sh
cd ${TEST_WD} && bash ./GNTE/scripts/clean.sh

perl -lane 'print $F[0], "\t", 1000000000.0/$F[2] if $F[2]; print if /script/' ${tmp_file} > ${BENCHRESULT_FILE}
#cd ${TEST_WD}/GNTE && bash -x ./generate.sh stopall miner
#cd test/GNTE/GNTE/scripts/node_miner_10.250.100.3/
#cd data/randomid
#replace storage.db3* file
#cd ${TEST_WD}/GNTE && bash -x ./generate.sh startall

