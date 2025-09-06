#!/usr/bin/env node
/**
 * Bitcoin Sprint Web - Smart Startup Script
 * Automatically detects backend tier and starts web app on correct port
 */

import { exec } from 'child_process';
import { promisify } from 'util';

const execAsync = promisify(exec);

// Tier configuration
const TIERS = {
  free: { port: 3000, backendPort: 8080 },
  business: { port: 3001, backendPort: 8082 },
  enterprise: { port: 3002, backendPort: 9000 }
};

/**
 * Check if a backend is running on a specific port
 */
async function checkBackend(port) {
  try {
    const response = await fetch(`http://localhost:${port}/health`, {
      method: 'GET',
      headers: { 'Content-Type': 'application/json' },
      signal: AbortSignal.timeout(2000)
    });
    return response.ok;
  } catch (error) {
    return false;
  }
}

/**
 * Detect which tier is currently running
 */
async function detectTier() {
  console.log('🔍 Detecting Bitcoin Sprint backend tier...');
  
  // Check in priority order: enterprise -> business -> free
  for (const [tierName, config] of Object.entries(TIERS).reverse()) {
    console.log(`   Checking ${tierName.toUpperCase()} tier (port ${config.backendPort})...`);
    
    if (await checkBackend(config.backendPort)) {
      console.log(`✅ Found active ${tierName.toUpperCase()} tier backend!`);
      return tierName;
    }
  }
  
  console.log('⚠️  No backend detected, defaulting to FREE tier');
  return 'free';
}

/**
 * Start the web application
 */
async function startWebApp() {
  try {
    // Get command line arguments
    const args = process.argv.slice(2);
    const isProduction = args.includes('--production') || args.includes('-p');
    const forceTier = args.find(arg => ['free', 'business', 'enterprise'].includes(arg));
    
    // Detect or use forced tier
    const tier = forceTier || await detectTier();
    const config = TIERS[tier];
    
    console.log(`🚀 Starting Bitcoin Sprint Web Dashboard...`);
    console.log(`   Tier: ${tier.toUpperCase()}`);
    console.log(`   Web Port: ${config.port}`);
    console.log(`   Backend Port: ${config.backendPort}`);
    console.log(`   Environment: ${isProduction ? 'PRODUCTION' : 'DEVELOPMENT'}`);
    
    // Set environment variables
    process.env.BITCOIN_SPRINT_TIER = tier;
    process.env.DETECTED_TIER = tier;
    process.env.WEB_PORT = config.port.toString();
    
    // Build the command
    const command = isProduction 
      ? `npm run start:${tier}`
      : `npm run dev:${tier}`;
    
    console.log(`   Command: ${command}`);
    console.log('');
    
    // Execute the command
    const child = exec(command, (error, stdout, stderr) => {
      if (error) {
        console.error(`❌ Error starting web app: ${error}`);
        process.exit(1);
      }
    });
    
    // Forward output
    child.stdout.on('data', (data) => process.stdout.write(data));
    child.stderr.on('data', (data) => process.stderr.write(data));
    
    // Handle termination
    process.on('SIGINT', () => {
      console.log('\n🛑 Shutting down Bitcoin Sprint Web Dashboard...');
      child.kill('SIGINT');
      process.exit(0);
    });
    
    process.on('SIGTERM', () => {
      console.log('\n🛑 Shutting down Bitcoin Sprint Web Dashboard...');
      child.kill('SIGTERM');
      process.exit(0);
    });
    
  } catch (error) {
    console.error(`❌ Failed to start web app: ${error.message}`);
    process.exit(1);
  }
}

// Show help
function showHelp() {
  console.log(`
🚀 Bitcoin Sprint Web - Smart Startup Script

Usage: node smart-start.js [options] [tier]

Options:
  --production, -p    Start in production mode
  --help, -h         Show this help message

Tiers:
  free               Force FREE tier (port 3000)
  business           Force BUSINESS tier (port 3001) 
  enterprise         Force ENTERPRISE tier (port 3002)

Examples:
  node smart-start.js                    # Auto-detect tier, development mode
  node smart-start.js --production       # Auto-detect tier, production mode
  node smart-start.js enterprise         # Force enterprise tier
  node smart-start.js business -p        # Force business tier, production

The script will automatically:
✅ Detect which Bitcoin Sprint backend is running
✅ Start the web app on the correct port for that tier
✅ Set appropriate environment variables
✅ Handle graceful shutdown
  `);
}

// Main execution
if (process.argv.includes('--help') || process.argv.includes('-h')) {
  showHelp();
} else {
  startWebApp().catch(console.error);
}
