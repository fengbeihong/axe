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

func initRedisClient(s *Server) {
	for _, redisCfg := range s.cfg.RedisClients {
		cfg := redisCfg
		globalRedisPoolMap[redisCfg.ServiceName] = initRedisPool(s, &cfg)
	}
}

func initRedisPool(s *Server, cfg *redisConfig) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     cfg.MaxIdle,
		IdleTimeout: time.Duration(cfg.IdleTimeout),
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", cfg.Address,
				redis.DialConnectTimeout(time.Duration(cfg.ConnTimeout)*time.Millisecond),
				redis.DialReadTimeout(time.Duration(cfg.ReadTimeout)*time.Millisecond),
				redis.DialWriteTimeout(time.Duration(cfg.WriteTimeout)*time.Millisecond),
				redis.DialPassword(cfg.Password),
			)
			if err != nil {
				s.Log.Error("init redis pool [%s] failed, address: %s, password: %s, error: %s", cfg.ServiceName, cfg.Address, cfg.Password, err.Error())
				return nil, err
			}
			if cfg.DB != 0 {
				_, err := c.Do("select", cfg.DB)
				if err != nil {
					s.Log.Error("init redis pool [%s] failed, address: %s, password: %s, error: %s", cfg.ServiceName, cfg.Address, cfg.Password, err.Error())
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
