package gohttpclient

import (
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
)

// Option defines the signature of the options configuration family of methods.
type Option func(c *Client)

// WithHTTPClient sets options for a custom http.Client instance.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		c.client = client
	}
}

// WithRequestTimeout sets the timeout for the entire request.
func WithRequestTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.requestTimeout = timeout
	}
}

// WithMaxBodySize sets the maximum limit on the size of data returned by the server.
func WithMaxBodySize(n uint64) Option {
	return func(c *Client) {
		c.maxBodySize = n
	}
}

// WithShouldRetryFunc sets the function that determines whether a retry is required.
func WithShouldRetryFunc(fn ShouldRetryFunc) Option {
	return func(c *Client) {
		c.retryOption.ShouldRetryFunc = fn
	}
}

// WithMaxRetry sets the maximum number of retries.
// When n=0, it means that no retry operation is performed, instead of retrying until success.
func WithMaxRetry(n uint64) Option {
	return func(c *Client) {
		c.retryOption.MaxRetry = n
	}
}

// WithRetryBackOff sets the retry policy.
// You can choose a constant retry interval, or use an exponential back off algorithm.
func WithRetryBackOff(b backoff.BackOff) Option {
	return func(c *Client) {
		c.retryOption.RetryBackOff = b
	}
}

// WithLoggerOption sets whether to enable the logging function to record the context information of the request.
func WithLoggerOption(option LoggerOption) Option {
	return func(c *Client) {
		c.loggerOption = option
	}
}

// WithRateLimitOption sets the rate-limiting configuration and limits the maximum number of requests per second.
func WithRateLimitOption(option RateLimitOption) Option {
	return func(c *Client) {
		c.rateLimitOption = option
	}
}

// WithHystrixOption sets the configuration of the circuit breaker.
func WithHystrixOption(option HystrixOption) Option {
	return func(c *Client) {
		c.hystrixOption = option
	}
}

// WithTraceOption sets the configuration for distributed call chain tracing.
func WithTraceOption(option TraceOption) Option {
	return func(c *Client) {
		c.traceOption = option
	}
}

// WithCacheOption sets the cache configuration.
func WithCacheOption(option CacheOption) Option {
	return func(c *Client) {
		c.cacheOption = option
	}
}
