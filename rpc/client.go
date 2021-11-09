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

func initRpcClient(s *Server) {
	for _, item := range s.cfg.RpcClients {
		if item.CallType == callTypeLocal {
			if err := item.loadEndpoints(); err != nil {
				s.Log.Error(err.Error())
				continue
			}
		}
		tmp := item
		clientConfigMap[item.ServiceName] = &tmp
	}
}

// load balancer
func (c *clientConfig) endpointByBalancer() string {
	// 暂时只有一种roundrobin负载均衡策略
	if len(c.EndpointStrList) == 0 {
		return ""
	}
	if c.Balancer == nil {
		return c.EndpointStrList[0]
	}
	v, err := c.Balancer.Pick()
	if err != nil {
		return c.EndpointStrList[0]
	}
	return v.(string)
}

func (c *clientConfig) loadEndpoints() error {
	arr := strings.Split(c.Endpoints, ",")
	if len(arr) == 0 {
		return fmt.Errorf("check endpoints failed, empty ip address, service_name: %s, endpoints: %s", c.ServiceName, c.Endpoints)
	}

	var l []interface{}
	for _, item := range arr {
		l = append(l, item)
	}

	// load balancer
	c.Balancer = NewBalancer(l)
	c.EndpointStrList = arr
	return nil
}

func getClientConfig(name string) *clientConfig {
	return clientConfigMap[name]
}
