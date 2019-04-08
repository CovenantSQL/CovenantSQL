#!/bin/bash

if [ -z "$WORKING_DIR" ]; then
    WORKING_DIR=/home/ubuntu/gopath/src/github.com/CovenantSQL/CovenantSQL
fi
if [ -z "$LOG_DIR" ]; then
    LOG_DIR=/data/logs
fi

#Collect logs
cd ${LOG_DIR}
cd ..
rm -rf logs.zip

docker logs covenantsql_bp_0 2> ${LOG_DIR}/covenantsql_bp_0.log
docker logs covenantsql_bp_1 2> ${LOG_DIR}/covenantsql_bp_1.log
docker logs covenantsql_bp_2 2> ${LOG_DIR}/covenantsql_bp_2.log
docker logs covenantsql_miner_0 2> ${LOG_DIR}/covenantsql_miner_0.log
docker logs covenantsql_miner_1 2> ${LOG_DIR}/covenantsql_miner_1.log
docker logs covenantsql_miner_2 2> ${LOG_DIR}/covenantsql_miner_2.log
docker logs covenantsql_miner_3 2> ${LOG_DIR}/covenantsql_miner_3.log

#Clean
killall 500million
killall sar

zip -r logs.zip logs

cd $WORKING_DIR
docker-compose down
sudo git clean -dfx
sudo rm -rf /data/node_miner_0
sudo rm -rf /data/node_miner_1
sudo rm -rf /data/node_miner_2
sudo rm -rf /data/node_miner_3

