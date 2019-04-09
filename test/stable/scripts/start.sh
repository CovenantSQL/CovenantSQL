#!/bin/bash
set -xeuo pipefail

if [ -z "$WORKING_DIR" ]; then
    WORKING_DIR=/home/ubuntu/gopath/src/github.com/CovenantSQL/CovenantSQL
fi
if [ -z "$RUNNING_DIR" ]; then
    RUNNING_DIR=$(cd `dirname $0`/..; pwd)
fi
if [ -z "$LOG_DIR" ]; then
    LOG_DIR=/data/logs
fi

#Prepare
cp -r ${RUNNING_DIR}/docker-compose.yml ${WORKING_DIR}
cp -r ${RUNNING_DIR}/node_miner_0 /data
cp -r ${RUNNING_DIR}/node_miner_1 /data
cp -r ${RUNNING_DIR}/node_miner_2 /data
cp -r ${RUNNING_DIR}/node_miner_3 /data

cd ${WORKING_DIR}
make docker_clean
make docker
make start

cd ${RUNNING_DIR}
docker cp covenantsql_bp_1:/app/cql ${RUNNING_DIR}
sleep 3s
${RUNNING_DIR}/cql create -config ${RUNNING_DIR}/node_c/config.yaml \
    -wait-tx-confirm -no-password '{"node":4,"advancepayment": 2000000000}' | tee dsn.txt
dsn=$(cat dsn.txt)

#Start
go build -o 500million
nohup ${RUNNING_DIR}/500million -config ${RUNNING_DIR}/node_c/config.yaml -dsn ${dsn} > ${LOG_DIR}/client.log 2>&1 &

#Monitor
nohup sar -uqrBWbdv -o ${LOG_DIR}/monitor_uqrBWbdv.log 5 2>&1 > ${LOG_DIR}/monitor_std.log &

