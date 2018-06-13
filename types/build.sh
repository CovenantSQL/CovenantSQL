#! /usr/bin/env bash

declare -r PB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
protoc -I="$PB_DIR" --go_out="$PB_DIR" "$PB_DIR"/*.proto
