package rpc

import (
	"fmt"
	"log"
	"net"
	"net/http"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"

	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/grpc-ecosystem/go-grpc-middleware/ratelimit"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc/reflection"

	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"

	"google.golang.org/grpc"
)

type Server struct {
	conf     *Config
	server   *grpc.Server
	listener net.Listener
	grpcAddr string
	logger   Logger
}

func initServer(cfg *Config) *Server {
	tcpAddr := fmt.Sprintf("%s:%d", cfg.Server.Address, cfg.Server.Port)
	l, err := net.Listen("tcp", tcpAddr)
	if err != nil {
		log.Fatalf("initServer failed, failed to listen: %v", err)
	}
	log.Printf("start rpc server, service_name: %s, address: %s", cfg.Server.ServiceName, tcpAddr)

	serverOption := makeMiddlewareInterceptor(cfg)
	s := grpc.NewServer(serverOption...)

	server := &Server{
		conf:     cfg,
		server:   s,
		listener: l,
		grpcAddr: tcpAddr,
		logger:   GlobalLogger,
	}

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

func (s *Server) GrpcServerEndpoint() string {
	return s.grpcAddr
}

func (s *Server) GrpcServer() *grpc.Server {
	return s.server
}

func (s *Server) ServeHttp(handler http.Handler) {
	if s.conf.Server.HttpPort == 0 {
		return
	}
	httpAddr := fmt.Sprintf("%s:%d", s.conf.Server.Address, s.conf.Server.HttpPort)
	log.Printf("start http server, service_name: %s, address: %s", s.conf.Server.ServiceName, httpAddr)
	go http.ListenAndServe(httpAddr, handler)
}

func (s *Server) Serve(options ...ServeOption) error {
	do := serveOptions{}
	for _, option := range options {
		option.f(&do)
	}

	defer func() {
		if GlobalTraceCloser != nil {
			GlobalTraceCloser.Close()
		}
	}()

	grpc_prometheus.Register(s.server)
	http.Handle("/metrics", promhttp.Handler())

	reflection.Register(s.server)
	return s.server.Serve(s.listener)
}
