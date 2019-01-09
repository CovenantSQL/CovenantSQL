#!/bin/bash

make -C ../../ clean && \
make -C ../../ use_all_cores
export miner_conf_dir=$PWD/../../test/bench_testnet/node_c
go test -bench=^BenchmarkCustomMiner1$ -benchtime=10s -run ^$ |tee custom_miner.log
go test -bench=^BenchmarkCustomMiner2$ -benchtime=10s -run ^$ |tee -a custom_miner.log
go test -bench=^BenchmarkCustomMiner3$ -benchtime=10s -run ^$ |tee -a custom_miner.log

go test -cpu=1 -bench=^BenchmarkCustomMiner1$ -benchtime=10s -run ^$ |tee -a custom_miner.log
go test -cpu=1 -bench=^BenchmarkCustomMiner2$ -benchtime=10s -run ^$ |tee -a custom_miner.log
go test -cpu=1 -bench=^BenchmarkCustomMiner3$ -benchtime=10s -run ^$ |tee -a custom_miner.log
