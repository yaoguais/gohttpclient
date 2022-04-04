package gohttpclient

import (
	"net/http"
	"strconv"

	"github.com/pkg/errors"
)

// BodySizeOption is used to set the maximum size of the server response data.
type BodySizeOption struct {
	MaxBodySize uint64
}

// NewBodySizeOption is used to create an option configuration,
// and the parameter maxBodySize sets the maximum number of bytes of data returned by the server.
// In detail, the restriction is implemented through
// the Content-Length field of the HTTP response header returned by the server.
// The limit can only limit honest servers.
func NewBodySizeOption(maxBodySize uint64) BodySizeOption {
	return BodySizeOption{MaxBodySize: maxBodySize}
}

func (o BodySizeOption) isEnabled() bool {
	return o.MaxBodySize > 0
}

// BodySizeHandler is the interceptor that the server returns the data size limit.
func BodySizeHandler(option BodySizeOption) RequestHandler {
	return func(req *http.Request, handlerFunc RequestHandlerFunc) (resp *http.Response, err error) {
		resp, err = handlerFunc(req)
		if err != nil {
			return
		}

		contentLengthStr := resp.Header.Get("Content-Length")
		contentLength, err := strconv.ParseUint(contentLengthStr, 10, 64)
		if err != nil {
			return nil, errors.Wrap(err, "Parse the data size of the response content")
		}

		if contentLength > option.MaxBodySize {
			return nil, errors.New("The server response data is too large")
		}
		return
	}
}
