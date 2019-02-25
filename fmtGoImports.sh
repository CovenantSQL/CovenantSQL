#!/usr/bin/env bash

PROJECT_DIR=$(cd $(dirname $0)/; pwd)
cd "$PROJECT_DIR" &&
    find . -type f -name '*.go' ! -path './vendor/*' ! -path './test/GNTE/GNTE/*' \
        -exec goimports -w -local 'github.com/CovenantSQL/CovenantSQL' {} +
