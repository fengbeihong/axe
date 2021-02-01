package rpc

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

func onExit() {
	//de-register if meet sighup
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL, syscall.SIGHUP, syscall.SIGQUIT)
	x := <-ch
	log.Println("warning: receive signal: ", x)

	// close task
	deregisterConsul()
	closeGrpc()
	closeDBClient()
	closeRedisClient()

	s, _ := strconv.Atoi(fmt.Sprintf("%d", x))
	os.Exit(s)
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
