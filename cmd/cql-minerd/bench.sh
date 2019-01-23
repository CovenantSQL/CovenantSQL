#!/bin/bash

make -C ../../ clean && \
make -C ../../ use_all_cores && \
go test -bench=^BenchmarkSQLite$ -benchtime=10s -run ^$ && \
go test -bench=^BenchmarkMinerOne$ -benchtime=10s -run ^$ && \
go test -bench=^BenchmarkMinerOneNoSign$ -benchtime=10s -run ^$ && \
go test -bench=^BenchmarkMinerTwo$ -benchtime=10s -run ^$ && \
go test -bench=^BenchmarkMinerTwoNoSign$ -benchtime=10s -run ^$ && \
go test -bench=^BenchmarkMinerThree$ -benchtime=10s -run ^$ && \
go test -bench=^BenchmarkMinerThreeNoSign$ -benchtime=10s -run ^$ && \
go test -bench=^BenchmarkMinerOneWithEventualConsistency$ -benchtime=10s -run ^$ && \
go test -bench=^BenchmarkMinerOneNoSignWithEventualConsistency$ -benchtime=10s -run ^$ && \
go test -bench=^BenchmarkMinerTwoWithEventualConsistency$ -benchtime=10s -run ^$ && \
go test -bench=^BenchmarkMinerTwoNoSignWithEventualConsistency$ -benchtime=10s -run ^$ && \
go test -bench=^BenchmarkMinerThreeWithEventualConsistency$ -benchtime=10s -run ^$ && \
go test -bench=^BenchmarkMinerThreeNoSignWithEventualConsistency$ -benchtime=10s -run ^$
