package gohttpclient

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRateLimitHandler(t *testing.T) {
	// The test requests the same address 5 times, up to 200 times per second.
	rate := 200
	wait := time.Second / time.Duration(rate)
	times := 5

	option := NewRateLimitOption(rate)
	handler := RateLimitHandler(option)

	handlerFunc := func(req *http.Request) (resp *http.Response, err error) {
		return &http.Response{
			Body: io.NopCloser(bytes.NewBufferString("hello world")),
		}, nil
	}

	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	startTime := time.Now()
	for i := 0; i < times; i++ {
		resp, err := handler(req, handlerFunc)
		require.Nil(t, err)
		require.NotNil(t, resp)
	}
	endTime := time.Now()
	minTakes := int64(wait/time.Millisecond) * int64(times-1)
	maxTakes := int64(wait/time.Millisecond) * int64(times)
	realTakes := int64(endTime.Sub(startTime) / time.Millisecond)
	require.True(t, minTakes <= realTakes && realTakes < maxTakes)
}

func TestRateLimitHandler_TwoURL(t *testing.T) {
	// The test requests two different addresses 5 times each, each address up to 200 times per second.
	rate := 200
	wait := time.Second / time.Duration(rate)
	times := 5

	option := NewRateLimitOption(rate)
	handler := RateLimitHandler(option)

	handlerFunc := func(req *http.Request) (resp *http.Response, err error) {
		return &http.Response{
			Body: io.NopCloser(bytes.NewBufferString("hello world")),
		}, nil
	}

	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	req2, _ := http.NewRequest(http.MethodPost, "https://example.com", nil)
	startTime := time.Now()
	for i := 0; i < times; i++ {
		resp, err := handler(req, handlerFunc)
		require.Nil(t, err)
		require.NotNil(t, resp)
		resp, err = handler(req2, handlerFunc)
		require.Nil(t, err)
		require.NotNil(t, resp)
	}
	endTime := time.Now()
	minTakes := int64(wait/time.Millisecond) * int64(times-1)
	maxTakes := int64(wait/time.Millisecond) * int64(times)
	realTakes := int64(endTime.Sub(startTime) / time.Millisecond)
	require.True(t, minTakes <= realTakes && realTakes < maxTakes)
}

func TestRateLimitHandler_ContextCancel(t *testing.T) {
	option := NewRateLimitOption(200)
	option.RateLimitFunc = func(req *http.Request, option RateLimitOption) error {
		return context.Canceled
	}
	handler := RateLimitHandler(option)

	handlerFunc := func(req *http.Request) (resp *http.Response, err error) {
		return &http.Response{
			Body: io.NopCloser(bytes.NewBufferString("hello world")),
		}, nil
	}

	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	resp, err := handler(req, handlerFunc)
	require.True(t, errors.Is(err, context.Canceled))
	require.Nil(t, resp)
}

func TestGetURLStringEndWithPath(t *testing.T) {
	cases := []struct {
		Input  string
		Output string
	}{
		{"", ""},
		{"https://examples.com/healthz?a=b", "https://examples.com/healthz"},
		{"https://examples.com/healthz?a=b#pointer", "https://examples.com/healthz"},
		{"https://username@examples.com/healthz?a=b", "https://examples.com/healthz"},
	}
	for _, c := range cases {
		u, err := url.Parse(c.Input)
		require.Nil(t, err)
		result := getURLStringEndWithPath(u)
		require.Equal(t, c.Output, result)
	}
}
