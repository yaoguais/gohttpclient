package gohttpclient

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	defaultLogMessage = "http client request"
	defaultLogger     = logrus.NewEntry(logrus.StandardLogger())
)

// LoggerFunc defines a function for logging.
type LoggerFunc func(req *http.Request, e LoggerEntry, option LoggerOption)

var defaultLoggerFunc LoggerFunc = func(req *http.Request, e LoggerEntry, option LoggerOption) {
	fields := logrus.Fields{
		"method":         e.Method,
		"url":            e.URL,
		"requestHeader":  copyHTTPHeader(e.RequestHeader),
		"requestBody":    string(e.RequestBody),
		"responseHeader": copyHTTPHeader(e.ResponseHeader),
		"responseBody":   string(e.ResponseBody),
		"statusCode":     e.StatusCode,
		"executeTime":    e.ExecuteTime.String(),
		"executeTimeMs":  e.ExecuteTime.Milliseconds(),
	}
	if e.StatusCode < 400 {
		option.Logger.WithFields(fields).Info(option.LogMessage)
		return
	}
	option.Logger.WithFields(fields).Error(option.LogMessage)
}

// LoggerOption is an option configuration for logging.
type LoggerOption struct {
	LogMessage        string
	LogRequestHeader  bool
	LogRequestBody    bool
	LogResponseHeader bool
	LogResponseBody   bool
	Logger            *logrus.Entry
	LoggerFunc        LoggerFunc
}

// HTTPHeader holds HTTP request and response headers.
type HTTPHeader map[string]string

// LoggerEntry is the entry that records the request context.
type LoggerEntry struct {
	Method         string
	URL            string
	RequestHeader  http.Header
	RequestBody    []byte
	ResponseHeader http.Header
	ResponseBody   []byte
	StatusCode     int
	ExecuteTime    time.Duration
	StartTime      time.Time
}

// NewLoggerOption creates a log option configuration.
// By default it will record the request body and the response body,
// which will have a certain performance loss, you can choose to turn it off.
func NewLoggerOption() LoggerOption {
	return LoggerOption{
		LogRequestHeader:  true,
		LogRequestBody:    true,
		LogResponseHeader: true,
		LogResponseBody:   true,
		LogMessage:        defaultLogMessage,
		Logger:            defaultLogger,
		LoggerFunc:        defaultLoggerFunc,
	}
}

func (o LoggerOption) isEnabled() bool {
	return o.Logger != nil
}

// LoggerHandler implements a logging interceptor that logs the request context.
func LoggerHandler(option LoggerOption) RequestHandler {
	return func(req *http.Request, handlerFunc RequestHandlerFunc) (resp *http.Response, err error) {
		startTime := time.Now()
		resp, err = handlerFunc(req)

		entry, loggerErr := getLoggerEntry(req, resp, option, startTime)
		if loggerErr != nil {
			logrus.WithError(loggerErr).Warn("gohttpclient build logger entry")
			return
		}

		option.LoggerFunc(req, entry, option)
		return
	}
}

func getLoggerEntry(req *http.Request, resp *http.Response, option LoggerOption, startTime time.Time) (entry LoggerEntry, err error) {
	if req == nil {
		err = errors.New("http.Request is nil")
		return
	}

	entry = LoggerEntry{
		Method:      req.Method,
		URL:         req.URL.String(),
		StartTime:   startTime,
		ExecuteTime: time.Now().Sub(startTime),
	}

	if option.LogRequestHeader {
		entry.RequestHeader = req.Header
	}

	if option.LogRequestBody && req != nil && req.Body != nil {
		entry.RequestBody, err = copyHTTPRequestBody(req)
		if err != nil {
			return
		}
	}

	if option.LogResponseHeader && resp != nil {
		entry.ResponseHeader = resp.Header
	}

	if option.LogResponseBody && resp != nil && resp.Body != nil {
		entry.ResponseBody, err = copyHTTPResponseBody(resp)
		if err != nil {
			return
		}
	}

	if resp != nil {
		entry.StatusCode = resp.StatusCode
	}

	return entry, nil
}

func copyHTTPRequestBody(req *http.Request) ([]byte, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	req.Body = io.NopCloser(bytes.NewBuffer(body))
	return body, nil
}

func copyHTTPResponseBody(resp *http.Response) ([]byte, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	resp.Body = io.NopCloser(bytes.NewBuffer(body))
	return body, nil
}

func copyHTTPHeader(h http.Header) HTTPHeader {
	if h == nil {
		return nil
	}
	m := make(HTTPHeader)
	for k := range h {
		m[k] = h.Get(k)
	}
	return m
}
