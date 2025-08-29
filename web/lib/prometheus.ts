import client from 'prom-client';
// Note: avoid importing updateState or maintenance at top-level to prevent circular
// dependency issues. We'll dynamically import them inside updateMaintenanceMetrics.

// Create a Registry
const register = new client.Registry();

// Add default metrics (memory, CPU, etc.)
client.collectDefaultMetrics({ register });

// Custom Maintenance Metrics
const maintenanceStatus = new client.Gauge({
  name: 'bitcoin_sprint_maintenance_mode',
  help: 'Current maintenance mode status (1 = enabled, 0 = disabled)',
  registers: [register],
});

const systemHealthStatus = new client.Gauge({
  name: 'bitcoin_sprint_system_health',
  help: 'System health status (1 = healthy, 0.5 = degraded, 0 = unhealthy)',
  registers: [register],
});

const healthChecksPassed = new client.Gauge({
  name: 'bitcoin_sprint_health_checks_passed',
  help: 'Number of health checks that passed',
  registers: [register],
});

const healthChecksFailed = new client.Gauge({
  name: 'bitcoin_sprint_health_checks_failed',
  help: 'Number of health checks that failed',
  registers: [register],
});

const systemVersionInfo = new client.Gauge({
  name: 'bitcoin_sprint_version_info',
  help: 'System version information',
  labelNames: ['version', 'rollback_status'],
  registers: [register],
});

const lastUpdateTimestamp = new client.Gauge({
  name: 'bitcoin_sprint_last_update_timestamp',
  help: 'Timestamp of last system update',
  registers: [register],
});

const maintenanceStartTime = new client.Gauge({
  name: 'bitcoin_sprint_maintenance_start_timestamp',
  help: 'Timestamp when maintenance mode was enabled',
  registers: [register],
});

const apiRequestsTotal = new client.Counter({
  name: 'bitcoin_sprint_api_requests_total',
  help: 'Total number of API requests',
  labelNames: ['endpoint', 'method', 'status_code'],
  registers: [register],
});

const apiRequestDuration = new client.Histogram({
  name: 'bitcoin_sprint_api_request_duration_seconds',
  help: 'Duration of API requests in seconds',
  labelNames: ['endpoint', 'method'],
  buckets: [0.1, 0.5, 1, 2, 5, 10],
  registers: [register],
});

const cacheHitsTotal = new client.Counter({
  name: 'bitcoin_sprint_cache_hits_total',
  help: 'Total number of cache hits',
  labelNames: ['cache_type'],
  registers: [register],
});

const cacheMissesTotal = new client.Counter({
  name: 'bitcoin_sprint_cache_misses_total',
  help: 'Total number of cache misses',
  labelNames: ['cache_type'],
  registers: [register],
});

// Entropy Bridge Metrics
const entropyBridgeAvailable = new client.Gauge({
  name: 'bitcoin_sprint_entropy_bridge_available',
  help: 'Entropy bridge availability status (1 = available, 0 = unavailable)',
  registers: [register],
});

const entropyBridgeRustAvailable = new client.Gauge({
  name: 'bitcoin_sprint_entropy_bridge_rust_available',
  help: 'Rust entropy bridge availability (1 = available, 0 = unavailable)',
  registers: [register],
});

const entropyBridgeFallbackMode = new client.Gauge({
  name: 'bitcoin_sprint_entropy_bridge_fallback_mode',
  help: 'Entropy bridge fallback mode status (1 = fallback active, 0 = primary active)',
  registers: [register],
});

const entropyBridgeSecretGenerationTotal = new client.Counter({
  name: 'bitcoin_sprint_entropy_bridge_secret_generation_total',
  help: 'Total number of admin secrets generated',
  labelNames: ['source', 'encoding'],
  registers: [register],
});

const entropyBridgeSecretGenerationDuration = new client.Histogram({
  name: 'bitcoin_sprint_entropy_bridge_secret_generation_duration_seconds',
  help: 'Duration of admin secret generation in seconds',
  labelNames: ['source', 'encoding'],
  buckets: [0.001, 0.005, 0.01, 0.05, 0.1, 0.5],
  registers: [register],
});

const entropyBridgeLastSecretGenerated = new client.Gauge({
  name: 'bitcoin_sprint_entropy_bridge_last_secret_generated_timestamp',
  help: 'Timestamp of last admin secret generation',
  registers: [register],
});

// Entropy Bridge Metrics Update Function
export async function updateEntropyBridgeMetrics(): Promise<void> {
  try {
    // Dynamically import the entropy bridge to avoid circular dependencies
    let entropyBridge: any = null;

    try {
      const eb = await import('./rust-entropy-bridge');
      if (eb && typeof eb.getEntropyBridge === 'function') {
        entropyBridge = eb.getEntropyBridge();
      }
    } catch (err) {
      console.debug('Could not import entropy bridge for metrics:', err);
    }

    if (entropyBridge) {
      const status = entropyBridge.getStatus();

      // Update availability metrics
      entropyBridgeAvailable.set(status.available ? 1 : 0);
      entropyBridgeRustAvailable.set(status.rustAvailable ? 1 : 0);
      entropyBridgeFallbackMode.set(status.fallbackMode ? 1 : 0);

      // Update last secret generation timestamp if available
      // This would be set when secrets are actually generated
      if ((global as any).secretLastGenerated) {
        entropyBridgeLastSecretGenerated.set((global as any).secretLastGenerated / 1000);
      }
    } else {
      // If entropy bridge is not available, set all metrics to 0
      entropyBridgeAvailable.set(0);
      entropyBridgeRustAvailable.set(0);
      entropyBridgeFallbackMode.set(1); // Fallback mode active
      entropyBridgeLastSecretGenerated.set(0);
    }

  } catch (error) {
    console.error('Failed to update entropy bridge metrics:', error);
    // Set error state
    entropyBridgeAvailable.set(0);
    entropyBridgeRustAvailable.set(0);
    entropyBridgeFallbackMode.set(1);
  }
}

