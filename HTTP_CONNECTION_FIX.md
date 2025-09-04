# Bitcoin Sprint API Server Connection Fix

This document outlines the fixes implemented to address the "Connection refused" errors with the Bitcoin Sprint API server and ensure reliable WebSocket connections to Ethereum and Solana relay networks.

## HTTP Server Connection Issues

### Root Causes Identified
1. **Insufficient HTTP Server Configuration**: The server was missing critical timeout settings and connection parameters
2. **Improper Socket Binding**: The server wasn't properly configuring TCP sockets for optimal connection management
3. **Missing Error Handling**: The server wasn't providing enough diagnostic information when connection issues occurred
4. **No Connection Retries**: Client connections weren't retrying, leading to false negatives on startup
5. **No Self-Testing**: The server needed a self-check mechanism to validate connections

### Implemented Fixes

#### 1. Enhanced HTTP Server Configuration
```go
s.srv = &http.Server{
    Addr:              addr,
    Handler:           handler,
    ReadHeaderTimeout: 5 * time.Second,
    ReadTimeout:       30 * time.Second,
    WriteTimeout:      60 * time.Second,
    IdleTimeout:       120 * time.Second,
    MaxHeaderBytes:    1 << 20, // 1 MB
    BaseContext: func(listener net.Listener) context.Context {
        baseCtx := context.Background()
        return baseCtx
    },
}
```

#### 2. Explicit TCP Socket Configuration
```go
listener, err := net.Listen("tcp", addr)
if err != nil {
    s.logger.Error("Failed to create listener", 
        zap.String("addr", addr), 
        zap.Error(err))
    return
}

// Set TCP keep-alive to detect dead connections
tcpListener, ok := listener.(*net.TCPListener)
if ok {
    _ = tcpListener.SetKeepAlive(true)
    _ = tcpListener.SetKeepAlivePeriod(3 * time.Minute)
}
```

#### 3. Self-Testing with Retries
```go
// Try connecting to our own server as a self-test with retries
client := http.Client{
    Timeout:   5 * time.Second,
    Transport: transport,
}

localURL := fmt.Sprintf("http://127.0.0.1:%d/health", s.cfg.APIPort)

// Retry up to 3 times with increasing delays
for attempt := 0; attempt < 3; attempt++ {
    resp, err = client.Get(localURL)
    if err == nil {
        // Success
        break
    }
    // Exponential backoff: 500ms, 1s, 2s
    time.Sleep(time.Duration(500*(1<<attempt)) * time.Millisecond)
}
```

#### 4. Connection Diagnostics
```go
// Try connecting via raw TCP socket to see if anything is listening
conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 2*time.Second)
if err != nil {
    logger.Error("TCP connection to local port failed", 
        zap.Int("port", port),
        zap.Error(err))
} else {
    conn.Close()
    logger.Info("TCP connection to local port succeeded, HTTP layer issue likely", 
        zap.Int("port", port))
}
```

#### 5. Enhanced Health Endpoint
The health endpoint now provides comprehensive information about the server state, including connection details, relay status, and server information.

## WebSocket Relay Connectivity Fixes

### Root Causes Identified
1. **Provider-Specific Requirements**: Different WebSocket providers (Cloudflare, Ankr) needed specific headers
2. **Ping/Pong Timeouts**: Default 20-second ping interval was too infrequent for some providers
3. **60-Second Timeouts**: Some providers (especially publicnode.com) had strict 60-second timeouts
4. **Ineffective Reconnection Logic**: The backoff strategy wasn't optimized for different endpoint behaviors

### Implemented Fixes

#### 1. Provider-Specific Headers
```go
// Endpoint-specific configuration
if strings.Contains(endpoint, "cloudflare") {
    // Cloudflare requires specific headers
    header.Set("Origin", "https://www.cloudflare-eth.com")
    header.Set("CF-Access-Client-Id", sr.cfg.Get("CF_ACCESS_CLIENT_ID", ""))
    header.Set("CF-Access-Client-Secret", sr.cfg.Get("CF_ACCESS_CLIENT_SECRET", ""))
} else if strings.Contains(endpoint, "ankr") {
    // Ankr API requires JWT or API key
    apiKey := sr.cfg.Get("ANKR_API_KEY", "")
    if apiKey != "" {
        header.Set("Authorization", "Bearer "+apiKey)
    }
    header.Set("Origin", "https://www.ankr.com")
}
```

