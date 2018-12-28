#!/bin/bash -x
set -e

param=$1

# TODO if not make
# TODO detect cpu

cd `dirname $0`

case $param in
    "bp")
        make bp
        ;;
    'miner')
        make miner
        ;;
    'client')
        make client
        ;;
    'observer')
        make observer
        ;;
    *)
        make
        ;;
esac

echo "done"

