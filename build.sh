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

tags="${platform} sqlite_omit_load_extension"
testtags="${platform} sqlite_omit_load_extension testbinary"

ldflags_role_bp="-X main.version=${version} -X github.com/CovenantSQL/CovenantSQL/conf.RoleTag=B ${GOLDFLAGS}"
ldflags_role_miner="-X main.version=${version} -X github.com/CovenantSQL/CovenantSQL/conf.RoleTag=M ${GOLDFLAGS}"
ldflags_role_client="-X main.version=${version} -X github.com/CovenantSQL/CovenantSQL/conf.RoleTag=C ${GOLDFLAGS}"

test_flags="-coverpkg github.com/CovenantSQL/CovenantSQL/... -cover -race -c"

cql_utils_pkgpath="github.com/CovenantSQL/CovenantSQL/cmd/cql-utils"
go build -ldflags "-X main.version=${version} ${GOLDFLAGS}"  -o bin/cql-utils ${cql_utils_pkgpath}

cqld_pkgpath="github.com/CovenantSQL/CovenantSQL/cmd/cqld"
CGO_ENABLED=1 go build -ldflags "${ldflags_role_bp}" --tags "${tags}" -o bin/cqld ${cqld_pkgpath}
CGO_ENABLED=1 go test ${test_flags} -tags "${testtags}" -ldflags "${ldflags_role_bp}" -o bin/cqld.test ${cqld_pkgpath}

cql_minerd_pkgpath="github.com/CovenantSQL/CovenantSQL/cmd/cql-minerd"
CGO_ENABLED=1 go build -ldflags "${ldflags_role_miner}" --tags "${tags}" -o bin/cql-minerd ${cql_minerd_pkgpath}
CGO_ENABLED=1 go test ${test_flags} -tags "${testtags}" -ldflags "${ldflags_role_miner}" -o bin/cql-minerd.test ${cql_minerd_pkgpath}

cql_observer_pkgpath="github.com/CovenantSQL/CovenantSQL/cmd/cql-observer"
go build -ldflags "${ldflags_role_client}" -o bin/cql-observer ${cql_observer_pkgpath}
go test ${test_flags} -tags 'testbinary' -ldflags "${ldflags_role_client}" -o bin/cql-observer.test ${cql_observer_pkgpath}

cli_pkgpath="github.com/CovenantSQL/CovenantSQL/cmd/cql"
CGO_ENABLED=1 go build -ldflags "${ldflags_role_client}" --tags "${tags}" -o bin/cql ${cli_pkgpath}

fuse_pkgpath="github.com/CovenantSQL/CovenantSQL/cmd/cql-fuse"
CGO_ENABLED=1 go build -ldflags "${ldflags_role_client}" --tags "${tags}" -o bin/cql-fuse ${fuse_pkgpath}

cql_adapter_pkgpath="github.com/CovenantSQL/CovenantSQL/cmd/cql-adapter"
CGO_ENABLED=1 go build -ldflags "${ldflags_role_client}" --tags "${tags}" -o bin/cql-adapter ${cql_adapter_pkgpath}

cql_faucet_pkgpath="github.com/CovenantSQL/CovenantSQL/cmd/cql-faucet"
CGO_ENABLED=1 go build -ldflags "${ldflags_role_client}" --tags "${tags}" -o bin/cql-faucet ${cql_faucet_pkgpath}

cql_mysql_adapter_pkgpath="github.com/CovenantSQL/CovenantSQL/cmd/cql-mysql-adapter"
CGO_ENABLED=1 go build -ldflags "${ldflags_role_client}" --tags "${tags}" -o bin/cql-mysql-adapter ${cql_mysql_adapter_pkgpath}

cql_explorer_pkgpath="github.com/CovenantSQL/CovenantSQL/cmd/cql-explorer"
CGO_ENABLED=1 go build -ldflags "${ldflags_role_client}" --tags "${tags}" -o bin/cql-explorer ${cql_explorer_pkgpath}

echo "done"

