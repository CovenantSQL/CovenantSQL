#!/bin/sh

go test -bench=^BenchmarkMinerTwo$ -benchtime=60s -run ^$
go tool pprof -text miner1.profile
go-torch --width=2400 miner1.profile
