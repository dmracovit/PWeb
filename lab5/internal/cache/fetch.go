package cache

import (
	"fmt"
	"os"
	"time"

	"github.com/dmracovit/PWeb/lab5/internal/client"
)

// Fetch is a cache-aware GET. It looks the URL up on disk and, when
// there is a fresh entry, serves it without touching the network. Stale
// entries that carry an ETag or Last-Modified validator are revalidated
// with a conditional request; a 304 response promotes the cached body
// back to fresh and a 200 replaces the cache.
//
// If c is nil this degrades to a plain client.Get.
func Fetch(c *Cache, rawURL string, opts client.Options) (*client.Response, bool, error) {
	if c == nil {
		resp, err := client.Get(rawURL, opts)
		return resp, false, err
	}

	accept := opts.Accept
	if accept == "" {
		accept = client.DefaultAccept
	}

	entry, hit := c.Get("GET", rawURL, accept)

	if hit && entry.Fresh() {
		if opts.Verbose {
			fmt.Fprintf(os.Stderr, "[cache hit] %s (expires %s)\n",
				rawURL, entry.ExpiresAt.Format(time.RFC3339))
		}
		return entryResponse(entry), true, nil
	}

	if hit && entry.CanRevalidate() {
		condOpts := withValidators(opts, entry)
		if opts.Verbose {
			fmt.Fprintf(os.Stderr, "[cache stale] revalidating %s (etag=%q)\n",
				rawURL, entry.ETag)
		}
		resp, err := client.Get(rawURL, condOpts)
		if err != nil {
			return nil, false, err
		}
		if resp.Status == 304 {
			if opts.Verbose {
				fmt.Fprintln(os.Stderr, "[cache 304] body reused from disk")
			}
			policy := ParsePolicy(resp.Headers)
			// merge fresh response headers (gives us new validators)
			for k, v := range resp.Headers {
				entry.Headers[k] = v
			}
			FillExpiry(entry, time.Now(), policy)
			_ = c.Put("GET", rawURL, accept, entry)

			out := entryResponse(entry)
			return out, true, nil
		}
		// 200 (or other) — fall through to caching the new response below
		store(c, rawURL, accept, resp, opts.Verbose)
		return resp, false, nil
	}

	resp, err := client.Get(rawURL, opts)
	if err != nil {
		return nil, false, err
	}
	store(c, rawURL, accept, resp, opts.Verbose)
	return resp, false, nil
}

func store(c *Cache, rawURL, accept string, resp *client.Response, verbose bool) {
	if resp.Status != 200 {
		return // only cache successful responses
	}
	policy := ParsePolicy(resp.Headers)
	if policy.NoStore {
		if verbose {
			fmt.Fprintf(os.Stderr, "[cache skip] no-store on %s\n", rawURL)
		}
		return
	}
	e := &Entry{
		Meta: Meta{
			Status:     resp.Status,
			StatusText: resp.StatusText,
			Headers:    resp.Headers,
		},
		Body: resp.Body,
	}
	FillExpiry(e, time.Now(), policy)
	if err := c.Put("GET", rawURL, accept, e); err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "[cache write failed] %v\n", err)
		}
		return
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "[cache store] %s (expires %s)\n",
			rawURL, e.ExpiresAt.Format(time.RFC3339))
	}
}

func entryResponse(e *Entry) *client.Response {
	return &client.Response{
		Status:     e.Status,
		StatusText: e.StatusText,
		Proto:      "HTTP/1.1",
		Headers:    e.Headers,
		Body:       e.Body,
	}
}

func withValidators(opts client.Options, e *Entry) client.Options {
	out := opts
	out.ExtraHeaders = map[string]string{}
	for k, v := range opts.ExtraHeaders {
		out.ExtraHeaders[k] = v
	}
	if e.ETag != "" {
		out.ExtraHeaders["If-None-Match"] = e.ETag
	}
	if e.LastModified != "" {
		out.ExtraHeaders["If-Modified-Since"] = e.LastModified
	}
	return out
}
