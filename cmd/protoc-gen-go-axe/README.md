# protoc-gen-go-axe

## installation
```
go install github.com/fengbeihong/axe/cmd/protoc-gen-go-axe
```

## generate
```
protoc -I $GOPATH/src  -I . --go_out=. --go-axe_out=. pb/echo.proto
```

You will get two files
```
echo.pb.go
echo_axe.pb.go
```