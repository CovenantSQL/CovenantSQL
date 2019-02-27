#!/bin/bash -x

OUTSIDE_BIN_DIR=$1

TEST_WD=$(cd $(dirname $0)/; pwd)
PROJECT_DIR=$(cd ${TEST_WD}/../../; pwd)

if [ -n "$OUTSIDE_BIN_DIR" ]; then
    BIN=${OUTSIDE_BIN_DIR}
else
    BIN=${PROJECT_DIR}/bin
fi

cp ${PROJECT_DIR}/test/integration/node_c/config.yaml ~/.cql/
cp ${PROJECT_DIR}/test/integration/node_c/private.key ~/.cql/

# start current version bp
${PROJECT_DIR}/bin/cqld -config ${PROJECT_DIR}/test/integration/node_0/config.yaml
${PROJECT_DIR}/bin/cqld -config ${PROJECT_DIR}/test/integration/node_1/config.yaml
${PROJECT_DIR}/bin/cqld -config ${PROJECT_DIR}/test/integration/node_2/config.yaml

# wait bp start
sleep 10

# start current version miner
${PROJECT_DIR}/bin/cql-minerd -config ${PROJECT_DIR}/test/integration/node_miner_0/config.yaml
${PROJECT_DIR}/bin/cql-minerd -config ${PROJECT_DIR}/test/integration/node_miner_0/config.yaml
${PROJECT_DIR}/bin/cql-minerd -config ${PROJECT_DIR}/test/integration/node_miner_0/config.yaml

cd ${TEST_WD}

${BIN}/cql -get-balance

${BIN}/cql -create 2 -wait-tx-confirm | tee dsn.txt

#get dsn
dsn=$(cat dsn.txt)

${BIN}/cql -dsn ${dsn} \
    -command 'create table test_for_new_account(column1 int);'

${BIN}/cql -dsn ${dsn} \
    -command 'show tables;' | tee result.log

grep "1 row" result.log

