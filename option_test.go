package gohttpclient

import (
	"net/http"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/stretchr/testify/require"
)

func TestWithHTTPClient(t *testing.T) {
	c := NewClient()
	httpClient := &http.Client{Timeout: 999 * time.Millisecond}
	WithHTTPClient(httpClient)(c)
	require.Equal(t, httpClient, c.client)
}

func TestWithRequestTimeout(t *testing.T) {
	c := NewClient()
	requestTimeout := 999 * time.Millisecond
	WithRequestTimeout(requestTimeout)(c)
	require.Equal(t, requestTimeout, c.requestTimeout)
}

func TestWithMaxBodySize(t *testing.T) {
	c := NewClient()
	maxBodySize := uint64(999)
	WithMaxBodySize(maxBodySize)(c)
	require.Equal(t, maxBodySize, c.maxBodySize)
}

func TestWithShouldRetryFunc(t *testing.T) {
	c := NewClient()
	shouldRetryFunc := func(req *http.Request, resp *http.Response, err error) bool { return true }
	WithShouldRetryFunc(shouldRetryFunc)(c)
	require.Equal(t, true, nil != c.retryOption.ShouldRetryFunc)
}

func TestWithMaxRetry(t *testing.T) {
	c := NewClient()
	maxRetry := uint64(999)
	WithMaxRetry(maxRetry)(c)
	require.Equal(t, maxRetry, c.retryOption.MaxRetry)
}

func TestWithRetryBackOff(t *testing.T) {
	c := NewClient()
	retryBackOff := backoff.NewConstantBackOff(999 * time.Millisecond)
	WithRetryBackOff(retryBackOff)(c)
	require.Equal(t, retryBackOff, c.retryOption.RetryBackOff)
}

func TestWithLoggerOption(t *testing.T) {
	c := NewClient()
	loggerOption := NewLoggerOption()
	// fix require.Equal
	loggerOption.LoggerFunc = nil
	WithLoggerOption(loggerOption)(c)
	require.Equal(t, loggerOption, c.loggerOption)
}

func TestWithRateLimitOption(t *testing.T) {
	c := NewClient()
	rateLimitOption := NewRateLimitOption(10)
	// fix require.Equal
	rateLimitOption.RateLimitConstructor = nil
	rateLimitOption.RateLimitFunc = nil
	WithRateLimitOption(rateLimitOption)(c)
	require.Equal(t, rateLimitOption, c.rateLimitOption)
}

func TestWithHystrixOption(t *testing.T) {
	c := NewClient()
	hystrixOption := NewHystrixOption()
	WithHystrixOption(hystrixOption)(c)
	require.Equal(t, true, c.hystrixOption.isEnabled())
}

func TestWithTraceOption(t *testing.T) {
	c := NewClient()
	traceOption := NewTraceOption()
	WithTraceOption(traceOption)(c)
	require.Equal(t, true, c.traceOption.isEnabled())
}

func TestWithCacheOption(t *testing.T) {
	c := NewClient()
	cacheOption := NewMemoryCacheOption()
	WithCacheOption(cacheOption)(c)
	require.Equal(t, true, c.cacheOption.isEnabled())
}
