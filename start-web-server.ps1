# Simple HTTP server to serve HTML dashboards
# This allows the HTML files to access the Bitcoin Sprint API without CORS issues

$port = 8888
$webRoot = Get-Location

Write-Host "Starting HTTP server on port $port..." -ForegroundColor Green
Write-Host "Web root: $webRoot" -ForegroundColor Yellow
Write-Host "" -ForegroundColor White
Write-Host "Available dashboards:" -ForegroundColor Cyan
Write-Host "  • Main Dashboard:     http://localhost:$port/dashboard.html" -ForegroundColor White
Write-Host "  • Entropy Monitor:    http://localhost:$port/entropy-monitor.html" -ForegroundColor White
Write-Host "  • SLA Test GUI:       http://localhost:$port/sla-test-gui.html" -ForegroundColor White
Write-Host "" -ForegroundColor White
Write-Host "Press Ctrl+C to stop the server..." -ForegroundColor Yellow
Write-Host ""

# Create HTTP listener
$listener = New-Object System.Net.HttpListener
$listener.Prefixes.Add("http://localhost:$port/")
$listener.Start()

try {
    while ($listener.IsListening) {
        $context = $listener.GetContext()
        $request = $context.Request
        $response = $context.Response
        
        $localPath = $request.Url.LocalPath
        if ($localPath -eq "/") {
            $localPath = "/dashboard.html"
        }
        
        $filePath = Join-Path $webRoot $localPath.TrimStart('/')
        
        Write-Host "$(Get-Date -Format 'HH:mm:ss') - $($request.HttpMethod) $localPath" -ForegroundColor Gray
        
        if (Test-Path $filePath -PathType Leaf) {
            $content = Get-Content $filePath -Raw -Encoding UTF8
            $response.ContentType = switch ([System.IO.Path]::GetExtension($filePath)) {
                ".html" { "text/html; charset=utf-8" }
                ".css"  { "text/css; charset=utf-8" }
                ".js"   { "application/javascript; charset=utf-8" }
                ".json" { "application/json; charset=utf-8" }
                default { "text/plain; charset=utf-8" }
            }
            
            # Add CORS headers to allow API access
            $response.Headers.Add("Access-Control-Allow-Origin", "*")
            $response.Headers.Add("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
            $response.Headers.Add("Access-Control-Allow-Headers", "Content-Type")
            
            $buffer = [System.Text.Encoding]::UTF8.GetBytes($content)
            $response.ContentLength64 = $buffer.Length
            $response.OutputStream.Write($buffer, 0, $buffer.Length)
        } else {
            $response.StatusCode = 404
            $content = "404 - File not found: $localPath"
            $buffer = [System.Text.Encoding]::UTF8.GetBytes($content)
            $response.ContentLength64 = $buffer.Length
            $response.OutputStream.Write($buffer, 0, $buffer.Length)
        }
        
        $response.OutputStream.Close()
    }
} finally {
    $listener.Stop()
    Write-Host "Server stopped." -ForegroundColor Red
}
