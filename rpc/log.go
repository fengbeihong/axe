package rpc

import (
	"log"
	"os"
)

type Logger interface {
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

func newLogger() Logger {
	return &logger{
		log: log.New(os.Stdout, "[rpc] ", log.LstdFlags),
	}
}

type logger struct {
	log *log.Logger
}

func (l *logger) Infof(format string, args ...interface{}) {
	l.log.Printf("[INFO] "+format, args...)
}

func (l *logger) Errorf(format string, args ...interface{}) {
	l.log.Printf("[ERROR] "+format, args...)
}

func initLogger(cfg *Config, opts ...InitOption) {
	options := initOptions{
		logger: newLogger(),
	}

	for _, opt := range opts {
		opt.f(&options)
	}

	cfg.Log = options.logger
}
