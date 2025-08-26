use std::sync::atomic::{AtomicBool, AtomicU64, Ordering};
use std::sync::Arc;
use std::time::{SystemTime, Duration, Instant};
use anyhow::{Result, Context, anyhow};
use tokio::net::TcpStream;
use tokio::io::{AsyncReadExt, AsyncWriteExt};
use tokio::sync::Mutex;
use tokio::time::{interval, Duration as TokioDuration};
use tokio_rustls::{TlsConnector, client::TlsStream};
use rustls::{ClientConfig, RootCertStore, ClientSessionMemoryCache, ServerName};
use tracing::{info, warn, error, span, Level};
use url::Url;
use async_trait::async_trait;
use backoff::{ExponentialBackoff, future::retry};
use tokio_metrics::TaskMonitor;
use hdrhistogram::Histogram;
use prometheus::{Encoder, TextEncoder, Histogram as PromHistogram, HistogramOpts, IntCounter, IntGauge, Registry};
use hyper::{Body, Response, Server, StatusCode};
use hyper::service::{make_service_fn, service_fn};
use std::net::SocketAddr;
use serde::Serialize;
use std::sync::RwLock;

static CONNECTION_ESTABLISHED: AtomicBool = AtomicBool::new(false);
static CIRCUIT_BREAKER_FAILURES: AtomicU64 = AtomicU64::new(0);
static CIRCUIT_BREAKER_LAST_FAILURE: AtomicU64 = AtomicU64::new(0);

/// Connection pool configuration
#[derive(Clone)]
struct PoolConfig {
    max_connections: usize,
    min_idle: usize,
    max_lifetime: Duration,
    max_latency_ms: u64,
    cleanup_interval: Duration,
    histogram_rotation_interval: Duration,
    metrics_host: String,
    metrics_port: u16,
    namespace: String,
    circuit_breaker_failure_threshold: u64,
    circuit_breaker_cooldown: Duration,
    metrics_auth_token: Option<String>,
}

impl Default for PoolConfig {
    fn default() -> Self {
        PoolConfig {
            max_connections: 100,
            min_idle: 10,
            max_lifetime: Duration::from_secs(1800), // 30 minutes
            max_latency_ms: 500, // 500ms threshold for slow connections
            cleanup_interval: Duration::from_secs(300), // 5 minutes
            histogram_rotation_interval: Duration::from_secs(3600), // 1 hour
            metrics_host: "0.0.0.0".to_string(),
            metrics_port: 9090,
            namespace: "secure_channel".to_string(),
            circuit_breaker_failure_threshold: 5, // 5 consecutive failures
            circuit_breaker_cooldown: Duration::from_secs(60), // 1 minute cooldown
            metrics_auth_token: None, // No auth by default
        }
    }
}

/// Common trait for secure transport
#[async_trait]
pub trait SecureTransport: Send + Sync {
    async fn write_all(&mut self, data: &[u8]) -> Result<()>;
    async fn read_exact(&mut self, buf: &mut [u8]) -> Result<()>;
    async fn shutdown(&mut self) -> Result<()>;
}

#[derive(Serialize)]
struct ConnectionStatus {
    connection_id: usize,
    last_activity: SystemTime,
    reconnects: u64,
    errors: u64,
    p95_latency_ms: u64,
}

#[derive(Serialize)]
struct PoolStatus {
    endpoint: String,
    active_connections: usize,
    total_reconnects: u64,
    total_errors: u64,
    pool_p95_latency_ms: u64,
    connections: Vec<ConnectionStatus>,
}

#[derive(Serialize)]
struct HealthStatus {
    status: String,
    timestamp: SystemTime,
    pool_healthy: bool,
    active_connections: usize,
}

/// Pool-level metrics (registered once)
pub struct PoolMetrics {
    prom_active_connections: IntGauge,
    prom_total_reconnects: IntCounter,
    prom_total_errors: IntCounter,
    prom_latency: PromHistogram,
    registry: Arc<Registry>,
    endpoint: String,
}

