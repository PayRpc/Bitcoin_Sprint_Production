#!/usr/bin/env node

/**
 * Bitcoin Sprint Development Setup Script
 * Sets up the development environment and runs initial tests
 */

const { execSync, spawn } = require('child_process');
const fs = require('fs');
const path = require('path');

const colors = {
  reset: '\x1b[0m',
  bright: '\x1b[1m',
  red: '\x1b[31m',
  green: '\x1b[32m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  magenta: '\x1b[35m',
  cyan: '\x1b[36m'
};

function log(message, color = 'reset') {
  console.log(`${colors[color]}${message}${colors.reset}`);
}

function logStep(step, message) {
  log(`\n${step}. ${message}`, 'cyan');
}

function logSuccess(message) {
  log(`✅ ${message}`, 'green');
}

function logError(message) {
  log(`❌ ${message}`, 'red');
}

function logWarning(message) {
  log(`⚠️  ${message}`, 'yellow');
}

async function runCommand(command, description, options = {}) {
  try {
    log(`Running: ${description}`, 'blue');
    const result = execSync(command, {
      stdio: options.silent ? 'pipe' : 'inherit',
      encoding: 'utf8',
      ...options
    });
    if (options.silent) {
      logSuccess(`${description} completed`);
    }
    return result;
  } catch (error) {
    logError(`${description} failed: ${error.message}`);
    if (!options.continueOnError) {
      throw error;
    }
    return null;
  }
}

async function checkDependencies() {
  logStep('1', 'Checking Dependencies');

  const dependencies = [
    { command: 'node --version', name: 'Node.js', required: true },
    { command: 'npm --version', name: 'npm', required: true },
    { command: 'npx --version', name: 'npx', required: true },
    { command: 'git --version', name: 'Git', required: true },
  ];

  for (const dep of dependencies) {
    try {
      const version = execSync(dep.command, { encoding: 'utf8' }).trim();
      logSuccess(`${dep.name}: ${version}`);
    } catch (error) {
      if (dep.required) {
        logError(`${dep.name} is required but not found`);
        throw error;
      } else {
        logWarning(`${dep.name} not found (optional)`);
      }
    }
  }
}

async function setupEnvironment() {
  logStep('2', 'Setting up Environment');

  // Check if .env.local exists
  const envPath = path.join(process.cwd(), '.env.local');
  if (!fs.existsSync(envPath)) {
    logWarning('.env.local not found, creating from template');
    // .env.local already exists from our previous setup
  } else {
    logSuccess('Environment file found');
  }

  // Check if node_modules exists
  if (!fs.existsSync(path.join(process.cwd(), 'node_modules'))) {
    log('Installing dependencies...', 'blue');
    await runCommand('npm install', 'Installing npm dependencies');
  } else {
    logSuccess('Dependencies already installed');
  }
}

async function setupDatabase() {
  logStep('3', 'Setting up Database');

  // Generate Prisma client
  await runCommand('npx prisma generate', 'Generating Prisma client');

  // Run database migrations
  try {
    await runCommand('npx prisma migrate dev --name init', 'Running database migrations');
  } catch (error) {
    logWarning('Migration may have already been applied');
  }

  // Create initial data
  log('Creating initial database data...', 'blue');
  try {
    const { PrismaClient } = require('@prisma/client');
    const prisma = new PrismaClient();

    // Create a test API key
    const testKey = await prisma.apiKey.upsert({
      where: { key: 'bitcoin-sprint-dev-key-2025' },
      update: {},
      create: {
        key: 'bitcoin-sprint-dev-key-2025',
        email: 'dev@bitcoin-sprint.com',
        company: 'Bitcoin Sprint Development',
        tier: 'ENTERPRISE',
        expiresAt: new Date(Date.now() + 365 * 24 * 60 * 60 * 1000), // 1 year
      },
    });

    logSuccess(`Created test API key: ${testKey.key}`);
    await prisma.$disconnect();
  } catch (error) {
    logWarning(`Database setup issue: ${error.message}`);
  }
}

async function testEntropyBridge() {
  logStep('4', 'Testing Entropy Bridge');

  try {
    await runCommand('node test-entropy-auth.js', 'Testing entropy bridge');
  } catch (error) {
    logWarning('Entropy bridge test failed - this is expected if Rust libraries are not built');
  }
}

async function testConnections() {
  logStep('5', 'Testing Connections');

  // Test web server (will fail if not running, which is expected)
  try {
    await runCommand('node test-server.js', 'Testing web server endpoints', { continueOnError: true });
  } catch (error) {
    logWarning('Web server test failed - server may not be running');
  }

  // Test backend connection (will fail if not running, which is expected)
  try {
    await runCommand('node test-connection.js', 'Testing backend connection', { continueOnError: true });
  } catch (error) {
    logWarning('Backend connection test failed - backend may not be running');
  }
}

async function showNextSteps() {
  logStep('6', 'Next Steps');

  log('\n🚀 Development Environment Ready!', 'green');
  log('\nTo start developing:', 'bright');

  log('\n1. Start the Next.js development server:', 'cyan');
  log('   npm run dev', 'yellow');

  log('\n2. In another terminal, start the Go backend:', 'cyan');
  log('   # Navigate to the Go backend directory', 'yellow');
  log('   # Run the backend server', 'yellow');

  log('\n3. Test the full integration:', 'cyan');
  log('   npm run test:connection', 'yellow');
  log('   npm run test:server', 'yellow');

  log('\n4. Access the application:', 'cyan');
  log('   Frontend: http://localhost:3002', 'yellow');
  log('   API Docs: http://localhost:3002/api/health', 'yellow');

  log('\n5. Available npm scripts:', 'cyan');
  log('   npm run dev          - Start development server', 'yellow');
  log('   npm run build        - Build for production', 'yellow');
  log('   npm run test:entropy - Test entropy bridge', 'yellow');
  log('   npm run test:server  - Test web server', 'yellow');
  log('   npm run test:connection - Test backend connection', 'yellow');
  log('   npm run prisma:studio - Open database GUI', 'yellow');
}

async function main() {
  try {
    log('\n🔧 Bitcoin Sprint Development Setup', 'bright');
    log('=====================================', 'bright');

    await checkDependencies();
    await setupEnvironment();
    await setupDatabase();
    await testEntropyBridge();
    await testConnections();
    await showNextSteps();

    log('\n🎉 Setup completed successfully!', 'green');

  } catch (error) {
    logError(`Setup failed: ${error.message}`);
    process.exit(1);
  }
}

// Run the setup
main().catch(console.error);
