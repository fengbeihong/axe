#!/usr/bin/env bash

#protoc -I $GOPATH/src  -I . --swagger_out=logtostderr=true:. echo.proto
protoc -I $GOPATH/src  -I . --go_out=. --go-axe_out=. pb/echo.proto