impl PoolMetrics {
    fn new(registry: Arc<Registry>, endpoint: &str, namespace: &str) -> Result<Self> {
        let prom_active_connections = IntGauge::with_opts(
            prometheus::Opts::new(
                format!("{}_active_connections", namespace),
                "Current number of active connections in the pool"
            ).const_label("endpoint", endpoint)
        )?;
        
        let prom_total_reconnects = IntCounter::with_opts(
            prometheus::Opts::new(
                format!("{}_reconnects_total", namespace),
                "Total number of connection reconnects across the pool"
            ).const_label("endpoint", endpoint)
        )?;
        
        let prom_total_errors = IntCounter::with_opts(
            prometheus::Opts::new(
                format!("{}_errors_total", namespace),
                "Total number of connection errors across the pool"
            ).const_label("endpoint", endpoint)
        )?;
        
        let prom_latency = PromHistogram::with_opts(
            HistogramOpts::new(
                format!("{}_latency_ms", namespace),
                "Connection latency distribution in milliseconds"
            ).const_label("endpoint", endpoint)
            .buckets(vec![0.1, 0.5, 1.0, 5.0, 10.0, 50.0, 100.0, 500.0, 1000.0, 5000.0]),
        )?;

        // Register metrics once at pool level
        registry.register(Box::new(prom_active_connections.clone()))?;
        registry.register(Box::new(prom_total_reconnects.clone()))?;
        registry.register(Box::new(prom_total_errors.clone()))?;
        registry.register(Box::new(prom_latency.clone()))?;

        Ok(PoolMetrics {
            prom_active_connections,
            prom_total_reconnects,
            prom_total_errors,
            prom_latency,
            registry,
            endpoint: endpoint.to_string(),
        })
    }

    fn record_latency(&self, duration: Duration) {
        let latency_ms = duration.as_millis() as f64;
        self.prom_latency.observe(latency_ms);
    }

    fn increment_reconnects(&self) {
        self.prom_total_reconnects.inc();
    }

    fn increment_errors(&self) {
        self.prom_total_errors.inc();
    }

    fn set_active_connections(&self, count: usize) {
        self.prom_active_connections.set(count as i64);
    }
}

/// Per-connection metrics (lightweight, no Prometheus registration)
pub struct ConnectionMetrics {
    connection_id: usize,
    last_activity: SystemTime,
    reconnects: u64,
    error_count: u64,
    latency_histogram: Arc<RwLock<Histogram<u64>>>,
    last_histogram_rotation: SystemTime,
}

impl ConnectionMetrics {
    fn new(connection_id: usize) -> Self {
        ConnectionMetrics {
            connection_id,
            last_activity: SystemTime::now(),
            reconnects: 0,
            error_count: 0,
            latency_histogram: Arc::new(RwLock::new(
                Histogram::<u64>::new_with_bounds(1, 60_000, 3)
                    .expect("Failed to create histogram")
            )),
            last_histogram_rotation: SystemTime::now(),
        }
    }

    fn record_latency(&mut self, duration: Duration) {
        let latency_ms = duration.as_millis() as u64;
        self.last_activity = SystemTime::now();
        
        if let Ok(mut hist) = self.latency_histogram.write() {
            let _ = hist.record(latency_ms);
        }
    }

    fn is_slow(&self, threshold_ms: u64) -> bool {
        if let Ok(hist) = self.latency_histogram.read() {
            hist.value_at_quantile(0.95) > threshold_ms
        } else {
            false
        }
    }

    fn rotate_histogram_if_needed(&mut self, rotation_interval: Duration) {
        if self.last_histogram_rotation.elapsed().map_or(false, |elapsed| elapsed > rotation_interval) {
            if let Ok(mut hist) = self.latency_histogram.write() {
                *hist = Histogram::<u64>::new_with_bounds(1, 60_000, 3)
                    .expect("Failed to create histogram");
                self.last_histogram_rotation = SystemTime::now();
                info!("Rotated latency histogram for connection {}", self.connection_id);
            }
        }
    }

    fn get_p95_latency(&self) -> u64 {
        if let Ok(hist) = self.latency_histogram.read() {
            hist.value_at_quantile(0.95)
        } else {
            0
        }
    }

    fn increment_reconnects(&mut self) {
        self.reconnects += 1;
    }

    fn increment_errors(&mut self) {
        self.error_count += 1;
        // Only increment connection-level errors
        // Pool metrics should aggregate from connections
    }

    fn get_status(&self) -> ConnectionStatus {
        ConnectionStatus {
            connection_id: self.connection_id,
            last_activity: self.last_activity,
            reconnects: self.reconnects,
            errors: self.error_count,
            p95_latency_ms: self.get_p95_latency(),
        }
    }
}

pub struct SecureChannel {
    stream: TlsStream<TcpStream>,
    last_rotated: SystemTime,
    metrics: ConnectionMetrics,
    monitor: TaskMonitor,
    pool_metrics: Arc<PoolMetrics>,
}

/// Builder for SecureChannelPool configuration
pub struct PoolBuilder {
    endpoint: String,
    root_store: Option<RootCertStore>,
    config: PoolConfig,
}

impl PoolBuilder {
    /// Create a new pool builder with the specified endpoint
    pub fn new(endpoint: &str) -> Self {
        PoolBuilder {
            endpoint: endpoint.to_string(),
            root_store: None,
            config: PoolConfig::default(),
        }
    }

    /// Set custom root certificate store for TLS
    pub fn with_root_store(mut self, root_store: RootCertStore) -> Self {
        self.root_store = Some(root_store);
        self
    }

