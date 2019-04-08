#!/bin/bash
if [ -z "$LOG_DIR" ]; then
    LOG_DIR=/data/logs
fi


go tool pprof -png -inuse_objects http://localhost:11112/debug/pprof/heap > ${LOG_DIR}/miner_0_objectinuse.png
go tool pprof -png http://localhost:11112/debug/pprof/heap > ${LOG_DIR}/miner_0_heapinuse.png

go tool pprof -png -inuse_objects http://localhost:11113/debug/pprof/heap > ${LOG_DIR}/miner_1_objectinuse.png
go tool pprof -png http://localhost:11113/debug/pprof/heap > ${LOG_DIR}/miner_1_heapinuse.png

go tool pprof -png -inuse_objects http://localhost:11114/debug/pprof/heap > ${LOG_DIR}/miner_2_objectinuse.png
go tool pprof -png http://localhost:11114/debug/pprof/heap > ${LOG_DIR}/miner_2_heapinuse.png

go tool pprof -png -inuse_objects http://localhost:11115/debug/pprof/heap > ${LOG_DIR}/miner_3_objectinuse.png
go tool pprof -png http://localhost:11115/debug/pprof/heap > ${LOG_DIR}/miner_3_heapinuse.png

go tool pprof -png -inuse_objects http://localhost:6061/debug/pprof/heap > ${LOG_DIR}/client_objectinuse.png
go tool pprof -png http://localhost:6061/debug/pprof/heap > ${LOG_DIR}/client_heapinuse.png
