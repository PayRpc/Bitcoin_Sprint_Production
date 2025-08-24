# Bitcoin Sprint - Build Tag Architecture
# =====================================

# 📁 File Structure:
# main.go        → Production-pure (no demo logic)
# demo.go        → Demo implementation (//go:build demo)
# production.go  → Production block poller (//go:build !demo)

# 🔨 Build Commands:
Write-Host "🎯 CLEAN BUILD TAG ARCHITECTURE WORKING!" -ForegroundColor Green
Write-Host ""

Write-Host "=== Production Build ===" -ForegroundColor Cyan
Write-Host "go build -o bitcoin-sprint-prod.exe"
Write-Host "→ Real Bitcoin Core nodes only"
Write-Host ""

Write-Host "=== Demo Build (30s blocks) ===" -ForegroundColor Yellow
Write-Host "go build -tags=demo -o bitcoin-sprint-demo.exe"
Write-Host "→ Simulated blocks every 30 seconds"
Write-Host ""

Write-Host "=== Fast Demo Build (5s blocks) ===" -ForegroundColor Magenta
Write-Host "go build -tags=demo -o bitcoin-sprint-fast.exe"
Write-Host "→ Run with: .\bitcoin-sprint-fast.exe --fast-demo"
Write-Host "→ Simulated blocks every 5 seconds for stress testing"
Write-Host ""

Write-Host "=== API Testing ===" -ForegroundColor Green
Write-Host "# Latest block"
Write-Host "curl -s http://localhost:8080/latest | python -m json.tool"
Write-Host ""
Write-Host "# System status"
Write-Host "curl -s http://localhost:8080/status | python -m json.tool"
Write-Host ""
Write-Host "# Live stream (Turbo only)"
Write-Host "curl http://localhost:8080/stream"
Write-Host ""

Write-Host "✅ Benefits:" -ForegroundColor Green
Write-Host "  → main.go stays production-pure"
Write-Host "  → Demo logic completely isolated"
Write-Host "  → Can ship 3 different binaries"
Write-Host "  → No more hacking main.go for testing"
Write-Host ""
