package gohttpclient

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cep21/circuit"
	"github.com/cep21/circuit/closers/hystrix"
	"github.com/sirupsen/logrus"
)

// HystrixContructor defines a function pointer to an instance of the circuit breaker.
type HystrixContructor func(req *http.Request, option HystrixOption) *circuit.Circuit

// defaultHystrixContructor is the default implementation of circuit breaker,
// and different domain names use different circuit breaker instances.
// The isolation strategy makes requests for different domain names not affect each other.
var defaultHystrixContructor HystrixContructor = func(req *http.Request, option HystrixOption) *circuit.Circuit {
	name := ""
	if req != nil && req.URL != nil {
		name = strings.ToLower(getURLStringEndWithHost(req.URL))
	}

	c := option.CircuitManager.GetCircuit(name)
	if c != nil {
		return c
	}
	c, err := option.CircuitManager.CreateCircuit(name)
	if err != nil { // Error: circuit with that name already exists
		c = option.CircuitManager.GetCircuit(name)
	}
	return c
}

var defaultHystrixFactory = hystrix.Factory{
	ConfigureOpener: hystrix.ConfigureOpener{
		RequestVolumeThreshold:   20,
		ErrorThresholdPercentage: 50,
	},
	ConfigureCloser: hystrix.ConfigureCloser{
		SleepWindow:                  5 * time.Second,
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

// HystrixOption is an option configuration for the circuit breaker.
type HystrixOption struct {
	CircuitManager    *circuit.Manager
	HystrixContructor HystrixContructor
}

// NewHystrixOption creates an option configuration for a circuit breaker.
// Circuit breakers use the Hystrix Pattern,
// which is a complex concept and requires a lot of effort to understand.
// The circuit breaker is divided into two states: closed circuit and open circuit.
// It is closed at first, but if there are many request failures,
// it will become open. In the open-circuit state,
// no more requests will be initiated,
// and after a period of time, the request will be passed to test whether the link is normal.
// If it is normal, it will become a closed state, and the request will be unblocked.
// The default settings are to request 20 times, and if half of them fail,
// it becomes an open-circuit state.
// Then wait for 5 seconds, then retries 1 time,
// and if successful, it changes back to the closed-circuit state.
func NewHystrixOption() HystrixOption {
	return HystrixOption{
		CircuitManager:    defaultCircuitManager,
		HystrixContructor: defaultHystrixContructor,
	}
}

func (h HystrixOption) isEnabled() bool {
	return h.HystrixContructor != nil && h.CircuitManager != nil
}

// HystrixHandler implements a circuit breaker interceptor.
func HystrixHandler(option HystrixOption) RequestHandler {
	return func(req *http.Request, handlerFunc RequestHandlerFunc) (resp *http.Response, err error) {
		c := option.HystrixContructor(req, option)
		err = c.Execute(getRequestContext(req), func(_ctx context.Context) error {
			resp, err = handlerFunc(req)
			return err
		}, func(_ctx context.Context, err error) error {
			return err
		})
		return
	}
}

func getURLStringEndWithHost(u *url.URL) string {
	v := url.URL{
		Scheme:      u.Scheme,
		Opaque:      "",
		User:        nil,
		Host:        u.Host,
		Path:        "",
		RawPath:     "",
		ForceQuery:  false,
		RawQuery:    "",
		Fragment:    "",
		RawFragment: "",
	}
	return v.String()
}
