package rpc

import (
	"fmt"
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
	cfg *Config
	hs  *httpServer
	gs  *grpcServer
	Log Logger
	Err error
}

type httpServer struct {
	s    *http.Server
	lis  net.Listener
	addr string
	None bool // 标记没有设置port，不想启动http服务时的情况
}

type grpcServer struct {
	s    *grpc.Server
	lis  net.Listener
	addr string
	None bool // 标记没有设置port，不想启动grpc服务时的情况
}

func (s *Server) GrpcAddr() string {
	return fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.GrpcPort)
}

func (s *Server) HttpAddr() string {
	return fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.HttpPort)
}

func initHttpServer(cfg *Config) (*httpServer, error) {
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.HttpPort)
	hs := &httpServer{
		addr: addr,
		s:    &http.Server{},
	}

	if cfg.Server.HttpPort == 0 {
		hs.None = true
		return hs, nil
	}

	l, err := net.Listen("tcp", addr)
	if err != nil {
		hs.None = true
		return hs, err
	}

	hs.lis = l
	return hs, nil
}

func initGrpcServer(cfg *Config) (*grpcServer, error) {
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.GrpcPort)
	gs := &grpcServer{
		addr: addr,
		s:    grpc.NewServer(makeMiddlewareInterceptor(cfg)...),
	}
	if cfg.Server.GrpcPort == 0 {
		gs.None = true
		return gs, nil
	}

	l, err := net.Listen("tcp", addr)
	if err != nil {
		gs.None = true
		return gs, err
	}

	gs.lis = l
	return gs, nil
}

// makeMiddlewareInterceptor Sending unary almost always faster. Use streaming to send big files.
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
	if cfg.RateLimit.Enabled {
		limiter := initRateLimit(cfg)
		siList = append(siList, ratelimit.StreamServerInterceptor(limiter))
		uiList = append(uiList, ratelimit.UnaryServerInterceptor(limiter))
	}

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

func (s *Server) GrpcServer() *grpc.Server {
	return s.gs.s
}

func (s *Server) HttpServer() *http.Server {
	return s.hs.s
}

func (s *Server) Serve(options ...ServeOption) error {
	if s == nil {
		return fmt.Errorf("grpc server is nil")
	}
	if s.gs.None && s.hs.None {
		return fmt.Errorf("both grpc and http server are nil")
	}

	if s.Err != nil {
		return s.Err
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
	if s.cfg.Consul.Enabled {
		if err := registerConsul(s.cfg); err != nil {
			s.Log.Error("register consul failed, error: %s", err.Error())
		}
	}

	if s.cfg.Metrics.Enabled {
		http.Handle("/metrics", promhttp.Handler())
	}

	if s.cfg.Pprof.Port != 0 {
		go func() {
			if err := http.ListenAndServe(fmt.Sprintf(":%d", s.cfg.Pprof.Port), nil); err != nil {
				s.Log.Error("init pprof with port [%d] failed, error: %s", s.cfg.Pprof.Port, err.Error())
			}
		}()
	}

	s.serveGrpc()
	s.serveHttp()

	onExit(onSignal())

	return nil
}

func (s *Server) serveGrpc() {
	if s.gs.None {
		return
	}

	if s.cfg.Metrics.Enabled {
		grpc_prometheus.Register(s.gs.s)
	}

	reflection.Register(s.gs.s)
	go func() {
		err := s.gs.s.Serve(s.gs.lis)
		if err != nil {
			s.Log.Error("grpc server serve failed, error: %s", err.Error())
		}
	}()
}

func (s *Server) serveHttp() {
	if s.hs.None {
		return
	}
	go func() {
		err := s.hs.s.Serve(s.hs.lis)
		if err != nil {
			s.Log.Error("http server serve failed, error: %s", err.Error())
		}
	}()
}

type ServeOption struct {
	f func(*serveOptions)
}

type serveOptions struct {
	// TODO
}
