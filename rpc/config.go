package rpc

import (
	"log"

	"github.com/creasty/defaults"

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
	Pprof        pprofConfig `toml:"pprof"`
	Server       serverConfig
	RateLimit    rateLimitConfig `toml:"rate_limit"`
	Consul       consulConfig
	Metrics      metricsConfig
	Trace        traceConfig
	RpcClients   []clientConfig `toml:"client"`
	DBClients    []dbConfig     `toml:"database"`
	RedisClients []redisConfig  `toml:"redis"`
}

type serverConfig struct {
	ServiceName string `toml:"service_name"`
	Host        string `toml:"host"`
	GrpcPort    int    `toml:"grpc_port"`
	HttpPort    int    `toml:"http_port"`
}

type rateLimitConfig struct {
	Enabled      bool
	Type         string `toml:"type" default:"always_pass"`
	FillInterval int    `toml:"fill_interval" default:"300"`
	Capacity     int64  `toml:"capacity" default:"3000"`
}

type pprofConfig struct {
	Port int `toml:"port"`
}

type consulConfig struct {
	Enabled bool
	Host    string `toml:"host"`
}

type metricsConfig struct {
	Enabled bool
	Type    string
}

type traceConfig struct {
	Enabled   bool
	Type      string
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

	EndpointStrList []string  `toml:"-"`
	Balancer        *Balancer `toml:"-"`
}

type redisConfig struct {
	ServiceName  string `toml:"service_name"`
	Address      string `toml:"address"`
	Password     string `toml:"password"`
	DB           int    `toml:"db"`
	MaxIdle      int    `toml:"max_idle"`
	IdleTimeout  int    `toml:"idle_timeout"`
	ConnTimeout  int    `toml:"conn_timeout"`
	ReadTimeout  int    `toml:"read_timeout"`
	WriteTimeout int    `toml:"write_timeout"`
}

type dbConfig struct {
	ServiceName string `toml:"service_name"`
	Host        string `toml:"host"`
	Port        int    `toml:"port" default:"3306"`
	Username    string `toml:"username"`
	Password    string `toml:"password"`
	Database    string `toml:"database"`
	EnableLog   bool   `toml:"enable_log"`
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
	setDefaultValue(GlobalConf)
	return GlobalConf
}

func setDefaultValue(cfg *Config) {
	ds(cfg)
	ds(&cfg.Server)
	ds(&cfg.Consul)
	ds(&cfg.Metrics)
	ds(&cfg.Trace)
	for _, item := range cfg.RpcClients {
		ds(&item)
	}
	for _, item := range cfg.DBClients {
		ds(&item)
	}
	for _, item := range cfg.RedisClients {
		ds(&item)
	}
}

func ds(o interface{}) {
	if err := defaults.Set(o); err != nil {
		panic(err)
	}
}
