import { UpdateState, UpdateStateSchema } from "@/lib/updateState";
import fs from "fs/promises";
import path from "path";
import pino from "pino";

const logger = pino({
  level: process.env.LOG_LEVEL || "info",
  base: { service: "sprint-maintenance" },
});

// ---------------------- Maintenance Operations ----------------------

export async function updateSystemState(
  version: string,
  rollback: boolean = false
): Promise<void> {
  const statePath = process.env.SPRINT_STATE_FILE || path.join(process.cwd(), "data", "update_state.json");
  const resolved = path.normalize(path.resolve(statePath));

  // Security check
  if (!resolved.startsWith(process.cwd()) && !resolved.startsWith("/var/lib/sprint")) {
    throw new Error(`Invalid state file path: ${resolved}`);
  }

  const newState: UpdateState = {
    version,
    last_updated: new Date().toISOString(),
    rollback,
  };

  // Validate the new state
  const validation = UpdateStateSchema.safeParse(newState);
  if (!validation.success) {
    logger.error({ errors: validation.error.errors }, "State validation failed");
    throw new Error("Invalid state data");
  }

  try {
    // Ensure directory exists
    await fs.mkdir(path.dirname(resolved), { recursive: true });
    
    // Write the new state
    await fs.writeFile(resolved, JSON.stringify(newState, null, 2), "utf-8");
    
    logger.info({ version, rollback, path: resolved }, "System state updated successfully");
    // Attempt to update Prometheus metrics (dynamic import to avoid circular import issues)
    try {
      const prom = await import('./prometheus');
      if (prom && typeof prom.updateMaintenanceMetrics === 'function') {
        await prom.updateMaintenanceMetrics();
      }
    } catch (err) {
      // Non-fatal: metrics update failed or module not available
      logger.debug({ err: (err as Error).message }, 'Prometheus metrics update skipped');
    }
  } catch (error: any) {
    logger.error({ error: error.message, path: resolved }, "Failed to update system state");
    throw new Error(`Failed to update system state: ${error.message}`);
  }
}

export async function createMaintenanceMode(
  reason: string = "System maintenance in progress"
): Promise<void> {
  const maintenancePath = path.join(process.cwd(), "data", "maintenance.json");
  
  const maintenanceState = {
    enabled: true,
    reason,
    started_at: new Date().toISOString(),
    estimated_duration: "30 minutes",
  };

  try {
    await fs.mkdir(path.dirname(maintenancePath), { recursive: true });
    await fs.writeFile(maintenancePath, JSON.stringify(maintenanceState, null, 2), "utf-8");
    
    logger.info({ reason }, "Maintenance mode enabled");
    // Update Prometheus metrics after enabling maintenance
    try {
      const prom = await import('./prometheus');
      if (prom && typeof prom.updateMaintenanceMetrics === 'function') {
        await prom.updateMaintenanceMetrics();
      }
    } catch (err) {
      logger.debug({ err: (err as Error).message }, 'Prometheus metrics update skipped');
    }
  } catch (error: any) {
    logger.error({ error: error.message }, "Failed to enable maintenance mode");
    throw new Error(`Failed to enable maintenance mode: ${error.message}`);
  }
}

export async function disableMaintenanceMode(): Promise<void> {
  const maintenancePath = path.join(process.cwd(), "data", "maintenance.json");
  
  try {
    await fs.unlink(maintenancePath);
    logger.info("Maintenance mode disabled");
    // Update Prometheus metrics after disabling maintenance
    try {
      const prom = await import('./prometheus');
      if (prom && typeof prom.updateMaintenanceMetrics === 'function') {
        await prom.updateMaintenanceMetrics();
      }
    } catch (err) {
      logger.debug({ err: (err as Error).message }, 'Prometheus metrics update skipped');
    }
  } catch (error: any) {
    if (error.code !== "ENOENT") {
      logger.error({ error: error.message }, "Failed to disable maintenance mode");
      throw new Error(`Failed to disable maintenance mode: ${error.message}`);
    }
    // File doesn't exist, maintenance mode is already disabled
    logger.info("Maintenance mode was already disabled");
    // Still attempt to refresh metrics to ensure state is accurate
    try {
      const prom = await import('./prometheus');
      if (prom && typeof prom.updateMaintenanceMetrics === 'function') {
        await prom.updateMaintenanceMetrics();
      }
    } catch (err) {
      logger.debug({ err: (err as Error).message }, 'Prometheus metrics update skipped');
    }
  }
}

export async function getMaintenanceStatus(): Promise<{ enabled: boolean; reason?: string; started_at?: string; estimated_duration?: string } | null> {
  const maintenancePath = path.join(process.cwd(), "data", "maintenance.json");
  
  try {
    const data = await fs.readFile(maintenancePath, "utf-8");
    return JSON.parse(data);
  } catch (error: any) {
    if (error.code === "ENOENT") {
      return { enabled: false };
    }
    throw new Error(`Failed to read maintenance status: ${error.message}`);
  }
}

// ---------------------- System Health Check ----------------------

export async function performSystemHealthCheck(): Promise<{
  status: "healthy" | "degraded" | "maintenance";
  checks: Record<string, { status: "pass" | "fail"; message: string }>;
  timestamp: string;
}> {
  const checks: Record<string, { status: "pass" | "fail"; message: string }> = {};

  // Check if maintenance mode is enabled
  const maintenance = await getMaintenanceStatus();
  if (maintenance?.enabled) {
    return {
      status: "maintenance",
      checks: {
        maintenance: {
          status: "fail",
          message: maintenance.reason || "System in maintenance mode"
        }
      },
      timestamp: new Date().toISOString(),
    };
  }

  // Check state file accessibility
  try {
    const statePath = process.env.SPRINT_STATE_FILE || path.join(process.cwd(), "data", "update_state.json");
    await fs.access(statePath);
    checks.state_file = { status: "pass", message: "State file accessible" };
  } catch {
    checks.state_file = { status: "fail", message: "State file not accessible" };
  }

  // Check data directory
  try {
    const dataPath = path.join(process.cwd(), "data");
    await fs.access(dataPath);
    checks.data_directory = { status: "pass", message: "Data directory accessible" };
  } catch {
    checks.data_directory = { status: "fail", message: "Data directory not accessible" };
  }

  // Check disk space (if available)
  try {
    const stats = await fs.stat(process.cwd());
    checks.filesystem = { status: "pass", message: "Filesystem accessible" };
  } catch {
    checks.filesystem = { status: "fail", message: "Filesystem issues detected" };
  }

  const failedChecks = Object.values(checks).filter(check => check.status === "fail");
  const status = failedChecks.length === 0 ? "healthy" : "degraded";

  // Update maintenance/health metrics after performing health check
  try {
    const prom = await import('./prometheus');
    if (prom && typeof prom.updateMaintenanceMetrics === 'function') {
      await prom.updateMaintenanceMetrics();
    }
  } catch (err) {
    logger.debug({ err: (err as Error).message }, 'Prometheus metrics update skipped');
  }

  return {
    status,
    checks,
    timestamp: new Date().toISOString(),
  };
}
