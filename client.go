package gohttpclient

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/opentracing-contrib/go-stdlib/nethttp"
)

// Doer is the interface for initiating requests, it needs to implement the Do method,
// and http.Client has implemented this interface.
type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

// Client is an HTTP request client and is fully compatible with the net.http package.
// And provides functions such as retry, rate limit, circuit breaker, cache, log, and trace.
// This package can be used as a basic toolkit for a microservice framework with HTTP requests as a carrier,
// or as a more secure library to limit the size of concurrent requests and downloaded data.
type Client struct {
	client          *http.Client
	requestTimeout  time.Duration
	maxBodySize     uint64
	retryOption     RetryOption
	loggerOption    LoggerOption
	rateLimitOption RateLimitOption
	hystrixOption   HystrixOption
	traceOption     TraceOption
	cacheOption     CacheOption
	requestHandler  RequestHandler
}

// NewClient creates a new HTTP request client.
// If no setting options are passed in, then it behaves exactly the same as the official package.
// You can use the WithXXX series of methods to configure options.
// It provides advanced functions such as retry, rate limit, circuit breaker, cache, log, and trace.
func NewClient(options ...Option) *Client {
	c := &Client{
		client:         &http.Client{},
		requestHandler: noOpRequestHandler,
	}
	for _, opt := range options {
		opt(c)
	}

	bodySizeOption := NewBodySizeOption(c.maxBodySize)

	var requestHandlers []RequestHandler

	getRequestHandlers := []struct {
		Enable  bool
		Handler RequestHandler
	}{
		{c.loggerOption.isEnabled(), LoggerHandler(c.loggerOption)},
		{c.retryOption.isEnabled(), RetryHandler(c.retryOption)},
		{c.rateLimitOption.isEnabled(), RateLimitHandler(c.rateLimitOption)},
		{c.hystrixOption.isEnabled(), HystrixHandler(c.hystrixOption)},
		{c.traceOption.isEnabled(), TraceHandler(c.traceOption)},
		{c.cacheOption.isEnabled(), CacheHandler(c.cacheOption)},
		{bodySizeOption.isEnabled(), BodySizeHandler(bodySizeOption)},
	}
	for _, g := range getRequestHandlers {
		if g.Enable {
			requestHandlers = append(requestHandlers, g.Handler)
		}
	}

	if len(requestHandlers) > 0 {
		c.requestHandler = ChainRequestHandlers(requestHandlers...)
	}
	if c.traceOption.isEnabled() {
		c.client.Transport = &nethttp.Transport{RoundTripper: c.client.Transport}
	}
	if c.requestTimeout > 0 {
		c.client.Timeout = c.requestTimeout
	}

	return c
}

// Do performs HTTP real requests.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.do(req)
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	return requestForDoer(c.client, c.requestHandler, req)
}

// Get initiates an HTTP GET request.
func (c *Client) Get(url string) (resp *http.Response, err error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// Post initiates an HTTP POST request.
func (c *Client) Post(url, contentType string, body io.Reader) (resp *http.Response, err error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return c.Do(req)
}

// PostForm initiates HTTP POST form data requests.
func (c *Client) PostForm(url string, data url.Values) (resp *http.Response, err error) {
	return c.Post(url, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
}

// Head initiates an HTTP HEAD request.
func (c *Client) Head(url string) (resp *http.Response, err error) {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}
