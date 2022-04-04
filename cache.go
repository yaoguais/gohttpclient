package gohttpclient

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/vmihailenco/msgpack/v5"
)

// ShouldCacheFunc is a function pointer to determine whether a request needs to be cached.
type ShouldCacheFunc func(*http.Request, *http.Response, error) bool

// CacheTTLFunc can configure different cache times for different requests.
type CacheTTLFunc func(*http.Request, *http.Response, error) time.Duration

// RequestHashFunc generates a hash value based on the context of the request as a cache key.
type RequestHashFunc func(*http.Request, *http.Response, error) []byte

// DefaultShouldCacheFunc is a function implemented by default to determine whether a request needs to be cached.
// By default, only successful requests with HTTP method GET
// and status code 200 will be cached for 5 minutes.
// The same complete request link will be treated as the same request and may be cached.
var DefaultShouldCacheFunc ShouldCacheFunc = func(req *http.Request, resp *http.Response, err error) bool {
	ok := req != nil && req.URL != nil && req.Method == http.MethodGet &&
		resp != nil && resp.StatusCode == http.StatusOK && err == nil
	return ok
}

// DefaultRequestHashFunc is a function implemented by default to generate different hash values as cache keys according to different requests.
var DefaultRequestHashFunc RequestHashFunc = func(req *http.Request, resp *http.Response, err error) []byte {
	ok := req != nil && req.URL != nil && req.Method == http.MethodGet
	if !ok {
		return nil
	}

	bv := []byte(req.URL.String())
	hasher := sha1.New()
	hasher.Write(bv)
	sha := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

	return []byte(sha)
}

// DefaultCacheTTLFunc is the default implemented function that sets the cache time based on the request context.
// By default, it caches all requests that need to be cached for 5 minutes.
var DefaultCacheTTLFunc CacheTTLFunc = func(*http.Request, *http.Response, error) time.Duration {
	return 5 * time.Minute
}

// CacheOption is the options structure that sets the cache.
type CacheOption struct {
	ShouldCacheFunc ShouldCacheFunc
	RequestHashFunc RequestHashFunc
	CacheTTLFunc    CacheTTLFunc
	Cacher          Cacher
	EncoderDecoder  RequestEntryEncoderDecoder
}

// NewCacheOption creates a new cache option and passes in a cache method.
// The cache function will save the content of the response,
// such as saving to memory, file, Redis, etc.
// The next time you initiate the same request,
// you don't need to actually execute the request, but extract it from the cache.
func NewCacheOption(cacher Cacher) CacheOption {
	return CacheOption{
		ShouldCacheFunc: DefaultShouldCacheFunc,
		RequestHashFunc: DefaultRequestHashFunc,
		CacheTTLFunc:    DefaultCacheTTLFunc,
		Cacher:          cacher,
		EncoderDecoder:  requestEntryEncoderDecoder{},
	}
}

// NewMemoryCacheOption creates a new cached option and caches the request and response data in memory.
func NewMemoryCacheOption() CacheOption {
	return NewCacheOption(NewMemoryCache())
}

func (o CacheOption) isEnabled() bool {
	return o.ShouldCacheFunc != nil && o.RequestHashFunc != nil &&
		o.CacheTTLFunc != nil && o.Cacher != nil && o.EncoderDecoder != nil
}

// CacheHandler is a cache interceptor that caches request content and server-side response content.
func CacheHandler(option CacheOption) RequestHandler {
	return func(req *http.Request, handlerFunc RequestHandlerFunc) (resp *http.Response, returnErr error) {
		hash := option.RequestHashFunc(req, nil, nil)
		if hash != nil {
			cacheValue, err := option.Cacher.Get(hash)
			if err == nil {
				re, err := option.EncoderDecoder.Decode(cacheValue)
				if err == nil {
					return re.Response, re.Error
				}
			}
		}

		resp, returnErr = handlerFunc(req)

		shouldCache := option.ShouldCacheFunc(req, resp, returnErr)
		if !shouldCache {
			return
		}

		hash = option.RequestHashFunc(req, resp, returnErr)
		if hash == nil {
			return
		}

		re := RequestEntry{
			Request:  req,
			Response: resp,
			Error:    returnErr,
		}
		cacheValue, err := option.EncoderDecoder.Encode(re)
		if err != nil {
			return nil, errors.Wrap(err, "Serialization request")
		}

		ttl := option.CacheTTLFunc(req, resp, returnErr)
		_ = option.Cacher.Set(hash, cacheValue, ttl)
		return
	}
}

