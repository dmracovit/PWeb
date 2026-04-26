# Lab 5 — `go2web`

A tiny HTTP client that talks raw TCP/TLS sockets — **no `net/http`, no
third-party HTTP libraries**. Written in Go.

![demo](assets/demo.gif)

## What it does

```text
go2web -u <URL>           Fetch the URL and print a human-readable response.
go2web -s <search-term>   Search the term and print the top 10 results.
go2web -h                 Show this help.
```

Extra flags:

| Flag         | What it does                                              |
| ------------ | --------------------------------------------------------- |
| `-v`         | verbose: dump request/response headers and cache decisions |
| `--no-cache` | bypass the on-disk cache for this request                  |

## Build & run

```bash
cd lab5
make build         # produces ./go2web
./go2web -h
```

The binary is a single self-contained executable; no runtime, no deps.

```bash
make install       # copies go2web to $GOPATH/bin
```

## Examples

```bash
# Plain HTTPS, HTML stripped to text
./go2web -u https://example.com

# JSON, pretty-printed via content negotiation
./go2web -u https://api.github.com/users/octocat

# Follows redirects (http → https in this case)
./go2web -v -u http://github.com

# Search, top 10 results — URLs are real and feed back into -u
./go2web -s coffee shop chisinau
./go2web -u <one of the URLs above>

# Second identical request comes from disk cache
./go2web -u https://api.github.com/users/octocat              # miss → store
./go2web -u https://api.github.com/users/octocat              # [cache hit]
./go2web --no-cache -u https://api.github.com/users/octocat   # forced miss
```

## How it works

```
lab5/
├── main.go                         # CLI parsing, dispatcher
└── internal/
    ├── client/
    │   ├── transport.go            # net.Dial + tls.Dial (no net/http)
    │   ├── request.go              # build raw HTTP/1.1 GET bytes
    │   ├── response.go             # parse status, headers, body
    │   ├── chunked.go              # Transfer-Encoding: chunked decoder
    │   ├── url.go                  # URL parsing + relative-Location resolve
    │   └── client.go               # high-level Get with redirect chain
    ├── cache/
    │   ├── cache.go                # SHA-256 keyed disk cache
    │   └── fetch.go                # cache-aware fetcher with revalidation
    ├── render/
    │   ├── html.go                 # HTML → plain text (golang.org/x/net/html)
    │   └── json.go                 # JSON pretty-printer
    └── search/
        └── search.go               # DuckDuckGo HTML scraping (top 10)
```

The TCP path is direct: open socket → write
`GET /path HTTP/1.1\r\nHost: ...\r\nConnection: close\r\n\r\n`
→ read everything back → parse status line, headers, body. Bodies that
arrive `chunked` are decoded byte-for-byte (`hex-size CRLF data CRLF
… 0 CRLF CRLF`); `Content-Encoding: gzip` is unwrapped through Go's
stdlib `compress/gzip`. Up to 10 redirects are followed; relative
`Location` headers are resolved against the current request URL with a
visited-set to break cycles.

The cache (`$HOME/.go2web/cache/`) is keyed on
`SHA-256(method|url|Accept)`. Entries are stored as a 4-byte big-endian
length, then JSON metadata, then raw body bytes. Cache-Control directives
are parsed: `no-store` skips writing, `no-cache`/`must-revalidate` force
revalidation on next access, `max-age=N` sets the expiry. If a stale
entry has an `ETag` or `Last-Modified` we send a conditional `GET`
(`If-None-Match` / `If-Modified-Since`); a `304` promotes the cached
body back to fresh, a `200` replaces the entry.

Search uses [DuckDuckGo's HTML endpoint][ddg] — server-rendered, no JS
required. Each result anchor (`<a class="result__a">`) carries the title
and a redirect URL of the form `/l/?uddg=<encoded-real-url>`; we peel
the wrapper so the URLs printed by `-s` can be passed straight back into
`-u`.

[ddg]: https://html.duckduckgo.com/html/

### What we explicitly don't use

- **`net/http`** — disallowed by the lab. We write to a `net.Conn` byte
  by byte and parse the protocol ourselves.
- HTTP/2, keep-alive, pipelining — overkill for the spec; we send
  `Connection: close` and read until EOF.
- Cookies, auth — not required.

### What is allowed and used

- `net` — TCP socket primitives.
- `crypto/tls` — TLS handshake; not an HTTP library.
- `compress/gzip` — body decompression after the HTTP layer.
- `encoding/json`, `net/url` — content rendering and URL string parsing.
- `golang.org/x/net/html` — third-party HTML tokenizer/parser. The lab
  hint explicitly permits "third-party libraries for parsing HTML".

## Score breakdown

| Item                                                     | Points |
| -------------------------------------------------------- | ------ |
| `-h`, `-u`, `-s`                                         | +6     |
| Search results accessible via the same CLI (`-u <link>`) | +1     |
| HTTP redirects                                           | +1     |
| HTTP cache (file-backed, conditional revalidation)       | +2     |
| Content negotiation (JSON + HTML)                        | +2     |
| **Total target**                                         | **12** |

## Recording the demo gif

The repo expects `lab5/assets/demo.gif`. Two options:

**asciinema → gif**

```bash
brew install asciinema agg
asciinema rec demo.cast        # run the commands below, then Ctrl-D
agg demo.cast assets/demo.gif
```

**QuickTime → gif**

Record screen with QuickTime, then convert with `ffmpeg`:

```bash
ffmpeg -i recording.mov -vf "fps=12,scale=900:-1" -loop 0 assets/demo.gif
```

A ready-to-replay script is at [`demo.sh`](demo.sh) — run it inside the
recording to walk through every feature.

## Git history

Five PRs, one feature per branch:

1. `feature/lab5-init` — Go module, Makefile, CLI skeleton with `-h`.
2. `feature/lab5-http-client` — TCP/TLS, chunked, gzip, redirects, HTML
   render, content negotiation.
3. `feature/lab5-search` — DuckDuckGo top-10.
4. `feature/lab5-cache` — file-backed cache with conditional GETs.
5. `feature/lab5-readme-demo` — this README and the demo gif.