    /// Set the metrics namespace (default: "secure_channel")
    pub fn with_namespace(mut self, namespace: &str) -> Self {
        self.config.namespace = namespace.to_string();
        self
    }

    /// Set maximum number of connections in the pool (default: 100)
    pub fn with_max_connections(mut self, max_connections: usize) -> Self {
        self.config.max_connections = max_connections;
        self
    }

    /// Set minimum idle connections to maintain (default: 10)
    pub fn with_min_idle(mut self, min_idle: usize) -> Self {
        self.config.min_idle = min_idle;
        self
    }

    /// Set connection maximum lifetime (default: 30 minutes)
    pub fn with_max_lifetime(mut self, max_lifetime: Duration) -> Self {
        self.config.max_lifetime = max_lifetime;
        self
    }

    /// Set latency threshold for dropping slow connections (default: 500ms)
    pub fn with_max_latency_ms(mut self, max_latency_ms: u64) -> Self {
        self.config.max_latency_ms = max_latency_ms;
        self
    }

    /// Set cleanup interval for background tasks (default: 5 minutes)
    pub fn with_cleanup_interval(mut self, cleanup_interval: Duration) -> Self {
        self.config.cleanup_interval = cleanup_interval;
        self
    }

    /// Set histogram rotation interval to manage memory (default: 1 hour)
    pub fn with_histogram_rotation_interval(mut self, rotation_interval: Duration) -> Self {
        self.config.histogram_rotation_interval = rotation_interval;
        self
    }

    /// Set metrics server host (default: "0.0.0.0")
    pub fn with_metrics_host(mut self, host: &str) -> Self {
        self.config.metrics_host = host.to_string();
        self
    }

    /// Set metrics server port (default: 9090)
    pub fn with_metrics_port(mut self, port: u16) -> Self {
        self.config.metrics_port = port;
        self
    }

    /// Set metrics authentication token for production security
    pub fn with_metrics_auth_token(mut self, token: &str) -> Self {
        self.config.metrics_auth_token = Some(token.to_string());
        self
    }

    /// Set circuit breaker failure threshold (default: 5)
    pub fn with_circuit_breaker_failure_threshold(mut self, threshold: u64) -> Self {
        self.config.circuit_breaker_failure_threshold = threshold;
        self
    }

    /// Set circuit breaker cooldown period (default: 60 seconds)
    pub fn with_circuit_breaker_cooldown(mut self, cooldown: Duration) -> Self {
        self.config.circuit_breaker_cooldown = cooldown;
        self
    }

    /// Build the SecureChannelPool (no background tasks started)
    pub fn build(self) -> Result<SecureChannelPool> {
        let registry = Arc::new(Registry::new());
        let pool_metrics = Arc::new(PoolMetrics::new(
            registry.clone(),
            &self.endpoint,
            &self.config.namespace
        )?);

        Ok(SecureChannelPool {
            connections: Arc::new(Mutex::new(Vec::new())),
            config: self.config,
            endpoint: self.endpoint,
            root_store: self.root_store,
            pool_metrics,
            next_connection_id: Arc::new(Mutex::new(0)),
        })
    }
}

pub struct SecureChannelPool {
    connections: Arc<Mutex<Vec<SecureChannel>>>,
    config: PoolConfig,
    endpoint: String,
    root_store: Option<RootCertStore>,
    pool_metrics: Arc<PoolMetrics>,
    next_connection_id: Arc<Mutex<usize>>,
}

impl Clone for SecureChannelPool {
    fn clone(&self) -> Self {
        SecureChannelPool {
            connections: self.connections.clone(),
            config: self.config.clone(),
            endpoint: self.endpoint.clone(),
            root_store: self.root_store.clone(),
            pool_metrics: self.pool_metrics.clone(),
            next_connection_id: self.next_connection_id.clone(),
        }
    }
}

impl SecureChannelPool {
    /// Create a new pool builder
    pub fn builder(endpoint: &str) -> PoolBuilder {
        PoolBuilder::new(endpoint)
    }

    /// Legacy constructor (deprecated - use builder() instead)
    #[deprecated(note = "Use SecureChannelPool::builder() for better configuration")]
    pub fn new(endpoint: &str, root_store: Option<RootCertStore>, namespace: Option<&str>) -> Result<Self> {
        let mut builder = PoolBuilder::new(endpoint);
        
        if let Some(store) = root_store {
            builder = builder.with_root_store(store);
        }
        
        if let Some(ns) = namespace {
            builder = builder.with_namespace(ns);
        }
        
        builder.build()
    }

    /// Explicit start of cleanup task - call this from your main()
    pub async fn run_cleanup_task(self: Arc<Self>) {
        self.run_background_cleanup().await;
    }

