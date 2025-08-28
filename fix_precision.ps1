# Precision fix for main.go - targeting exact syntax errors
$backupFile = "cmd\sprintd\main.go.backup_precision"
Copy-Item "cmd\sprintd\main.go" $backupFile
Write-Host "Created backup: $backupFile"

# Read the file as an array of lines for precise line editing
$lines = Get-Content "cmd\sprintd\main.go"

# Fix composite literal around line 3607
for ($i = 0; $i -lt $lines.Length; $i++) {
    # Fix malformed composite literal with missing braces
    if ($lines[$i] -match "hashBytes := sha256\.Sum256" -and $i -gt 3600) {
        # Found the problematic section - need to fix the composite literal structure
        $lines[$i-3] = "		return &UnifiedRequest{"
        $lines[$i-2] = "			Chain:     ca.chain,"
        $lines[$i-1] = "			Method:    method,"
        $lines[$i] = "			Params:    map[string]interface{}{`"params`": params},"
        $lines[$i+1] = "			RequestID: func() string {"
        $lines[$i+2] = "				hashBytes := sha256.Sum256([]byte(time.Now().String()))"
        $lines[$i+3] = "				return hex.EncodeToString(hashBytes[:16])"
        $lines[$i+4] = "			}(),"
        $lines[$i+5] = "			Metadata: map[string]string{`"chain`": ca.chain},"
        $lines[$i+6] = "		}, nil"
        break
    }
}

# Write the fixed content back
$lines | Set-Content "cmd\sprintd\main.go"

Write-Host "Applied precision fixes to main.go"

# Test compilation
Write-Host "`nTesting compilation..."
$result = go build -o sprintd cmd/sprintd/main.go 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host "SUCCESS: Compilation successful!" -ForegroundColor Green
} else {
    Write-Host "Compilation errors remain:" -ForegroundColor Yellow
    $result | Select-Object -First 5
    Write-Host "`nTo restore backup: Copy-Item $backupFile cmd\sprintd\main.go"
}
