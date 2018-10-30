#!/bin/bash

../../build.sh && \
go test -bench=^BenchmarkMinerGNTETwo$ -benchtime=10s -run ^$
