// Package client implements a tiny HTTP/1.1 client over raw TCP/TLS
// sockets. It supports GET only, plus chunked transfer-encoding, gzip
// content-encoding, redirect following, and content negotiation through
// the caller-supplied Accept header.
package client

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

const (
	defaultAccept = "application/json, text/html;q=0.9, */*;q=0.5"
	maxRedirects  = 10
)

type Options struct {
	Verbose bool
	Accept  string
	// extra request headers (e.g. If-None-Match for cache revalidation)
	ExtraHeaders map[string]string
}

// Get performs a GET against rawURL, following up to 10 redirects.
func Get(rawURL string, opts Options) (*Response, error) {
	visited := make(map[string]struct{})
	current := rawURL
	for hop := 0; hop <= maxRedirects; hop++ {
		if _, seen := visited[current]; seen {
			return nil, fmt.Errorf("redirect loop at %s", current)
		}
		visited[current] = struct{}{}

		resp, err := doGet(current, opts)
		if err != nil {
			return nil, err
		}

		if resp.Status >= 300 && resp.Status < 400 {
			loc := resp.Headers["location"]
			if loc == "" {
				return resp, nil
			}
			next, err := Resolve(current, loc)
			if err != nil {
				return nil, fmt.Errorf("resolve redirect: %w", err)
			}
			if opts.Verbose {
				fmt.Fprintf(os.Stderr, "[redirect %d] %d → %s\n", hop+1, resp.Status, next)
			}
			current = next
			continue
		}
		return resp, nil
	}
	return nil, errors.New("too many redirects")
}

func doGet(rawURL string, opts Options) (*Response, error) {
	u, err := ParseURL(rawURL)
	if err != nil {
		return nil, err
	}

	accept := opts.Accept
	if accept == "" {
		accept = defaultAccept
	}

	req := newRequest(u, accept, opts.ExtraHeaders)

	if opts.Verbose {
		fmt.Fprintf(os.Stderr, "→ GET %s\n", rawURL)
		dumpHeaders(os.Stderr, "  ", req.Headers)
	}

	conn, err := dial(u)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", u.Host, err)
	}
	defer conn.Close()

	if _, err := conn.Write(req.bytes()); err != nil {
		return nil, fmt.Errorf("write request: %w", err)
	}

	resp, err := readResponse(conn)
	if err != nil {
		return nil, err
	}

	if opts.Verbose {
		fmt.Fprintf(os.Stderr, "← %d %s\n", resp.Status, resp.StatusText)
		dumpHeaders(os.Stderr, "  ", resp.Headers)
	}
	return resp, nil
}

func dumpHeaders(w io.Writer, prefix string, h map[string]string) {
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(w, "%s%s: %s\n", prefix, k, h[k])
	}
}

// IsJSON returns true if the response Content-Type advertises JSON.
func IsJSON(ct string) bool {
	ct = strings.ToLower(ct)
	return strings.HasPrefix(ct, "application/json") ||
		strings.HasSuffix(strings.SplitN(ct, ";", 2)[0], "+json")
}

// IsHTML returns true for text/html and application/xhtml+xml.
func IsHTML(ct string) bool {
	ct = strings.ToLower(ct)
	return strings.HasPrefix(ct, "text/html") ||
		strings.HasPrefix(ct, "application/xhtml")
}
