package gohttpclient

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChainRequestHandlers(t *testing.T) {
	var result []string

	handler1 := func(req *http.Request, handlerFunc RequestHandlerFunc) (*http.Response, error) {
		result = append(result, "handler1")
		return handlerFunc(req)
	}
	handler2 := func(req *http.Request, handlerFunc RequestHandlerFunc) (*http.Response, error) {
		result = append(result, "handler2")
		return handlerFunc(req)
	}
	handler3 := func(req *http.Request, handlerFunc RequestHandlerFunc) (*http.Response, error) {
		result = append(result, "handler3")
		return handlerFunc(req)
	}
	handlerFunc := func(req *http.Request) (resp *http.Response, err error) {
		result = append(result, "handlerFunc")
		return &http.Response{}, nil
	}

	handler := ChainRequestHandlers(handler1, handler2, handler3)

	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	resp, err := handler(req, handlerFunc)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.Equal(t, []string{"handler1", "handler2", "handler3", "handlerFunc"}, result)
}

func TestChainRequestHandlers_NoHandler(t *testing.T) {
	handler := ChainRequestHandlers()
	require.NotNil(t, handler)
}

func TestChainRequestHandlers_OneHandler(t *testing.T) {
	handler1 := func(req *http.Request, handlerFunc RequestHandlerFunc) (*http.Response, error) {
		return handlerFunc(req)
	}

	handler := ChainRequestHandlers(handler1)
	require.NotNil(t, handler)
}

func TestGetRequestContext(t *testing.T) {
	require.NotNil(t, getRequestContext(nil))
	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	require.NotNil(t, getRequestContext(req))
}
