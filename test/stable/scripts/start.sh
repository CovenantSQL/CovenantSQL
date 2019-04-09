#!/bin/bash
set -xeuo pipefail

role=$1

if [ -z "$WORKING_DIR" ]; then
    WORKING_DIR=/home/ubuntu/gopath/src/github.com/CovenantSQL/CovenantSQL
fi
if [ -z "$RUNNING_DIR" ]; then
    RUNNING_DIR=$(cd `dirname $0`/..; pwd)
fi
if [ -z "$LOG_DIR" ]; then
    LOG_DIR=/data/logs
fi

cd ${WORKING_DIR}

case $role in
    bp)
        # start bp
        docker-compose up --no-start covenantsql_bp_0 covenantsql_bp_1 covenantsql_bp_2
        docker-compose start covenantsql_bp_0 covenantsql_bp_1 covenantsql_bp_2
        ;;
    miner0)
        cp -r ${RUNNING_DIR}/node_miner_0 /data
        # start miner
        docker-compose up --no-start covenantsql_miner_0
        docker-compose start covenantsql_miner_0
        ;;
    miner1)
        cp -r ${RUNNING_DIR}/node_miner_1 /data
        # start miner
        docker-compose up --no-start covenantsql_miner_1
        docker-compose start covenantsql_miner_1
        ;;
    miner2)
        cp -r ${RUNNING_DIR}/node_miner_2 /data
        # start miner
        docker-compose up --no-start covenantsql_miner_2
        docker-compose start covenantsql_miner_2
        ;;
    miner3)
        cp -r ${RUNNING_DIR}/node_miner_3 /data
        # start miner
        docker-compose up --no-start covenantsql_miner_3
        docker-compose start covenantsql_miner_3
        ;;
    client)
        cd ${RUNNING_DIR}
        docker cp covenantsql_bp_1:/app/cql ${RUNNING_DIR}
        sleep 3s
        ${RUNNING_DIR}/cql create -config ${RUNNING_DIR}/node_c/config.yaml \
            -wait-tx-confirm -no-password '{"node":4,"advancepayment": 2000000000}' | tee dsn.txt
        dsn=$(cat dsn.txt)

        #Start client
        go build -o 500million
        nohup ${RUNNING_DIR}/500million -config ${RUNNING_DIR}/node_c/config.yaml -dsn ${dsn} > ${LOG_DIR}/client.log 2>&1 &
        ;;
esac


