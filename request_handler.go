package gohttpclient

import (
	"context"
	"net/http"
)

// RequestHandler defines interceptors for requests.
type RequestHandler func(req *http.Request, handlerFunc RequestHandlerFunc) (resp *http.Response, err error)

// RequestHandlerFunc defines the handler function for request interception.
type RequestHandlerFunc func(req *http.Request) (resp *http.Response, err error)

var noOpRequestHandler RequestHandler = func(req *http.Request, handlerFunc RequestHandlerFunc) (*http.Response, error) {
	return handlerFunc(req)
}

var noOpRequestHandlerFunc RequestHandlerFunc = func(req *http.Request) (resp *http.Response, err error) {
	return &http.Response{}, nil
}

func requestForDoer(doer Doer, handler RequestHandler, req *http.Request) (*http.Response, error) {
	return handler(req, func(curReq *http.Request) (*http.Response, error) {
		return doer.Do(curReq)
	})
}

// ChainRequestHandlers merges multiple interceptors sequentially into a single interceptor.
func ChainRequestHandlers(handlers ...RequestHandler) RequestHandler {
	n := len(handlers)

	if n == 0 {
		return noOpRequestHandler
	}

	if n == 1 {
		return handlers[0]
	}

	return func(req *http.Request, handlerFunc RequestHandlerFunc) (*http.Response, error) {
		currHandlerFunc := handlerFunc
		for i := n - 1; i > 0; i-- {
			innerHandlerFunc, i := currHandlerFunc, i
			currHandlerFunc = func(req *http.Request) (*http.Response, error) {
				return handlers[i](req, innerHandlerFunc)
			}

		}
		return handlers[0](req, currHandlerFunc)
	}
}

func getRequestContext(req *http.Request) context.Context {
	if req != nil {
		return req.Context()
	}
	return context.Background()
}
