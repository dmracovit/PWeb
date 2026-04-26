package client

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// readChunked decodes an HTTP/1.1 Transfer-Encoding: chunked body.
// Format per RFC 7230 §4.1:
//
//	chunk = chunk-size [chunk-ext] CRLF chunk-data CRLF
//	last-chunk = 1*("0") [chunk-ext] CRLF
//	(then optional trailer headers, then CRLF)
func readChunked(r *bufio.Reader) ([]byte, error) {
	var body bytes.Buffer

	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("read chunk size: %w", err)
		}
		sizeStr := strings.TrimRight(line, "\r\n")
		if i := strings.Index(sizeStr, ";"); i >= 0 {
			sizeStr = sizeStr[:i] // drop chunk extension
		}
		sizeStr = strings.TrimSpace(sizeStr)

		size, err := strconv.ParseInt(sizeStr, 16, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid chunk size %q: %w", sizeStr, err)
		}

		if size == 0 {
			// drain trailers up to the empty line
			for {
				t, err := r.ReadString('\n')
				if err != nil {
					return nil, err
				}
				if strings.TrimRight(t, "\r\n") == "" {
					return body.Bytes(), nil
				}
			}
		}

		chunk := make([]byte, size)
		if _, err := io.ReadFull(r, chunk); err != nil {
			return nil, fmt.Errorf("read chunk body: %w", err)
		}
		body.Write(chunk)

		// trailing CRLF after chunk
		if _, err := r.Discard(2); err != nil {
			return nil, err
		}
	}
}
