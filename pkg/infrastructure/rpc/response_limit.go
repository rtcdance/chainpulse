package rpc

import (
	"fmt"
	"io"
	"net/http"
)

const DefaultMaxResponseSize = 50 * 1024 * 1024

type ResponseSizeLimitRoundTripper struct {
	Proxied       http.RoundTripper
	MaxSizeBytes  int64
}

func (rt *ResponseSizeLimitRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	proxied := rt.Proxied
	if proxied == nil {
		proxied = http.DefaultTransport
	}

	resp, err := proxied.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	maxSize := rt.MaxSizeBytes
	if maxSize <= 0 {
		maxSize = DefaultMaxResponseSize
	}

	if resp.ContentLength > maxSize {
		resp.Body.Close()
		return nil, fmt.Errorf("RPC response too large: Content-Length=%d exceeds max=%d", resp.ContentLength, maxSize)
	}

	resp.Body = &limitedReadCloser{
		ReadCloser: resp.Body,
		remaining:  maxSize,
	}

	return resp, nil
}

type limitedReadCloser struct {
	io.ReadCloser
	remaining int64
}

func (l *limitedReadCloser) Read(p []byte) (int, error) {
	if l.remaining <= 0 {
		return 0, fmt.Errorf("RPC response body exceeds max size limit")
	}
	if int64(len(p)) > l.remaining {
		p = p[:l.remaining]
	}
	n, err := l.ReadCloser.Read(p)
	l.remaining -= int64(n)
	return n, err
}