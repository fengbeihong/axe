package rpc

import (
	"context"
	"fmt"
	"time"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"

	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"

	"google.golang.org/grpc"
)

var grpcPools map[string]*grpc.ClientConn

func init() {
	grpcPools = make(map[string]*grpc.ClientConn)
}

// DialService for rpc dial
func DialService(ctx context.Context, serviceName string) (*grpc.ClientConn, error) {
	conf := getClientConfig(serviceName)
	if conf == nil {
		return nil, ServiceConfigNotFound
	}

	if conf.ProtoType != protoTypeRpc {
		return nil, ServiceConfigInvalidProto
	}

	// TODO to optimize
	if grpcPools[serviceName] != nil {
		return grpcPools[serviceName], nil
	}

	opts := makeDialOption(conf)

	var conn *grpc.ClientConn
	var err error
	switch conf.CallType {
	case callTypeLocal:
		conn, err = dialWithLocal(ctx, conf, opts...)
	case callTypeConsul:
		conn, err = dialWithConsul(ctx, conf, opts...)
	default:
		conn, err = dialWithLocal(ctx, conf, opts...)
	}
	if err != nil {
		return nil, err
	}
	grpcPools[serviceName] = conn
	return conn, nil
}

func makeDialOption(conf *clientConfig) []grpc.DialOption {
	var streamInterceptorList []grpc.StreamClientInterceptor
	var unaryInterceptorList []grpc.UnaryClientInterceptor
	if GlobalConf.Metrics.Enabled {
		streamInterceptorList = append(streamInterceptorList, grpc_prometheus.StreamClientInterceptor)
		unaryInterceptorList = append(unaryInterceptorList, grpc_prometheus.UnaryClientInterceptor)
	}
	if GlobalConf.Trace.Enabled {
		streamInterceptorList = append(streamInterceptorList, grpc_opentracing.StreamClientInterceptor())
		unaryInterceptorList = append(unaryInterceptorList, grpc_opentracing.UnaryClientInterceptor())
	}

	// retry times
	opts := []grpc_retry.CallOption{
		grpc_retry.WithMax(conf.RetryTimes),
		grpc_retry.WithPerRetryTimeout(time.Duration(conf.RetryTimeout) * time.Millisecond),
	}

	return []grpc.DialOption{
		grpc.WithChainStreamInterceptor(streamInterceptorList...),
		grpc.WithChainUnaryInterceptor(unaryInterceptorList...),
		grpc.WithStreamInterceptor(grpc_retry.StreamClientInterceptor(opts...)),
		grpc.WithUnaryInterceptor(grpc_retry.UnaryClientInterceptor(opts...)),
		grpc.WithInsecure(),
	}
}

func dialWithConsul(ctx context.Context, cfg *clientConfig, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(cfg.Timeout)*time.Millisecond)
	defer cancel()

	conn, err := grpc.DialContext(ctx, fmt.Sprintf("consul://%s:8500/%s", GlobalConf.Consul.Host, cfg.ServiceName), opts...)
	if err != nil {
		return nil, fmt.Errorf("dialWithConsul, dial with context failed: %s", err.Error())
	}

	return conn, nil
}

func dialWithLocal(ctx context.Context, cfg *clientConfig, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(cfg.Timeout)*time.Millisecond)
	defer cancel()

	conn, err := grpc.DialContext(ctx, cfg.endpointByBalancer(), opts...)
	if err != nil {
		return nil, fmt.Errorf("dialWithLocal, dial with context failed: %s", err.Error())
	}

	return conn, nil
}
