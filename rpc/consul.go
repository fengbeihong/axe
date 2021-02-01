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
)

const (
	defaultConsulHost = "127.0.0.1"
	defaultInterval   = 5
	defaultTTL        = 100
)

func registerConsul(cfg *Config) error {
	if cfg.Consul.Host == "" {
		cfg.Consul.Host = defaultConsulHost
	}
	consulConfig := api.DefaultConfig()
	consulConfig.Address = cfg.Consul.Host
	client, err := api.NewClient(consulConfig)
	if err != nil {
		return fmt.Errorf("wonaming: create consul client error: %v", err)
	}

	consulAgent = client.Agent()
	consulServiceID = fmt.Sprintf("%s-%d", cfg.Server.Host, cfg.Server.Port)
	interval := time.Duration(defaultInterval) * time.Second
	ttl := time.Duration(defaultTTL) * time.Second

	// async update ttl
	go asyncUpdateTTL()

	// register service
	regService := &api.AgentServiceRegistration{
		ID:      consulServiceID,
		Name:    cfg.Server.ServiceName,
		Address: cfg.Server.Host,
		Port:    cfg.Server.Port,
	}
	err = consulAgent.ServiceRegister(regService)
	if err != nil {
		return fmt.Errorf("warning: initial register service '%s' host to consul error: %s", cfg.Server.ServiceName, err.Error())
	}

	// agent service check
	regCheck := &api.AgentCheckRegistration{
		ID:        consulServiceID,
		Name:      cfg.Server.ServiceName,
		ServiceID: consulServiceID,
		AgentServiceCheck: api.AgentServiceCheck{
			Interval: interval.String(),
			TTL:      ttl.String(),
			Status:   api.HealthPassing,
		},
	}
	err = consulAgent.CheckRegister(regCheck)
	if err != nil {
		return fmt.Errorf("warning: initial register service check to consul error: %s", err.Error())
	}

	return nil
}

func asyncUpdateTTL() {
	ticker := time.NewTicker(defaultInterval)
	for {
		<-ticker.C
		err := consulAgent.UpdateTTL(consulServiceID, "", api.HealthPassing)
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
