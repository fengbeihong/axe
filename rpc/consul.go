package rpc

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/consul/api"
	_ "github.com/mbobakov/grpc-consul-resolver"
)

var (
	consulAgent     *api.Agent
	consulServiceID string
	consulCheckID   string
)

const (
	defaultConsulHost = "127.0.0.1"
	defaultInterval   = 10 * time.Second
	defaultTTL        = 30 * time.Second
	defaultTimeout    = 2 * time.Second
)

func registerConsul(cfg *Config) error {
	if cfg.Consul.Host == "" {
		cfg.Consul.Host = defaultConsulHost
	}
	cc := api.DefaultConfig()
	cc.Address = fmt.Sprintf("%s:8500", cfg.Consul.Host)
	client, err := api.NewClient(cc)
	if err != nil {
		return fmt.Errorf("waring: create consul client error: %v", err)
	}

	agentIP := GetLocalIP()
	if agentIP == "" {
		return fmt.Errorf("warning: init consul failed, get local ip empty")
	}

	consulAgent = client.Agent()
	consulServiceID = fmt.Sprintf("%s-%d", agentIP, cfg.Server.GrpcPort)
	consulCheckID = consulServiceID + "-ttl"

	// register service
	regService := &api.AgentServiceRegistration{
		ID:      consulServiceID,
		Name:    cfg.Server.ServiceName,
		Address: agentIP,
		Port:    cfg.Server.GrpcPort,
	}

	err = consulAgent.ServiceRegister(regService)
	if err != nil {
		return fmt.Errorf("warning: register service '%s' to consul error: %s", cfg.Server.ServiceName, err.Error())
	}

	err = consulAgent.CheckRegister(&api.AgentCheckRegistration{
		ID:        consulCheckID,
		Name:      cfg.Server.ServiceName,
		ServiceID: consulServiceID,
		AgentServiceCheck: api.AgentServiceCheck{
			CheckID:  consulCheckID,
			Status:   api.HealthPassing,
			TTL:      defaultTTL.String(),
			Interval: defaultInterval.String(),
			Timeout:  defaultTimeout.String(),
		},
	})
	if err != nil {
		return fmt.Errorf("warning: register check '%s' to consul error: %s", cfg.Server.ServiceName, err.Error())
	}

	// async update ttl
	go asyncUpdateTTL()

	return nil
}

func asyncUpdateTTL() {
	ticker := time.NewTicker(defaultTTL)
	for {
		<-ticker.C
		err := consulAgent.UpdateTTL(consulCheckID, "", api.HealthPassing)
		if err != nil {
			log.Println("warning: update ttl of service error: ", err.Error())
		}
	}
}

func deregisterConsul() {
	if consulAgent == nil {
		return
	}

	err := consulAgent.ServiceDeregister(consulServiceID)
	if err != nil {
		log.Println("warning: deregister service error: ", err.Error())
	} else {
		log.Println("warning: deregister service from consul server.")
	}

	err = consulAgent.CheckDeregister(consulServiceID)
	if err != nil {
		log.Println("warning: deregister check error: ", err.Error())
	}
}
