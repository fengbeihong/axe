package rpc

import (
	"log"
	"os"
)

var gLogger Logger

type Logger interface {
	Info(format string, args ...interface{})
	Error(format string, args ...interface{})
}

func init()  {
	setGLogger(defaultLogger())
}

func setGLogger(l Logger)  {
	gLogger = l
}

func defaultLogger() Logger {
	return &myLogger{
		log: log.New(os.Stdout, "[rpc] ", log.LstdFlags),
	}
}

type myLogger struct {
	log *log.Logger
}

func (l *myLogger) Info(format string, args ...interface{}) {
	l.log.Printf("[INFO] "+format, args...)
}

func (l *myLogger) Error(format string, args ...interface{}) {
	l.log.Printf("[ERROR] "+format, args...)
}
