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

yes | ${BIN}/cql generate

#label myself
sed 's/0.0.0.0:15151/testnet_compatibility/g' ~/.cql/config.yaml > ~/.cql/config1.yaml

mv ~/.cql/config1.yaml ~/.cql/config.yaml

#get wallet addr
wallet=$(grep "WalletAddress" ~/.cql/config.yaml | awk '{print $2}')

#transfer some coin to above address
${BIN}/cql transfer -config ${PROJECT_DIR}/conf/testnet/config.yaml -wait-tx-confirm \
    -address ${wallet} -amount 100000000 -type Particle

${BIN}/cql wallet

# create database only in miner00 and miner01
${BIN}/cql create -wait-tx-confirm -node 2 \
    -target-miners 'ba0ba731c7a76ccef2c1170f42038f7e228dfb474ef0190dfe35d9a37911ed37,1a7b0959bbd0d0ec529278a61c0056c277bffe75b2646e1699b46b10a90210be'

#get dsn
dsn=$(cat ~/.cql/.dsn | tail -n1)
if [ -z "$dsn" ]; then
    exit 1
fi

${BIN}/cql console \
    -command 'create table test_for_new_account(column1 int);' \
    ${dsn}

${BIN}/cql console -command 'show tables;' ${dsn} | tee result.log

grep "1 row" result.log
