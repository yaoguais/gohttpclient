package gohttpclient

import (
	"fmt"
	"net/http"

	"github.com/opentracing-contrib/go-stdlib/nethttp"
	"github.com/opentracing/opentracing-go"
)

// TraceComponentNameFunc defines a function that gets the name of the tracking component by request.
type TraceComponentNameFunc func(req *http.Request) string

// DefaultTraceComponentNameFunc is a default function that defines the name of a tracked component by request.
var DefaultTraceComponentNameFunc TraceComponentNameFunc = func(req *http.Request) string {
	if req == nil || req.URL == nil {
		return "HTTP NULL"
	}
	return fmt.Sprintf("HTTP %s %s", req.Method, req.URL.Path)
}

// TraceOption defines an option configuration for distributed tracing.
type TraceOption struct {
	Enabled               bool
	Tracer                opentracing.Tracer
	ComponentName         string
	ComponentNameFunc     TraceComponentNameFunc
	ClientConnectionTrace bool
}

// NewTraceOption creates a new option configuration for distributed tracing.
// Distributed tracing integrates with the Jaeger project,
// you can visit jaegertracing.io for more information.
func NewTraceOption() TraceOption {
	return TraceOption{
		Enabled:               true,
		Tracer:                opentracing.GlobalTracer(),
		ComponentName:         "HTTP Client",
		ComponentNameFunc:     DefaultTraceComponentNameFunc,
		ClientConnectionTrace: false,
	}
}

func (t TraceOption) isEnabled() bool {
	return t.Enabled
}

// TraceHandler creates a distributed tracing interceptor that can record and display call chain information through opentracing.
func TraceHandler(option TraceOption) RequestHandler {
	return func(req *http.Request, handlerFunc RequestHandlerFunc) (resp *http.Response, err error) {
		opts := []nethttp.ClientOption{
			nethttp.ComponentName(option.ComponentName),
			nethttp.OperationName(option.ComponentNameFunc(req)),
			nethttp.ClientTrace(option.ClientConnectionTrace),
		}

		req, ht := nethttp.TraceRequest(option.Tracer, req, opts...)
		defer ht.Finish()

		return handlerFunc(req)
	}
}