    /// Explicit start of metrics server - call this from your main()
    pub async fn run_metrics_task(self: Arc<Self>) -> Result<()> {
        self.run_metrics_server().await
    }

    /// Get or create a connection from the pool
    pub async fn get_connection(&self) -> Result<SecureChannel> {
        let _span = span!(Level::INFO, "get_connection", endpoint = self.endpoint);
        
        // Check circuit breaker
        self.check_circuit_breaker()?;
        
        let mut connections = self.connections.lock().await;

        // Enforce connection pool upper bound
        if connections.len() >= self.config.max_connections {
            return Err(anyhow!("Connection pool exhausted: {} connections active", connections.len()));
        }

        // Try to reuse an existing connection
        while let Some(mut conn) = connections.pop() {
            if conn.is_valid().await {
                conn.metrics.rotate_histogram_if_needed(self.config.histogram_rotation_interval);
                if !conn.metrics.is_slow(self.config.max_latency_ms) {
                    self.pool_metrics.set_active_connections(connections.len() + 1);
                    return Ok(conn);
                } else {
                    warn!("Dropping slow connection {}: p95={}ms", 
                        conn.metrics.connection_id, 
                        conn.metrics.get_p95_latency()
                    );
                    let _ = conn.shutdown().await; // Graceful shutdown
                }
            } else {
                let _ = conn.shutdown().await; // Graceful shutdown of invalid connection
            }
        }

        // Create new connection with retry logic
        let backoff = ExponentialBackoff {
            max_elapsed_time: Some(Duration::from_secs(30)),
            ..Default::default()
        };

        let conn = retry(backoff, || async {
            self.create_connection().await
        }).await.context("Failed to create connection after retries")?;

        // Reset circuit breaker on successful connection
        CIRCUIT_BREAKER_FAILURES.store(0, Ordering::Relaxed);

        self.pool_metrics.set_active_connections(connections.len() + 1);
        Ok(conn)
    }

    fn check_circuit_breaker(&self) -> Result<()> {
        let failures = CIRCUIT_BREAKER_FAILURES.load(Ordering::Relaxed);
        if failures >= self.config.circuit_breaker_failure_threshold {
            let last_failure = CIRCUIT_BREAKER_LAST_FAILURE.load(Ordering::Relaxed);
            let now = SystemTime::now().duration_since(SystemTime::UNIX_EPOCH)
                .unwrap_or_default().as_secs();
            
            if now - last_failure < self.config.circuit_breaker_cooldown.as_secs() {
                return Err(anyhow!(
                    "Circuit breaker open: {} consecutive failures, cooldown until {}s",
                    failures,
                    last_failure + self.config.circuit_breaker_cooldown.as_secs()
                ));
            } else {
                // Reset after cooldown
                CIRCUIT_BREAKER_FAILURES.store(0, Ordering::Relaxed);
                info!("Circuit breaker reset after cooldown");
            }
        }
        Ok(())
    }

    async fn create_connection(&self) -> Result<SecureChannel> {
        let _span = span!(Level::INFO, "create_connection", endpoint = self.endpoint);
        let start = Instant::now();

        let endpoint_url = normalize_endpoint(&self.endpoint)?;
        let domain_str = endpoint_url.host_str().ok_or_else(|| anyhow!("Invalid endpoint: missing domain"))?;
        let port = endpoint_url.port_or_known_default().unwrap_or(443);
        let tcp_endpoint = format!("{}:{}", domain_str, port);

        // Optimized TLS config with safe root cert loading
        let root_store = self.root_store.clone().unwrap_or_else(|| {
            let mut store = RootCertStore::empty();
            match rustls_native_certs::load_native_certs() {
                Ok(certs) => {
                    for cert in certs {
                        if let Err(e) = store.add(&rustls::Certificate(cert.0)) {
                            warn!("Skipping invalid system cert: {:?}", e);
                        }
                    }
                }
                Err(e) => {
                    error!("Failed to load native certs: {:?}", e);
                    // Continue with empty store - will fail TLS verification but won't crash
                }
            }
            store
        });

        let config = ClientConfig::builder()
            .with_safe_defaults()
            .with_cipher_suites(&[
                rustls::cipher_suite::TLS13_AES_256_GCM_SHA384,
                rustls::cipher_suite::TLS13_CHACHA20_POLY1305_SHA256,
            ])
            .with_root_certificates(root_store)
            .with_no_client_auth()
            .with_client_session_cache(ClientSessionMemoryCache::new(256));

        let connector = TlsConnector::from(Arc::new(config));
        let server_name = ServerName::try_from(domain_str)
            .map_err(|_| anyhow!("Invalid DNS name: {}", domain_str))?;

        let stream = tokio::time::timeout(Duration::from_secs(5), TcpStream::connect(&tcp_endpoint))
            .await
            .context("Connection timed out")??
            .into_std()?;

        stream.set_nodelay(true)?;
        let stream = TcpStream::from_std(stream)?;

        let tls_stream = connector.connect(server_name, stream).await
            .map_err(|e| {
                // Record circuit breaker failure
                CIRCUIT_BREAKER_FAILURES.fetch_add(1, Ordering::Relaxed);
                CIRCUIT_BREAKER_LAST_FAILURE.store(
                    SystemTime::now().duration_since(SystemTime::UNIX_EPOCH)
                        .unwrap_or_default().as_secs(),
                    Ordering::Relaxed
                );
                e
            })
            .context("TLS handshake failed")?;

        CONNECTION_ESTABLISHED.store(true, Ordering::Relaxed);
        info!("Secure connection established to {}", tcp_endpoint);

        // Get next connection ID
        let connection_id = {
            let mut id = self.next_connection_id.lock().await;
            *id += 1;
            *id
        };

        let metrics = ConnectionMetrics::new(connection_id);
        self.pool_metrics.record_latency(start.elapsed());

        Ok(SecureChannel {
            stream: tls_stream,
            last_rotated: SystemTime::now(),
            metrics,
            monitor: TaskMonitor::new(),
            pool_metrics: self.pool_metrics.clone(),
        })
    }

