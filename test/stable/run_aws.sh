#!/bin/bash
set -xeo pipefail

export WORKING_DIR=/home/ubuntu/gopath/src/github.com/CovenantSQL/CovenantSQL
export RUNNING_DIR=$(cd `dirname $0`; pwd)
export LOG_DIR=/data/logs

if [[ ! -d "$LOG_DIR" ]]; then
	echo "$LOG_DIR not exist"
    exit 1
fi

./scripts/clean.sh

#Monitor
nohup sar -uqrBWbdv -o ${LOG_DIR}/monitor_uqrBWbdv.log 5 2>&1 > ${LOG_DIR}/monitor_std.log &

#Prepare
cp -r ${RUNNING_DIR}/docker-compose.yml ${WORKING_DIR}

cd ${WORKING_DIR}
make docker

cd ${RUNNING_DIR}
if [[ -z "$1" ]]; then
    ./scripts/start.sh bp
    ./scripts/start.sh miner0
    ./scripts/start.sh miner1
    ./scripts/start.sh miner2
    ./scripts/start.sh miner3
    ./scripts/start.sh client
else
    ./scripts/start.sh $1
fi
