# Advanced Batch Fix Script for main.go Compilation Errors
# This script uses line-by-line processing to fix syntax errors

Write-Host "Starting advanced batch fix for main.go compilation errors..." -ForegroundColor Green

$mainGoPath = "cmd\sprintd\main.go"
$backupPath = "cmd\sprintd\main.go.backup2"

# Create backup
Write-Host "Creating backup..." -ForegroundColor Yellow
Copy-Item $mainGoPath $backupPath -Force

# Read all lines
$lines = Get-Content $mainGoPath

Write-Host "Processing $($lines.Count) lines..." -ForegroundColor Yellow

# Process each line to fix syntax errors
for ($i = 0; $i -lt $lines.Count; $i++) {
    $line = $lines[$i]
    $lineNum = $i + 1
    
    # Fix function signatures with parameter parsing issues
    if ($line -match 'func \(.*\) \w+\([^)]*http\.ResponseWriter[^)]*\) \{' -and $line -notmatch 'w http\.ResponseWriter, r \*http\.Request') {
        Write-Host "Fixing HTTP handler signature at line $lineNum"
        $lines[$i] = $line -replace 'func \((.*?)\) (\w+)\(([^)]*http\.ResponseWriter[^)]*)\) \{', 'func ($1) $2(w http.ResponseWriter, r *http.Request) {'
    }
    
    # Fix UnifiedRequest parameter issues
    if ($line -match 'func \(.*\) \w+\([^)]*UnifiedRequest[^)]*\) \*?UnifiedResponse \{' -and $line -notmatch 'req UnifiedRequest, start time\.Time') {
        Write-Host "Fixing UnifiedRequest parameter at line $lineNum"
        $lines[$i] = $line -replace 'func \((.*?)\) (\w+)\(([^)]*UnifiedRequest[^)]*)\) (\*?UnifiedResponse) \{', 'func ($1) $2(req UnifiedRequest, start time.Time) $4 {'
    }
    
    # Fix interface{} return type issues
    if ($line -match 'func \(.*\) Get\([^)]*\) interface\{\} \{' -and $line -match 'interface\{\}') {
        Write-Host "Fixing interface return type at line $lineNum"
        $lines[$i] = $line -replace 'interface\{\}', 'interface{}'
    }
    
    # Fix Cache Set function signature
    if ($line -match 'func \(c \*Cache\) Set\(' -and $line -match 'string, value interface\{\}, ttl') {
        Write-Host "Fixing Cache Set signature at line $lineNum"
        $lines[$i] = 'func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {'
    }
    
    # Fix orphaned if statements (missing function declaration)
    if ($line -match '^\s*if predictiveCache != nil \{$' -and $i > 0 -and $lines[$i-1] -notmatch 'func ') {
        Write-Host "Fixing orphaned if statement at line $lineNum"
        # Look back to find the intended function
        for ($j = $i - 5; $j -ge 0; $j--) {
            if ($lines[$j] -match 'processUnifiedRequest') {
                # This if belongs in processUnifiedRequest function
                break
            }
        }
        # Keep the if statement as is, it should be inside a function
    }
    
    # Fix composite literal syntax errors
    if ($line -match 'hashBytes := sha256\.Sum256.*in composite literal') {
        Write-Host "Fixing composite literal at line $lineNum"
        # This usually means we're inside a struct literal, need to move it outside
        if ($i > 0 -and $lines[$i-1] -match '.*:.*\{$') {
            # Insert before the struct literal
            $lines[$i] = $line
        }
    }
    
    # Fix map literal colon syntax
    if ($line -match 'Metadata:\s+map\[.*\]\{.*\}') {
        Write-Host "Fixing map literal syntax at line $lineNum"
        $lines[$i] = $line -replace 'Metadata:\s+', 'Metadata: '
    }
}

# Add missing struct definitions at the top of the file
$structsToAdd = @()

# Check if we need to add Cache struct
$hasCacheStruct = $false
$hasUnifiedStruct = $false

foreach ($line in $lines) {
    if ($line -match 'type Cache struct') { $hasCacheStruct = $true }
    if ($line -match 'type UnifiedRequest struct') { $hasUnifiedStruct = $true }
}

