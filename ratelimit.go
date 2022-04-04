package gohttpclient

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"go.uber.org/ratelimit"
)

// RateLimitConstructor defines the constructor of a rate limiter.
type RateLimitConstructor func() ratelimit.Limiter

// RateLimitFunc enforces the rate limit.
type RateLimitFunc func(req *http.Request, option RateLimitOption) error

// defaultRateLimitFunc gets a request token, and if no token is currently available, it waits.
var defaultRateLimitFunc RateLimitFunc = func(req *http.Request, option RateLimitOption) error {
	key := ""
	if req != nil && req.URL != nil {
		key = fmt.Sprintf("%s %s", req.Method, strings.ToLower(getURLStringEndWithPath(req.URL)))
	}

	val, _ := option.RateLimits.LoadOrStore(key, option.RateLimitConstructor())
	rl := val.(ratelimit.Limiter)
	_ = rl.Take()

	return nil
}

// RateLimitAllRequestsFunc enforces a rate limit, each request is included in the rate limit,
// and it does not distinguish the domain name of the request.
var RateLimitAllRequestsFunc RateLimitFunc = func(req *http.Request, option RateLimitOption) error {
	key := "__all__"

	val, _ := option.RateLimits.LoadOrStore(key, option.RateLimitConstructor())
	rl := val.(ratelimit.Limiter)
	_ = rl.Take()

	return nil
}

// RateLimitOption defines a rate limit option configuration.
type RateLimitOption struct {
	Rate                 int
	RateLimitConstructor RateLimitConstructor
	RateLimits           *sync.Map
	RateLimitFunc        RateLimitFunc
}

func (r RateLimitOption) isEnabled() bool {
	return r.RateLimits != nil
}

// NewRateLimitOption creates a rate limit option configuration.
// The parameter rate defines the maximum number of requests per second.
// If it exceeds maximum times, the excess requests will wait until the next second to execute.
// The requested address needs to be specially explained,
// and the parameters after the link question mark will be omitted.
// Different requested addresses have different capacity of maximum times per second.
// Of course, you can also customize the algorithm
// and stipulate that different domain names use different capacity limits.
func NewRateLimitOption(rate int) RateLimitOption {
	return RateLimitOption{
		Rate: rate,
		RateLimitConstructor: func() ratelimit.Limiter {
			return ratelimit.New(rate)
		},
		RateLimits:    &sync.Map{},
		RateLimitFunc: defaultRateLimitFunc,
	}
}

// RateLimitHandler creates a rate-limiting interceptor that limits the maximum number of requests per second.
func RateLimitHandler(option RateLimitOption) RequestHandler {
	return func(req *http.Request, handlerFunc RequestHandlerFunc) (resp *http.Response, err error) {
		err = option.RateLimitFunc(req, option)
		if err != nil {
			return
		}
		return handlerFunc(req)
	}
}

func getURLStringEndWithPath(u *url.URL) string {
	v := url.URL{
		Scheme:      u.Scheme,
		Opaque:      "",
		User:        nil,
		Host:        u.Host,
		Path:        u.Path,
		RawPath:     u.RawPath,
		ForceQuery:  u.ForceQuery,
		RawQuery:    "",
		Fragment:    "",
		RawFragment: "",
	}
	return v.String()
}
