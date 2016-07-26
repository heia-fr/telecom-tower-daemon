#!/usr/bin/env bash

APP=telecom-tower-server

mkdir -p /vagrant/_build
cd /vagrant/_build

mkdir -p go/src
mkdir -p go/pkg
mkdir -p go/bin

export GOPATH=$(pwd)/go

go get -d github.com/supcik-go/ws2811
cd $GOPATH/src/github.com/supcik-go/ws2811

git clone https://github.com/jgarff/rpi_ws281x.git

cd rpi_ws281x
patch -i /vagrant/01_cross.patch

export CC=arm-linux-gnueabihf-gcc
export CXX=arm-linux-gnueabihf-g++
export LD=arm-linux-gnueabihf-g++
export AR=arm-linux-gnueabihf-ar
export STRIP=arm-linux-gnueabihf-strip

scons

cd ..
export CGO_ENABLED=1
export GOOS=linux
export GOARCH=arm
export CC_FOR_TARGET=arm-linux-gnueabihf-gcc
export CXX_FOR_TARGET=arm-linux-gnueabihf-g++

export CPATH=$GOPATH/src/github.com/supcik-go/ws2811/rpi_ws281x
export LIBRARY_PATH=$GOPATH/src/github.com/supcik-go/ws2811/rpi_ws281x

sudo -E go install std
patch -i /vagrant/02_go.patch
# go build

cd $GOPATH
mkdir -p src/github.com/heia-fr/${APP}
cp /vagrant/*.go src/github.com/heia-fr/${APP}/

cd $GOPATH/src/github.com/heia-fr/${APP}
go get --tags physical .
go build --tags physical
# go install --tags physical

file ${GOPATH}/src/github.com/heia-fr/${APP}/${APP}
cp ${GOPATH}/src/github.com/heia-fr/${APP}/${APP} /vagrant/${APP}.raspbian-release