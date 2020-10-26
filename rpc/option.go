package rpc

type InitOption struct {
	f func(*initOptions)
}

type initOptions struct {
	logger Logger
}

// WithLogger init logger
func WithLogger(logger Logger) InitOption {
	return InitOption{func(i *initOptions) {
		i.logger = logger
	}}
}

type ServeOption struct {
	f func(*serveOptions)
}

type serveOptions struct {
	// TODO
}
