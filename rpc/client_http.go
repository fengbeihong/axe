package rpc

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type httpclientOption struct {
	cfg         *clientConfig
	method      string
	ctx         context.Context
	serviceName string
	uri         string
	headers     map[string]string
	body        io.Reader
}

func HttpGet(ctx context.Context, serviceName string, uri string, headers map[string]string, body ...io.Reader) ([]byte, error) {
	var b io.Reader
	if len(body) == 1 {
		b = body[0]
	} else {
		b = nil
	}
	return httpOperation(&httpclientOption{
		method:      "GET",
		ctx:         ctx,
		serviceName: serviceName,
		uri:         uri,
		body:        b,
		headers:     headers,
	})
}

func HttpPost(ctx context.Context, serviceName string, uri string, headers map[string]string, body ...io.Reader) ([]byte, error) {
	var b io.Reader
	if len(body) == 1 {
		b = body[0]
	} else {
		b = nil
	}
	return httpOperation(&httpclientOption{
		method:      "POST",
		ctx:         ctx,
		serviceName: serviceName,
		uri:         uri,
		body:        b,
		headers:     headers,
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

	if !strings.HasPrefix(domain, "http://") && !strings.HasPrefix(domain, "https://") {
		domain = "http://" + domain
	}

	url := domain + opt.uri

	c := &http.Client{
		Timeout:   time.Duration(opt.cfg.RetryTimeout) * time.Millisecond,
		Transport: http.DefaultTransport,
	}
	req, err := http.NewRequest(opt.method, url, opt.body)
	if err != nil {
		return nil, fmt.Errorf("failed to execute http request, service_name: %s, url: %s, error: %s", opt.serviceName, url, err.Error())
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range opt.headers {
		req.Header.Set(k, v)
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed, service name: %s, url: %s, error: %s", opt.serviceName, url, err.Error())
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("http request failed, service name: %s, url: %s, code: %d, status: %s", opt.serviceName, url, resp.StatusCode, resp.Status)
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("http request read body failed, service name: %s, url: %s, error: %s", opt.serviceName, url, err.Error())
	}

	return b, nil
}
