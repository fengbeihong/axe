package rpc

func InitRpc(filePath string, opts ...InitOption) *Server {
	cfg := initConfig(filePath)

	initLogger(opts...)

	registerConsul(cfg)

	initClient(cfg)

	// init rpc server
	return initServer(cfg)
}

func InitRpcSimple(filePath string, opts ...InitOption) {
	cfg := initConfig(filePath)

	initLogger(opts...)

	registerConsul(cfg)

	initClient(cfg)
}
