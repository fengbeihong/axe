package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"path"
	"runtime"
	"time"

	pb "github.com/fengbeihong/rpc-go/demo/pb"
	"github.com/fengbeihong/rpc-go/rpc"
)

type echoServer struct {
	pb.UnsafeEchoServiceServer
}

func (s *echoServer) Echo(ctx context.Context, req *pb.EchoRequest) (*pb.EchoResponse, error) {
	return &pb.EchoResponse{Value: "echo1_" + req.GetValue()}, nil
}

func (s *echoServer) Echo2(ctx context.Context, req *pb.EchoRequest) (*pb.EchoResponse, error) {
	return &pb.EchoResponse{Value: "echo2_" + req.GetValue()}, nil
}

func middleware1(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("middleware1 - Before Handler")
		next.ServeHTTP(w, r)
		log.Println("middleware1 - After Handler")
	})
}

func middleware2(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("middleware2 - Before Handler")
		next.ServeHTTP(w, r)
		log.Println("middleware2 - After Handler")
	})
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

	// 也可以使用默认logger
	// s, _ := rpc.InitRpc(cfgPath)
	s, err := rpc.NewServer(cfgPath, rpc.WithLogger(&MyLogger{}))
	if err != nil {
		log.Fatalf("failed to new server: %s", err.Error())
	}

	// register rpc
	pb.RegisterEchoServiceServer(s.GrpcServer(), &echoServer{})
	// register http, pattern和handler会自动生成
	pb.RegisterEchoServiceHttpServer(s.HttpServer(), &echoServer{}, []pb.MiddlewareFunc{middleware1, middleware2}...)

	// 调用client的例子
	//go clientExample()

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
