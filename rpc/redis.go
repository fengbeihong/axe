package rpc

import (
	"context"
	"fmt"
	"time"

	"github.com/opentracing/opentracing-go"

	"github.com/garyburd/redigo/redis"
)

var globalRedisPoolMap map[string]*redis.Pool

func init() {
	globalRedisPoolMap = make(map[string]*redis.Pool)
}

func initRedisClient(globalCfg *Config) {
	for _, redisCfg := range globalCfg.RedisClients {
		cfg := redisCfg
		globalRedisPoolMap[redisCfg.ServiceName] = initRedisPool(globalCfg, &cfg)
	}
}

func initRedisPool(globalCfg *Config, cfg *redisConfig) *redis.Pool {
	var opts []redis.DialOption
	if cfg.ConnTimeout != 0 {
		opts = append(opts, redis.DialConnectTimeout(time.Duration(cfg.ConnTimeout)*time.Millisecond))
	}
	if cfg.ReadTimeout != 0 {
		opts = append(opts, redis.DialReadTimeout(time.Duration(cfg.ReadTimeout)*time.Millisecond))
	}
	if cfg.WriteTimeout != 0 {
		opts = append(opts, redis.DialWriteTimeout(time.Duration(cfg.WriteTimeout)*time.Millisecond))
	}
	if cfg.Password != "" {
		opts = append(opts, redis.DialPassword(cfg.Password))
	}
	return &redis.Pool{
		MaxIdle:     cfg.MaxIdle,
		IdleTimeout: time.Duration(cfg.IdleTimeout),
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", cfg.Address, opts...)
			if err != nil {
				globalCfg.Log.Errorf("init redis pool [%s] failed, address: %s, password: %s, error: %s", cfg.ServiceName, cfg.Address, cfg.Password, err.Error())
				return nil, err
			}
			if cfg.DB != 0 {
				_, err := c.Do("select", cfg.DB)
				if err != nil {
					globalCfg.Log.Errorf("init redis pool [%s] failed, address: %s, password: %s, error: %s", cfg.ServiceName, cfg.Address, cfg.Password, err.Error())
					return nil, err
				}
			}

			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}

func GetRedisConn(serviceName string) (redis.Conn, error) {
	redisPool, ok := globalRedisPoolMap[serviceName]
	if !ok || redisPool == nil {
		return nil, fmt.Errorf("can't find redis client with name: '%s'", serviceName)
	}
	return redisPool.Get(), nil
}

func DoRedis(ctx context.Context, serviceName, cmd string, args ...interface{}) (reply interface{}, err error) {
	redisPool, ok := globalRedisPoolMap[serviceName]
	if !ok || redisPool == nil {
		return nil, fmt.Errorf("can't find redis client with name: '%s'", serviceName)
	}
	conn := redisPool.Get()
	defer conn.Close()

	return conn.Do(cmd, args...)
}

func DoRedisWithTrace(ctx context.Context, serviceName, cmd string, args ...interface{}) (reply interface{}, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "do_redis")
	defer span.Finish()
	redisPool, ok := globalRedisPoolMap[serviceName]
	if !ok || redisPool == nil {
		return nil, fmt.Errorf("can't find redis client with name: '%s'", serviceName)
	}
	conn := redisPool.Get()
	defer conn.Close()

	return conn.Do(cmd, args...)
}