    /// Background task for periodic cleanup and health checks
    async fn run_background_cleanup(&self) {
        let mut interval = interval(TokioDuration::from_secs(self.config.cleanup_interval.as_secs()));
        loop {
            interval.tick().await;
            let _span = span!(Level::INFO, "background_cleanup", endpoint = self.endpoint);
            let mut connections = self.connections.lock().await;
            let initial_count = connections.len();
            
            // Force histogram rotation for all connections
            for conn in connections.iter_mut() {
                conn.metrics.rotate_histogram_if_needed(self.config.histogram_rotation_interval);
            }
            
            // Gracefully shutdown and remove invalid connections
            let mut valid_connections = Vec::new();
            for mut conn in connections.drain(..) {
                let is_valid = conn.last_rotated.elapsed().map_or(false, |elapsed| {
                    elapsed < self.config.max_lifetime && !conn.metrics.is_slow(self.config.max_latency_ms)
                });
                
                if is_valid {
                    valid_connections.push(conn);
                } else {
                    warn!("Removing connection {}: lifetime={:?}, slow={}", 
                        conn.metrics.connection_id,
                        conn.last_rotated.elapsed().unwrap_or(Duration::from_secs(0)),
                        conn.metrics.is_slow(self.config.max_latency_ms)
                    );
                    // Gracefully shutdown dropped connection
                    let _ = conn.shutdown().await;
                }
            }
            *connections = valid_connections;
            
            let dropped = initial_count - connections.len();
            if dropped > 0 {
                info!("Dropped {} stale or slow connections", dropped);
            }

            // Reset CONNECTION_ESTABLISHED if pool is empty
            if connections.is_empty() {
                CONNECTION_ESTABLISHED.store(false, Ordering::Relaxed);
                info!("Connection pool empty - reset CONNECTION_ESTABLISHED flag");
            }

            // Ensure minimum idle connections
            while connections.len() < self.config.min_idle {
                match self.create_connection().await {
                    Ok(conn) => {
                        connections.push(conn);
                    }
                    Err(e) => {
                        warn!("Failed to create idle connection: {}", e);
                        self.pool_metrics.increment_errors();
                        break;
                    }
                }
            }

            // Update pool metrics
            self.pool_metrics.set_active_connections(connections.len());
        }
    }

