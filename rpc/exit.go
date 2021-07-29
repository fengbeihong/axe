package rpc

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

func onSignal() int {
	//de-register if meet sighup
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL, syscall.SIGHUP, syscall.SIGQUIT)
	x := <-ch
	code, _ := strconv.Atoi(fmt.Sprintf("%d", x))
	log.Println("warning: receive signal: ", x)
	return code
}

func onExit(code int) {
	// close task
	deregisterConsul()
	closeGrpc()
	closeDBClient()
	closeRedisClient()

	os.Exit(code)
}

func closeGrpc() {
	for _, conn := range grpcPools {
		if conn != nil {
			conn.Close()
		}
	}
}
func closeDBClient() {
	// TODO
}

func closeRedisClient() {
	for _, m := range globalRedisPoolMap {
		if m != nil {
			m.Close()
		}
	}
}
