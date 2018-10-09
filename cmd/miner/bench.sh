#!/bin/bash

../../build.sh && \
go test -bench=^BenchmarkSingleMiner -benchtime=10s -run ^$
