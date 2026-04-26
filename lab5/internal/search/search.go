// Package search performs a top-10 web search by scraping DuckDuckGo's
// HTML-only endpoint. We avoid Google/Bing because they require JS or
// produce results inside heavily-obfuscated markup.
package search

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/dmracovit/PWeb/lab5/internal/client"
)

const (
	endpoint   = "https://html.duckduckgo.com/html/?q="
	maxResults = 10

	// DuckDuckGo blocks our default UA. Send a plausible browser string —
	// the lab disallows HTTP libraries, not realistic User-Agent values.
	browserUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) " +
		"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0 Safari/537.36"
)

type Result struct {
	Title   string
	URL     string
	Snippet string
}

// Search returns the first ~10 search results for query.
func Search(query string, opts client.Options) ([]Result, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("empty search query")
	}

	if opts.ExtraHeaders == nil {
		opts.ExtraHeaders = map[string]string{}
	}
	opts.ExtraHeaders["User-Agent"] = browserUA
	opts.Accept = "text/html"

	resp, err := client.Get(endpoint+url.QueryEscape(query), opts)
	if err != nil {
		return nil, err
	}
	if resp.Status != 200 {
		return nil, fmt.Errorf("search engine returned HTTP %d", resp.Status)
	}
	return parse(bytes.NewReader(resp.Body))
}

func parse(r io.Reader) ([]Result, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}
	var out []Result
	walk(doc, &out)
	if len(out) > maxResults {
		out = out[:maxResults]
	}
	return out, nil
}

func walk(n *html.Node, out *[]Result) {
	if len(*out) >= maxResults*2 {
		// stop early — we'll trim later
		return
	}
	if n.Type == html.ElementNode && n.DataAtom == atom.A && hasClass(n, "result__a") {
		title := strings.TrimSpace(textOf(n))
		href := decodeRedirect(getAttr(n, "href"))
		if title != "" && href != "" {
			snippet := findSnippet(n)
			*out = append(*out, Result{Title: title, URL: href, Snippet: snippet})
		}
		return
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walk(c, out)
	}
}

// findSnippet walks up to the enclosing result container, then back down
// to a node carrying the result__snippet class.
func findSnippet(titleAnchor *html.Node) string {
	container := titleAnchor.Parent
	for container != nil {
		if hasClass(container, "result") || hasClass(container, "result__body") {
			break
		}
		container = container.Parent
	}
	if container == nil {
		return ""
	}
	if snip := findFirstWithClass(container, "result__snippet"); snip != nil {
		return strings.TrimSpace(textOf(snip))
	}
	return ""
}

func findFirstWithClass(n *html.Node, cls string) *html.Node {
	if hasClass(n, cls) {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if r := findFirstWithClass(c, cls); r != nil {
			return r
		}
	}
	return nil
}

// decodeRedirect peels DuckDuckGo's /l/?uddg=<encoded-real-url> wrapper so
// the URLs we print can actually be passed back into `go2web -u`.
func decodeRedirect(href string) string {
	q := strings.Index(href, "?")
	if q < 0 {
		return href
	}
	vals, err := url.ParseQuery(href[q+1:])
	if err != nil {
		return href
	}
	if real := vals.Get("uddg"); real != "" {
		return real
	}
	return href
}

func textOf(n *html.Node) string {
	var b strings.Builder
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	// collapse internal whitespace for readable single-line strings
	return strings.Join(strings.Fields(b.String()), " ")
}

func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func hasClass(n *html.Node, cls string) bool {
	if n == nil || n.Type != html.ElementNode {
		return false
	}
	for _, a := range n.Attr {
		if a.Key != "class" {
			continue
		}
		for _, c := range strings.Fields(a.Val) {
			if c == cls {
				return true
			}
		}
	}
	return false
}
