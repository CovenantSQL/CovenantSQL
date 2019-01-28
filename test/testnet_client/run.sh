#!/bin/bash -x

set -e

TEST_WD=$(cd $(dirname $0)/; pwd)
PROJECT_DIR=$(cd ${TEST_WD}/../../; pwd)

echo ${PROJECT_DIR}

# Build
# cd ${PROJECT_DIR} && make clean
# cd ${PROJECT_DIR} && make use_all_cores

cd ${TEST_WD}
echo -ne "y\n" | ${PROJECT_DIR}/bin/cql-utils -tool confgen -skip-master-key
${PROJECT_DIR}/bin/cql-utils -tool addrgen -skip-master-key | tee wallet.txt

#get wallet addr
wallet=$(awk '{print $3}' wallet.txt)

#transfer some coin to above address
${PROJECT_DIR}/bin/cql -config ${PROJECT_DIR}/conf/testnet/config.yaml -transfer \
    '{"addr":"'${wallet}'", "amount":"100000000 Particle"}' -wait-tx-confirm

${PROJECT_DIR}/bin/cql -get-balance

${PROJECT_DIR}/bin/cql -create 2 -wait-tx-confirm | tee dsn.txt

#get dsn
dsn=$(cat dsn.txt)

${PROJECT_DIR}/bin/cql -dsn ${dsn} \
    -command 'create table test_for_new_account(column1 int);'

${PROJECT_DIR}/bin/cql -dsn ${dsn} \
    -command 'show tables;' | tee result.log

grep "1 row" result.log
