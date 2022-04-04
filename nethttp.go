package gohttpclient

import (
	"io"
	"net/http"
	"net/url"
)

// DefaultClient is the default implementation of the client,
// the same as the official http package.
var DefaultClient = NewClient()

// Get initiates an HTTP GET request.
func Get(url string) (resp *http.Response, err error) {
	return DefaultClient.Get(url)
}

// Post initiates an HTTP POST request.
func Post(url, contentType string, body io.Reader) (resp *http.Response, err error) {
	return DefaultClient.Post(url, contentType, body)
}

// PostForm initiates HTTP POST form data requests.
func PostForm(url string, data url.Values) (resp *http.Response, err error) {
	return DefaultClient.PostForm(url, data)
}

// Head initiates an HTTP HEAD request.
func Head(url string) (resp *http.Response, err error) {
	return DefaultClient.Head(url)
}
