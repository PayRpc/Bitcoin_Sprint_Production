# Simple Batch Fix Script for main.go Compilation Errors
Write-Host "Starting simple batch fix for main.go compilation errors..." -ForegroundColor Green

$mainGoPath = "cmd\sprintd\main.go"
$backupPath = "cmd\sprintd\main.go.backup3"

# Create backup
Copy-Item $mainGoPath $backupPath -Force
Write-Host "Backup created: $backupPath" -ForegroundColor Yellow

# Read content as single string
$content = Get-Content $mainGoPath -Raw

Write-Host "Applying targeted fixes..." -ForegroundColor Yellow

# Fix 1: Broken function signatures - use simpler approach
Write-Host "Fix 1: Function signatures"
$content = $content -replace 'func \(c \*Cache\) Set\(key string, value interface\{\}, ttl time\.Duration\) \{', 'func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {'

# Fix 2: Add missing Cache struct if not present
if ($content -notmatch 'type Cache struct') {
    Write-Host "Fix 2: Adding Cache struct"
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
    $content = $content -replace '(type BlockEvent struct \{[^}]+\})', "`$1$cacheStruct"
}

# Fix 3: Add missing Unified structs if not present
if ($content -notmatch 'type UnifiedRequest struct') {
    Write-Host "Fix 3: Adding Unified structs"
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
	Result    interface{}            `json:"result,omitempty"`
	Error     *UnifiedError          `json:"error,omitempty"`
	Chain     string                 `json:"chain"`
	Method    string                 `json:"method,omitempty"`
	RequestID string                 `json:"request_id"`
	Timing    *ResponseTiming        `json:"timing,omitempty"`
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
    # Insert before first function after imports
    $content = $content -replace '(func [A-Za-z])', "$unifiedStructs`$1"
}

# Write back
$content | Set-Content $mainGoPath -Encoding UTF8

Write-Host "Testing compilation..." -ForegroundColor Yellow
$result = & go build -o sprintd cmd/sprintd/main.go 2>&1

if ($LASTEXITCODE -eq 0) {
    Write-Host "SUCCESS: main.go compiles!" -ForegroundColor Green
    Remove-Item $backupPath -Force
} else {
    Write-Host "Still has errors. Manual fixes needed:" -ForegroundColor Red
    $result | Write-Host -ForegroundColor Cyan
    Write-Host "Restore backup: Copy-Item $backupPath $mainGoPath -Force" -ForegroundColor Yellow
}

Write-Host "Script completed." -ForegroundColor Green
