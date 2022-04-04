package gohttpclient

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ClientTestSuite struct {
	suite.Suite
	done chan bool
	addr string
	url  string
}

func (suite *ClientTestSuite) SetupSuite() {
	suite.done = make(chan bool)
	suite.addr = ":19998"
	path := "/client"
	suite.url = fmt.Sprintf("http://localhost%s%s", suite.addr, path)

	handlerFunc := func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			panic(err)
		}
		fmt.Fprint(w, r.Form.Encode())
	}

	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc(path, handlerFunc)
		srv := &http.Server{Addr: suite.addr, Handler: mux}
		go func() {
			<-suite.done
			srv.Close()
		}()
		srv.ListenAndServe()
	}()
}

func (suite *ClientTestSuite) TearDownSuite() {
	close(suite.done)
}

func (suite *ClientTestSuite) TestNewClient() {
	t := suite.T()
	c := NewClient()
	require.NotNil(t, c)
}

func (suite *ClientTestSuite) TestNewClient_WithRetry() {
	t := suite.T()
	c := NewClient(
		WithMaxRetry(5),
		WithRetryBackOff(backoff.NewConstantBackOff(time.Second)),
		WithShouldRetryFunc(func(req *http.Request, resp *http.Response, err error) bool {
			return !(err == nil && resp != nil && resp.StatusCode < 500)
		}),
	)
	require.NotNil(t, c)
}

func (suite *ClientTestSuite) TestNewClient_WithLogger() {
	t := suite.T()
	c := NewClient(WithLoggerOption(NewLoggerOption()))
	require.NotNil(t, c)
}

func (suite *ClientTestSuite) TestNewClient_WithRateLimit() {
	t := suite.T()
	c := NewClient(WithRateLimitOption(NewRateLimitOption(10)))
	require.NotNil(t, c)
}

func (suite *ClientTestSuite) TestNewClient_WithHystrix() {
	t := suite.T()
	c := NewClient(WithHystrixOption(NewHystrixOption()))
	require.NotNil(t, c)
}

func (suite *ClientTestSuite) TestNewClient_WithTrace() {
	t := suite.T()
	c := NewClient(WithTraceOption(NewTraceOption()))
	require.NotNil(t, c)
}

func (suite *ClientTestSuite) TestNewClient_WithCache() {
	t := suite.T()
	c := NewClient(WithCacheOption(NewMemoryCacheOption()))
	require.NotNil(t, c)
}

func (suite *ClientTestSuite) TestGet() {
	t := suite.T()
	query := "foo=bar&foo2=bar2"
	uri := fmt.Sprintf("%s?%s", suite.url, query)

	fns := []func() (*http.Response, error){
		func() (*http.Response, error) {
			return NewClient().Get(uri)
		},
		func() (*http.Response, error) {
			return Get(uri)
		},
	}
	for _, fn := range fns {
		resp, err := fn()
		require.Nil(t, err)
		require.NotNil(t, resp)
		respBody, _ := io.ReadAll(resp.Body)
		require.Equal(t, query, string(respBody))
	}
}

func (suite *ClientTestSuite) TestPost() {
	t := suite.T()
	query := "foo=bar&foo2=bar2"

	fns := []func() (*http.Response, error){
		func() (*http.Response, error) {
			return NewClient().Post(suite.url, "application/x-www-form-urlencoded", strings.NewReader(query))
		},
		func() (*http.Response, error) {
			return Post(suite.url, "application/x-www-form-urlencoded", strings.NewReader(query))
		},
	}
	for _, fn := range fns {
		resp, err := fn()
		require.Nil(t, err)
		require.NotNil(t, resp)
		respBody, _ := io.ReadAll(resp.Body)
		require.Equal(t, query, string(respBody))
	}
}

func (suite *ClientTestSuite) TestPostForm() {
	t := suite.T()
	values := url.Values{
		"foo":  []string{"bar"},
		"foo2": []string{"bar2"},
	}
	query := values.Encode()

	fns := []func() (*http.Response, error){
		func() (*http.Response, error) {
			return NewClient().PostForm(suite.url, values)
		},
		func() (*http.Response, error) {
			return PostForm(suite.url, values)
		},
	}
	for _, fn := range fns {
		resp, err := fn()
		require.Nil(t, err)
		require.NotNil(t, resp)
		respBody, _ := io.ReadAll(resp.Body)
		require.Equal(t, query, string(respBody))
	}
}

func (suite *ClientTestSuite) TestHead() {
	t := suite.T()
	fns := []func() (*http.Response, error){
		func() (*http.Response, error) {
			return NewClient().Head(suite.url)
		},
		func() (*http.Response, error) {
			return Head(suite.url)
		},
	}
	for _, fn := range fns {
		resp, err := fn()
		require.Nil(t, err)
		require.NotNil(t, resp)
	}
}

func (suite *ClientTestSuite) TestClient_InvalidURL() {
	t := suite.T()
	fns := []func() (*http.Response, error){
		func() (*http.Response, error) {
			return NewClient().Get("😭://")
		},
		func() (*http.Response, error) {
			return NewClient().Post("😭://", "application/x-www-form-urlencoded", nil)
		},
		func() (*http.Response, error) {
			return NewClient().Head("😭://")
		},
		func() (*http.Response, error) {
			return Get("😭://")
		},
		func() (*http.Response, error) {
			return Post("😭://", "application/x-www-form-urlencoded", nil)
		},
		func() (*http.Response, error) {
			return Head("😭://")
		},
	}
	for _, fn := range fns {
		resp, err := fn()
		require.NotNil(t, err)
		require.Nil(t, resp)
	}
}

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}
