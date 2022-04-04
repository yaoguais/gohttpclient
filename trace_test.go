package gohttpclient

import (
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/opentracing-contrib/go-stdlib/nethttp"
	"github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
	"github.com/uber/jaeger-client-go/log"
	"github.com/uber/jaeger-lib/metrics"
)

type TraceTestSuite struct {
	suite.Suite
	done chan bool
	addr string
	url  string
}

func (suite *TraceTestSuite) SetupSuite() {
	suite.done = make(chan bool)
	suite.addr = ":19999"
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

func (suite *TraceTestSuite) TearDownSuite() {
	close(suite.done)
}

func (suite *TraceTestSuite) TestTraceHandler() {
	t := suite.T()

	tracer, closer, err := getTestTracer()
	require.Nil(t, nil)
	require.NotNil(t, closer)
	defer closer.Close()

	option := NewTraceOption()
	option.Tracer = tracer
	handler := TraceHandler(option)

	req, err := http.NewRequest(http.MethodGet, suite.url, nil)
	require.Nil(t, err)

	hc := &http.Client{Transport: &nethttp.Transport{}}

	resp, err := handler(req, hc.Do)
	require.Nil(t, err)
	require.NotNil(t, resp)
	traceID := req.Header.Get("Uber-Trace-Id")
	require.NotEmpty(t, traceID)
}

func TestTraceTestSuite(t *testing.T) {
	suite.Run(t, new(TraceTestSuite))
}

func getTestTracer() (opentracing.Tracer, io.Closer, error) {
	cfg := config.Configuration{
		Sampler: &config.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans: true,
		},
	}

	return cfg.New(
		"test",
		config.Logger(log.StdLogger),
		config.Metrics(metrics.NullFactory),
	)
}
