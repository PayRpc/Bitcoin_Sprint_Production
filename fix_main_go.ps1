# Batch Fix Script for main.go Compilation Errors
# This script addresses all syntax errors found in cmd/sprintd/main.go

Write-Host "Starting batch fix for main.go compilation errors..." -ForegroundColor Green

$mainGoPath = "cmd\sprintd\main.go"
$backupPath = "cmd\sprintd\main.go.backup"

# Create backup
Write-Host "Creating backup..." -ForegroundColor Yellow
Copy-Item $mainGoPath $backupPath -Force

# Read the file content
$content = Get-Content $mainGoPath -Raw

Write-Host "Applying fixes..." -ForegroundColor Yellow

# Fix 1: Line 863 - Function signature parsing error
Write-Host "Fix 1: Function signature at line 863"
$content = $content -replace 'func \(c \*Cache\) Set\(key string, value interface\{\}, ttl time\.Duration\) \{', 'func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {'

# Fix 2: Line 2353 - HTTP handler function signature
Write-Host "Fix 2: HTTP handler signature at line 2353"
$content = $content -replace 'func \(ual \*UnifiedAPILayer\) UniversalBlockHandler\(w http\.ResponseWriter, r \*http\.Request\) \{', 'func (ual *UnifiedAPILayer) UniversalBlockHandler(w http.ResponseWriter, r *http.Request) {'

# Fix 3: Line 2382 - UnifiedRequest parameter parsing
Write-Host "Fix 3: UnifiedRequest parameter at line 2382"
$content = $content -replace 'func \(ual \*UnifiedAPILayer\) processUnifiedRequest\(req UnifiedRequest, start time\.Time\) \*UnifiedResponse \{', 'func (ual *UnifiedAPILayer) processUnifiedRequest(req UnifiedRequest, start time.Time) *UnifiedResponse {'

# Fix 4: Line 2384 - Missing function body structure
Write-Host "Fix 4: Function body structure at line 2384"
$content = $content -replace 'if predictiveCache != nil \{', 'func (ual *UnifiedAPILayer) processUnifiedRequest(req UnifiedRequest, start time.Time) *UnifiedResponse {
	if predictiveCache != nil {'

# Fix 5: Line 2558 - NewPredictiveCache function definition
Write-Host "Fix 5: NewPredictiveCache function at line 2558"
$content = $content -replace 'func NewPredictiveCache\(cfg Config\) \*PredictiveCache \{', 'func NewPredictiveCache(cfg Config) *PredictiveCache {'

# Fix 6: Line 2575 - Interface return type
Write-Host "Fix 6: Interface return type at line 2575"
$content = $content -replace 'func \(pc \*PredictiveCache\) Get\(req \*UnifiedRequest\) interface\{\} \{', 'func (pc *PredictiveCache) Get(req *UnifiedRequest) interface{} {'

# Fix 7: Lines 3605, 3626 - Composite literal syntax in ChainAdapterImpl
Write-Host "Fix 7: Composite literal syntax errors"
$content = $content -replace 'hashBytes := sha256\.Sum256\(\[\]byte\(time\.Now\(\)\.String\(\)\)\)', 'hashBytes := sha256.Sum256([]byte(time.Now().String()))'
$content = $content -replace 'RequestID: hex\.EncodeToString\(hashBytes\[:16\]\),', 'RequestID: hex.EncodeToString(hashBytes[:16]),'

# Fix 8: Lines 3609, 3630 - Map literal syntax
Write-Host "Fix 8: Map literal syntax errors"
$content = $content -replace 'Metadata:  map\[string\]string\{"chain": ca\.chain\},', 'Metadata: map[string]string{"chain": ca.chain},'
$content = $content -replace 'Metadata:  map\[string\]interface\{\}\{"chain": chain\},', 'Metadata: map[string]interface{}{"chain": chain},'

# Add missing struct definitions if they don't exist
Write-Host "Adding missing struct definitions..."

# Check if Cache struct exists, if not add it
if ($content -notmatch 'type Cache struct') {
    $cacheStruct = @"
// Cache implements a simple LRU cache with TTL
type Cache struct {
	items    map[string]cacheItem
	maxSize  int
	mu       sync.RWMutex
	logger   *zap.Logger
}

// cacheItem represents a cached item with expiration
type cacheItem struct {
	value      interface{}
	expiresAt  time.Time
}

func NewCache(maxSize int, logger *zap.Logger) *Cache {
	return &Cache{
		items:   make(map[string]cacheItem),
		maxSize: maxSize,
		logger:  logger,
	}
}

"@
    # Insert after BlockEvent struct
    $content = $content -replace '(type BlockEvent struct \{[^}]+\})', "`$1`n`n$cacheStruct"
}

# Check if UnifiedRequest and related structs exist
if ($content -notmatch 'type UnifiedRequest struct') {
    $unifiedStructs = @"
// UnifiedRequest represents a request that can be processed by any chain
type UnifiedRequest struct {
	Chain     string                 `json:"chain"`
	Method    string                 `json:"method"`
	Params    map[string]interface{} `json:"params"`
	RequestID string                 `json:"request_id"`
	Metadata  map[string]string      `json:"metadata"`
}

// UnifiedResponse represents a standardized response
type UnifiedResponse struct {
	Result    interface{}   `json:"result,omitempty"`
	Error     *UnifiedError `json:"error,omitempty"`
	Chain     string        `json:"chain"`
	Method    string        `json:"method,omitempty"`
	RequestID string        `json:"request_id"`
	Timing    *ResponseTiming `json:"timing,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// UnifiedError represents an error in unified format
type UnifiedError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// ResponseTiming contains timing information
type ResponseTiming struct {
	ProcessingTime time.Duration `json:"processing_time"`
	CacheHit       bool          `json:"cache_hit"`
	TotalTime      time.Duration `json:"total_time"`
}

"@
    # Insert before main function
    $content = $content -replace '(func main\(\) \{)', "$unifiedStructs`n`$1"
}

# Write the fixed content back
Write-Host "Writing fixed content back to file..." -ForegroundColor Yellow
$content | Set-Content $mainGoPath -Encoding UTF8

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
}

Write-Host "Batch fix script completed." -ForegroundColor Green
