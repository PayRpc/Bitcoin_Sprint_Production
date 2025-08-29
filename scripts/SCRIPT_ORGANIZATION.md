# Bitcoin Sprint Scripts Organization Plan
# ======================================

## 📁 scripts/startup/ - System Startup Scripts
- start-system.bat (MAIN - keep in root for easy access)
- start-system.ps1 (MAIN - keep in root for easy access)
- start-backend.ps1
- start-backend-simple.ps1
- start-complete-system.bat
- start-docker-metrics-server.ps1
- start-fastapi.ps1
- start-metrics-server.ps1

## 📁 scripts/testing/ - Testing & Validation Scripts
- test-api.ps1
- test-integration.ps1
- test-production-readiness.bat
- test-service.sh
- validate-system.bat (MAIN - keep in root for easy access)
- validate-acceleration-layer.ps1
- validate-competitive-advantage.ps1
- customer-api-simulation.ps1 (KEEP - this is the working version)
- customer-api-simulation-clean.ps1 (DELETE - empty)
- customer-api-simulation-fixed.ps1 (DELETE - empty)
- customer-api-simulation-new.ps1 (DELETE - empty)
- real-data-test.ps1
- automated-test.ps1
- comprehensive-test.ps1
- multichain_sla_testing.ps1

## 📁 scripts/business/ - Business Analysis Scripts
- business-analysis.ps1
- business-summary.ps1
- api-architecture-analysis.ps1
- infrastructure_status_report.ps1
- latency-validation-report.ps1

## 📁 scripts/deployment/ - Deployment & Production Scripts
- deploy-grafana.ps1
- deploy-solana.ps1
- deploy_with_turbo_validation.sh
- package-production.ps1
- production-turbo-validator.ps1

## 📁 scripts/monitoring/ - Monitoring & Metrics Scripts
- monitor-entropy.ps1
- bitcoin-core-monitoring.ps1
- bitcoin-core-monitoring-simulated.ps1
- solana-demo.ps1
- solana-load-test.ps1
- quick-load-test.ps1
- register-metrics-server-task.ps1

## 📁 scripts/maintenance/ - Maintenance & Utility Scripts
- cleanup-repo.bat
- cleanup-safe.ps1
- safe-cleanup.bat
- analyze-cleanup.bat
- memory-profile.ps1
- manage-platform.ps1
- backend-manager.ps1
- check-status.bat

## 📁 scripts/development/ - Development & Build Scripts
- fix_main_advanced.ps1
- fix_main_go.ps1
- fix_precision.ps1
- fix_simple.ps1
- fix_surgical.ps1
- generate-enterprise-key.ps1
- activate-turbo-mode.ps1
- debug_generate_key.py (keep in root or move to tools/)

## 📁 Keep in Root (Essential Scripts)
- start-system.bat (main startup)
- start-system.ps1 (main startup)
- validate-system.bat (main validation)
- setup-bitcoin-core.ps1 (setup script)

## 🗑️ Scripts to DELETE (Empty or Duplicate)
- customer-api-simulation-clean.ps1
- customer-api-simulation-fixed.ps1
- customer-api-simulation-new.ps1
