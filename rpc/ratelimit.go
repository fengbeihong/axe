package rpc

import (
	"time"

	grpcRateLimit "github.com/grpc-ecosystem/go-grpc-middleware/ratelimit"
	"github.com/juju/ratelimit"
)

const (
	LimiterAlwaysPass = "always_pass"
	LimiterNoBlock    = "no_block"
)

var limiterMap = map[string]grpcRateLimit.Limiter{
	LimiterAlwaysPass: &alwaysPassLimiter{},
	LimiterNoBlock:    &rateLimitNoBlock{},
}

func name2Limiter(typeName string) grpcRateLimit.Limiter {
	return limiterMap[typeName]
}

// alwaysPassLimiter is an example limiter which implements Limiter interface.
// It does not limit any request because Limit function always returns false.
type alwaysPassLimiter struct{}

func (*alwaysPassLimiter) Limit() bool {
	return false
}

var tb *ratelimit.Bucket

func initRateLimit(cfg *Config) grpcRateLimit.Limiter {
	if cfg.RateLimit.Type != LimiterAlwaysPass {
		tb = ratelimit.NewBucket(time.Duration(cfg.RateLimit.FillInterval), cfg.RateLimit.Capacity)
	}
	limiter := name2Limiter(cfg.RateLimit.Type)
	return limiter
}

// rateLimitNoBlock 如果桶中没有token，不block，直接返回
type rateLimitNoBlock struct{}

func (*rateLimitNoBlock) Limit() bool {
	if tb == nil {
		return false
	}
	count := tb.TakeAvailable(1)
	if count == 0 {
		return true
	} else {
		return false
	}
}
