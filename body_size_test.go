package gohttpclient

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestBodySizeHandler(t *testing.T) {
	option := NewBodySizeOption(10)
	handler := BodySizeHandler(option)

	handlerFunc := func(req *http.Request) (resp *http.Response, err error) {
		responseBody := "hello world"
		return &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Length": []string{strconv.Itoa(len(responseBody))},
			},
			Body: io.NopCloser(bytes.NewBufferString(responseBody)),
		}, nil
	}

	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	resp, err := handler(req, handlerFunc)
	require.NotNil(t, err)
	require.Nil(t, resp)
	require.Equal(t, "The server response data is too large", err.Error())
}

func TestBodySizeHandler_BodySizeIsOK(t *testing.T) {
	option := NewBodySizeOption(11)
	handler := BodySizeHandler(option)

	handlerFunc := func(req *http.Request) (resp *http.Response, err error) {
		responseBody := "hello world"
		return &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Length": []string{strconv.Itoa(len(responseBody))},
			},
			Body: io.NopCloser(bytes.NewBufferString(responseBody)),
		}, nil
	}

	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	resp, err := handler(req, handlerFunc)
	require.Nil(t, err)
	require.NotNil(t, resp)
}

func TestBodySizeHandler_InvalidContentLengthString(t *testing.T) {
	option := NewBodySizeOption(10)
	handler := BodySizeHandler(option)

	handlerFunc := func(req *http.Request) (resp *http.Response, err error) {
		responseBody := "hello world"
		return &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Length": []string{"Foo123"},
			},
			Body: io.NopCloser(bytes.NewBufferString(responseBody)),
		}, nil
	}

	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	resp, err := handler(req, handlerFunc)
	require.NotNil(t, err)
	require.Nil(t, resp)
	require.True(t, strings.HasPrefix(err.Error(), "Parse the data size of the response content"))
}

func TestBodySizeHandler_HandlerFuncError(t *testing.T) {
	option := NewBodySizeOption(10)
	handler := BodySizeHandler(option)

	handlerFunc := func(req *http.Request) (resp *http.Response, err error) {
		return nil, errors.New("response is invalid")
	}

	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	resp, err := handler(req, handlerFunc)
	require.NotNil(t, err)
	require.Nil(t, resp)
	require.True(t, strings.HasPrefix(err.Error(), "response is invalid"))
}
