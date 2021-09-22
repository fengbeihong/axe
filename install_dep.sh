#!/usr/bin/env bash

go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.26
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1


dir=$(mktemp -d)
git clone https://github.com/go-swagger/go-swagger "$dir"
cd "$dir"
go install ./cmd/swagger


cd $GOPATH
mkdir -p google/api
curl https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/annotations.proto > google/api/annotations.proto
curl https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/http.proto > google/api/http.proto

