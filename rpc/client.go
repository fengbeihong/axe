package rpc

import (
	"fmt"
	"strings"
)

var (
	ServiceConfigNotFound     = fmt.Errorf("service not found")
	ServiceConfigInvalidProto = fmt.Errorf("service found, but invalid proto type")
)

var clientConfigMap map[string]*clientConfig

func init() {
	clientConfigMap = make(map[string]*clientConfig)
}

func initRpcClient(cfg *Config) {
	for _, item := range cfg.RpcClients {
		if item.CallType == callTypeLocal {
			if err := item.checkEndpoints(); err != nil {
				cfg.Log.Errorf("%v", err)
				continue
			}
		}
		tmp := item
		clientConfigMap[item.ServiceName] = &tmp
	}
}

// load balancer
func (c *clientConfig) endpointByBalancer() string {
	// 暂时没有实现负载均衡
	if len(c.EndpointsArr) == 0 {
		return ""
	}
	return c.EndpointsArr[0]
}

func (c *clientConfig) checkEndpoints() error {
	arr := strings.Split(c.Endpoints, ",")
	if len(arr) == 0 {
		return fmt.Errorf("check endpoints failed, empty ip address, service_name: %s, endpoints: %s", c.ServiceName, c.Endpoints)
	}

	c.EndpointsArr = arr
	return nil
}

func getClientConfig(name string) *clientConfig {
	return clientConfigMap[name]
}
