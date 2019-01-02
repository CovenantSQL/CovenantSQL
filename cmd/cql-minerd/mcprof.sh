#!/bin/sh -x

../../cleanupDB.sh
make -C ../../ clean
make -C ../../ use_all_cores

(go test -bench=^BenchmarkMinerTwo$ -benchtime=40s -run ^$ &) && \
 (sleep 25 && DSN=`cat .dsn` go test '-bench=^BenchmarkClientOnly$' -benchtime=20s -run '^$')