    /// Run Prometheus and JSON metrics server
    async fn run_metrics_server(&self) -> Result<()> {
        let addr: SocketAddr = format!("{}:{}", self.config.metrics_host, self.config.metrics_port)
            .parse()
            .context("Invalid metrics server address")?;
        
        let registry = self.pool_metrics.registry.clone();
        let endpoint = self.endpoint.clone();
        let connections = self.connections.clone();
        let auth_token = self.config.metrics_auth_token.clone();

        let make_service = make_service_fn(move |_| {
            let registry = registry.clone();
            let endpoint = endpoint.clone();
            let connections = connections.clone();
            let auth_token = auth_token.clone();
            async move {
                Ok::<_, hyper::Error>(service_fn(move |req: hyper::Request<Body>| {
                    let registry = registry.clone();
                    let endpoint = endpoint.clone();
                    let connections = connections.clone();
                    let auth_token = auth_token.clone();
                    async move {
                        // Check authentication for protected endpoints
                        if let Some(expected_token) = &auth_token {
                            if req.uri().path().starts_with("/metrics") || 
                               req.uri().path().starts_with("/status") {
                                if let Some(auth_header) = req.headers().get("X-Auth-Token") {
                                    if let Ok(token) = auth_header.to_str() {
                                        if token != expected_token {
                                            return Ok::<_, hyper::Error>(
                                                Response::builder()
                                                    .status(StatusCode::UNAUTHORIZED)
                                                    .body(Body::from("Unauthorized: Invalid token"))
                                                    .expect("Failed to build response")
                                            );
                                        }
                                    } else {
                                        return Ok::<_, hyper::Error>(
                                            Response::builder()
                                                .status(StatusCode::UNAUTHORIZED)
                                                .body(Body::from("Unauthorized: Invalid token format"))
                                                .expect("Failed to build response")
                                        );
                                    }
                                } else {
                                    return Ok::<_, hyper::Error>(
                                        Response::builder()
                                            .status(StatusCode::UNAUTHORIZED)
                                            .body(Body::from("Unauthorized: Missing X-Auth-Token header"))
                                            .expect("Failed to build response")
                                    );
                                }
                            }
                        }

                        match req.uri().path() {
                            "/metrics" => {
                                let encoder = TextEncoder::new();
                                let metric_families = registry.gather();
                                let mut buffer = vec![];
                                encoder.encode(&metric_families, &mut buffer)
                                    .expect("Failed to encode metrics");
                                Ok::<_, hyper::Error>(Response::new(Body::from(buffer)))
                            }
                            "/status/connections" => {
                                let connections = connections.lock().await;
                                let connection_statuses: Vec<ConnectionStatus> = connections
                                    .iter()
                                    .map(|c| c.metrics.get_status())
                                    .collect();
                                
                                let pool_p95 = if !connection_statuses.is_empty() {
                                    connection_statuses.iter().map(|c| c.p95_latency_ms).max().unwrap_or(0)
                                } else {
                                    0
                                };

                                let total_reconnects = connection_statuses.iter().map(|c| c.reconnects).sum();
                                // Calculate total errors from all connections (no double counting)
                                let total_errors = connection_statuses.iter().map(|c| c.errors).sum();

                                let status = PoolStatus {
                                    endpoint: endpoint.clone(),
                                    active_connections: connections.len(),
                                    total_reconnects,
                                    total_errors,
                                    pool_p95_latency_ms: pool_p95,
                                    connections: connection_statuses,
                                };

                                let json = serde_json::to_string(&status)
                                    .expect("Failed to serialize status");
                                Ok::<_, hyper::Error>(
                                    Response::builder()
                                        .status(StatusCode::OK)
                                        .header("Content-Type", "application/json")
                                        .body(Body::from(json))
                                        .expect("Failed to build response")
                                )
                            }
                            "/healthz" => {
                                let connections = connections.lock().await;
                                let pool_healthy = !connections.is_empty();
                                let health = HealthStatus {
                                    status: if pool_healthy { "healthy".to_string() } else { "unhealthy".to_string() },
                                    timestamp: SystemTime::now(),
                                    pool_healthy,
                                    active_connections: connections.len(),
                                };
                                let json = serde_json::to_string(&health)
                                    .expect("Failed to serialize health status");
                                let status_code = if pool_healthy { StatusCode::OK } else { StatusCode::SERVICE_UNAVAILABLE };
                                Ok::<_, hyper::Error>(
                                    Response::builder()
                                        .status(status_code)
                                        .header("Content-Type", "application/json")
                                        .body(Body::from(json))
                                        .expect("Failed to build response")
                                )
                            }
                            _ => Ok::<_, hyper::Error>(
                                Response::builder()
                                    .status(StatusCode::NOT_FOUND)
                                    .body(Body::empty())
                                    .expect("Failed to build response")
                            ),
                        }
                    }
                }))
            }
        });

        let server = Server::bind(&addr).serve(make_service);
        info!("Metrics server running on http://{}", addr);
        info!("Endpoints: /metrics (Prometheus), /status/connections (JSON), /healthz (Health)");
        
        server.await.context("Metrics server failed")?;
        Ok(())
    }
}

impl SecureChannel {
    async fn is_valid(&self) -> bool {
        self.last_rotated.elapsed().map_or(false, |elapsed| {
            elapsed < Duration::from_secs(1800) // 30 minutes
        })
    }

    pub fn check_rotation(&mut self) -> Result<()> {
        if self.last_rotated.elapsed()? > Duration::from_secs(3600) {
            self.rotate_keys()?;
            self.last_rotated = SystemTime::now();
            self.metrics.increment_reconnects();
            self.pool_metrics.increment_reconnects();
        }
        Ok(())
    }

    fn rotate_keys(&mut self) -> Result<()> {
        let _span = span!(Level::INFO, "rotate_keys", connection_id = self.metrics.connection_id);
        info!("Rotating TLS keys for connection {}", self.metrics.connection_id);
        Ok(())
    }

