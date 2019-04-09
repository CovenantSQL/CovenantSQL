#!/bin/bash
set -xeuo pipefail

export WORKING_DIR=/home/ubuntu/gopath/src/github.com/CovenantSQL/CovenantSQL
export RUNNING_DIR=$(cd `dirname $0`; pwd)
export LOG_DIR=/data/logs

if [[ ! -d "$LOG_DIR" ]]; then
	echo "$LOG_DIR not exist"
    exit 1
fi

./scripts/clean.sh
./scripts/start.sh
