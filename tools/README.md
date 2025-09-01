# Tools Directory

This directory contains management and utility scripts for the Bitcoin Sprint project.

## Available Scripts

### CI/CD Management
- **`cicd-manager.ps1`** - Comprehensive CI/CD pipeline setup, testing, and management
  - `-Setup`: Validate CI/CD environment and check required components
  - `-Test`: Test workflow syntax and component functionality
  - `-Validate`: Validate CI/CD configuration and file integrity
  - `-Cleanup`: Clean up CI/CD artifacts and temporary files

### Usage Examples

```powershell
# Setup and validate CI/CD environment
.\tools\cicd-manager.ps1 -Setup

# Test CI/CD components
.\tools\cicd-manager.ps1 -Test

# Validate configuration
.\tools\cicd-manager.ps1 -Validate
```

## Organization

Scripts are organized by function:
- **CI/CD**: Continuous integration and deployment tools
- **Management**: Project management and maintenance scripts
- **Utilities**: General-purpose helper scripts

## Contributing

When adding new scripts to this directory:
1. Include comprehensive help documentation
2. Add usage examples
3. Update this README with the new script information
4. Follow PowerShell best practices and error handling
