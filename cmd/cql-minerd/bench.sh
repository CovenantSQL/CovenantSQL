#!/bin/bash

make -C ../../ && \
go test -bench=^BenchmarkSQLite$ -benchtime=10s -run ^$ && \
go test -bench=^BenchmarkMinerOne$ -benchtime=10s -run ^$ && \
go test -bench=^BenchmarkMinerOneNoSign$ -benchtime=10s -run ^$ && \
go test -bench=^BenchmarkMinerTwo$ -benchtime=10s -run ^$ && \
go test -bench=^BenchmarkMinerTwoNoSign$ -benchtime=10s -run ^$ && \
go test -bench=^BenchmarkMinerThree$ -benchtime=10s -run ^$ && \
go test -bench=^BenchmarkMinerThreeNoSign$ -benchtime=10s -run ^$