#### 2. More Aggressive Ping/Pong Cycle
```go
// Enhanced ping loop with more frequent pings
go func() {
    pingTicker := time.NewTicker(15 * time.Second)  // More frequent pings
    heartbeatTicker := time.NewTicker(50 * time.Second) // Send heartbeat before timeout
    defer pingTicker.Stop()
    defer heartbeatTicker.Stop()
    
    for {
        select {
        case <-pingTicker.C:
            // Send ping with timestamp
            pingData := fmt.Sprintf("ping-%d", time.Now().Unix())
            err := wc.Conn.WriteControl(websocket.PingMessage, []byte(pingData), time.Now().Add(5*time.Second))
            
        case <-heartbeatTicker.C:
            // Send a heartbeat message to keep connection alive
            // This is especially important for publicnode.com which has a 60s timeout
            sr.sendHeartbeat(wc)
        }
    }
}()
```

#### 3. Intelligent Reconnection Logic
```go
// If this is a Cloudflare or Ankr endpoint and we have at least one working connection,
// use a longer backoff to avoid unnecessary reconnection attempts
isProblematicEndpoint := strings.Contains(endpoint, "cloudflare") || 
                        strings.Contains(endpoint, "ankr") || 
                        strings.Contains(endpoint, "api.mainnet-beta.solana.com")

var attempt int
if isProblematicEndpoint && activeConnections > 0 {
    // Use higher starting backoff for problematic endpoints if we have other working connections
    attempt = sr.backoff[endpoint] + 2
    if attempt > 8 {
        attempt = 8 // Cap at ~256s for problematic endpoints
    }
} else {
    // Standard backoff for primary endpoints or when we have no connections
    attempt = sr.backoff[endpoint] + 1
    if attempt > 6 {
        attempt = 6 // Cap at ~32s
    }
}
```

## New Diagnostic Tools

### HTTP Connection Tester
Created a new tool (`http-connection-tester.go`) that:
- Tests raw TCP connections to verify port availability
- Makes HTTP requests with configurable retries and timeouts
- Shows detailed diagnostic information for connection failures
- Helps identify whether issues are at the network, TCP, or HTTP layer

### Network Diagnostics Tool
Enhanced the existing network diagnostics tool to:
- Test HTTP server bindings on multiple interfaces
- Validate relay endpoints with comprehensive tests
- Check DNS resolution for all endpoints
- Verify loopback interfaces are working correctly

## How to Use

### Starting the Server
Use the new `start-api-server.ps1` script to:
1. Run pre-flight network diagnostics
2. Check for port availability
3. Start the server with proper environment configuration
4. Perform post-start connection tests if needed

```powershell
.\start-api-server.ps1 -Port 9000 -Host 0.0.0.0 -Debug
```

### Testing Connections
Use the HTTP connection tester to verify server connectivity:

```powershell
go run .\tools\http-connection-tester.go -host 127.0.0.1 -port 9000 -path /health -v -headers
```

## Verification Steps

1. Start the server using the new script
2. Check the logs for successful binding and self-test
3. Use the HTTP connection tester to verify external connectivity
4. Monitor the relay connections to ensure they remain stable
5. Check the enhanced health endpoint for comprehensive status information

## Future Enhancements

1. **Circuit Breaker Pattern**: Implement a circuit breaker for problematic endpoints
2. **Connection Pool Management**: Add more sophisticated connection pool management for relays
3. **Metrics Tracking**: Track connection success rates and latency for each endpoint
4. **Adaptive Timeouts**: Implement adaptive timeouts based on historical connection performance
5. **Health-Based Load Balancing**: Prefer healthier endpoints based on connection statistics
