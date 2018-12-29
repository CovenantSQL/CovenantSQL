#!/bin/sh -x

../../cleanupDB.sh
make -C ../../ clean
make -C ../../ use_all_cores

go test -bench=^BenchmarkMinerTwo$ -benchtime=15s -run ^$
go tool pprof -text miner1.profile > pprof.txt
go tool pprof -svg miner1.profile > tree.svg
go-torch -t 180 --width=2400 miner1.profile
