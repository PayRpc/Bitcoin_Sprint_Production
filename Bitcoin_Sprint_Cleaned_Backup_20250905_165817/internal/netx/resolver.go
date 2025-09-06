package netx

import (
	"context"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"
)

// CustomResolver returns a net.Resolver that prefers the Go resolver and
// uses the DNS servers configured in SPRINT_DNS (comma-separated list)
// or falls back to Cloudflare/Google if not set.
func CustomResolver() *net.Resolver {
	dnsEnv := os.Getenv("SPRINT_DNS")
	if dnsEnv == "" {
		dnsEnv = "1.1.1.1:53,8.8.8.8:53"
	}
	servers := strings.Split(dnsEnv, ",")

	// Use a short dialer for resolver lookups
	dialer := &net.Dialer{Timeout: 5 * time.Second}

	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			// Try each configured DNS server in order
			for _, s := range servers {
				conn, err := dialer.DialContext(ctx, "udp", s)
				if err == nil {
					return conn, nil
				}
			}
			// Last-ditch: use default behavior
			return dialer.DialContext(ctx, network, address)
		},
	}
}

// DialContextWithResolver returns a DialContext function suitable for
// wiring into websocket.Dialer.NetDialContext or http.Transport.
// It resolves hostnames using the CustomResolver and then attempts to
// dial the resolved IPs with a short timeout, applying a simple
// exponential backoff with jitter in callers.
func DialContextWithResolver(ctx context.Context, network, address string) (net.Conn, error) {
	d := &net.Dialer{Timeout: 10 * time.Second}
	r := CustomResolver()

	host, port, err := net.SplitHostPort(address)
	if err != nil {
		// If the address isn't host:port, fall back to direct dial
		return d.DialContext(ctx, network, address)
	}

	ips, err := r.LookupIPAddr(ctx, host)
	if err != nil || len(ips) == 0 {
		return d.DialContext(ctx, network, address)
	}

	var lastErr error
	for _, ip := range ips {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		addr := net.JoinHostPort(ip.IP.String(), port)
		conn, err := d.DialContext(ctx, network, addr)
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, ctx.Err()
}

// DialerWithResolver returns a function matching the signature expected by
// websocket.Dialer.NetDialContext
func DialerWithResolver() func(ctx context.Context, network, address string) (net.Conn, error) {
	// Seed math/rand for jitter
	rand.Seed(time.Now().UnixNano())
	return DialContextWithResolver
}
