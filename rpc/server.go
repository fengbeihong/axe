package rpc

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/ratelimit"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type Server struct {
	conf         *Config
	server       *grpc.Server
	listenerGrpc ListenerWrap
	listenerHttp ListenerWrap
}

type ListenerWrap struct {
	Type     string
	Listener net.Listener
	Error    error
	Addr     string
}

func initServer(cfg *Config) *Server {
	server := &Server{
		conf:         cfg,
		listenerGrpc: ListenerWrap{Type: "grpc"},
		listenerHttp: ListenerWrap{Type: "http"},
	}

	// grpc
	server.listenerGrpc.Addr = fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server.listenerGrpc.Listener, server.listenerGrpc.Error = net.Listen("tcp", server.listenerGrpc.Addr)
	if server.listenerGrpc.Error != nil {
		return server
	}
	log.Printf("start rpc server, service_name: %s, address: %s", cfg.Server.ServiceName, server.listenerGrpc.Addr)

	// http
	server.listenerHttp.Addr = fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.HttpPort)
	if hasSetHttpPort(cfg) { // 0是没有配置http_port，可以不启动，非0则认为必须启动成功http端口
		server.listenerHttp.Listener, server.listenerHttp.Error = net.Listen("tcp", server.listenerHttp.Addr)
		if server.listenerHttp.Error != nil {
			return server
		}
		log.Printf("start http server, service_name: %s, address: %s", cfg.Server.ServiceName, server.listenerHttp.Addr)
	}

	serverOption := makeMiddlewareInterceptor(cfg)
	server.server = grpc.NewServer(serverOption...)

	return server
}

func makeMiddlewareInterceptor(cfg *Config) []grpc.ServerOption {
	var siList []grpc.StreamServerInterceptor
	var uiList []grpc.UnaryServerInterceptor

	// metrics
	if cfg.Metrics.Enabled {
		siList = append(siList, grpc_prometheus.StreamServerInterceptor)
		uiList = append(uiList, grpc_prometheus.UnaryServerInterceptor)
	}

	// trace
	if cfg.Trace.Enabled {
		t := initJaeger(cfg)
		opts := []grpc_opentracing.Option{
			grpc_opentracing.WithTracer(t),
		}
		siList = append(siList, grpc_opentracing.StreamServerInterceptor(opts...))
		uiList = append(uiList, grpc_opentracing.UnaryServerInterceptor(opts...))
	}

	// rate limit
	limiter := &alwaysPassLimiter{}
	siList = append(siList, ratelimit.StreamServerInterceptor(limiter))
	uiList = append(uiList, ratelimit.UnaryServerInterceptor(limiter))

	// panic recovery
	opts := []grpc_recovery.Option{
		grpc_recovery.WithRecoveryHandler(func(p interface{}) (err error) {
			return status.Errorf(codes.Unknown, "panic triggered: %v", p)
		}),
	}
	siList = append(siList, grpc_recovery.StreamServerInterceptor(opts...))
	uiList = append(uiList, grpc_recovery.UnaryServerInterceptor(opts...))

	return []grpc.ServerOption{
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(siList...)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(uiList...)),
	}
}

func (s *Server) GrpcServerAddr() string {
	return s.listenerGrpc.Addr
}

func (s *Server) HttpServerAddr() string {
	return s.listenerHttp.Addr
}

func (s *Server) GrpcListener() net.Listener {
	return s.listenerGrpc.Listener
}

func (s *Server) HttpListener() net.Listener {
	return s.listenerHttp.Listener
}

func (s *Server) GrpcServer() *grpc.Server {
	return s.server
}

func (s *Server) Error() error {
	if s.listenerHttp.Error != nil || s.listenerGrpc.Error != nil {
		return s.handleError()
	}
	return nil
}

func hasSetHttpPort(cfg *Config) bool {
	return cfg.Server.HttpPort != 0
}

func (s *Server) handleError() error {
	errMsg := "grpc server is nil, %s net.Listen failed, %s addr:[%s], error:%s"
	var lw ListenerWrap
	if hasSetHttpPort(s.conf) {
		// 没有设置http port，
		lw = s.listenerGrpc
	} else {
		if s.listenerGrpc.Error != nil {
			lw = s.listenerGrpc
		}
		if s.listenerHttp.Error != nil {
			lw = s.listenerHttp
		}
	}
	errMsg = fmt.Sprintf(errMsg, lw.Type, lw.Type, lw.Addr, lw.Error)
	return errors.New(errMsg)
}

func (s *Server) Serve(options ...ServeOption) error {
	if s.server == nil {
		return s.handleError()
	}

	do := serveOptions{}
	for _, option := range options {
		option.f(&do)
	}

	defer func() {
		if GlobalTraceCloser != nil {
			GlobalTraceCloser.Close()
		}
	}()

	// 只有非dev环境，并且开关打开，才执行consul的注册和metrics的监听
	if s.conf.Consul.Enabled {
		registerConsul(s.conf)
	}

	if s.conf.Metrics.Enabled {
		grpc_prometheus.Register(s.server)
		http.Handle("/metrics", promhttp.Handler())
	}

	if s.conf.Pprof.Port != 0 {
		go http.ListenAndServe(fmt.Sprintf(":%d", s.conf.Pprof.Port), nil)
	}

	reflection.Register(s.server)
	go func() {
		err := s.server.Serve(s.listenerGrpc.Listener)
		if err != nil {
			log.Println("grpc serve failed, error:", err.Error())
		}
	}()

	onExit(onSignal())

	return nil
}
