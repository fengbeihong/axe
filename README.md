### 说明

自研的轻量级rpc框架，没有加对多种组件的适配，比如metrics就是prometheus，trace就是jaeger。

主要特点就是把所有可能用到的配置项尽量集中到配置文件，main函数只用少量代码就可以启动服务。

并且，可以同时注册grpc接口和http接口，和grpc-gateway不同，不经过proxy转发，也不需要在proto文件额外定义。

### 安装protoc-gen-go-axe
> 自定义的`protoc-gen-go`插件

`protoc-gen-go-axe`是基于`protoc-gen-go-grpc`v1.1版本修改的，参考 https://github.com/grpc/grpc-go/tree/cmd/protoc-gen-go-grpc%2Fv1.1.0/cmd/protoc-gen-go-grpc

```
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.26
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1

go install github.com/fengbeihong/cmd/protoc-gen-go-axe
```

### 安装其他依赖

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

### 生成pb文件
> 会生成 `xxx.pb.go` `xxx_axe.pb.go` 两种文件
```
cd $GOPATH/src/github.com/fengbeihong/axe/demo/
./gen_pb.sh
```

##### 启动 swagger UI

> 也可以使用 https://editor.swagger.io/

```
cd $GOPATH/src/github.com/fengbeihong/axe/demo
swagger serve --host=0.0.0.0 --port=9000 ./pb/echo.swagger.json
```

##### 启动demo服务
> toml文件规范 https://github.com/LongTengDao/TOML/
```
cd $GOPATH/src/github.com/fengbeihong/axe/demo/
go run main.go
```

##### 测试
> `/EchoService/Echo`是用`protoc-gen-go-axe`工具自动生成的path名称，和`proto`文件里的定义对应
```
curl -XPOST --data '{"value":"testvalue"}' http://localhost:9901/EchoService/Echo
```

### TODO
- [x] 自定义protoc-gen-go工具，可以通过普通的proto文件，除了生成grpc的code之外，还可以生成注册http接口的pattern的code.
- [ ] swagger集成到服务内，只要启动服务，直接访问url即可获取接口描述信息，可以利用pb工具
- [ ] 使用endpoint调用依赖服务时的负载均衡功能

