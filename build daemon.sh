#!/bin/bash
GOPATH_OLD=$GOPATH
PWD=$(pwd)
GOPATH=$PWD:$GOPATH_OLD
cd src/daemon
go build daemon.go
go install daemon.go
cd ../..
echo "build daemon"
go build daemon_demo.go
echo $GOPATH
echo $GOPATH_OLD
echo $PWD
export GOPATH=$GOPATH_OLD
