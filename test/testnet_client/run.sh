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

echo -ne "y\n" | ${BIN}/cql generate -no-password config

#get wallet addr
${BIN}/cql wallet -no-password | tee wallet.txt
wallet=$(awk '{print $3}' wallet.txt)

#transfer some coin to above address
${BIN}/cql transfer -config ${PROJECT_DIR}/conf/testnet/config.yaml -wait-tx-confirm -no-password \
    '{"addr":"'${wallet}'", "amount":"100000000 Particle"}'

${BIN}/cql wallet -no-password -balance all

${BIN}/cql create -wait-tx-confirm -no-password '{"node":2}' | tee dsn.txt

#get dsn
dsn=$(cat dsn.txt)
if [ -z "$dsn" ]; then
    exit 1
fi

${BIN}/cql console -no-password \
    -command 'create table test_for_new_account(column1 int);' \
    -dsn ${dsn}

${BIN}/cql console -dsn ${dsn} -no-password \
    -command 'show tables;' | tee result.log

grep "1 row" result.log
