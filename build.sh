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

idminer_pkgpath="gitlab.com/thunderdb/ThunderDB/cmd/idminer"
go build -ldflags "-X main.version=${version} ${GOLDFLAGS}"  -o bin/idminer ${idminer_pkgpath}

thunderdbd_pkgpath="gitlab.com/thunderdb/ThunderDB/cmd/thunderdbd"
CGO_ENABLED=1 go build -ldflags "-X main.version=${version} -X gitlab.com/thunderdb/ThunderDB/conf.RoleTag=B ${GOLDFLAGS}" --tags "${platform} sqlite_omit_load_extension" -o bin/thunderdbd ${thunderdbd_pkgpath}

miner_pkgpath="gitlab.com/thunderdb/ThunderDB/cmd/miner"
CGO_ENABLED=1 go build -ldflags "-X main.version=${version} -X gitlab.com/thunderdb/ThunderDB/conf.RoleTag=M ${GOLDFLAGS}" --tags ${platform}" sqlite_omit_load_extension" -o bin/thunderminerd ${miner_pkgpath}

#echo "build thunderdbd-linux"
#GOOS=linux GOARCH=amd64   go build -ldflags "-X main.version=${version}"  -o bin/thunderdbd-linux ${pkgpath}

#echo "build thunderdbd-osx"
#GOOS=darwin GOARCH=amd64  go build -ldflags "-X main.version=${version}"  -o bin/thunderdbd-osx ${pkgpath}

#echo "build thunderdbd-windows"
#GOOS=windows GOARCH=386  go build -ldflags "-X main.version=${version}"  -o bin/thunderdbd-win ${pkgpath}

echo "done"

