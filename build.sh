#!/bin/bash
set -e

branch=`git rev-parse --abbrev-ref HEAD`
commitid=`git rev-parse --short HEAD`
builddate=`date +%Y%m%d-%H%M%S`

function getversion() {
    echo $branch-$commitid-$builddate
}

cd `dirname $0`

version=`getversion`
pkgpath="github.com/thunderdb/ThunderDB/cmd/thunderdbd"

go build -ldflags "-X main.version=${version}"  -o bin/thunderdbd ${pkgpath}

#echo "build thunderdbd-linux"
#GOOS=linux GOARCH=amd64   go build -ldflags "-X main.version=${version}"  -o bin/thunderdbd-linux ${pkgpath}

#echo "build thunderdbd-osx"
#GOOS=darwin GOARCH=amd64  go build -ldflags "-X main.version=${version}"  -o bin/thunderdbd-osx ${pkgpath}

#echo "build thunderdbd-windows"
#GOOS=windows GOARCH=386  go build -ldflags "-X main.version=${version}"  -o bin/thunderdbd-win ${pkgpath}

echo "done"

