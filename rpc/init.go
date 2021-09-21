package rpc

import (
	"os"

	_ "go.uber.org/automaxprocs"
)

func init() {
	// https://developer.aliyun.com/article/238940
	// http://tbg.github.io/golang-static-linking-bug?spm=a2c6h.12873639.0.0.52933341DDCBdG
	os.Setenv("GODEBUG", "netdns=go")
}

func NewServer(filePath string, opts ...InitOption) (*Server, error) {
	s := &Server{
		cfg: initConfig(filePath),
		Log: defaultLogger(),
	}

	for _, opt := range opts {
		opt.f(s)
	}

	setGLogger(s.Log)

	initRpcClient(s)

	initRedisClient(s)

	initDBClient(s)

	s.gs, s.Err = initGrpcServer(s.cfg)
	if s.Err != nil {
		return s, s.Err
	}
	s.hs, s.Err = initHttpServer(s.cfg)
	if s.Err != nil {
		return s, s.Err
	}

	return s, s.Err
}

type InitOption struct {
	f func(*Server)
}

// WithLogger init logger
func WithLogger(l Logger) InitOption {
	return InitOption{func(s *Server) {
		s.Log = l
	}}
}
