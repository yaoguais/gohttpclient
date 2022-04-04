package gohttpclient

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestLoggerRequestHander(t *testing.T) {
	var resultEntry LoggerEntry
	option := NewLoggerOption()
	option.LoggerFunc = func(req *http.Request, e LoggerEntry, option LoggerOption) {
		resultEntry = e
	}
	handler := LoggerHandler(option)

	handlerFunc := func(req *http.Request) (resp *http.Response, err error) {
		return &http.Response{
			StatusCode: 200,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewBufferString("hello world")),
		}, nil
	}

	url := "https://example.com"
	query := "foo=bar&foo2=bar2"
	req, _ := http.NewRequest(http.MethodPost, url, strings.NewReader(query))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := handler(req, handlerFunc)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.Equal(t, http.MethodPost, resultEntry.Method)
	require.Equal(t, url, resultEntry.URL)
	require.Equal(t, http.Header{"Content-Type": []string{"application/x-www-form-urlencoded"}}, resultEntry.RequestHeader)
	require.Equal(t, query, string(resultEntry.RequestBody))
	require.Equal(t, http.Header{"Content-Type": []string{"application/json"}}, resultEntry.ResponseHeader)
	require.Equal(t, "hello world", string(resultEntry.ResponseBody))
	require.Equal(t, 200, resultEntry.StatusCode)
	require.True(t, resultEntry.ExecuteTime > 0)
	require.True(t, resultEntry.StartTime.UnixNano() > 0)
}

type testErrReader struct{}

func (testErrReader) Read([]byte) (n int, err error) {
	return 0, errors.New("error found")
}

func TestCopyHTTPBody_ReadError(t *testing.T) {
	fns := []func() ([]byte, error){
		func() ([]byte, error) {
			return copyHTTPRequestBody(&http.Request{Body: io.NopCloser(&testErrReader{})})
		},
		func() ([]byte, error) {
			return copyHTTPResponseBody(&http.Response{Body: io.NopCloser(&testErrReader{})})
		},
	}
	for _, fn := range fns {
		_, err := fn()
		require.NotNil(t, err)
	}
}

func TestCopyHTTPHeader(t *testing.T) {
	require.Nil(t, copyHTTPHeader(nil))
	h := copyHTTPHeader(http.Header{"Foo": []string{"bar"}})
	require.Equal(t, HTTPHeader{"Foo": "bar"}, h)
}
