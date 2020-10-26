#!/usr/bin/env bash


go install \
    github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway \
    github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger \
    github.com/golang/protobuf/protoc-gen-go \
    github.com/grpc-ecosystem/go-grpc-middleware \
    github.com/grpc-ecosystem/go-grpc-prometheus


dir=$(mktemp -d)
git clone https://github.com/go-swagger/go-swagger "$dir"
cd "$dir"
go install ./cmd/swagger


cd $GOPATH
mkdir -p google/api
curl https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/annotations.proto > google/api/annotations.proto
curl https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/http.proto > google/api/http.proto

