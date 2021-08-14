package rpc

import (
	"context"
	"testing"
)

var redisTestConfig = &Server{
	cfg: &Config{
		RedisClients: []redisConfig{
			{
				ServiceName:  "test",
				Address:      "127.0.0.1:6379",
				Password:     "",
				MaxIdle:      1,
				IdleTimeout:  200,
				ConnTimeout:  200,
				ReadTimeout:  200,
				WriteTimeout: 200,
			},
		},
	},
	Log: defaultLogger(),
}

func TestRedisConn(t *testing.T) {
	initRedisClient(redisTestConfig)

	c, err := GetRedisConn("test")
	if err != nil {
		t.Error("get redis conn error: %s\n", err.Error())
	}
	defer c.Close()

	_, err = c.Do("SET", "test", "1")
	if err != nil {
		t.Error("redis do error: %s\n", err.Error())
	}
}

func TestDoRedis(t *testing.T) {
	initRedisClient(redisTestConfig)

	_, err := DoRedis(context.Background(), "test", "SET", "test", "1")
	if err != nil {
		t.Error("get redis conn error: %s\n", err.Error())
	}
}