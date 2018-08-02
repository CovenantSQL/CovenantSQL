#!/bin/sh

TEST_WD=$(cd $(dirname $0)/; pwd)
PROJECT_DIR=$(cd $TEST_WD/../../; pwd)
echo $PROJECT_DIR

cd $PROJECT_DIR && ./build.sh && \
    cp -r ./bin $TEST_WD/GNTE/scripts

cd $TEST_WD && cp -r ./conf/* ./GNTE/scripts

cd $TEST_WD/GNTE && bash -x ./build.sh
cd $TEST_WD/GNTE && bash -x ./generate.sh ./scripts/gnte.yaml

