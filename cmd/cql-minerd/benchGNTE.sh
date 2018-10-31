#!/bin/bash

../../build.sh && \
go test -bench=^BenchmarkMinerGNTE1$ -benchtime=10s -run ^$ && \
go test -bench=^BenchmarkMinerGNTE2$ -benchtime=10s -run ^$ && \
go test -bench=^BenchmarkMinerGNTE3$ -benchtime=10s -run ^$ && \
go test -bench=^BenchmarkMinerGNTE4$ -benchtime=10s -run ^$ && \
go test -bench=^BenchmarkMinerGNTE8$ -benchtime=10s -run ^$
