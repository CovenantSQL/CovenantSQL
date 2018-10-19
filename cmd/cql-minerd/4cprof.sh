#!/bin/sh -x

../../cleanupDB.sh
../../build.sh

(go test -bench=^BenchmarkMinerTwo$ -benchtime=80s -run ^$ &) && \
 (sleep 20 && DSN=`cat .dsn` go test '-bench=^BenchmarkClientOnly$' -benchtime=60s -run '^$' &) && \
 (sleep 20 && DSN=`cat .dsn` go test '-bench=^BenchmarkClientOnly$' -benchtime=60s -run '^$' &) && \
 (sleep 20 && DSN=`cat .dsn` go test '-bench=^BenchmarkClientOnly$' -benchtime=60s -run '^$' &) && \
 (sleep 20 && DSN=`cat .dsn` go test '-bench=^BenchmarkClientOnly$' -benchtime=60s -run '^$')

