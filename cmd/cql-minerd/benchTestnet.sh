#!/bin/bash

make -C ../../ clean && \
make -C ../../ use_all_cores
go test -bench=^BenchmarkTestnetMiner1$ -benchtime=10s -run ^$ |tee testnet.log
go test -bench=^BenchmarkTestnetMiner2$ -benchtime=10s -run ^$ |tee -a testnet.log
go test -bench=^BenchmarkTestnetMiner3$ -benchtime=10s -run ^$ |tee -a testnet.log

go test -cpu=1 -bench=^BenchmarkTestnetMiner1$ -benchtime=10s -run ^$ |tee -a testnet.log
go test -cpu=1 -bench=^BenchmarkTestnetMiner2$ -benchtime=10s -run ^$ |tee -a testnet.log
go test -cpu=1 -bench=^BenchmarkTestnetMiner3$ -benchtime=10s -run ^$ |tee -a testnet.log
