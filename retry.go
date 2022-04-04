package gohttpclient

import (
	"context"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/pkg/errors"
)

// ShouldRetryFunc defines a function that determines whether a retry is required.
type ShouldRetryFunc func(*http.Request, *http.Response, error) bool

// defaultShouldRetryFunc is the default function that determines whether to retry by default.
// If the request fails or the response status code is greater than or equal to 500, it will be retried.
var defaultShouldRetryFunc ShouldRetryFunc = func(req *http.Request, resp *http.Response, err error) bool {
	ok := err == nil && resp != nil && resp.StatusCode < 500
	return !ok
}

// RetryOption defines a retry option configuration.
type RetryOption struct {
	ShouldRetryFunc ShouldRetryFunc
	MaxRetry        uint64
	RetryBackOff    backoff.BackOff
}

// NewRetryOption creates a retry options configuration.
// Set the request to repeat up to maxRetry times when it fails.
// You can set a constant retry interval, or you can use the Exponential backoff algorithm.
// Exponential backoff is an algorithm that uses feedback to multiplicatively decrease the rate of some process,
// in order to gradually find an acceptable rate.
// The default algorithm for judging retry is that the
// HTTP status code is greater than or equal to 500 before retrying.
func NewRetryOption(maxRetry uint64, retryBackOff backoff.BackOff) RetryOption {
	return RetryOption{
		ShouldRetryFunc: defaultShouldRetryFunc,
		MaxRetry:        maxRetry,
		RetryBackOff:    retryBackOff,
	}
}

func (r RetryOption) isEnabled() bool {
	return r.ShouldRetryFunc != nil && r.RetryBackOff != nil && r.MaxRetry > 0
}

// RetryHandler creates a retry interceptor that can set the maximum number of retries, and the time interval between each retry.
func RetryHandler(option RetryOption) RequestHandler {
	return func(req *http.Request, handlerFunc RequestHandlerFunc) (resp *http.Response, err error) {
		if option.MaxRetry == 0 {
			return handlerFunc(req)
		}

		b := newFromBackOff(option.RetryBackOff)
		b = backoff.WithMaxRetries(b, option.MaxRetry)

		fn := func() bool {
			resp, err = handlerFunc(req)
			defer func() {
				if err != nil && resp != nil {
					if resp.Body != nil {
						_ = resp.Body.Close()
					}
					resp = nil
				}
			}()
			should := option.ShouldRetryFunc(req, resp, err)
			if !should {
				return false
			}
			d := b.NextBackOff()
			if d == backoff.Stop {
				return false
			}
			if err2 := sleepContext(getRequestContext(req), d); err2 != nil {
				err = errors.Wrapf(err2, "%v", err)
				return false
			}
			return true
		}

		for fn() {
			// fix revive
			_ = true
		}
		return
	}
}

func newFromBackOff(b backoff.BackOff) backoff.BackOff {
	var b2 backoff.BackOff
	switch v := b.(type) {
	case *backoff.ExponentialBackOff:
		v2 := *v
		b2 = &v2
	case *backoff.ConstantBackOff:
		v2 := *v
		b2 = &v2
	case *backoff.StopBackOff:
		v2 := *v
		b2 = &v2
	case *backoff.ZeroBackOff:
		v2 := *v
		b2 = &v2
	default:
		panic("undefind backoff")
	}
	b2.Reset()
	return b2
}

func sleepContext(ctx context.Context, wait time.Duration) error {
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
