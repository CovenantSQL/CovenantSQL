#!/bin/bash

protoc -I=. --go_out=. *.proto