// Update metrics function
export async function updateMaintenanceMetrics(): Promise<void> {
  try {
    // Dynamically import maintenance and updateState helpers to prevent circular imports
    let maintenance: any = null;
    let health: any = null;
    let updateState: any = null;

    try {
      const m = await import('./maintenance');
      if (m && typeof m.getMaintenanceStatus === 'function') {
        maintenance = await m.getMaintenanceStatus();
      }
      if (m && typeof m.performSystemHealthCheck === 'function') {
        health = await m.performSystemHealthCheck();
      }
    } catch (err) {
      console.debug('Could not import maintenance helpers for metrics:', err);
    }

    try {
      const u = await import('./updateState');
      if (u && typeof u.getUpdateState === 'function') {
        updateState = await u.getUpdateState();
      }
    } catch (err) {
      console.debug('Could not import updateState for metrics:', err);
    }

    // Update maintenance status
    maintenanceStatus.set(maintenance?.enabled ? 1 : 0);
    if (maintenance?.enabled && maintenance.started_at) {
      maintenanceStartTime.set(new Date(maintenance.started_at).getTime() / 1000);
    } else {
      maintenanceStartTime.set(0);
    }

    // Update health metrics
    if (health) {
      const healthValue = health.status === 'healthy' ? 1 : 
                         health.status === 'degraded' ? 0.5 : 0;
      systemHealthStatus.set(healthValue);

      const passedChecks = Object.values(health.checks).filter((check: any) => check.status === 'pass').length;
      const failedChecks = Object.values(health.checks).filter((check: any) => check.status === 'fail').length;
      
      healthChecksPassed.set(passedChecks);
      healthChecksFailed.set(failedChecks);
    }

    // Update version info
    if (updateState) {
      systemVersionInfo.labels(updateState.version, updateState.rollback ? 'rollback' : 'normal').set(1);
      lastUpdateTimestamp.set(new Date(updateState.last_updated).getTime() / 1000);
    }

    // Update entropy bridge metrics
    await updateEntropyBridgeMetrics();

  } catch (error) {
    console.error('Failed to update maintenance metrics:', error);
  }
}

// Metrics collection functions
export function recordApiRequest(endpoint: string, method: string, statusCode: number, duration: number): void {
  apiRequestsTotal.labels(endpoint, method, statusCode.toString()).inc();
  apiRequestDuration.labels(endpoint, method).observe(duration);
}

export function recordCacheHit(cacheType: string): void {
  cacheHitsTotal.labels(cacheType).inc();
}

export function recordCacheMiss(cacheType: string): void {
  cacheMissesTotal.labels(cacheType).inc();
}

// Entropy Bridge Metrics Functions
export function recordEntropySecretGeneration(source: 'rust' | 'nodejs' | 'env', encoding: 'base64' | 'hex' | 'raw', duration: number): void {
  entropyBridgeSecretGenerationTotal.labels(source, encoding).inc();
  entropyBridgeSecretGenerationDuration.labels(source, encoding).observe(duration);
  entropyBridgeLastSecretGenerated.set(Date.now() / 1000);
}

// Initialize metrics update interval
let metricsInterval: NodeJS.Timeout | null = null;

export function startMetricsCollection(intervalMs: number = 30000): void {
  if (metricsInterval) {
    clearInterval(metricsInterval);
  }
  
  // Update metrics immediately
  updateMaintenanceMetrics();
  
  // Set up periodic updates
  metricsInterval = setInterval(updateMaintenanceMetrics, intervalMs);
}

export function stopMetricsCollection(): void {
  if (metricsInterval) {
    clearInterval(metricsInterval);
    metricsInterval = null;
  }
}

// Export the registry for Prometheus endpoint
export { register };

// Export individual metrics for manual updates
export const metrics = {
  maintenanceStatus,
  systemHealthStatus,
  healthChecksPassed,
  healthChecksFailed,
  systemVersionInfo,
  lastUpdateTimestamp,
  maintenanceStartTime,
  apiRequestsTotal,
  apiRequestDuration,
  cacheHitsTotal,
  cacheMissesTotal,
  // Entropy Bridge Metrics
  entropyBridgeAvailable,
  entropyBridgeRustAvailable,
  entropyBridgeFallbackMode,
  entropyBridgeSecretGenerationTotal,
  entropyBridgeSecretGenerationDuration,
  entropyBridgeLastSecretGenerated,
};

// Auto-start metrics collection when module is imported
if (typeof window === 'undefined') { // Only in Node.js environment
  startMetricsCollection();
}
