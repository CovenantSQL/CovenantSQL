#!/bin/bash
if [ -z "$LOG_DIR" ]; then
    LOG_DIR=/data/logs/mempprof
fi

mkdir -p ${LOG_DIR}

now=`date +%H_%M`

export GOROOT=/home/ubuntu/go
export GOBIN=$GOROOT/bin
export GOPATH=/home/ubuntu/gopath
export PATH=$GOBIN:$PATH

cd ${LOG_DIR}

which go > ${LOG_DIR}/whichgo

go tool pprof -png -inuse_objects http://localhost:11112/debug/pprof/heap > \
    ${LOG_DIR}/${now}_miner_0_objectinuse.png 2>>${LOG_DIR}/error.log
go tool pprof -png http://localhost:11112/debug/pprof/heap > \
    ${LOG_DIR}/${now}_miner_0_heapinuse.png 2>>${LOG_DIR}/error.log
go tool pprof -png http://localhost:11112/debug/pprof/goroutine > \
    ${LOG_DIR}/${now}_miner_0_goroutine.png 2>>${LOG_DIR}/error.log

go tool pprof -png -inuse_objects http://localhost:11113/debug/pprof/heap > ${LOG_DIR}/${now}_miner_1_objectinuse.png
go tool pprof -png http://localhost:11113/debug/pprof/heap > ${LOG_DIR}/${now}_miner_1_heapinuse.png
go tool pprof -png http://localhost:11113/debug/pprof/goroutine > ${LOG_DIR}/${now}_miner_1_goroutine.png

go tool pprof -png -inuse_objects http://localhost:11114/debug/pprof/heap > ${LOG_DIR}/${now}_miner_2_objectinuse.png
go tool pprof -png http://localhost:11114/debug/pprof/heap > ${LOG_DIR}/${now}_miner_2_heapinuse.png
go tool pprof -png http://localhost:11114/debug/pprof/goroutine > ${LOG_DIR}/${now}_miner_2_goroutine.png

go tool pprof -png -inuse_objects http://localhost:11115/debug/pprof/heap > ${LOG_DIR}/${now}_miner_3_objectinuse.png
go tool pprof -png http://localhost:11115/debug/pprof/heap > ${LOG_DIR}/${now}_miner_3_heapinuse.png
go tool pprof -png http://localhost:11115/debug/pprof/goroutine > ${LOG_DIR}/${now}_miner_3_goroutine.png

go tool pprof -png -inuse_objects http://localhost:6061/debug/pprof/heap > ${LOG_DIR}/${now}_client_objectinuse.png
go tool pprof -png http://localhost:6061/debug/pprof/heap > ${LOG_DIR}/${now}_client_heapinuse.png
go tool pprof -png http://localhost:6061/debug/pprof/goroutine > ${LOG_DIR}/${now}_client_goroutine.png
