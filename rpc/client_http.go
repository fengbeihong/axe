package rpc

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

type httpclientOption struct {
	cfg         *clientConfig
	method      string
	ctx         context.Context
	serviceName string
	uri         string
	body        io.Reader
}

func HttpGet(ctx context.Context, serviceName string, uri string) ([]byte, error) {
	return httpOperation(&httpclientOption{
		method:      "GET",
		ctx:         ctx,
		serviceName: serviceName,
		uri:         uri,
		body:        nil,
	})
}

func HttpPost(ctx context.Context, serviceName string, uri string, body io.Reader) ([]byte, error) {
	return httpOperation(&httpclientOption{
		method:      "POST",
		ctx:         ctx,
		serviceName: serviceName,
		uri:         uri,
		body:        body,
	})
}

func httpOperation(opt *httpclientOption) (b []byte, err error) {
	cfg := getClientConfig(opt.serviceName)
	if cfg == nil {
		return nil, ServiceConfigNotFound
	}

	if cfg.ProtoType != protoTypeHttp {
		return nil, ServiceConfigInvalidProto
	}

	opt.cfg = cfg

	c := make(chan struct{})
	go func() {
		b, err = httpDoWithRetry(opt)
		c <- struct{}{}
	}()

	select {
	case <-c:
	case <-time.After(time.Duration(cfg.Timeout) * time.Millisecond):
		return nil, fmt.Errorf("http operation timeout for total %d ms, retry %d times", cfg.Timeout, cfg.RetryTimes)
	}

	return
}

func httpDoWithRetry(opt *httpclientOption) (b []byte, err error) {
	for i := 0; i < int(opt.cfg.RetryTimes); i++ {
		b, err = httpDo(opt)
		if err == nil {
			break
		}
	}
	return
}

// TODO tracer breaker ...
func httpDo(opt *httpclientOption) ([]byte, error) {
	domain := opt.cfg.endpointByBalancer()

	url := fmt.Sprintf("http://%s%s", domain, opt.uri)

	c := &http.Client{
		Timeout:   time.Duration(opt.cfg.RetryTimeout) * time.Millisecond,
		Transport: http.DefaultTransport,
	}
	req, err := http.NewRequest(opt.method, url, opt.body)
	if err != nil {
		GlobalConf.Log.Errorf("failed to execute http request, service_name: %s, url: %s, error: %s", opt.serviceName, url, err.Error())
		return nil, err
	}
	resp, err := c.Do(req)
	if err != nil {
		GlobalConf.Log.Errorf("http request failed, service name: %s, url: %s, error: %s", opt.serviceName, url, err.Error())
		return nil, err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		GlobalConf.Log.Errorf("http request read body failed, service name: %s, url: %s, error: %s", opt.serviceName, url, err.Error())
	}

	return b, nil
}