// RequestEntry is a structure that stores the request context.
type RequestEntry struct {
	Request  *http.Request
	Response *http.Response
	Error    error
}

// RequestEntryEncoderDecoder is an interface to serialize and deserialize the request context.
type RequestEntryEncoderDecoder interface {
	Encode(entry RequestEntry) ([]byte, error)
	Decode([]byte) (RequestEntry, error)
}

// HTTPRequestResponse is an intermediate temporary structure for the request context.
type HTTPRequestResponse struct {
	Method         string
	URL            string
	RequestHeader  map[string]string
	RequestBody    []byte
	Status         string
	StatusCode     int
	Proto          string
	ProtoMajor     int
	ProtoMinor     int
	ResponseHeader map[string]string
	ResponseBody   []byte
	Error          []byte
}

type requestEntryEncoderDecoder struct {
}

// Encode serializes the request context into a byte array.
func (m requestEntryEncoderDecoder) Encode(entry RequestEntry) ([]byte, error) {
	r := entry.Request
	w := entry.Response

	if r == nil {
		return nil, errors.New("Request not found in RequestEntry")
	}
	if r.URL == nil {
		return nil, errors.New("URL not found in RequestEntry")
	}

	var (
		requestBody  []byte
		responseBody []byte
		err          error
	)

	if r.Body != nil {
		requestBody, err = copyHTTPRequestBody(r)
		if err != nil {
			return nil, err
		}
	}

	e := HTTPRequestResponse{
		Method:        r.Method,
		URL:           r.URL.String(),
		RequestHeader: httpHeaderToMap(r.Header),
		RequestBody:   requestBody,
	}

	if w != nil && w.Body != nil {
		responseBody, err = copyHTTPResponseBody(w)
		if err != nil {
			return nil, err
		}
	}

	if w != nil {
		e.Status = w.Status
		e.StatusCode = w.StatusCode
		e.Proto = w.Proto
		e.ProtoMajor = w.ProtoMajor
		e.ProtoMinor = w.ProtoMinor
		e.ResponseHeader = httpHeaderToMap(w.Header)
		e.ResponseBody = responseBody
	}

	if entry.Error != nil {
		e.Error = []byte(entry.Error.Error())
	}

	return msgpack.Marshal(&e)
}

// Decode deserializes the byte array into the request context.
func (m requestEntryEncoderDecoder) Decode(value []byte) (re RequestEntry, err error) {
	var e HTTPRequestResponse
	err = msgpack.Unmarshal(value, &e)
	if err != nil {
		return
	}

	req, err := http.NewRequest(e.Method, e.URL, bytes.NewReader(e.RequestBody))
	if err != nil {
		return
	}

	var resp *http.Response

	if e.StatusCode > 0 {
		resp = &http.Response{
			Status:        e.Proto,
			StatusCode:    e.StatusCode,
			Proto:         e.Proto,
			ProtoMajor:    e.ProtoMajor,
			ProtoMinor:    e.ProtoMinor,
			Body:          ioutil.NopCloser(bytes.NewBuffer(e.ResponseBody)),
			ContentLength: int64(len(e.ResponseBody)),
			Request:       req,
			Header:        mapToHTTPHeader(e.ResponseHeader),
		}
	}

	var entryError error
	if e.Error != nil {
		entryError = errors.New(string(e.Error))
	}

	return RequestEntry{
		Request:  req,
		Response: resp,
		Error:    entryError,
	}, nil
}

func httpHeaderToMap(header http.Header) map[string]string {
	m := make(map[string]string)
	for key := range header {
		value := header.Get(key)
		m[key] = value
	}
	return m
}

func mapToHTTPHeader(m map[string]string) http.Header {
	header := make(http.Header)
	for key, value := range m {
		header.Set(key, value)
	}
	return header
}
