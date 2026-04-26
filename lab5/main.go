package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/dmracovit/PWeb/lab5/internal/client"
	"github.com/dmracovit/PWeb/lab5/internal/render"
)

const usage = `go2web — a tiny HTTP client over raw TCP sockets.

Usage:
  go2web -u <URL>           Fetch the URL and print a human-readable response.
  go2web -s <search-term>   Search the term and print the top 10 results.
  go2web -h                 Show this help.

Flags:
  -v                        Verbose: show request/response headers.
  --no-cache                Bypass the on-disk cache for this request.

Examples:
  go2web -u https://example.com
  go2web -u https://api.github.com/users/octocat
  go2web -s coffee shop chisinau
`

func main() {
	help := flag.Bool("h", false, "show help")
	url := flag.String("u", "", "fetch URL")
	search := flag.String("s", "", "search term (or use positional args after -s)")
	verbose := flag.Bool("v", false, "verbose mode")
	noCache := flag.Bool("no-cache", false, "bypass cache")

	flag.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	flag.Parse()

	if *help || (flag.NFlag() == 0 && flag.NArg() == 0) {
		fmt.Print(usage)
		return
	}

	if *url != "" && *search != "" {
		fail("use only one of -u or -s, not both")
	}

	switch {
	case *url != "":
		runFetch(*url, *verbose, *noCache)
	case *search != "":
		query := *search
		if rest := flag.Args(); len(rest) > 0 {
			query = strings.TrimSpace(query + " " + strings.Join(rest, " "))
		}
		runSearch(query, *verbose, *noCache)
	default:
		fail("nothing to do — pass -u <URL>, -s <term>, or -h")
	}
}

func runFetch(rawURL string, verbose, noCache bool) {
	resp, err := client.Get(rawURL, client.Options{Verbose: verbose})
	if err != nil {
		fail("fetch failed: " + err.Error())
	}

	if resp.Status >= 400 {
		fmt.Fprintf(os.Stderr, "go2web: HTTP %d %s\n", resp.Status, resp.StatusText)
	}

	ct := resp.ContentType()
	switch {
	case client.IsJSON(ct):
		fmt.Print(render.JSON(resp.Body))
	case client.IsHTML(ct):
		fmt.Println(render.HTML(bytes.NewReader(resp.Body)))
	default:
		os.Stdout.Write(resp.Body)
	}
}

func runSearch(query string, verbose, noCache bool) {
	// implemented in feature/lab5-search
	fmt.Fprintln(os.Stderr, "search not implemented yet:", query)
}

func fail(msg string) {
	fmt.Fprintln(os.Stderr, "go2web:", msg)
	fmt.Fprintln(os.Stderr, "run `go2web -h` for usage.")
	os.Exit(1)
}
