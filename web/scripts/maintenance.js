#!/usr/bin/env node

/**
 * Bitcoin Sprint Maintenance CLI Tool
 * 
 * Usage:
 *   node scripts/maintenance.js status
 *   node scripts/maintenance.js enable "System update in progress"
 *   node scripts/maintenance.js disable
 *   node scripts/maintenance.js update v1.2.0
 *   node scripts/maintenance.js update v1.1.0 --rollback
 *   node scripts/maintenance.js health
 */

const fs = require('fs/promises');
const path = require('path');

// Simple logger for CLI
const logger = {
  info: (msg) => console.log(`ℹ️  ${msg}`),
  success: (msg) => console.log(`✅ ${msg}`),
  error: (msg) => console.error(`❌ ${msg}`),
  warn: (msg) => console.warn(`⚠️  ${msg}`),
};

// CLI Functions
async function getMaintenanceStatus() {
  try {
    const maintenancePath = path.join(process.cwd(), 'data', 'maintenance.json');
    const data = await fs.readFile(maintenancePath, 'utf-8');
    const status = JSON.parse(data);
    
    logger.info('Maintenance Status:');
    console.log(JSON.stringify(status, null, 2));
    
    if (status.enabled) {
      logger.warn('System is currently in maintenance mode');
    } else {
      logger.success('System is operational');
    }
  } catch (error) {
    if (error.code === 'ENOENT') {
      logger.success('Maintenance mode is disabled');
    } else {
      logger.error(`Failed to check maintenance status: ${error.message}`);
      process.exit(1);
    }
  }
}

async function enableMaintenance(reason = 'System maintenance in progress') {
  try {
    const maintenancePath = path.join(process.cwd(), 'data', 'maintenance.json');
    const maintenanceState = {
      enabled: true,
      reason,
      started_at: new Date().toISOString(),
      estimated_duration: '30 minutes',
    };

    await fs.mkdir(path.dirname(maintenancePath), { recursive: true });
    await fs.writeFile(maintenancePath, JSON.stringify(maintenanceState, null, 2), 'utf-8');
    
    logger.success(`Maintenance mode enabled: ${reason}`);
  } catch (error) {
    logger.error(`Failed to enable maintenance mode: ${error.message}`);
    process.exit(1);
  }
}

async function disableMaintenance() {
  try {
    const maintenancePath = path.join(process.cwd(), 'data', 'maintenance.json');
    await fs.unlink(maintenancePath);
    logger.success('Maintenance mode disabled');
  } catch (error) {
    if (error.code === 'ENOENT') {
      logger.info('Maintenance mode was already disabled');
    } else {
      logger.error(`Failed to disable maintenance mode: ${error.message}`);
      process.exit(1);
    }
  }
}

async function updateSystemState(version, rollback = false) {
  try {
    const statePath = path.join(process.cwd(), 'data', 'update_state.json');
    const newState = {
      version,
      last_updated: new Date().toISOString(),
      rollback,
    };

    await fs.mkdir(path.dirname(statePath), { recursive: true });
    await fs.writeFile(statePath, JSON.stringify(newState, null, 2), 'utf-8');
    
    const action = rollback ? 'rolled back to' : 'updated to';
    logger.success(`System state ${action} version ${version}`);
  } catch (error) {
    logger.error(`Failed to update system state: ${error.message}`);
    process.exit(1);
  }
}

async function checkHealth() {
  try {
    logger.info('Performing system health check...');
    
    const checks = {};
    
    // Check maintenance mode
    try {
      const maintenancePath = path.join(process.cwd(), 'data', 'maintenance.json');
      await fs.access(maintenancePath);
      const data = await fs.readFile(maintenancePath, 'utf-8');
      const maintenance = JSON.parse(data);
      if (maintenance.enabled) {
        checks.maintenance = { status: 'fail', message: maintenance.reason || 'System in maintenance mode' };
      } else {
        checks.maintenance = { status: 'pass', message: 'Not in maintenance mode' };
      }
    } catch (error) {
      if (error.code === 'ENOENT') {
        checks.maintenance = { status: 'pass', message: 'Not in maintenance mode' };
      } else {
        checks.maintenance = { status: 'fail', message: 'Cannot check maintenance status' };
      }
    }

    // Check state file
    try {
      const statePath = path.join(process.cwd(), 'data', 'update_state.json');
      await fs.access(statePath);
      checks.state_file = { status: 'pass', message: 'State file accessible' };
    } catch {
      checks.state_file = { status: 'fail', message: 'State file not accessible' };
    }

    // Check data directory
    try {
      const dataPath = path.join(process.cwd(), 'data');
      await fs.access(dataPath);
      checks.data_directory = { status: 'pass', message: 'Data directory accessible' };
    } catch {
      checks.data_directory = { status: 'fail', message: 'Data directory not accessible' };
    }

    const health = {
      status: Object.values(checks).every(check => check.status === 'pass') ? 'healthy' : 'degraded',
      checks,
      timestamp: new Date().toISOString(),
    };

    logger.info('Health Check Results:');
    console.log(JSON.stringify(health, null, 2));
    
    if (health.status === 'healthy') {
      logger.success('System is healthy');
    } else {
      logger.warn('System has issues that need attention');
    }
  } catch (error) {
    logger.error(`Health check failed: ${error.message}`);
    process.exit(1);
  }
}

// CLI Entry Point
async function main() {
  const args = process.argv.slice(2);
  const command = args[0];

  try {
    switch (command) {
      case 'status':
        await getMaintenanceStatus();
        break;

      case 'enable':
        const reason = args[1] || 'System maintenance in progress';
        await enableMaintenance(reason);
        break;

      case 'disable':
        await disableMaintenance();
        break;

      case 'update':
        const version = args[1];
        const isRollback = args.includes('--rollback');
        if (!version) {
          logger.error('Version is required for update command');
          process.exit(1);
        }
        await updateSystemState(version, isRollback);
        break;

      case 'health':
        await checkHealth();
        break;

      default:
        logger.info('Bitcoin Sprint Maintenance CLI');
        logger.info('');
        logger.info('Commands:');
        logger.info('  status                           - Check maintenance status');
        logger.info('  enable [reason]                  - Enable maintenance mode');
        logger.info('  disable                          - Disable maintenance mode');
        logger.info('  update <version> [--rollback]    - Update system state');
        logger.info('  health                           - Perform health check');
        logger.info('');
        logger.info('Examples:');
        logger.info('  node scripts/maintenance.js status');
        logger.info('  node scripts/maintenance.js enable "Deploying new features"');
        logger.info('  node scripts/maintenance.js update v1.2.0');
        logger.info('  node scripts/maintenance.js update v1.1.0 --rollback');
        logger.info('  node scripts/maintenance.js health');
    }
  } catch (error) {
    logger.error(`Command failed: ${error.message}`);
    process.exit(1);
  }
}

if (require.main === module) {
  main();
}
