# Surgical fixes for main.go compilation errors
# Create backup first
$backupFile = "cmd\sprintd\main.go.backup_surgical"
Copy-Item "cmd\sprintd\main.go" $backupFile
Write-Host "Created backup: $backupFile"

$content = Get-Content "cmd\sprintd\main.go" -Raw

# Fix 1: Remove duplicate processUnifiedRequest function definition (line 2383)
$content = $content -replace 'func \(ual \*UnifiedAPILayer\) processUnifiedRequest\(req UnifiedRequest, start time\.Time\) \*UnifiedResponse \{\s*func \(ual \*UnifiedAPILayer\) processUnifiedRequest\(req UnifiedRequest, start time\.Time\) \*UnifiedResponse \{', 'func (ual *UnifiedAPILayer) processUnifiedRequest(req UnifiedRequest, start time.Time) *UnifiedResponse {'

# Fix 2: Add missing closing brace for UniversalBlockHandler function before processUnifiedRequest
$content = $content -replace 'response := ual\.processUnifiedRequest\(req, start\)\s*if latencyOptimizer != nil \{\s*func \(ual \*UnifiedAPILayer\) processUnifiedRequest', 'response := ual.processUnifiedRequest(req, start)
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func (ual *UnifiedAPILayer) processUnifiedRequest'

# Fix 3: Fix Cache Set function call syntax (line 863)
$content = $content -replace 'Cache\.Set\(key, result, entry\.TTL\)\.', 'Cache.Set(key, result, entry.TTL)'

# Fix 4: Fix composite literal syntax errors
$content = $content -replace '\s*Status:\s*"success",\s*Data:\s*cached,\s*}', '
        Status: "success",
        Data:   cached,
    }'

# Fix 5: Fix HTTP handler definitions
$content = $content -replace 'func \(s \*HTTPServer\) handleUnified\(w http\.ResponseWriter, r \*http\.Request\) \{', 'func (s *HTTPServer) handleUnified(w http.ResponseWriter, r *http.Request) {'

# Fix 6: Fix map literal syntax
$content = $content -replace 'map\[string\]interface\{\}\{\s*"error":\s*err\.Error\(\),\s*\}', 'map[string]interface{}{
        "error": err.Error(),
    }'

# Fix 7: Fix interface{} return type syntax
$content = $content -replace 'return\s+interface\{\}\{', 'return map[string]interface{}{'

# Fix 8: Fix Cache initialization
$content = $content -replace 'Cache:\s+cache\.New\(5\*time\.Minute, 10\*time\.Minute\),', 'Cache: cache.New(5*time.Minute, 10*time.Minute),'

# Write the fixed content
$content | Set-Content "cmd\sprintd\main.go" -NoNewline

Write-Host "Applied surgical fixes to main.go"

# Test compilation
Write-Host "`nTesting compilation..."
$result = go build -o sprintd cmd/sprintd/main.go 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host "SUCCESS: Compilation successful!" -ForegroundColor Green
} else {
    Write-Host "Compilation failed with errors:" -ForegroundColor Red
    Write-Host $result
    Write-Host "`nTo restore backup: Copy-Item $backupFile cmd\sprintd\main.go"
}
