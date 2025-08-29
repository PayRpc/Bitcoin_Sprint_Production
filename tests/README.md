# Bitcoin Sprint Test Suite

This directory contains the organized test suite for Bitcoin Sprint, structured by test type and purpose.

## Directory Structure

### `/integration/`

Integration tests that validate end-to-end functionality and multi-chain operations.

- `sprint_test.go` - Core Sprint functionality tests
- `multichain_infrastructure_test.go` - Multi-chain infrastructure validation
- `simple_multichain_test.go` - Basic multi-chain operations
- `multichain_demo_server.go` - Demo server for multi-chain testing
- `multichain_zmq_test.go` - ZMQ-based multi-chain tests
- `multichain_sla_test.ps1` - PowerShell SLA testing script
- `real_zmq_sla_test.ps1` - Real ZMQ SLA validation

### `/performance/`

Performance and benchmarking tests.

- `performance_validation.go` - Comprehensive performance tracking and SLA validation

### `/solana/`

Solana-specific customer simulation and testing.

- `customer-solana-simulation.ps1` - Complete DeFi analytics customer journey simulation

## Running Tests

### Go Tests

```bash
# Run all integration tests
cd tests/integration
go run multichain_infrastructure_test.go

# Run performance tests
cd tests/performance
go run performance_validation.go
```

### PowerShell Tests

```powershell
# Run SLA tests
.\tests\integration\multichain_sla_test.ps1
.\tests\integration\real_zmq_sla_test.ps1
```

## Test Categories

- **Integration Tests**: Validate multi-chain functionality, API endpoints, and system integration
- **Performance Tests**: Measure latency, throughput, and SLA compliance
- **SLA Tests**: Validate service level agreements and performance guarantees

## Notes

- All tests use realistic load patterns and production-ready configurations
- Performance tests include automatic adaptation and optimization triggers
- SLA tests validate sub-5ms response times for enterprise tier operations
