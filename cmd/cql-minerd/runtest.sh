#!/usr/bin/env bash

root=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
cd "${root}/../../"
make bin/cql-minerd.test
make bin/cqld.test
./cleanupDB.sh
go test -run TestFullProcess -v github.com/CovenantSQL/CovenantSQL/cmd/cql-minerd
gocovmerge $(find cmd/cql-minerd/ -name "*.cover.out") | grep -F -v '_gen.go' > coverage.txt && rm -f cover.out
go tool cover -html coverage.txt
