package gohttpclient

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/stretchr/testify/require"
)

func TestRetryRequestHandler(t *testing.T) {
	// Retry 3 times, each time interval is 5ms, the first 2 times fail, the 3rd time succeeds.
	maxRetry := uint64(3)
	backOffWait := 5 * time.Millisecond
	curRetry := uint64(0)
	options := NewRetryOption(maxRetry, backoff.NewConstantBackOff(backOffWait))
	options.ShouldRetryFunc = func(req *http.Request, resp *http.Response, err error) bool {
		curRetry++
		return curRetry < maxRetry
	}
	handler := RetryHandler(options)

	handlerFunc := func(req *http.Request) (resp *http.Response, err error) {
		return &http.Response{
			Body: io.NopCloser(bytes.NewBufferString("hello world")),
		}, nil
	}

	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	startTime := time.Now()
	resp, err := handler(req, handlerFunc)
	endTime := time.Now()
	require.Nil(t, err)
	require.NotNil(t, resp)
	// Actual retries 2 times, a total of 10ms.
	minTakes := int64(backOffWait/time.Millisecond) * int64(maxRetry-1)
	maxTakes := int64(backOffWait/time.Millisecond) * int64(maxRetry)
	realTakes := int64(endTime.Sub(startTime) / time.Millisecond)
	require.True(t, minTakes <= realTakes && realTakes < maxTakes)
}

func TestRetryRequestHandler_AllFailed(t *testing.T) {
	// Retry 3 times, each time interval is 5ms, all 3 times fail.
	maxRetry := uint64(3)
	backOffWait := 5 * time.Millisecond
	options := NewRetryOption(maxRetry, backoff.NewConstantBackOff(backOffWait))
	options.ShouldRetryFunc = func(req *http.Request, resp *http.Response, err error) bool {
		return true
	}
	handler := RetryHandler(options)

	handlerFunc := func(req *http.Request) (resp *http.Response, err error) {
		return &http.Response{
			Body: io.NopCloser(bytes.NewBufferString("hello world")),
		}, nil
	}

	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	startTime := time.Now()
	resp, err := handler(req, handlerFunc)
	endTime := time.Now()
	require.Nil(t, err)
	require.NotNil(t, resp)
	minTakes := int64(backOffWait/time.Millisecond) * int64(maxRetry)
	maxTakes := int64(backOffWait/time.Millisecond) * int64(maxRetry+1)
	realTakes := int64(endTime.Sub(startTime) / time.Millisecond)
	require.True(t, minTakes <= realTakes && realTakes < maxTakes)
}

func TestRetryRequestHandler_NoFailed(t *testing.T) {
	// Retry 3 times with 5ms interval each time. Success the first time.
	maxRetry := uint64(3)
	backOffWait := 5 * time.Millisecond
	options := NewRetryOption(maxRetry, backoff.NewConstantBackOff(backOffWait))
	options.ShouldRetryFunc = func(req *http.Request, resp *http.Response, err error) bool {
		return false
	}
	handler := RetryHandler(options)

	handlerFunc := func(req *http.Request) (resp *http.Response, err error) {
		return &http.Response{
			Body: io.NopCloser(bytes.NewBufferString("hello world")),
		}, nil
	}

	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	startTime := time.Now()
	resp, err := handler(req, handlerFunc)
	endTime := time.Now()
	require.Nil(t, err)
	require.NotNil(t, resp)
	maxTakes := int64(backOffWait / time.Millisecond)
	realTakes := int64(endTime.Sub(startTime) / time.Millisecond)
	require.True(t, realTakes < maxTakes)
}

func TestRetryRequestHandler_ContextCancel(t *testing.T) {
	options := NewRetryOption(3, backoff.NewConstantBackOff(5*time.Millisecond))
	options.ShouldRetryFunc = func(req *http.Request, resp *http.Response, err error) bool {
		return true
	}
	handler := RetryHandler(options)

	handlerFunc := func(req *http.Request) (resp *http.Response, err error) {
		return &http.Response{
			Body: io.NopCloser(bytes.NewBufferString("hello world")),
		}, nil
	}

	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	ctx, cancel := context.WithCancel(context.Background())
	req = req.WithContext(ctx)
	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()
	resp, err := handler(req, handlerFunc)
	require.True(t, errors.Is(err, context.Canceled))
	require.Nil(t, resp)
}

func TestNewFromBackOff(t *testing.T) {
	exponentialBackOff := backoff.NewExponentialBackOff()
	exponentialBackOff.RandomizationFactor = 0
	inits := []backoff.BackOff{
		exponentialBackOff,
		backoff.NewConstantBackOff(time.Second),
		&backoff.StopBackOff{},
		&backoff.ZeroBackOff{},
	}
	initNextBackOffs := []time.Duration{}
	for _, b := range inits {
		b.Reset()
		d := b.NextBackOff()
		initNextBackOffs = append(initNextBackOffs, d)
	}

	exponentialBackOff1 := backoff.NewExponentialBackOff()
	exponentialBackOff1.RandomizationFactor = 0
	bs := []backoff.BackOff{
		exponentialBackOff1,
		backoff.NewConstantBackOff(time.Second),
		&backoff.StopBackOff{},
		&backoff.ZeroBackOff{},
	}
	nextBackOffs := []time.Duration{}
	for _, b := range bs {
		b.NextBackOff()
		b2 := newFromBackOff(b)
		d := b2.NextBackOff()
		nextBackOffs = append(nextBackOffs, d)
	}
	require.Equal(t, initNextBackOffs, nextBackOffs)
}

type testBackOff struct{}

func (b *testBackOff) Reset() {}

func (b *testBackOff) NextBackOff() time.Duration { return 0 }

func TestNewFromBackOff_NotDefined(t *testing.T) {
	var errmsg string
	defer func() {
		if r := recover(); r != nil {
			errmsg = fmt.Sprintf("%v", r)
		}
	}()

	_ = newFromBackOff(&testBackOff{})
	require.Equal(t, "undefind backoff", errmsg)
}
