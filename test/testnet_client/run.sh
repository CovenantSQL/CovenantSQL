#!/bin/bash -x
set -e

TEST_WD=$(cd $(dirname $0)/; pwd)
PROJECT_DIR=$(cd ${TEST_WD}/../../; pwd)
BIN=${PROJECT_DIR}/bin

echo ${PROJECT_DIR}

# Build
# cd ${PROJECT_DIR} && make clean
# cd ${PROJECT_DIR} && make use_all_cores

cd ${TEST_WD}

echo -ne "y\n" | ${BIN}/cql-utils -tool confgen -skip-master-key

#get wallet addr
${BIN}/cql-utils -tool addrgen -skip-master-key | tee wallet.txt
wallet=$(awk '{print $3}' wallet.txt)

#transfer some coin to above address
${BIN}/cql transfer -config ${PROJECT_DIR}/conf/testnet/config.yaml -wait-tx-confirm \
    '{"addr":"'${wallet}'", "amount":"100000000 Particle"}'

${BIN}/cql balance

${BIN}/cql create -wait-tx-confirm '{"node":2}' | tee dsn.txt

#get dsn
dsn=$(cat dsn.txt)

${BIN}/cql console -dsn ${dsn} \
    -command 'create table test_for_new_account(column1 int);'

${BIN}/cql console -dsn ${dsn} \
    -command 'show tables;' | tee result.log

grep "1 row" result.log