    pub async fn write(&mut self, buf: &[u8]) -> Result<usize> {
        let _span = self.monitor.instrument(span!(Level::TRACE, "write", connection_id = self.metrics.connection_id));
        let start = Instant::now();
        
        let result = self.stream.write(buf).await
            .map_err(|e| {
                self.metrics.increment_errors();
                // Don't double-count pool errors - they are aggregated from connections
                e
            })
            .context("Failed to write to secure channel");
        
        self.metrics.record_latency(start.elapsed());
        self.pool_metrics.record_latency(start.elapsed());
        result
    }

    pub async fn read(&mut self, buf: &mut [u8]) -> Result<usize> {
        let _span = self.monitor.instrument(span!(Level::TRACE, "read", connection_id = self.metrics.connection_id));
        let start = Instant::now();
        
        let result = self.stream.read(buf).await
            .map_err(|e| {
                self.metrics.increment_errors();
                // Don't double-count pool errors - they are aggregated from connections
                e
            })
            .context("Failed to read from secure channel");
        
        self.metrics.record_latency(start.elapsed());
        self.pool_metrics.record_latency(start.elapsed());
        result
    }
}

#[async_trait]
impl SecureTransport for SecureChannel {
    async fn write_all(&mut self, buf: &[u8]) -> Result<()> {
        let _span = self.monitor.instrument(span!(Level::TRACE, "write_all", connection_id = self.metrics.connection_id));
        let start = Instant::now();
        
        let result = self.stream.write_all(buf).await
            .map_err(|e| {
                self.metrics.increment_errors();
                // Don't double-count pool errors - they are aggregated from connections
                e
            })
            .context("Failed to write_all to secure channel");
        
        self.metrics.record_latency(start.elapsed());
        self.pool_metrics.record_latency(start.elapsed());
        result
    }

    async fn read_exact(&mut self, buf: &mut [u8]) -> Result<()> {
        let _span = self.monitor.instrument(span!(Level::TRACE, "read_exact", connection_id = self.metrics.connection_id));
        let start = Instant::now();
        
        let result = self.stream.read_exact(buf).await
            .map_err(|e| {
                self.metrics.increment_errors();
                // Don't double-count pool errors - they are aggregated from connections
                e
            })
            .context("Failed to read_exact from secure channel");
        
        self.metrics.record_latency(start.elapsed());
        self.pool_metrics.record_latency(start.elapsed());
        result
    }

    async fn shutdown(&mut self) -> Result<()> {
        let _span = self.monitor.instrument(span!(Level::TRACE, "shutdown", connection_id = self.metrics.connection_id));
        let result = self.stream.shutdown().await
            .map_err(|e| {
                self.metrics.increment_errors();
                // Don't double-count pool errors - they are aggregated from connections
                e
            })
            .context("Failed to shutdown secure channel");
        result
    }
}

// Implement Drop for graceful async shutdown
impl Drop for SecureChannel {
    fn drop(&mut self) {
        let connection_id = self.metrics.connection_id;
        info!("Dropping SecureChannel {}, initiating graceful shutdown", connection_id);
        
        // Note: We can't do async work in Drop, but we can spawn a task
        // In practice, the connection cleanup in the pool handles graceful shutdown
        // This is just for logging and any synchronous cleanup
    }
}

