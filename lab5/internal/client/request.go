package client

import (
	"bytes"
	"fmt"
	"sort"
)

const userAgent = "go2web/1.0"

// Request is a tiny representation of an outgoing HTTP/1.1 GET request.
// We only need GET — go2web never sends bodies.
type Request struct {
	URL     *URL
	Headers map[string]string
}

func newRequest(u *URL, accept string, extra map[string]string) *Request {
	h := map[string]string{
		"Host":            u.Host,
		"User-Agent":      userAgent,
		"Accept":          accept,
		"Accept-Encoding": "gzip",
		"Connection":      "close",
	}
	for k, v := range extra {
		h[k] = v
	}
	return &Request{URL: u, Headers: h}
}

func (r *Request) bytes() []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, "GET %s HTTP/1.1\r\n", r.URL.Path)

	keys := make([]string, 0, len(r.Headers))
	for k := range r.Headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&b, "%s: %s\r\n", k, r.Headers[k])
	}
	b.WriteString("\r\n")
	return b.Bytes()
}
