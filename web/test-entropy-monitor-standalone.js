#!/usr/bin/env node

/**
 * Entropy Monitor Standalone Test
 * Tests the entropy monitoring setup without requiring the full web app
 */

import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

async function testEntropyMonitoringStandalone() {
  console.log('🔐 Bitcoin Sprint Entropy Monitor Standalone Test');
  console.log('================================================');
  console.log('');

  let testsPassed = 0;
  let totalTests = 0;

  // Test 1: Check if prometheus.ts exists and has entropy metrics
  totalTests++;
  console.log('Test 1: Checking Prometheus metrics configuration...');
  try {
    const prometheusPath = path.join(__dirname, 'lib', 'prometheus.ts');
    if (fs.existsSync(prometheusPath)) {
      const prometheusContent = fs.readFileSync(prometheusPath, 'utf8');

      const entropyMetrics = [
        'bitcoin_sprint_entropy_bridge_available',
        'bitcoin_sprint_entropy_bridge_rust_available',
        'bitcoin_sprint_entropy_bridge_fallback_mode',
        'bitcoin_sprint_entropy_secret_generation_total',
        'bitcoin_sprint_entropy_secret_generation_duration_seconds',
        'bitcoin_sprint_entropy_quality_score',
        'bitcoin_sprint_entropy_admin_auth_attempts_total',
        'bitcoin_sprint_entropy_admin_auth_duration_seconds'
      ];

      let foundMetrics = 0;
      entropyMetrics.forEach(metric => {
        if (prometheusContent.includes(metric)) {
          foundMetrics++;
        }
      });

      if (foundMetrics === entropyMetrics.length) {
        console.log('✅ All entropy metrics found in prometheus.ts');
        testsPassed++;
      } else {
        console.log(`⚠️  Found ${foundMetrics}/${entropyMetrics.length} entropy metrics`);
      }
    } else {
      console.log('❌ prometheus.ts not found');
    }
  } catch (error) {
    console.log('❌ Error checking prometheus.ts:', error.message);
  }

  // Test 2: Check if entropy-status API endpoint exists
  totalTests++;
  console.log('');
  console.log('Test 2: Checking entropy-status API endpoint...');
  try {
    const apiPath = path.join(__dirname, 'pages', 'api', 'admin', 'entropy-status.ts');
    if (fs.existsSync(apiPath)) {
      const apiContent = fs.readFileSync(apiPath, 'utf8');

      if (apiContent.includes('recordEntropyQualityScore') &&
          apiContent.includes('recordEntropySecretGeneration')) {
        console.log('✅ Entropy-status API endpoint has metrics recording');
        testsPassed++;
      } else {
        console.log('⚠️  Entropy-status API endpoint missing some metrics recording');
      }
    } else {
      console.log('❌ entropy-status.ts not found');
    }
  } catch (error) {
    console.log('❌ Error checking entropy-status.ts:', error.message);
  }

  // Test 3: Check if Grafana dashboard exists
  totalTests++;
  console.log('');
  console.log('Test 3: Checking Grafana dashboard configuration...');
  try {
    const dashboardPath = path.join(__dirname, '..', 'grafana', 'dashboards', 'grafana-dashboard-entropy-bridge.json');
    if (fs.existsSync(dashboardPath)) {
      const dashboardContent = fs.readFileSync(dashboardPath, 'utf8');

      if (dashboardContent.includes('Bitcoin Sprint - Entropy Bridge Monitor') &&
          dashboardContent.includes('bitcoin_sprint_entropy')) {
        console.log('✅ Grafana dashboard configured for entropy monitoring');
        testsPassed++;
      } else {
        console.log('⚠️  Grafana dashboard missing entropy monitoring configuration');
      }
    } else {
      console.log('❌ Grafana dashboard not found');
    }
  } catch (error) {
    console.log('❌ Error checking Grafana dashboard:', error.message);
  }

  // Test 4: Check if provisioning configuration exists
  totalTests++;
  console.log('');
  console.log('Test 4: Checking Grafana provisioning configuration...');
  try {
    const provisioningPath = path.join(__dirname, '..', 'grafana', 'provisioning', 'dashboards', 'dashboards.yml');
    if (fs.existsSync(provisioningPath)) {
      const provisioningContent = fs.readFileSync(provisioningPath, 'utf8');

      if (provisioningContent.includes('/opt/bitcoin-sprint/grafana/dashboards')) {
        console.log('✅ Grafana provisioning configured for entropy dashboard directory');
        testsPassed++;
      } else {
        console.log('⚠️  Grafana provisioning missing correct dashboard directory path');
      }
    } else {
      console.log('❌ Grafana provisioning configuration not found');
    }
  } catch (error) {
    console.log('❌ Error checking Grafana provisioning:', error.message);
  }

  // Test 5: Check if test script exists
  totalTests++;
  console.log('');
  console.log('Test 5: Checking test script configuration...');
  try {
    const packagePath = path.join(__dirname, 'package.json');
    if (fs.existsSync(packagePath)) {
      const packageContent = fs.readFileSync(packagePath, 'utf8');

      if (packageContent.includes('test:monitor') &&
          packageContent.includes('test-entropy-monitor.js')) {
        console.log('✅ Test script configured in package.json');
        testsPassed++;
      } else {
        console.log('⚠️  Test script not properly configured in package.json');
      }
    } else {
      console.log('❌ package.json not found');
    }
  } catch (error) {
    console.log('❌ Error checking package.json:', error.message);
  }

  // Summary
  console.log('');
  console.log('📊 Test Summary');
  console.log('===============');
  console.log(`✅ ${testsPassed}/${totalTests} tests passed`);

  if (testsPassed === totalTests) {
    console.log('');
    console.log('🎉 All entropy monitoring components are properly configured!');
    console.log('');
    console.log('Next steps:');
    console.log('1. Start the web application: npm run dev');
    console.log('2. Start Grafana: docker-compose up grafana');
    console.log('3. Access dashboard at: http://localhost:3000');
    console.log('4. Run full test: npm run test:monitor');
  } else {
    console.log('');
    console.log('⚠️  Some components need attention. Check the errors above.');
  }

  console.log('');
  console.log('📖 For detailed setup instructions, see: ENTROPY_MONITOR_README.md');
}

testEntropyMonitoringStandalone().catch(console.error);
