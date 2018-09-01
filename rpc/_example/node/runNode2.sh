#!/bin/bash

WORKING_DIR=$(cd $(dirname $0)/; pwd)
cd ${WORKING_DIR} && go run main.go ../conf/node2.yaml