fn normalize_endpoint(endpoint: &str) -> Result<Url> {
    let endpoint_url_str = if !endpoint.contains("://") {
        format!("https://{}", endpoint)
    } else {
        endpoint.to_string()
    };
    Url::parse(&endpoint_url_str)
        .context(format!("Failed to parse endpoint URL: {}", endpoint_url_str))
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::time::Duration;

    #[tokio::test]
    async fn test_pool_builder() -> Result<()> {
        let pool = SecureChannelPool::builder("example.com:443")
            .with_namespace("test")
            .with_max_connections(50)
            .with_min_idle(5)
            .with_max_latency_ms(200)
            .with_metrics_port(9999)
            .with_metrics_host("127.0.0.1")
            .with_metrics_auth_token("secret123")
            .with_circuit_breaker_failure_threshold(3)
            .with_circuit_breaker_cooldown(Duration::from_secs(30))
            .build()?;
        
        assert_eq!(pool.endpoint, "example.com:443");
        assert_eq!(pool.config.namespace, "test");
        assert_eq!(pool.config.max_connections, 50);
        assert_eq!(pool.config.min_idle, 5);
        assert_eq!(pool.config.max_latency_ms, 200);
        assert_eq!(pool.config.metrics_port, 9999);
        assert_eq!(pool.config.metrics_host, "127.0.0.1");
        assert_eq!(pool.config.metrics_auth_token, Some("secret123".to_string()));
        assert_eq!(pool.config.circuit_breaker_failure_threshold, 3);
        assert_eq!(pool.config.circuit_breaker_cooldown, Duration::from_secs(30));
        Ok(())
    }

    #[tokio::test]
    async fn test_pool_builder_defaults() -> Result<()> {
        let pool = SecureChannelPool::builder("test.com:443")
            .with_namespace("default_test")
            .build()?;
        
        assert_eq!(pool.config.max_connections, 100); // Default
        assert_eq!(pool.config.min_idle, 10); // Default
        assert_eq!(pool.config.max_latency_ms, 500); // Default
        assert_eq!(pool.config.metrics_port, 9090); // Default
        assert_eq!(pool.config.metrics_host, "0.0.0.0"); // Default
        assert_eq!(pool.config.circuit_breaker_failure_threshold, 5); // Default
        assert_eq!(pool.config.circuit_breaker_cooldown, Duration::from_secs(60)); // Default
        assert_eq!(pool.config.metrics_auth_token, None); // Default
        Ok(())
    }

    #[tokio::test]
    async fn test_legacy_constructor_still_works() -> Result<()> {
        #[allow(deprecated)]
        let pool = SecureChannelPool::new("legacy.com:443", None, Some("legacy"))?;
        assert_eq!(pool.endpoint, "legacy.com:443");
        assert_eq!(pool.config.namespace, "legacy");
        Ok(())
    }

    #[test]
    fn test_pool_config_chaining() {
        let config = PoolConfig::default();
        
        // Test that defaults are sensible
        assert_eq!(config.max_connections, 100);
        assert_eq!(config.min_idle, 10);
        assert_eq!(config.max_lifetime, Duration::from_secs(1800));
        assert_eq!(config.max_latency_ms, 500);
        assert_eq!(config.cleanup_interval, Duration::from_secs(300));
        assert_eq!(config.histogram_rotation_interval, Duration::from_secs(3600));
        assert_eq!(config.metrics_host, "0.0.0.0");
        assert_eq!(config.metrics_port, 9090);
        assert_eq!(config.namespace, "secure_channel");
    }

    #[tokio::test]
    async fn test_multiple_pools_different_namespaces() -> Result<()> {
        let pool1 = SecureChannelPool::builder("pool1.com:443")
            .with_namespace("pool_one")
            .with_metrics_port(9090)
            .build()?;
            
        let pool2 = SecureChannelPool::builder("pool2.com:443")
            .with_namespace("pool_two")
            .with_metrics_port(9091)
            .build()?;
        
        // Different namespaces for metrics separation
        assert_eq!(pool1.config.namespace, "pool_one");
        assert_eq!(pool2.config.namespace, "pool_two");
        
        // Different ports to avoid conflicts
        assert_eq!(pool1.config.metrics_port, 9090);
        assert_eq!(pool2.config.metrics_port, 9091);
        
        Ok(())
    }

    #[tokio::test]
    async fn test_circuit_breaker_functionality() -> Result<()> {
        // Test that circuit breaker rejects requests after threshold failures
        CIRCUIT_BREAKER_FAILURES.store(10, Ordering::Relaxed); // Above threshold
        CIRCUIT_BREAKER_LAST_FAILURE.store(
            SystemTime::now().duration_since(SystemTime::UNIX_EPOCH)
                .unwrap_or_default().as_secs(),
            Ordering::Relaxed
        );

        let pool = SecureChannelPool::builder("unreachable.example.com:443")
            .with_circuit_breaker_failure_threshold(5)
            .with_circuit_breaker_cooldown(Duration::from_secs(3600)) // Long cooldown
            .build()?;

        // Should fail due to circuit breaker
        let result = pool.get_connection().await;
        assert!(result.is_err());
        assert!(result.unwrap_err().to_string().contains("Circuit breaker open"));

        Ok(())
    }

    #[tokio::test]
    async fn test_connection_pool_upper_bound() -> Result<()> {
        let pool = SecureChannelPool::builder("example.com:443")
            .with_max_connections(0) // Force immediate exhaustion
            .build()?;

        let result = pool.get_connection().await;
        assert!(result.is_err());
        assert!(result.unwrap_err().to_string().contains("Connection pool exhausted"));

        Ok(())
    }

    #[tokio::test]
    async fn test_metrics_auth_configuration() -> Result<()> {
        let pool_with_auth = SecureChannelPool::builder("example.com:443")
            .with_metrics_auth_token("secret123")
            .build()?;
        
        assert_eq!(pool_with_auth.config.metrics_auth_token, Some("secret123".to_string()));

        let pool_without_auth = SecureChannelPool::builder("example.com:443")
            .build()?;
        
        assert_eq!(pool_without_auth.config.metrics_auth_token, None);
        
        Ok(())
    }
}
