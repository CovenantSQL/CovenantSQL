#!/bin/bash

protoc -I=. --go_out=../sqlchain sqlchaintypes.proto
