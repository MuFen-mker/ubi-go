package utils

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/http2"
)

// NewBrowserClient creates an HTTP client that mimics a real Chrome browser's
// TLS fingerprint using uTLS with full HTTP/2 support.
// If proxyURL is non-empty, traffic is tunneled through that HTTP CONNECT proxy.
func NewBrowserClient(proxyURL string) (*http.Client, error) {
	jar, _ := cookiejar.New(nil)

	dialFunc := func(ctx context.Context, network, addr string) (net.Conn, error) {
		return (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, network, addr)
	}

	if proxyURL != "" {
		parsed, err := url.Parse(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL %q: %w", proxyURL, err)
		}

		dialFunc = func(ctx context.Context, network, addr string) (net.Conn, error) {
			proxyConn, err := (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, "tcp", parsed.Host)
			if err != nil {
				return nil, fmt.Errorf("failed to connect to proxy: %w", err)
			}

			connectReq := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n", addr, addr)
			if parsed.User != nil {
				pass, _ := parsed.User.Password()
				creds := base64.StdEncoding.EncodeToString([]byte(parsed.User.Username() + ":" + pass))
				connectReq += fmt.Sprintf("Proxy-Authorization: Basic %s\r\n", creds)
			}
			connectReq += "\r\n"

			if _, err := proxyConn.Write([]byte(connectReq)); err != nil {
				proxyConn.Close()
				return nil, fmt.Errorf("failed to send CONNECT: %w", err)
			}

			buf := make([]byte, 4096)
			n, err := proxyConn.Read(buf)
			if err != nil {
				proxyConn.Close()
				return nil, fmt.Errorf("failed to read CONNECT response: %w", err)
			}
			resp := string(buf[:n])
			if len(resp) < 12 || resp[9] != '2' {
				proxyConn.Close()
				return nil, fmt.Errorf("proxy CONNECT failed: %s", resp)
			}

			return proxyConn, nil
		}
	}

	transport := &http2.Transport{
		DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
			conn, err := dialFunc(ctx, network, addr)
			if err != nil {
				return nil, err
			}

			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				conn.Close()
				return nil, err
			}

			uconn := utls.UClient(conn, &utls.Config{
				ServerName: host,
			}, utls.HelloChrome_Auto)

			if err := uconn.HandshakeContext(ctx); err != nil {
				conn.Close()
				return nil, err
			}

			return uconn, nil
		},
	}

	return &http.Client{
		Transport: transport,
		Jar:       jar,
		Timeout:   15 * time.Second,
	}, nil
}
