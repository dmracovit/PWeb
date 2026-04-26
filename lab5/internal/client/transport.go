package client

import (
	"crypto/tls"
	"net"
	"time"
)

const (
	dialTimeout = 10 * time.Second
	readTimeout = 30 * time.Second
)

// dial opens a raw TCP (http) or TLS-wrapped TCP (https) connection.
// We deliberately do not use net/http — only socket-level packages.
func dial(u *URL) (net.Conn, error) {
	addr := net.JoinHostPort(u.Host, u.Port)
	dialer := &net.Dialer{Timeout: dialTimeout}

	if u.Scheme == "https" {
		return tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
			ServerName: u.Host,
		})
	}
	return dialer.Dial("tcp", addr)
}
