#!/bin/bash

WORKING_DIR=$(cd $(dirname $0)/; pwd)
echo "1.run runTracker.sh"
echo "2.run runNode2.sh"
echo "you can connect to node2 now"
echo "the node id of node2 is 000005aa62048f85da4ae9698ed59c14ec0d48a88a07c15a32265634e7e64ade"
cd ${WORKING_DIR} && go run main.go ../conf/node1.yaml