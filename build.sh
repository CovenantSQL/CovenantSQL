#!/bin/bash -x
set -e

param=$1

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

tags="${platform} sqlite_omit_load_extension"
testtags="${platform} sqlite_omit_load_extension testbinary"

ldflags_role_bp="-X main.version=${version} -X github.com/CovenantSQL/CovenantSQL/conf.RoleTag=B ${GOLDFLAGS}"
ldflags_role_miner="-X main.version=${version} -X github.com/CovenantSQL/CovenantSQL/conf.RoleTag=M ${GOLDFLAGS}"
ldflags_role_client="-X main.version=${version} -X github.com/CovenantSQL/CovenantSQL/conf.RoleTag=C ${GOLDFLAGS}"

test_flags="-coverpkg github.com/CovenantSQL/CovenantSQL/... -cover -race -c"

export CGO_ENABLED=1

cql_utils_pkgpath="github.com/CovenantSQL/CovenantSQL/cmd/cql-utils"
go build -ldflags "-X main.version=${version} ${GOLDFLAGS}"  -o bin/cql-utils ${cql_utils_pkgpath}


build_bp() {
    cqld_pkgpath="github.com/CovenantSQL/CovenantSQL/cmd/cqld"
    go build -ldflags "${ldflags_role_bp}" --tags "${tags}" -o bin/cqld ${cqld_pkgpath}
    go test ${test_flags} -tags "${testtags}" -ldflags "${ldflags_role_bp}" -o bin/cqld.test ${cqld_pkgpath}
}

build_miner() {
    cql_minerd_pkgpath="github.com/CovenantSQL/CovenantSQL/cmd/cql-minerd"
    go build -ldflags "${ldflags_role_miner}" --tags "${tags}" -o bin/cql-minerd ${cql_minerd_pkgpath}
    go test ${test_flags} -tags "${testtags}" -ldflags "${ldflags_role_miner}" -o bin/cql-minerd.test ${cql_minerd_pkgpath}
}


build_client() {
    cql_observer_pkgpath="github.com/CovenantSQL/CovenantSQL/cmd/cql-observer"
    go build -ldflags "${ldflags_role_client}" -o bin/cql-observer ${cql_observer_pkgpath}
    go test ${test_flags} -tags 'testbinary' -ldflags "${ldflags_role_client}" -o bin/cql-observer.test ${cql_observer_pkgpath}
    
    cli_pkgpath="github.com/CovenantSQL/CovenantSQL/cmd/cql"
    go build -ldflags "${ldflags_role_client}" --tags "${tags}" -o bin/cql ${cli_pkgpath}
    
    fuse_pkgpath="github.com/CovenantSQL/CovenantSQL/cmd/cql-fuse"
    go build -ldflags "${ldflags_role_client}" --tags "${tags}" -o bin/cql-fuse ${fuse_pkgpath}
    
    cql_adapter_pkgpath="github.com/CovenantSQL/CovenantSQL/cmd/cql-adapter"
    go build -ldflags "${ldflags_role_client}" --tags "${tags}" -o bin/cql-adapter ${cql_adapter_pkgpath}
    
    cql_faucet_pkgpath="github.com/CovenantSQL/CovenantSQL/cmd/cql-faucet"
    go build -ldflags "${ldflags_role_client}" --tags "${tags}" -o bin/cql-faucet ${cql_faucet_pkgpath}
    
    cql_mysql_adapter_pkgpath="github.com/CovenantSQL/CovenantSQL/cmd/cql-mysql-adapter"
    go build -ldflags "${ldflags_role_client}" --tags "${tags}" -o bin/cql-mysql-adapter ${cql_mysql_adapter_pkgpath}
    
    cql_explorer_pkgpath="github.com/CovenantSQL/CovenantSQL/cmd/cql-explorer"
    go build -ldflags "${ldflags_role_client}" --tags "${tags}" -o bin/cql-explorer ${cql_explorer_pkgpath}
}

case $param in
    "bp")
        build_bp
        ;;
    'miner')
        build_miner
        ;;
    'client')
        build_client
        ;;
    *)
        build_bp
        build_miner
        build_client
        ;;
esac

echo "done"

