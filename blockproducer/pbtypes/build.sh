#!/bin/bash

protoc -I=. --go_out=. blockproducertypes.proto
