#!/usr/bin/env bash

protoc -I $GOPATH/src  -I . --swagger_out=logtostderr=true:. echo.proto
protoc -I $GOPATH/src -I . --grpc-gateway_out=logtostderr=true:. echo.proto
protoc -I $GOPATH/src -I . --go_out=plugins=grpc:. echo.proto
