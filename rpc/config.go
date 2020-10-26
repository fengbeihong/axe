package rpc

import (
	"log"

	"github.com/BurntSushi/toml"
)

const (
	protoTypeRpc  = "rpc"
	protoTypeHttp = "http"
)

const (
	callTypeConsul = "consul"
	callTypeLocal  = "local"
)

const (
	BalanceTypeRandom     = "random"
	BalanceTypeRoundRobin = "roundrobin"
)

type Config struct {
	Server  serverConfig
	Log     logConfig
	Metrics metricsConfig
	Trace   traceConfig
	Clients []clientConfig `toml:"client"`
}

type serverConfig struct {
	ServiceName string `toml:"service_name"`
	Address     string
	Port        int
	HttpPort    int `toml:"http_port"`
}

type logConfig struct {
}

type metricsConfig struct {
	Type    string
	Enabled bool
}

type traceConfig struct {
	Type      string
	Enabled   bool
	AgentPort int `toml:"agent_port"`
}

type clientConfig struct {
	ServiceName  string `toml:"service_name"`
	ProtoType    string `toml:"proto"`             // 协议名称 rpc或http
	CallType     string `toml:"type"`              // 调用方式 consul或local
	Endpoints    string `toml:"endpoints"`         // 指定的调用ip端口，当type为local时使用
	BalanceType  string `toml:"balance_type"`      // 负载均衡类型 round robin或random
	Timeout      int    `toml:"timeout"`           // 超时时间，是总体的超时，包含多次重试后的超时
	RetryTimes   uint   `toml:"retry_times"`       // 重试次数
	RetryTimeout int    `toml:"per_retry_timeout"` // 每次调用(包含第一次请求)的超时

	EndpointsArr []string `toml:"-"`
}

var GlobalConf *Config

func parseConfig(filePath string) *Config {
	var cfg Config
	if _, err := toml.DecodeFile(filePath, &cfg); err != nil {
		log.Fatalf("parse config file failed, file path: %s, error: %v", filePath, err)
	}
	return &cfg
}

func initConfig(filePath string) *Config {
	GlobalConf = parseConfig(filePath)
	return GlobalConf
}
