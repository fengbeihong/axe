package rpc

import "os"

func init() {
	os.Setenv("GODEBUG", "netdns=go")
}

func InitRpc(filePath string, opts ...InitOption) *Server {
	cfg := initConfig(filePath)

	initLogger(cfg, opts...)

	initRpcClient(cfg)

	initRedisClient(cfg)

	initDBClient(cfg)

	// init rpc server
	return initServer(cfg)
}

func InitRpcSimple(filePath string, opts ...InitOption) {
	cfg := initConfig(filePath)

	initLogger(cfg, opts...)

	initRpcClient(cfg)
}
