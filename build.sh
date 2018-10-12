#!/bin/bash -x
set -e

branch=`git rev-parse --abbrev-ref HEAD`
commitid=`git rev-parse --short HEAD`
builddate=`date +%Y%m%d%H%M%S`

platform=''
unamestr=`uname`
if [[ "$unamestr" == 'Linux' ]]; then
   platform='linux'
fi

function getversion() {
    echo $branch-$commitid-$builddate
}

cd `dirname $0`

version=`getversion`

cql_utils_pkgpath="github.com/CovenantSQL/CovenantSQL/cmd/cql-utils"
go build -ldflags "-X main.version=${version} ${GOLDFLAGS}"  -o bin/cql-utils ${cql_utils_pkgpath}

cqld_pkgpath="github.com/CovenantSQL/CovenantSQL/cmd/cqld"
CGO_ENABLED=1 go build -ldflags "-X main.version=${version} -X github.com/CovenantSQL/CovenantSQL/conf.RoleTag=B ${GOLDFLAGS}" -tags "${platform} sqlite_omit_load_extension" -o bin/cqld ${cqld_pkgpath}
CGO_ENABLED=1 go test -coverpkg github.com/CovenantSQL/CovenantSQL/... -cover -race -c -tags "${platform} sqlite_omit_load_extension testbinary" -ldflags "-X main.version=${version} -X github.com/CovenantSQL/CovenantSQL/conf.RoleTag=B ${GOLDFLAGS}" -o bin/cqld.test ${cqld_pkgpath}

cql_miner_pkgpath="github.com/CovenantSQL/CovenantSQL/cmd/cql-miner"
CGO_ENABLED=1 go build -ldflags "-X main.version=${version} -X github.com/CovenantSQL/CovenantSQL/conf.RoleTag=M ${GOLDFLAGS}" --tags ${platform}" sqlite_omit_load_extension" -o bin/cql-miner ${cql_miner_pkgpath}
CGO_ENABLED=1 go test -coverpkg github.com/CovenantSQL/CovenantSQL/... -cover -race -c -tags "${platform} sqlite_omit_load_extension testbinary" -ldflags "-X main.version=${version} -X github.com/CovenantSQL/CovenantSQL/conf.RoleTag=M ${GOLDFLAGS}" -o bin/cql-miner.test ${cql_miner_pkgpath}

cql_observer_pkgpath="github.com/CovenantSQL/CovenantSQL/cmd/cql-observer"
go build -ldflags "-X main.version=${version} -X github.com/CovenantSQL/CovenantSQL/conf.RoleTag=C ${GOLDFLAGS}" -o bin/covenantcql-observer ${cql_observer_pkgpath}
go test -coverpkg github.com/CovenantSQL/CovenantSQL/... -cover -race -c -tags 'testbinary' -ldflags "-X main.version=${version} -X github.com/CovenantSQL/CovenantSQL/conf.RoleTag=C ${GOLDFLAGS}" -o bin/covenantcql-observer.test ${cql_observer_pkgpath}

# hotfix_pkgpath="github.com/CovenantSQL/CovenantSQL/cmd/hotfix/msgpack-20180824"
# CGO_ENABLED=1 go build -ldflags "${GOLDFLAGS}" -o bin/hotfix_msgpack_20180824 ${hotfix_pkgpath}

cli_pkgpath="github.com/CovenantSQL/CovenantSQL/cmd/cql"
CGO_ENABLED=1 go build -ldflags "-X main.version=${version} -X github.com/CovenantSQL/CovenantSQL/conf.RoleTag=C ${GOLDFLAGS}" --tags ${platform}" sqlite_omit_load_extension" -o bin/cql ${cli_pkgpath}

cql_adapter_pkgpath="github.com/CovenantSQL/CovenantSQL/cmd/cql-adapter"
CGO_ENABLED=1 go build -ldflags "-X main.version=${version} -X github.com/CovenantSQL/CovenantSQL/conf.RoleTag=C ${GOLDFLAGS}" --tags ${platform}" sqlite_omit_load_extension" -o bin/cql-adapter ${cql_adapter_pkgpath}

cql_faucet_pkgpath="github.com/CovenantSQL/CovenantSQL/cmd/cql-faucet"
CGO_ENABLED=1 go build -ldflags "-X main.version=${version} -X github.com/CovenantSQL/CovenantSQL/conf.RoleTag=C ${GOLDFLAGS}" --tags ${platform}" sqlite_omit_load_extension" -o bin/cql-faucet ${cql_faucet_pkgpath}

echo "done"

