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
nohup ${BPBIN} -config node_0/config.yaml -log-level debug >${LOGS_DIR}/bp0.log 2>&1 &
nohup ${BPBIN} -config node_1/config.yaml -log-level debug >${LOGS_DIR}/bp1.log 2>&1 &
nohup ${BPBIN} -config node_2/config.yaml -log-level debug >${LOGS_DIR}/bp2.log 2>&1 &

# wait bp start
sleep 20

# start miner
nohup ${MINERBIN} -config node_miner_0/config.yaml >${LOGS_DIR}/miner0.log 2>&1 &
nohup ${MINERBIN} -config node_miner_1/config.yaml >${LOGS_DIR}/miner1.log 2>&1 &
nohup ${MINERBIN} -config node_miner_2/config.yaml >${LOGS_DIR}/miner2.log 2>&1 &

# wait miner start
sleep 20

# use new client and bp_node_c node config to approve miners for public service
(
    ${NEW_BIN_DIR}/cql rpc -config node_bp_c/config.yaml -bp -name 'MCC.AddTx' -wait-tx-confirm \
        -req '{"TxType": 13, "Miner": "ba0ba731c7a76ccef2c1170f42038f7e228dfb474ef0190dfe35d9a37911ed37", "Enabled": 1}'
    ${NEW_BIN_DIR}/cql rpc -config node_bp_c/config.yaml -bp -name 'MCC.AddTx' -wait-tx-confirm \
        -req '{"TxType": 13, "Miner": "1a7b0959bbd0d0ec529278a61c0056c277bffe75b2646e1699b46b10a90210be", "Enabled": 1}'
    ${NEW_BIN_DIR}/cql rpc -config node_bp_c/config.yaml -bp -name 'MCC.AddTx' -wait-tx-confirm \
        -req '{"TxType": 13, "Miner": "9235bc4130a2ed4e6c35ea189dab35198ebb105640bedb97dd5269cc80863b16", "Enabled": 1}'
) || [[ "${test_case}" == "bp" ]]

${CLIENTBIN} wallet -config node_c/config.yaml
${CLIENTBIN} create -config node_c/config.yaml -wait-tx-confirm -db-node 2

#get dsn
dsn=$(cat node_c/.dsn | tail -n1)
if [ -z "$dsn" ]; then
    exit 1
fi

${CLIENTBIN} console -config ${PROJECT_DIR}/test/integration/node_c/config.yaml \
    -command 'create table test_for_new_account(column1 int);' ${dsn}

${CLIENTBIN} console -config ${PROJECT_DIR}/test/integration/node_c/config.yaml \
    -command 'show tables;' ${dsn} | tee result.log

grep "1 row" result.log

