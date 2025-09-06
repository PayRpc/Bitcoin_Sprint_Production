// Package netkit provides enhanced networking utilities for Bitcoin Sprint
// Implements Happy-Eyeballs dialing, tuned TCP options, and connection management
package netkit

import (
	"context"
	"net"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// ConnectionConfig holds configuration for enhanced connections
type ConnectionConfig struct {
	Timeout        time.Duration
	KeepAlive      time.Duration
	KeepAliveIdle  time.Duration
	KeepAliveCount int
	KeepAliveIntvl time.Duration
	UserTimeout    time.Duration
	NoDelay        bool
	HappyEyeballs  bool
	MaxConcurrency int
}

// DefaultConfig returns a production-ready connection configuration
func DefaultConfig() *ConnectionConfig {
	return &ConnectionConfig{
		Timeout:        30 * time.Second,
		KeepAlive:      30 * time.Second,
		KeepAliveIdle:  10 * time.Second,
		KeepAliveCount: 4,
		KeepAliveIntvl: 10 * time.Second,
		UserTimeout:    20 * time.Second,
		NoDelay:        true,
		HappyEyeballs:  true,
		MaxConcurrency: 4,
	}
}

// Dialer provides enhanced TCP dialing with Happy-Eyeballs and tuned options
type Dialer struct {
	config *ConnectionConfig
	logger *zap.Logger
}

// NewDialer creates a new enhanced dialer
func NewDialer(config *ConnectionConfig, logger *zap.Logger) *Dialer {
	if config == nil {
		config = DefaultConfig()
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Dialer{
		config: config,
		logger: logger,
	}
}

// Dial connects to the address with enhanced options
func (d *Dialer) Dial(network, address string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address)
}

// DialContext connects to the address with context and enhanced options
func (d *Dialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	if network != "tcp" && network != "tcp4" && network != "tcp6" {
		// Fallback to standard dial for non-TCP
		return (&net.Dialer{Timeout: d.config.Timeout}).DialContext(ctx, network, address)
	}

	if !d.config.HappyEyeballs {
		// Use tuned dial without Happy-Eyeballs
		return d.dialTuned(ctx, network, address)
	}

	// Happy-Eyeballs: resolve and try multiple addresses in parallel
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	// Resolve all addresses
	addrs, err := net.LookupHost(host)
	if err != nil {
		return nil, err
	}

	if len(addrs) == 0 {
		return nil, &net.DNSError{Err: "no such host", Name: host}
	}

	// Convert to SocketAddr
	var sockAddrs []net.TCPAddr
	for _, addr := range addrs {
		if tcpAddr, err := net.ResolveTCPAddr(network, net.JoinHostPort(addr, port)); err == nil {
			sockAddrs = append(sockAddrs, *tcpAddr)
		}
	}

	if len(sockAddrs) == 0 {
		return nil, &net.DNSError{Err: "no valid addresses", Name: host}
	}

	// Try connections in parallel (up to MaxConcurrency)
	resultChan := make(chan net.Conn, 1)
	errorChan := make(chan error, len(sockAddrs))
	var wg sync.WaitGroup

	maxConcurrent := d.config.MaxConcurrency
	if len(sockAddrs) < maxConcurrent {
		maxConcurrent = len(sockAddrs)
	}

	for i := 0; i < maxConcurrent; i++ {
		wg.Add(1)
		go func(addr net.TCPAddr) {
			defer wg.Done()
			conn, err := d.dialTunedAddr(ctx, &addr)
			if err != nil {
				errorChan <- err
				return
			}
			select {
			case resultChan <- conn:
			default:
				conn.Close() // Another connection succeeded first
			}
		}(sockAddrs[i])
	}

	// Wait for first success or all failures
	go func() {
		wg.Wait()
		close(errorChan)
	}()

	select {
	case conn := <-resultChan:
		d.logger.Debug("Happy-Eyeballs connection established", zap.String("address", address))
		return conn, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(d.config.Timeout):
		return nil, &net.OpError{Op: "dial", Net: network, Source: nil, Addr: nil, Err: syscall.ETIMEDOUT}
	}
}

// dialTuned establishes a connection with tuned TCP options
func (d *Dialer) dialTuned(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	addr, err := net.ResolveTCPAddr(network, net.JoinHostPort(host, port))
	if err != nil {
		return nil, err
	}

	return d.dialTunedAddr(ctx, addr)
}

// dialTunedAddr establishes a connection to a specific address with tuned options
func (d *Dialer) dialTunedAddr(ctx context.Context, addr *net.TCPAddr) (net.Conn, error) {
	// Use standard Go net package for cross-platform compatibility
	dialer := &net.Dialer{
		Timeout: d.config.Timeout,
	}

	// Set keepalive if supported
	if d.config.KeepAlive > 0 {
		dialer.KeepAlive = d.config.KeepAlive
	}

	conn, err := dialer.DialContext(ctx, "tcp", addr.String())
	if err != nil {
		return nil, err
	}

	// Set TCP_NODELAY for lower latency
	if tcpConn, ok := conn.(*net.TCPConn); ok && d.config.NoDelay {
		tcpConn.SetNoDelay(true)
	}

	return conn, nil
}

// setSocketOptions configures TCP socket with optimized settings
// Note: This function is kept for future use with low-level socket operations
func (d *Dialer) setSocketOptions(fd int) {
	// TCP_NODELAY - disable Nagle's algorithm for lower latency
	if d.config.NoDelay {
		// syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, syscall.TCP_NODELAY, 1)
	}

	// TCP keepalive settings
	if d.config.KeepAlive > 0 {
		// syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_KEEPALIVE, 1)
		// Note: Keepalive settings are platform-specific and may require additional syscalls
	}

	// TCP_USER_TIMEOUT (Linux-specific) - timeout stalled connections
	if d.config.UserTimeout > 0 {
		// This is Linux-specific, so we use a platform check
		// syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, 0x12, int(d.config.UserTimeout.Milliseconds()))
	}
}

// DialTimeout is a convenience function for simple timeout-based dialing
func DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	config := DefaultConfig()
	config.Timeout = timeout
	dialer := NewDialer(config, nil)
	return dialer.Dial(network, address)
}

// DialHappy is a convenience function for Happy-Eyeballs dialing
func DialHappy(address string, timeout time.Duration) (net.Conn, error) {
	config := DefaultConfig()
	config.Timeout = timeout
	dialer := NewDialer(config, nil)
	return dialer.Dial("tcp", address)
}
