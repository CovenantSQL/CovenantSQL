#!/usr/bin/env bash

PROJECT_DIR=$(cd $(dirname $0)/; pwd)

echo "install HashStablePack cmd: hsp"
go get -v -u github.com/CovenantSQL/HashStablePack/hsp

echo ${PROJECT_DIR}

cd ${PROJECT_DIR} && go generate ./...

#cd ${PROJECT_DIR} && hsp -file proto
#cd ${PROJECT_DIR} && hsp -file blockproducer/types
#cd ${PROJECT_DIR} && hsp -file worker/types
#cd ${PROJECT_DIR} && hsp -file sqlchain/types
#cd ${PROJECT_DIR} && hsp -file kayak/types.go
