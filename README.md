## 说明

### 安装依赖

##### grpc-gateway
```
go install \
    github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway \
    github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger \
    github.com/golang/protobuf/protoc-gen-go
```

##### swagger
```
dir=$(mktemp -d) 
git clone https://github.com/go-swagger/go-swagger "$dir" 
cd "$dir"
go install ./cmd/swagger
```

##### annotation
```
cd $GOPATH
mkdir -p google/api    
curl https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/annotations.proto > google/api/annotations.proto     
curl https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/http.proto > google/api/http.proto
```

##### 其他一些golang库，如果已经有可以忽略
```
mkdir -p $GOPATH/golang.org/x
Cd $GOPATH/golang.org/x
Git clone https://github.com/golang/net
Git clone https://github.com/golang/text
Git clone https://github.com/golang/sys
```

### 执行生成命令
> 会生成 `.pb.go` `.pb.gw.go` `.swagger.json` 三种文件
```
PB_PATH=$GOPATH/src/github.com/fengbeihong/rpc-go/demo/pb
cd $PB_PATH

protoc -I $GOPATH/src -I $PB_PATH/ --go_out=plugins=grpc:$PB_PATH/ $PB_PATH/echo.proto
protoc -I $GOPATH/src -I $PB_PATH/ --grpc-gateway_out=logtostderr=true:$PB_PATH/ $PB_PATH/echo.proto
protoc -I $GOPATH/src -I $PB_PATH/ --swagger_out=logtostderr=true:$PB_PATH/ $PB_PATH/echo.proto
```

##### 启动 swagger UI

> 也可以使用 https://editor.swagger.io/

```
cd $GOPATH/src/github.com/fengbeihong/rpc-go/demo/pb
swagger serve --host=0.0.0.0 --port=9000 ./echo.swagger.json
```

##### 启动demo服务
> toml文件规范 https://github.com/LongTengDao/TOML/
```
cd $GOPATH/src/github.com/fengbeihong/rpc-go/demo/
go run main.go
```

##### 测试
```
curl -XPOST --data '{"value":"testvalue"}' http://localhost:9901/v1/example/echo
```