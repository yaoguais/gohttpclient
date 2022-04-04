package gohttpclient

import (
	"bytes"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
)

func TestCacheHandler(t *testing.T) {
	option := NewMemoryCacheOption()
	option.CacheTTLFunc = func(*http.Request, *http.Response, error) time.Duration {
		return 300 * time.Millisecond
	}

	handler := CacheHandler(option)
	realRequestTimes := 0
	responseHeader := http.Header{"X-Test": []string{"OK"}}
	responseBody := "hello world"
	handlerFunc := func(req *http.Request) (resp *http.Response, err error) {
		realRequestTimes++
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     responseHeader,
			Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
		}, nil
	}

	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	resp, err := handler(req, handlerFunc)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.Equal(t, 1, realRequestTimes)
	require.Equal(t, responseHeader, resp.Header)
	respBody, err := copyHTTPResponseBody(resp)
	require.Nil(t, err)
	require.Equal(t, string(responseBody), string(respBody))

	for i := 0; i < 3; i++ {
		req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
		resp, err := handler(req, handlerFunc)
		require.Nil(t, err)
		require.NotNil(t, resp)
		require.Equal(t, 1, realRequestTimes)
		require.Equal(t, responseHeader, resp.Header)
		respBody, err := copyHTTPResponseBody(resp)
		require.Nil(t, err)
		require.Equal(t, string(responseBody), string(respBody))
	}

	time.Sleep(350 * time.Millisecond)

	req, _ = http.NewRequest(http.MethodGet, "https://example.com", nil)
	resp, err = handler(req, handlerFunc)
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.Equal(t, 2, realRequestTimes)
	require.Equal(t, responseHeader, resp.Header)
	respBody, err = copyHTTPResponseBody(resp)
	require.Nil(t, err)
	require.Equal(t, string(responseBody), string(respBody))
}

func TestRequestEntryEncoderDecoder(t *testing.T) {
	m := requestEntryEncoderDecoder{}

	req1, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	req2, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)

	resp1 := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString("hello world"))}

	es := []RequestEntry{
		{
			Request:  req1,
			Response: resp1,
		},
		{
			Request:  req2,
			Response: nil,
			Error:    errors.New("invalid response"),
		},
	}

	for _, e := range es {
		value, err := m.Encode(e)
		require.Nil(t, err)
		require.NotNil(t, value)

		e2, err := m.Decode(value)
		require.Nil(t, err)
		require.Equal(t, e.Error != nil, e2.Error != nil)
	}
}

func TestRequestEntryEncoderDecoder_EncodeWithInvalidInput(t *testing.T) {
	m := requestEntryEncoderDecoder{}

	req1, _ := http.NewRequest(http.MethodGet, "https://example.com", &testErrReader{})
	req2, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)

	resp1 := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(&testErrReader{})}

	es := []RequestEntry{
		{},
		{
			Request: &http.Request{},
		},
		{
			Request: req1,
		},
		{
			Request:  req2,
			Response: resp1,
		},
	}

	for _, e := range es {
		value, err := m.Encode(e)
		require.NotNil(t, err)
		require.Nil(t, value)
	}
}

func TestRequestEntryEncoderDecoder_DecodeWithInvalidInput(t *testing.T) {
	m := requestEntryEncoderDecoder{}

	re, err := m.Decode(nil)
	require.NotNil(t, err)
	require.Nil(t, re.Request)

	e := HTTPRequestResponse{Method: "()"}
	value, err := msgpack.Marshal(&e)
	require.Nil(t, err)
	require.NotNil(t, value)

	re, err = m.Decode(value)
	require.NotNil(t, err)
	require.Nil(t, re.Request)
}
