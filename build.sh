#!/bin/bash -x
set -e

branch=`git rev-parse --abbrev-ref HEAD`
commitid=`git rev-parse --short HEAD`
builddate=`date +%Y%m%d%H%M%S`

function getversion() {
    echo $branch-$commitid-$builddate
}

cd `dirname $0`

version=`getversion`

idminer_pkgpath="gitlab.com/thunderdb/ThunderDB/cmd/idminer"
go build -ldflags "-X main.version=${version}"  -o bin/idminer ${idminer_pkgpath}

thunderdbd_pkgpath="gitlab.com/thunderdb/ThunderDB/cmd/thunderdbd"
go build -ldflags "-X main.version=${version} -X gitlab.com/thunderdb/ThunderDB/conf.RoleTag=B"  -o bin/thunderdbd ${thunderdbd_pkgpath}

miner_pkgpath="gitlab.com/thunderdb/ThunderDB/cmd/miner"
go build -ldflags "-X main.version=${version} -X gitlab.com/thunderdb/ThunderDB/conf.RoleTag=M"  -o bin/thunderminerd ${miner_pkgpath}

#echo "build thunderdbd-linux"
#GOOS=linux GOARCH=amd64   go build -ldflags "-X main.version=${version}"  -o bin/thunderdbd-linux ${pkgpath}

#echo "build thunderdbd-osx"
#GOOS=darwin GOARCH=amd64  go build -ldflags "-X main.version=${version}"  -o bin/thunderdbd-osx ${pkgpath}

#echo "build thunderdbd-windows"
#GOOS=windows GOARCH=386  go build -ldflags "-X main.version=${version}"  -o bin/thunderdbd-win ${pkgpath}

echo "done"

