#!/usr/bin/env bash

APP=telecom-tower-server
REPO=github.com/heia-fr
BASE=/vagrant
APP_SUFFIX=.raspbian-release

cd ${BASE}/_build
export GOPATH=$(pwd)/go

export CC=arm-linux-gnueabihf-gcc
export CXX=arm-linux-gnueabihf-g++
export LD=arm-linux-gnueabihf-g++
export AR=arm-linux-gnueabihf-ar
export STRIP=arm-linux-gnueabihf-strip

export CGO_ENABLED=1
export GOOS=linux
export GOARCH=arm
export CC_FOR_TARGET=arm-linux-gnueabihf-gcc
export CXX_FOR_TARGET=arm-linux-gnueabihf-g++

export CPATH=${GOPATH}/src/github.com/supcik-go/ws2811/rpi_ws281x
export LIBRARY_PATH=${GOPATH}/src/github.com/supcik-go/ws2811/rpi_ws281x

cd ${GOPATH}
cp ${BASE}/*.go src/${REPO}/${APP}/

cd $GOPATH/src/${REPO}/${APP}
go get --tags physical .
go build --tags physical
# go install --tags physical

cp ${GOPATH}/src/${REPO}/${APP}/${APP} ${BASE}/${APP}${APP_SUFFIX}
file ${BASE}/${APP}${APP_SUFFIX}