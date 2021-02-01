package rpc

import (
	"fmt"
	"io"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	"github.com/uber/jaeger-lib/metrics"
)

const traceTypeJaeger = "jaeger"

var GlobalTraceCloser io.Closer

type traceLogger struct {
}

func (l *traceLogger) Infof(format string, args ...interface{}) {
	GlobalConf.Log.Infof("[INFO] "+format, args...)
}

func (l *traceLogger) Error(msg string) {
	GlobalConf.Log.Errorf(msg)
}

func initJaeger(rpcConf *Config) opentracing.Tracer {
	if rpcConf.Trace.Type != traceTypeJaeger {
		return nil
	}

	cfg := jaegercfg.Configuration{
		ServiceName: rpcConf.Server.ServiceName,
		Sampler: &jaegercfg.SamplerConfig{
			Type:  jaeger.SamplerTypeRateLimiting,
			Param: 1.0, // 1秒采集一次
		},
		// 配置agent收集
		Reporter: &jaegercfg.ReporterConfig{
			LocalAgentHostPort: fmt.Sprintf("127.0.0.1:%d", rpcConf.Trace.AgentPort),
		},
		Disabled: !rpcConf.Trace.Enabled,
	}
	t, closer, err := cfg.NewTracer(
		jaegercfg.Logger(&traceLogger{}),
		jaegercfg.Metrics(metrics.NullFactory),
	)
	if t == nil || closer == nil || err != nil {
		GlobalConf.Log.Errorf("could not initialize jaeger tracer, tracer: %v, closer: %v, error: %v", t, closer, err)
		return nil
	}
	GlobalTraceCloser = closer
	opentracing.SetGlobalTracer(t)
	return t
}
