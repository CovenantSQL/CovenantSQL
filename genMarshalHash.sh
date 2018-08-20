#!/usr/bin/env bash

PROJECT_DIR=$(cd $(dirname $0)/; pwd)

if [[ -x HashStablePack ]]; then
    echo "install HashStablePack"
    go get -u github.com/CovenantSQL/HashStablePack
fi

echo ${PROJECT_DIR}

cd ${PROJECT_DIR} && go generate ./...

#cd ${PROJECT_DIR} && HashStablePack -file proto
#cd ${PROJECT_DIR} && HashStablePack -file blockproducer/types
#cd ${PROJECT_DIR} && HashStablePack -file worker/types
#cd ${PROJECT_DIR} && HashStablePack -file sqlchain/types
#cd ${PROJECT_DIR} && HashStablePack -file kayak/types.go
