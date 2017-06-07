#!/bin/bash
cd `dirname $0`

ROOT_DIR=`pwd`

mygopath=$(dirname $(dirname $ROOT_DIR))
sysgopath=$GOPATH

if [ "$sysgopath" != "" ]; then
export GOPATH=$mygopath:$sysgopath
else
export GOPATH=$mygopath
fi

echo "GOPATH:$GOPATH"

APPNAME=app.bin

if [ "$1" != "" ]; then
    APPNAME=$1.app.bin
fi

go build -o bin/$APPNAME

if [ $? != 0 ]; then
    exit 1
fi

export GOPATH=$sysgopath
