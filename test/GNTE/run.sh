#!/bin/sh

TEST_WD=$(cd $(dirname $0)/; pwd)
PROJECT_DIR=$(cd $TEST_WD/../../; pwd)
echo $PROJECT_DIR

cd $PROJECT_DIR && ./build.sh && \
    cp -r ./bin $TEST_WD/GNTE/scripts && \
    cp -r ./conf/* $TEST_WD/GNTE/scripts

