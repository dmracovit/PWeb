package client

import (
	"fmt"
	"net/url"
	"strings"
)

// URL is a minimal parsed URL — we only care about the bits the request
// builder and TCP dialer need. We use net/url for the heavy lifting; that
// package is pure string parsing, not an HTTP client.
type URL struct {
	Raw    string
	Scheme string
	Host   string // host without port
	Port   string
	Path   string // path + raw query, ready for the request line
}

func ParseURL(raw string) (*URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("unsupported scheme %q (only http and https)", u.Scheme)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("missing host in URL %q", raw)
	}

	host := u.Hostname()
	port := u.Port()
	if port == "" {
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}

	path := u.RequestURI()
	if path == "" {
		path = "/"
	}

	return &URL{
		Raw:    raw,
		Scheme: u.Scheme,
		Host:   host,
		Port:   port,
		Path:   path,
	}, nil
}

// Resolve resolves a (possibly relative) Location header against the
// current request URL.
func Resolve(base, ref string) (string, error) {
	b, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	r, err := url.Parse(ref)
	if err != nil {
		return "", err
	}
	return b.ResolveReference(r).String(), nil
}
