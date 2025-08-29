Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Bitcoin Sprint FastAPI Gateway" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

Write-Host "Starting FastAPI Gateway on port 8000..." -ForegroundColor Green
Write-Host ""

Write-Host "Configuration:" -ForegroundColor Magenta
Write-Host "  - Backend URL: http://localhost:8080" -ForegroundColor White
Write-Host "  - Gateway URL: http://localhost:8000" -ForegroundColor White
Write-Host "  - Docs URL: http://localhost:8000/docs" -ForegroundColor White
Write-Host ""

Write-Host "API Keys for testing:" -ForegroundColor Magenta
Write-Host "  - Free: demo-key-free" -ForegroundColor White
Write-Host "  - Pro: demo-key-pro" -ForegroundColor White
Write-Host "  - Enterprise: demo-key-enterprise" -ForegroundColor White
Write-Host ""

Write-Host "Starting server..." -ForegroundColor Green
Write-Host ""

# Start the FastAPI server
python app.py
