# Bitcoin Sprint - Project Structure Guidelines

## 🎯 **MAINTAIN CLEAN ENTERPRISE ARCHITECTURE**

This document defines the **mandatory folder structure** for Bitcoin Sprint to prevent clutter accumulation and maintain professional organization.

## 📁 **Organized Directory Structure**

```
Bitcoin_Sprint/
├── 📁 cmd/                    # Go application entry points
├── 📁 internal/               # Go internal packages
├── 📁 secure/rust/           # Rust FFI security library
├── 📁 config/                # Main configuration files
├── 📁 docker/                # Docker compose & configs
├── 📁 scripts/               # Organized scripts
│   └── 📁 powershell/       # PowerShell automation
├── 📁 web/                   # Next.js web interface
├── 📁 logs/                  # Application logs
├── 📁 data/                  # Data storage
├── 📁 docs/                  # Documentation
└── 📄 [essential root files] # Only critical files
```

## 🚫 **PROHIBITED: Root Directory Clutter**

**Never place these in root directory:**

### PowerShell Scripts
- ❌ `/*.ps1` files in root
- ✅ Use `scripts/powershell/` instead

### Docker/YAML Files  
- ❌ `docker-compose*.yml` in root
- ❌ `monitoring*.yml` in root
- ✅ Use `docker/` or `config/` instead

### Legacy Simple API
- ❌ `simple_api/` directory
- ❌ Any `simple-*` files
- ✅ Use enterprise web API in `web/pages/api/`

### Empty/Corrupted Files
- ❌ 0-byte files (corruption indicator)
- ❌ `-empty.*` pattern files
- ✅ Immediate removal required

## 🛡️ **Protection Mechanisms**

### 1. **Git Ignore Protection**
`.gitignore` prevents:
- Loose PowerShell scripts
- Root directory YAML files  
- Legacy simple_api returns
- Empty file patterns

### 2. **Essential Root Files Only**
Allowed in root directory:
- `go.mod`, `go.sum` (Go modules)
- `Makefile` (build system)
- `README.md` (documentation)
- `LICENSE` (legal)
- Configuration files (minimal set)

### 3. **Cleanup Verification**
Regular checks for:
- Empty files (corruption)
- Duplicate configs
- Misplaced scripts
- Legacy artifacts

## 🎯 **Enterprise Quality Standards**

### ✅ **Clean Architecture Achieved**
- **32 → 7** PowerShell scripts (organized)
- **11 → 3** Docker configs (consolidated) 
- **16** empty files removed
- **0** root directory clutter

### 🔧 **Core Enterprise Components**
1. **CustomerKeyManager** - Authentication system
2. **SecureBuf** - Memory protection & FFI
3. **Web API** - Next.js enterprise interface
4. **Rust FFI** - Hardware-backed security
5. **Monitoring** - Professional observability

## 📋 **Maintenance Commands**

### Check for violations:
```powershell
# Find loose PowerShell scripts
Get-ChildItem -Filter "*.ps1" | Where-Object {$_.Directory.Name -eq "BItcoin_Sprint"}

# Find loose YAML files  
Get-ChildItem -Filter "*.yml" | Where-Object {$_.Directory.Name -eq "BItcoin_Sprint"}

# Find empty files
Get-ChildItem -Recurse | Where-Object {$_.Length -eq 0}
```

### Cleanup if needed:
```powershell
# Move scripts to proper location
Move-Item "*.ps1" "scripts/powershell/"

# Move Docker configs
Move-Item "docker-compose*.yml" "docker/"

# Remove empty files
Get-ChildItem | Where-Object {$_.Length -eq 0} | Remove-Item
```

## 🎖️ **Quality Assurance**

This structure ensures:
- **Professional appearance** for enterprise clients
- **Maintainable codebase** for development teams  
- **Scalable architecture** for feature expansion
- **Security compliance** for audit requirements

---

**⚠️ CRITICAL:** Violation of this structure indicates potential:
- Development environment corruption
- Automated script malfunction  
- Build system degradation
- Security component failure

**Always maintain this clean enterprise architecture!**
