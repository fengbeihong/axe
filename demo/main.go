package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"runtime"
	"time"

	pb "github.com/fengbeihong/rpc-go/demo/pb"

	"github.com/fengbeihong/rpc-go/rpc"
)

type echoServer struct {
}

func (s *echoServer) Echo(ctx context.Context, req *pb.EchoRequest) (*pb.EchoResponse, error) {
	return &pb.EchoResponse{Value: req.GetValue()}, nil
}

func EchoWithHttpWrapper(w http.ResponseWriter, req *http.Request) {
	data, err := ioutil.ReadAll(req.Body)
	defer req.Body.Close()
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	var reqData pb.EchoRequest
	if len(data) != 0 {
		err = json.Unmarshal(data, &reqData)
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}
	}
	var es echoServer
	respData, err := es.Echo(context.Background(), &reqData)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	b, err := json.Marshal(respData)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	w.Write(b)
}

/////////////////////////////
// 下面是自定义logger的例子，需要实现rpc.Logger接口，可以对接到不同服务使用的log库上
/////////////////////////////

type MyLogger struct {
}

func (m *MyLogger) Info(format string, args ...interface{}) {
	log.Printf(format, args...)
}

func (m *MyLogger) Error(format string, args ...interface{}) {
	log.Printf(format, args...)
}

/////////////////////////////

func getCurrentFilePath() string {
	_, filePath, _, _ := runtime.Caller(1)
	return filePath
}
func main() {
	cfgPath := path.Join(path.Dir(getCurrentFilePath()), "rpc.toml")

	// grpc server自定义生成
	// TODO http server也自定义生成
	s, _ := rpc.InitRpc(cfgPath, rpc.WithLogger(&MyLogger{}))
	// 也可以使用默认logger
	// s := rpc.InitRpc("./rpc.toml")

	// register rpc
	// 注册rpc server只能传grpc server对象
	// TODO 所以注册http server需要再单独出一个register方法，还是要人手动调用
	pb.RegisterEchoServiceServer(s.GrpcServer(), &echoServer{})

	// TODO 这些代码，应该直接在pb.go里生成，放在http的register方法里
	// TODO wrapper方法，最终调用的是echoServer的Echo
	http.HandleFunc("/echo", EchoWithHttpWrapper)

	// TODO 由上面的http server提供一个类似于grpc server的serve方法
	go http.ListenAndServe(s.HttpAddr(), nil)

	// 调用client的例子
	go clientExample()

	if err := s.Serve(); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func clientExample() {
	time.Sleep(time.Duration(2) * time.Second)
	//clientExampleRpcConsul()
	clientExampleRpcLocal()
	clientExampleHttpLocal()
}

// 通过consul调用启动的rpc服务
func clientExampleRpcConsul() {
	conn, err := rpc.DialService(context.Background(), "rpcservername")
	if err != nil {
		log.Println("clientExampleRpcConsul error: ", err)
		return
	}
	defer conn.Close()
	c := pb.NewEchoServiceClient(conn)

	r, err := c.Echo(context.Background(), &pb.EchoRequest{Value: "call rpc server with consul"})
	if err != nil {
		log.Println("clientExampleRpcConsul error: ", err)
		return
	}
	log.Println("clientExampleRpcConsul succeed, response: ", r.Value)
}

// 通过local配置调用启动的rpc服务
func clientExampleRpcLocal() {
	conn, err := rpc.DialService(context.Background(), "rpcservername_local")
	if err != nil {
		log.Println("clientExampleRpcLocal error: ", err)
		return
	}
	defer conn.Close()
	c := pb.NewEchoServiceClient(conn)

	r, err := c.Echo(context.Background(), &pb.EchoRequest{Value: "call rpc server with local"})
	if err != nil {
		log.Println("clientExampleRpcLocal error: ", err)
		return
	}
	log.Println("clientExampleRpcLocal succeed, response: ", r.Value)
}

// 通过local配置调用http服务
func clientExampleHttpLocal() {
	data := map[string]interface{}{
		"value": "call http server with local",
	}
	bb, _ := json.Marshal(data)
	body := bytes.NewReader(bb)
	b, err := rpc.HttpPost(context.Background(), "rpcservername_http", "/echo", nil, body)
	if err != nil {
		log.Println("clientExampleHttpLocal error: ", err)
		return
	}
	log.Println("clientExampleHttpLocal succeed, response: ", string(b))
}