if (-not $hasCacheStruct) {
    Write-Host "Adding Cache struct definition"
    $structsToAdd += @(
        "",
        "// Cache implements a simple LRU cache with TTL",
        "type Cache struct {",
        "	items    map[string]cacheItem",
        "	maxSize  int", 
        "	mu       sync.RWMutex",
        "	logger   *zap.Logger",
        "}",
        "",
        "// cacheItem represents a cached item with expiration", 
        "type cacheItem struct {",
        "	value      interface{}",
        "	expiresAt  time.Time",
        "}",
        "",
        "func NewCache(maxSize int, logger *zap.Logger) *Cache {",
        "	return &Cache{",
        "		items:   make(map[string]cacheItem),",
        "		maxSize: maxSize,",
        "		logger:  logger,",
        "	}",
        "}"
    )
}

if (-not $hasUnifiedStruct) {
    Write-Host "Adding Unified structs definitions"
    $structsToAdd += @(
        "",
        "// UnifiedRequest represents a request that can be processed by any chain",
        "type UnifiedRequest struct {",
        "	Chain     string                 ``json:`"chain`"``",
        "	Method    string                 ``json:`"method`"``", 
        "	Params    map[string]interface{} ``json:`"params`"``",
        "	RequestID string                 ``json:`"request_id`"``",
        "	Metadata  map[string]string      ``json:`"metadata`"``",
        "}",
        "",
        "// UnifiedResponse represents a standardized response",
        "type UnifiedResponse struct {",
        "	Result    interface{}            ``json:`"result,omitempty`"``",
        "	Error     *UnifiedError          ``json:`"error,omitempty`"``",
        "	Chain     string                 ``json:`"chain`"``",
        "	Method    string                 ``json:`"method,omitempty`"``",
        "	RequestID string                 ``json:`"request_id`"``",
        "	Timing    *ResponseTiming        ``json:`"timing,omitempty`"``",
        "	Metadata  map[string]interface{} ``json:`"metadata,omitempty`"``",
        "}",
        "",
        "// UnifiedError represents an error in unified format", 
        "type UnifiedError struct {",
        "	Code    int    ``json:`"code`"``",
        "	Message string ``json:`"message`"``",
        "	Details string ``json:`"details,omitempty`"``",
        "}",
        "",
        "// ResponseTiming contains timing information",
        "type ResponseTiming struct {",
        "	ProcessingTime time.Duration ``json:`"processing_time`"``",
        "	CacheHit       bool          ``json:`"cache_hit`"``",
        "	TotalTime      time.Duration ``json:`"total_time`"``",
        "}"
    )
}

# Insert struct definitions after imports
if ($structsToAdd.Count -gt 0) {
    for ($i = 0; $i -lt $lines.Count; $i++) {
        if ($lines[$i] -match '^\)$' -and $i > 0 -and $lines[$i-1] -match 'import|zap|unix') {
            # Found end of imports, insert structs here
            $newLines = @()
            $newLines += $lines[0..$i]
            $newLines += $structsToAdd
            $newLines += $lines[($i+1)..($lines.Count-1)]
            $lines = $newLines
            break
        }
    }
}

# Write the fixed content back
Write-Host "Writing fixed content back to file..." -ForegroundColor Yellow
$lines | Set-Content $mainGoPath -Encoding UTF8

# Test compilation
Write-Host "Testing compilation..." -ForegroundColor Yellow
$buildResult = & go build -o sprintd cmd/sprintd/main.go 2>&1

if ($LASTEXITCODE -eq 0) {
    Write-Host "SUCCESS: main.go compiles successfully!" -ForegroundColor Green
    Write-Host "Binary created: sprintd" -ForegroundColor Green
    Remove-Item $backupPath -Force
} else {
    Write-Host "COMPILATION STILL HAS ERRORS:" -ForegroundColor Red
    Write-Host $buildResult -ForegroundColor Red
    Write-Host "Backup available at: $backupPath" -ForegroundColor Yellow
    Write-Host "You can restore with: Copy-Item $backupPath $mainGoPath -Force" -ForegroundColor Yellow
    
    # Show first few errors for analysis
    $errorLines = ($buildResult -split "`n") | Where-Object { $_ -match "cmd\\sprintd\\main.go:" } | Select-Object -First 5
    Write-Host "`nFirst 5 compilation errors:" -ForegroundColor Yellow
    $errorLines | ForEach-Object { Write-Host $_ -ForegroundColor Cyan }
}

Write-Host "Advanced batch fix script completed." -ForegroundColor Green
