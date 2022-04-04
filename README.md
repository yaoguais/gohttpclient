# Go HTTP Client

![Build Status](https://github.com/yaoguais/gohttpclient/actions/workflows/ci.yml/badge.svg)
[![codecov](https://codecov.io/gh/yaoguais/gohttpclient/branch/main/graph/badge.svg?token=11OX1Yevrr)](https://codecov.io/gh/yaoguais/gohttpclient)
[![Go Report Card](https://goreportcard.com/badge/github.com/yaoguais/gohttpclient)](https://goreportcard.com/report/github.com/yaoguais/gohttpclient)
[![GoDoc](https://pkg.go.dev/badge/github.com/yaoguais/gohttpclient?status.svg)](https://pkg.go.dev/github.com/yaoguais/gohttpclient?tab=doc)

An HTTP client package that is 100% compatible with the official library, and provides functions such as retry, rate limit, circuit breaker, cache, log, and trace.
This package can be used as a basic toolkit for a microservice framework with HTTP requests as a carrier, 
or as a more secure library to limit the size of concurrent requests and downloaded data.

## Installation

Use go get.

```sh
$ go get -u github.com/yaoguais/gohttpclient
```

Then import the package into your own code.

```go
import 	"github.com/yaoguais/gohttpclient"
```

## Quick start

```go
package main

import (
	"github.com/yaoguais/gohttpclient"
)

func main() {
	c := gohttpclient.NewClient()
	resp, err := c.Get("http://examples.com/ping")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
}
```

## Extensions

- [Retry mechanism for requests](#retry-mechanism-for-requests)
- [Mechanism for requesting traffic throttling](#mechanism-for-requesting-traffic-throttling)
- [Circuit breaker mechanism to prevent avalanches](#circuit-breaker-mechanism-to-prevent-avalanches)
- [Cache and reuse client request and response content](#cache-and-reuse-client-request-and-response-content)
- [Recording of client request logs](#recording-of-client-request-logs)
- [Distributed tracing and analysis of cross-process transactions](#distributed-tracing-and-analysis-of-cross-process-transactions)
- [Customize the official HTTP client instance](#customize-the-official-http-client-instance)
- [Limit the timeout period for requests](#limit-the-timeout-period-for-requests)
- [Limit the size of the client download response content](#limit-the-size-of-the-client-download-response-content)


### Retry mechanism for requests

```go
package main

import "github.com/cenkalti/backoff/v4"

func main() {
	// Set the request to repeat up to 10 times when it fails,
	// and retry 1 time per second interval.
	// The default algorithm for judging retry is that the
	// HTTP status code is greater than or equal to 500 before retrying.
	// Or you can use gohttpclient.WithShouldRetryFunc() to set the algorithm to judge the retry by yourself.
	c := gohttpclient.NewClient(
		gohttpclient.WithMaxRetry(10),
		gohttpclient.WithRetryBackOff(backoff.NewConstantBackOff(time.Second)),
	)
	c.Get("http://examples.com/ping")
	// You can also choose to use the Exponential backoff algorithm.
	// Exponential backoff is an algorithm that uses feedback to multiplicatively decrease the rate of some process,
	// in order to gradually find an acceptable rate.
	c = gohttpclient.NewClient(
		gohttpclient.WithMaxRetry(10),
		gohttpclient.WithRetryBackOff(backoff.NewExponentialBackOff()),
	)
}
```

### Mechanism for requesting traffic throttling

```go
package main

import "github.com/yaoguais/gohttpclient"

func main() {
	// For the requested address, a maximum of 10 concurrent requests are made per second.
	// If it exceeds 10 times, the excess requests will wait until the next second to execute.
	// The requested address needs to be specially explained,
	// and the parameters after the link question mark will be omitted.
	// Different requested addresses have different capacity of 10 times per second.
	// Of course, you can also customize the algorithm
	// and stipulate that different domain names use different capacity limits.
	option := gohttpclient.NewRateLimitOption(10)
	c := gohttpclient.NewClient(
		gohttpclient.WithRateLimitOption(option),
	)
	c.Get("http://examples.com/ping")
}
```

### Circuit breaker mechanism to prevent avalanches

```go
package main

import "github.com/yaoguais/gohttpclient"

func main() {
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
	option := gohttpclient.NewHystrixOption()
	c := gohttpclient.NewClient(
		gohttpclient.WithHystrixOption(option),
	)
	c.Get("http://examples.com/ping")
}
```

### Cache and reuse client request and response content

```go
package main

import "github.com/yaoguais/gohttpclient"

func main() {
	// The cache function will save the content of the response,
	// such as saving to memory, file, Redis, etc.
	// The next time you initiate the same request,
	// you don't need to actually execute the request, but extract it from the cache.
	// By default, only successful requests with HTTP method GET
	// and status code 200 will be cached for 5 minutes.
	// The same complete request link will be treated as the same request and may be cached.
	option := gohttpclient.NewMemoryCacheOption()
	c := gohttpclient.NewClient(
		gohttpclient.WithCacheOption(option),
	)
	c.Get("http://examples.com/ping")
}
```

### Recording of client request logs

```go
package main

import "github.com/yaoguais/gohttpclient"

func main() {
	// The log function can record the context of the request,
	// such as request content, response content, response status code, request time, etc.
	// All contexts are logged by default, and you can also close them yourself.
	option := gohttpclient.NewLoggerOption()
	c := gohttpclient.NewClient(
		gohttpclient.WithLoggerOption(option),
	)
	c.Get("http://examples.com/ping")
}
```

Of course, you can also set the output in JSON format,

```go
func main() {
	l := logrus.New()
	l.SetFormatter(&logrus.JSONFormatter{})
	logger := logrus.NewEntry(l)

	option := gohttpclient.NewLoggerOption()
	option.Logger = logger
	c := gohttpclient.NewClient(
		gohttpclient.WithLoggerOption(option),
	)
	c.Get("http://examples.com/ping")
}
```

and the format of the output content is as follows

```json
{
  "executeTime": "2.073556152s",
  "executeTimeMs": 2073,
  "level": "error",
  "method": "GET",
  "msg": "http client request",
  "requestBody": "",
  "requestHeader": {},
  "responseBody": "\r\n<!DOCTYPE html>...",
  "responseHeader": {},
  "statusCode": 404,
  "time": "1970-01-01T00:00:00+00:00",
  "url": "http://examples.com/ping"
}
```

### Distributed tracing and analysis of cross-process transactions

```go
package main

import (
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
	"github.com/yaoguais/gohttpclient"
)

func main() {
	// Distributed tracing integrates with the Jaeger project,
	// you can visit jaegertracing.io for more information.
	// First we set the sampling frequency,
	// and the server address where the tracking data is stored.
	cfg := &config.Configuration{
		Sampler: &config.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans:           true,
			LocalAgentHostPort: "localhost:6831",
		},
	}

	// Then we create a Tracer instance and set it with global default options.
	tracer, closer, err := cfg.New("serviceName", config.Logger(jaeger.StdLogger))
	if err != nil {
		panic(err)
	}
	defer closer.Close()
	opentracing.SetGlobalTracer(tracer)

	// Finally, use our client to initiate a request,
	// and the entire call chain will be connected in series through Jaeger.
	option := gohttpclient.NewTraceOption()
	c := gohttpclient.NewClient(
		gohttpclient.WithTraceOption(option),
	)
	c.Get("http://examples.com/ping")
}
```

### Customize the official HTTP client instance

```go
package main

import "net/http"
import "github.com/yaoguais/gohttpclient"

func main() {
	client := &http.Client{}
	c := gohttpclient.NewClient(
		gohttpclient.WithHTTPClient(client),
	)
	c.Get("http://examples.com/ping")
}
```

### Limit the timeout period for requests

```go
package main

import "time"
import "github.com/yaoguais/gohttpclient"

func main() {
	c := gohttpclient.NewClient(
		gohttpclient.WithRequestTimeout(5 * time.Second),
	)
	c.Get("http://examples.com/ping")
}
```

### Limit the size of the client download response content

```go
package main

import "github.com/yaoguais/gohttpclient"

func main() {
	// Set a limit to the size of the data returned by the client to no more than 10MB.
	// In detail, the restriction is implemented through
	// the Content-Length field of the HTTP response header returned by the server.
	// The limit can only limit honest servers.
	c := gohttpclient.NewClient(
		gohttpclient.WithMaxBodySize(10 * 1024 * 1024),
	)
	c.Get("http://examples.com/ping")
}
```

### License

    Copyright 2013 Mir Ikram Uddin

    Licensed under the Apache License, Version 2.0 (the "License");
    you may not use this file except in compliance with the License.
    You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

    Unless required by applicable law or agreed to in writing, software
    distributed under the License is distributed on an "AS IS" BASIS,
    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
    See the License for the specific language governing permissions and
    limitations under the License.
