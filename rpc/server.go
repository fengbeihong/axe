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
	cfg         *Config
	server      *grpc.Server
	Log         Logger
	Err         error
}

func initGrpcServer(cfg *Config) *grpc.Server {
	opts := makeMiddlewareInterceptor(cfg)
	return grpc.NewServer(opts...)
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

func (s *Server) GrpcAddr() string {
	return fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.Port)
}

func (s *Server) HttpAddr() string {
	return fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.HttpPort)
}

func (s *Server) GrpcServer() *grpc.Server {
	return s.server
}

func (s *Server) Serve(options ...ServeOption) error {
	if s == nil || s.server == nil {
		return fmt.Errorf("grpc server is nil")
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

	// init listener
	lis, err := net.Listen("tcp", s.GrpcAddr())
	if err != nil {
		s.Log.Error("grpc listen with addr [%s] failed, error: ", err.Error())
		return err
	}

	// 只有非dev环境，并且开关打开，才执行consul的注册和metrics的监听
	if s.cfg.Consul.Enabled {
		if err := registerConsul(s.cfg); err != nil {
			s.Log.Error("register consul failed, error: %s", err.Error())
		}
	}

	if s.cfg.Metrics.Enabled {
		grpc_prometheus.Register(s.server)
		http.Handle("/metrics", promhttp.Handler())
	}

	if s.cfg.Pprof.Port != 0 {
		go func() {
			if err := http.ListenAndServe(fmt.Sprintf(":%d", s.cfg.Pprof.Port), nil); err != nil {
				s.Log.Error("init pprof with port [%d] failed, error: %s", s.cfg.Pprof.Port, err.Error())
			}
		}()
	}

	reflection.Register(s.server)
	go func() {
		err := s.server.Serve(lis)
		if err != nil {
			s.Log.Error("grpc server serve failed, error: %s", err.Error())
		}
	}()

	onExit(onSignal())

	return nil
}

type ServeOption struct {
	f func(*serveOptions)
}

type serveOptions struct {
	// TODO
}
