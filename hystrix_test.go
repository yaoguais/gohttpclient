package gohttpclient

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/cep21/circuit"
	"github.com/cep21/circuit/closers/hystrix"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestHystrixHandler(t *testing.T) { //revive:disable:cyclomatic
	// Test requests for the same domain name, first succeed 50 times,
	// then error 50 times, repeat 3 times

	option := NewHystrixOption()
	option.CircuitManager = getTestCircuitManager()
	handler := HystrixHandler(option)

	requestTimes := 0
	errResponseTimes := errors.New("requestTimes error")

	handlerFunc := func(req *http.Request) (resp *http.Response, err error) {
		requestTimes++
		ok := (requestTimes >= 1 && requestTimes <= 50) ||
			(requestTimes >= 101 && requestTimes <= 150) ||
			(requestTimes >= 201 && requestTimes <= 250)
		if !ok {
			return nil, errResponseTimes
		}
		return &http.Response{
			Body: io.NopCloser(bytes.NewBufferString("hello world")),
		}, nil
	}

	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	for i := 1; i <= 119; i++ {
		resp, err := handler(req, handlerFunc)
		if i >= 1 && i <= 50 {
			require.Nilf(t, err, "#%d", i)
			require.NotNilf(t, resp, "#%d", i)
		} else if i >= 51 && i <= 100 {
			require.Equalf(t, errResponseTimes, err, "#%d", i)
			require.Nilf(t, resp, "#%d", i)
		} else if i >= 101 && i <= 119 {
			require.NotNilf(t, err, "#%d", i)
			require.Equalf(t, "circuit is open: concurrencyReached=false circuitOpen=true", err.Error(), "#%d", i)
			require.Nilf(t, resp, "#%d", i)
			// After the open circuit, there is no actual response, but the number of requests should be accumulated
			requestTimes++
		}
	}

	time.Sleep(500 * time.Millisecond)

	for i := 120; i <= 300; i++ {
		resp, err := handler(req, handlerFunc)
		if i == 120 {
			// This time the request is attempted, and the circuit breaker is closed when successful.
			// However, this request does not record the total number of successful requests
			// and clears the number of incorrect requests and successful requests
			require.Nilf(t, err, "#%d", i)
			require.NotNilf(t, resp, "#%d", i)
		} else if i >= 121 && i <= 150 {
			require.Nilf(t, err, "#%d", i)
			require.NotNilf(t, resp, "#%d", i)
		} else if i >= 151 && i <= 180 {
			require.Equalf(t, errResponseTimes, err, "#%d", i)
			require.Nilf(t, resp, "#%d", i)
		} else if i >= 180 && i <= 300 {
			require.NotNilf(t, err, "#%d", i)
			require.Equalf(t, "circuit is open: concurrencyReached=false circuitOpen=true", err.Error(), "#%d", i)
			require.Nilf(t, resp, "#%d", i)
			// After the open circuit, there is no actual response, but the number of requests should be accumulated
			requestTimes++
		}
	}
}

func TestGetURLStringEndWithHost(t *testing.T) {
	cases := []struct {
		Input  string
		Output string
	}{
		{"", ""},
		{"https://examples.com/healthz?a=b", "https://examples.com"},
		{"https://examples.com/healthz?a=b#pointer", "https://examples.com"},
		{"https://username@examples.com/healthz?a=b", "https://examples.com"},
	}
	for _, c := range cases {
		u, err := url.Parse(c.Input)
		require.Nil(t, err)
		result := getURLStringEndWithHost(u)
		require.Equal(t, c.Output, result)
	}
}

func getTestCircuitManager() *circuit.Manager {
	var defaultHystrixFactory = hystrix.Factory{
		ConfigureOpener: hystrix.ConfigureOpener{
			RequestVolumeThreshold:   20,
			ErrorThresholdPercentage: 50,
		},
		ConfigureCloser: hystrix.ConfigureCloser{
			SleepWindow:                  300 * time.Millisecond,
			HalfOpenAttempts:             1,
			RequiredConcurrentSuccessful: 1,
		},
	}

	var defaultCircuitManager = &circuit.Manager{
		DefaultCircuitProperties: []circuit.CommandPropertiesConstructor{
			defaultHystrixFactory.Configure,
			func(_circuitName string) circuit.Config {
				return circuit.Config{
					General: circuit.GeneralConfig{
						GoLostErrors: func(err error, panics interface{}) {
							logrus.WithError(err).WithField("panic", panics).Warn("gohttpclient hystrix lost errros")
						},
					},
					Execution: circuit.ExecutionConfig{
						Timeout:               -1,
						MaxConcurrentRequests: -1,
					},
					Fallback: circuit.FallbackConfig{
						MaxConcurrentRequests: -1,
					},
					Metrics: circuit.MetricsCollectors{},
				}
			},
		},
	}

	return defaultCircuitManager
}
