#!/bin/bash -x
test_case=$1
set -e

TEST_WD=$(cd $(dirname $0)/; pwd)
PROJECT_DIR=$(cd ${TEST_WD}/../../; pwd)

OLD_BIN_DIR=${CACHE_DIR}/${PREV_VERSION}
NEW_BIN_DIR=${PROJECT_DIR}/bin

LOGS_DIR=${PIPELINE_CACHE}/logs/old_${test_case}
mkdir -p ${LOGS_DIR}

case $test_case in
    "client")
        CLIENTBIN=${OLD_BIN_DIR}/cql
        BPBIN=${NEW_BIN_DIR}/cqld
        MINERBIN=${NEW_BIN_DIR}/cql-minerd
        ;;
    "bp")
        CLIENTBIN=${NEW_BIN_DIR}/cql
        BPBIN=${OLD_BIN_DIR}/cqld
        MINERBIN=${NEW_BIN_DIR}/cql-minerd
        ;;
    "miner")
        CLIENTBIN=${NEW_BIN_DIR}/cql
        BPBIN=${NEW_BIN_DIR}/cqld
        MINERBIN=${OLD_BIN_DIR}/cql-minerd
        ;;
    *)
        exit 1
        ;;
esac

cd ${TEST_WD}
# start bp
nohup ${BPBIN} -config node_0/config.yaml >${LOGS_DIR}/bp0.log 2>&1 &
nohup ${BPBIN} -config node_1/config.yaml >${LOGS_DIR}/bp1.log 2>&1 &
nohup ${BPBIN} -config node_2/config.yaml >${LOGS_DIR}/bp2.log 2>&1 &

# wait bp start
sleep 20

# start miner
nohup ${MINERBIN} -config node_miner_0/config.yaml >${LOGS_DIR}/miner0.log 2>&1 &
nohup ${MINERBIN} -config node_miner_1/config.yaml >${LOGS_DIR}/miner1.log 2>&1 &
nohup ${MINERBIN} -config node_miner_2/config.yaml >${LOGS_DIR}/miner2.log 2>&1 &

# wait miner start
sleep 20

${CLIENTBIN} wallet -config node_c/config.yaml -balance all -no-password

${CLIENTBIN} create -config node_c/config.yaml -wait-tx-confirm -no-password '{"node":2}' | tail -n1 | tee dsn.txt

#get dsn
dsn=$(cat dsn.txt)
if [ -z "$dsn" ]; then
    exit 1
fi

${CLIENTBIN} console -config ${PROJECT_DIR}/test/integration/node_c/config.yaml -dsn ${dsn} \
    -command 'create table test_for_new_account(column1 int);' -no-password

${CLIENTBIN} console -config ${PROJECT_DIR}/test/integration/node_c/config.yaml -dsn ${dsn} \
    -command 'show tables;' -no-password | tee result.log

grep "1 row" result.log

