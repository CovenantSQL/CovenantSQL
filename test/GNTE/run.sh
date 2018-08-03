#!/bin/bash

TEST_WD=$(cd $(dirname $0)/; pwd)
PROJECT_DIR=$(cd $TEST_WD/../../; pwd)
echo $PROJECT_DIR

cd $PROJECT_DIR && ./build.sh

if [ -d $TEST_WD/GNTE/scripts/bin ];then
    mv $TEST_WD/GNTE/scripts/bin{,.bak}
fi

cd $PROJECT_DIR && cp ./cleanupDB.sh $TEST_WD/GNTE/scripts
cd $TEST_WD && bash ./GNTE/scripts/cleanupDB.sh
cd $PROJECT_DIR && cp -r ./bin $TEST_WD/GNTE/scripts

cd $TEST_WD && cp -r ./conf/* ./GNTE/scripts

cd $TEST_WD/GNTE && bash -x ./build.sh
cd $TEST_WD/GNTE && bash -x ./generate.sh ./scripts/gnte.yaml
rm -rf $TEST_WD/GNTE/scripts/bin.bak
