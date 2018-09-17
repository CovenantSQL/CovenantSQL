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

idminer_pkgpath="github.com/CovenantSQL/CovenantSQL/cmd/idminer"
go build -ldflags "-X main.version=${version} ${GOLDFLAGS}"  -o bin/idminer ${idminer_pkgpath}

covenantsqld_pkgpath="github.com/CovenantSQL/CovenantSQL/cmd/covenantsqld"
CGO_ENABLED=1 go build -ldflags "-X main.version=${version} -X github.com/CovenantSQL/CovenantSQL/conf.RoleTag=B ${GOLDFLAGS}" -tags "${platform} sqlite_omit_load_extension" -o bin/covenantsqld ${covenantsqld_pkgpath}
CGO_ENABLED=1 go test -coverpkg github.com/CovenantSQL/CovenantSQL/... -cover -race -c -tags "${platform} sqlite_omit_load_extension testbinary" -ldflags "-X main.version=${version} -X github.com/CovenantSQL/CovenantSQL/conf.RoleTag=B ${GOLDFLAGS}" -o bin/covenantsqld.test ${covenantsqld_pkgpath}

miner_pkgpath="github.com/CovenantSQL/CovenantSQL/cmd/miner"
CGO_ENABLED=1 go build -ldflags "-X main.version=${version} -X github.com/CovenantSQL/CovenantSQL/conf.RoleTag=M ${GOLDFLAGS}" --tags ${platform}" sqlite_omit_load_extension" -o bin/covenantminerd ${miner_pkgpath}
CGO_ENABLED=1 go test -coverpkg github.com/CovenantSQL/CovenantSQL/... -cover -race -c -tags "${platform} sqlite_omit_load_extension testbinary" -ldflags "-X main.version=${version} -X github.com/CovenantSQL/CovenantSQL/conf.RoleTag=M ${GOLDFLAGS}" -o bin/covenantminerd.test ${miner_pkgpath}

observer_pkgpath="github.com/CovenantSQL/CovenantSQL/cmd/observer"
go build -ldflags "-X main.version=${version} -X github.com/CovenantSQL/CovenantSQL/conf.RoleTag=C ${GOLDFLAGS}" -o bin/covenantobserver ${observer_pkgpath}
go test -coverpkg github.com/CovenantSQL/CovenantSQL/... -cover -race -c -tags 'testbinary' -ldflags "-X main.version=${version} -X github.com/CovenantSQL/CovenantSQL/conf.RoleTag=C ${GOLDFLAGS}" -o bin/covenantobserver.test ${observer_pkgpath}

# hotfix_pkgpath="github.com/CovenantSQL/CovenantSQL/cmd/hotfix/msgpack-20180824"
# CGO_ENABLED=1 go build -ldflags "${GOLDFLAGS}" -o bin/hotfix_msgpack_20180824 ${hotfix_pkgpath}

cli_pkgpath="github.com/CovenantSQL/CovenantSQL/cmd/cli"
CGO_ENABLED=1 go build -ldflags "-X main.version=${version} -X github.com/CovenantSQL/CovenantSQL/conf.RoleTag=C ${GOLDFLAGS}" --tags ${platform}" sqlite_omit_load_extension" -o bin/covenantcli ${cli_pkgpath}

#echo "build covenantsqld-linux"
#GOOS=linux GOARCH=amd64   go build -ldflags "-X main.version=${version}"  -o bin/covenantsqld-linux ${pkgpath}

#echo "build covenantsqld-osx"
#GOOS=darwin GOARCH=amd64  go build -ldflags "-X main.version=${version}"  -o bin/covenantsqld-osx ${pkgpath}

#echo "build covenantsqld-windows"
#GOOS=windows GOARCH=386  go build -ldflags "-X main.version=${version}"  -o bin/covenantsqld-win ${pkgpath}

echo "done"

