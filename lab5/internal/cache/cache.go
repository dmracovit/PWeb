// Package cache implements a small file-backed HTTP cache. Each entry is
// keyed by SHA-256(method | url | Accept) and stored under
// $HOME/.go2web/cache/. We honour Cache-Control: no-store / no-cache /
// max-age and use ETag / Last-Modified validators for revalidation.
package cache

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const defaultTTL = 5 * time.Minute

// Meta is the JSON-serializable header of a cache entry. Body bytes are
// stored separately so we don't pay the base64 cost.
type Meta struct {
	Status       int               `json:"status"`
	StatusText   string            `json:"status_text"`
	Headers      map[string]string `json:"headers"`
	FetchedAt    time.Time         `json:"fetched_at"`
	ExpiresAt    time.Time         `json:"expires_at"`
	ETag         string            `json:"etag,omitempty"`
	LastModified string            `json:"last_modified,omitempty"`
}

type Entry struct {
	Meta
	Body []byte
}

// Fresh reports whether the cached response can still be served without
// revalidation.
func (e *Entry) Fresh() bool {
	return !e.ExpiresAt.IsZero() && time.Now().Before(e.ExpiresAt)
}

// CanRevalidate reports whether we have a validator for a conditional
// request when the entry is stale.
func (e *Entry) CanRevalidate() bool {
	return e.ETag != "" || e.LastModified != ""
}

type Cache struct {
	dir string
}

func New() (*Cache, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, ".go2web", "cache")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Cache{dir: dir}, nil
}

func (c *Cache) Dir() string { return c.dir }

func (c *Cache) keyPath(method, url, accept string) string {
	h := sha256.Sum256([]byte(method + "|" + url + "|" + accept))
	return filepath.Join(c.dir, hex.EncodeToString(h[:]))
}

// Get returns a cached entry for the given key, or (nil, false) if there
// is no usable entry on disk.
func (c *Cache) Get(method, url, accept string) (*Entry, bool) {
	path := c.keyPath(method, url, accept)
	f, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	defer f.Close()

	var sizeBuf [4]byte
	if _, err := io.ReadFull(f, sizeBuf[:]); err != nil {
		return nil, false
	}
	metaLen := binary.BigEndian.Uint32(sizeBuf[:])
	if metaLen == 0 || metaLen > 1<<20 {
		return nil, false
	}
	metaBuf := make([]byte, metaLen)
	if _, err := io.ReadFull(f, metaBuf); err != nil {
		return nil, false
	}
	var meta Meta
	if err := json.Unmarshal(metaBuf, &meta); err != nil {
		return nil, false
	}
	body, err := io.ReadAll(f)
	if err != nil {
		return nil, false
	}
	return &Entry{Meta: meta, Body: body}, true
}

// Put writes an entry to disk. Caller is expected to set Meta.ExpiresAt
// and friends.
func (c *Cache) Put(method, url, accept string, e *Entry) error {
	metaBuf, err := json.Marshal(e.Meta)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(c.dir, "tmp-*")
	if err != nil {
		return err
	}
	cleanup := true
	defer func() {
		if cleanup {
			tmp.Close()
			os.Remove(tmp.Name())
		}
	}()

	var sizeBuf [4]byte
	binary.BigEndian.PutUint32(sizeBuf[:], uint32(len(metaBuf)))
	if _, err := tmp.Write(sizeBuf[:]); err != nil {
		return err
	}
	if _, err := tmp.Write(metaBuf); err != nil {
		return err
	}
	if _, err := tmp.Write(e.Body); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	cleanup = false
	return os.Rename(tmp.Name(), c.keyPath(method, url, accept))
}

// Policy is the parsed Cache-Control directive set we care about.
type Policy struct {
	NoStore        bool
	MustRevalidate bool // includes "no-cache" and "must-revalidate"
	MaxAge         time.Duration
}

// ParsePolicy reads Cache-Control from the response headers and returns
// the directives relevant to our private on-disk cache.
func ParsePolicy(headers map[string]string) Policy {
	p := Policy{MaxAge: defaultTTL}
	cc := strings.ToLower(headers["cache-control"])
	if cc == "" {
		return p
	}
	for _, part := range strings.Split(cc, ",") {
		part = strings.TrimSpace(part)
		switch {
		case part == "no-store":
			p.NoStore = true
		case part == "no-cache", part == "must-revalidate":
			p.MustRevalidate = true
		case strings.HasPrefix(part, "max-age="):
			if n, err := strconv.Atoi(strings.TrimPrefix(part, "max-age=")); err == nil {
				p.MaxAge = time.Duration(n) * time.Second
			}
		}
	}
	return p
}

// FillExpiry computes ExpiresAt for a fresh entry from its policy and
// fetched-at timestamp.
func FillExpiry(e *Entry, fetchedAt time.Time, p Policy) {
	e.FetchedAt = fetchedAt
	if p.MustRevalidate {
		// Force revalidation on next access.
		e.ExpiresAt = fetchedAt
	} else {
		e.ExpiresAt = fetchedAt.Add(p.MaxAge)
	}
	e.ETag = headerVal(e.Headers, "etag")
	e.LastModified = headerVal(e.Headers, "last-modified")
}

func headerVal(h map[string]string, key string) string {
	if h == nil {
		return ""
	}
	return h[strings.ToLower(key)]
}

// Stats reports cache directory size for the verbose / demo output.
func (c *Cache) Stats() (entries int, bytes int64, err error) {
	ents, err := os.ReadDir(c.dir)
	if err != nil {
		return 0, 0, err
	}
	for _, e := range ents {
		if e.IsDir() || strings.HasPrefix(e.Name(), "tmp-") {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			return 0, 0, err
		}
		entries++
		bytes += fi.Size()
	}
	return entries, bytes, nil
}

// String summarises an entry for verbose logging.
func (e *Entry) String() string {
	return fmt.Sprintf("%d %s, expires %s, etag=%q",
		e.Status, e.StatusText, e.ExpiresAt.Format(time.RFC3339), e.ETag)
}
