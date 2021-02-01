package rpc

import (
	"fmt"
	"log"
	"net"
	"net/http"

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
	conf     *Config
	server   *grpc.Server
	listener net.Listener
	grpcAddr string
	httpAddr string
}

func initServer(cfg *Config) *Server {
	tcpAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	httpAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.HttpPort)
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
		httpAddr: httpAddr,
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

func (s *Server) GrpcServerAddr() string {
	return s.grpcAddr
}

func (s *Server) HttpServerAddr() string {
	return s.httpAddr
}

func (s *Server) GrpcServer() *grpc.Server {
	return s.server
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

	registerConsul(s.conf)
	grpc_prometheus.Register(s.server)
	http.Handle("/metrics", promhttp.Handler())

	reflection.Register(s.server)
	return s.server.Serve(s.listener)
}
