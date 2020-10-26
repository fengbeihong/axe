package rpc

import (
	"fmt"
	"time"

	"google.golang.org/grpc/resolver"
)

const (
	defaultConsulAgentAddr = "127.0.0.1:8500"
	defaultTTL             = 100
)

func registerConsul(cfg *Config) {
	ips, _ := LocalIPv4s()
	err := Register(cfg.Server.ServiceName, ips[0], cfg.Server.Port, defaultConsulAgentAddr, time.Second*10, defaultTTL)
	if err != nil {
		fmt.Println(fmt.Errorf("register consul failed, error: %v", err))
	}
}

func generateSchema(serviceName string) (schema string, err error) {
	builder := NewConsulBuilder(defaultConsulAgentAddr)
	target := resolver.Target{Scheme: builder.Scheme(), Endpoint: serviceName}
	_, err = builder.Build(target, NewConsulClientConn(), resolver.BuildOptions{})
	if err != nil {
		return builder.Scheme(), err
	}
	schema = builder.Scheme()
	return
}
