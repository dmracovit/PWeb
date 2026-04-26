package client

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

// Response is a parsed HTTP/1.1 response. Headers are lower-cased to make
// case-insensitive lookups trivial.
type Response struct {
	Status     int
	StatusText string
	Proto      string
	Headers    map[string]string
	Body       []byte
}

func (r *Response) ContentType() string {
	return r.Headers["content-type"]
}

const (
	maxResponseSize = 10 * 1024 * 1024 // 10 MB cap
)

func readResponse(conn net.Conn) (*Response, error) {
	_ = conn.SetReadDeadline(time.Now().Add(readTimeout))
	r := bufio.NewReader(conn)

	statusLine, err := r.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("read status line: %w", err)
	}
	statusLine = strings.TrimRight(statusLine, "\r\n")

	parts := strings.SplitN(statusLine, " ", 3)
	if len(parts) < 2 {
		return nil, fmt.Errorf("malformed status line: %q", statusLine)
	}
	code, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("non-numeric status code in %q", statusLine)
	}
	resp := &Response{
		Proto:   parts[0],
		Status:  code,
		Headers: map[string]string{},
	}
	if len(parts) == 3 {
		resp.StatusText = parts[2]
	}

	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("read header: %w", err)
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		idx := strings.Index(line, ":")
		if idx <= 0 {
			continue
		}
		k := strings.ToLower(strings.TrimSpace(line[:idx]))
		v := strings.TrimSpace(line[idx+1:])
		// duplicate Set-Cookie etc. is fine to overwrite for our purposes
		resp.Headers[k] = v
	}

	body, err := readBody(r, resp.Headers)
	if err != nil {
		return nil, err
	}

	if strings.EqualFold(resp.Headers["content-encoding"], "gzip") && len(body) > 0 {
		body, err = gunzip(body)
		if err != nil {
			return nil, fmt.Errorf("gunzip: %w", err)
		}
	}
	resp.Body = body
	return resp, nil
}

func readBody(r *bufio.Reader, headers map[string]string) ([]byte, error) {
	te := strings.ToLower(headers["transfer-encoding"])
	if strings.Contains(te, "chunked") {
		return readChunked(r)
	}
	if cl, ok := headers["content-length"]; ok {
		n, err := strconv.Atoi(cl)
		if err != nil || n < 0 {
			return nil, fmt.Errorf("invalid content-length %q", cl)
		}
		if n > maxResponseSize {
			n = maxResponseSize
		}
		buf := make([]byte, n)
		_, err = io.ReadFull(r, buf)
		if err != nil && err != io.ErrUnexpectedEOF {
			return nil, err
		}
		return buf, nil
	}
	// Connection: close — read to EOF, with a size cap
	limited := io.LimitReader(r, maxResponseSize)
	return io.ReadAll(limited)
}

func gunzip(body []byte) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	return io.ReadAll(io.LimitReader(gz, maxResponseSize))
}
