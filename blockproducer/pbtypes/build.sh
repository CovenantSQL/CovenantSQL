#!/bin/bash

protoc -I=. --go_out=. sqlchaintypes.proto
