# Bitcoin Sprint - Environment Protection Script
# Ensures enterprise configuration stays permanent and protected

param(
    [switch]$Install,
    [switch]$Check,
    [switch]$Restore
)

Write-Host "🛡️ Bitcoin Sprint Environment Protection" -ForegroundColor Cyan
Write-Host "========================================"

$ErrorActionPreference = "Stop"

# Define paths
$envFile = ".env"
$enterpriseTemplate = ".env.enterprise"
$protectedBackup = ".env.enterprise.protected"
$checksumFile = ".env.checksum"

function Get-FileChecksum {
    param($FilePath)
    if (Test-Path $FilePath) {
        return (Get-FileHash $FilePath -Algorithm SHA256).Hash
    }
    return $null
}

function Create-ProtectedBackup {
    Write-Host "📦 Creating protected enterprise template..." -ForegroundColor Yellow
    
    if (Test-Path $envFile) {
        Copy-Item $envFile $protectedBackup -Force
        $checksum = Get-FileChecksum $envFile
        Set-Content $checksumFile $checksum
        
        # Make protected backup read-only
        Set-ItemProperty $protectedBackup -Name IsReadOnly -Value $true
        
        Write-Host "✅ Protected backup created: $protectedBackup" -ForegroundColor Green
        Write-Host "✅ Checksum stored: $checksum" -ForegroundColor Green
    } else {
        Write-Host "❌ .env file not found!" -ForegroundColor Red
        exit 1
    }
}

function Check-Environment {
    Write-Host "🔍 Checking environment integrity..." -ForegroundColor Yellow
    
    if (-not (Test-Path $envFile)) {
        Write-Host "❌ .env file missing!" -ForegroundColor Red
        return $false
    }
    
    if (-not (Test-Path $checksumFile)) {
        Write-Host "⚠️ No checksum file found - environment not protected" -ForegroundColor Yellow
        return $false
    }
    
    $currentChecksum = Get-FileChecksum $envFile
    $expectedChecksum = Get-Content $checksumFile -Raw
    $expectedChecksum = $expectedChecksum.Trim()
    
    if ($currentChecksum -eq $expectedChecksum) {
        Write-Host "✅ Environment integrity verified" -ForegroundColor Green
        
        # Check critical enterprise settings
        $content = Get-Content $envFile -Raw
        $criticalChecks = @(
            @{ Pattern = "TIER=enterprise"; Name = "Enterprise Tier" },
            @{ Pattern = "RATE_LIMIT_REQUESTS_PER_SECOND=500"; Name = "Enterprise Rate Limit" },
            @{ Pattern = "DATABASE_URL=./enterprise_tier.db"; Name = "Enterprise Database" },
            @{ Pattern = "ENABLE_KERNEL_BYPASS=true"; Name = "Kernel Bypass" },
            @{ Pattern = "ENABLE_ENTROPY_MONITORING=true"; Name = "Entropy Monitoring" }
        )
        
        $allPassed = $true
        foreach ($check in $criticalChecks) {
            if ($content -match [regex]::Escape($check.Pattern)) {
                Write-Host "  ✅ $($check.Name): OK" -ForegroundColor Green
            } else {
                Write-Host "  ❌ $($check.Name): MISSING" -ForegroundColor Red
                $allPassed = $false
            }
        }
        
        return $allPassed
    } else {
        Write-Host "❌ Environment has been modified!" -ForegroundColor Red
        Write-Host "   Expected: $expectedChecksum" -ForegroundColor Gray
        Write-Host "   Current:  $currentChecksum" -ForegroundColor Gray
        return $false
    }
}

function Restore-Environment {
    Write-Host "🔧 Restoring enterprise environment..." -ForegroundColor Yellow
    
    if (-not (Test-Path $protectedBackup)) {
        if (Test-Path $enterpriseTemplate) {
            Write-Host "⚠️ Using .env.enterprise template" -ForegroundColor Yellow
            Copy-Item $enterpriseTemplate $envFile -Force
        } else {
            Write-Host "❌ No protected backup or template found!" -ForegroundColor Red
            exit 1
        }
    } else {
        # Temporarily remove read-only to copy
        Set-ItemProperty $protectedBackup -Name IsReadOnly -Value $false
        Copy-Item $protectedBackup $envFile -Force
        Set-ItemProperty $protectedBackup -Name IsReadOnly -Value $true
    }
    
    # Update checksum
    $newChecksum = Get-FileChecksum $envFile
    Set-Content $checksumFile $newChecksum
    
    Write-Host "✅ Enterprise environment restored!" -ForegroundColor Green
}

function Install-Protection {
    Write-Host "📥 Installing environment protection..." -ForegroundColor Yellow
    
    # Create protected backup
    Create-ProtectedBackup
    
    # Add to gitignore to protect sensitive data
    $gitignoreContent = Get-Content ".gitignore" -Raw -ErrorAction SilentlyContinue
    if ($gitignoreContent -notmatch "\.env\.enterprise\.protected") {
        Add-Content ".gitignore" "`n# Environment Protection`n.env.enterprise.protected`n.env.checksum"
        Write-Host "✅ Added protection files to .gitignore" -ForegroundColor Green
    }
    
    # Create scheduled check
    $taskScript = @"
# Environment Protection Check
if (-not (.\scripts\powershell\protect-environment.ps1 -Check)) {
    Write-Host "🚨 Environment corruption detected! Restoring..." -ForegroundColor Red
    .\scripts\powershell\protect-environment.ps1 -Restore
}
"@
    
    Set-Content "scripts\powershell\check-environment.ps1" $taskScript
    Write-Host "✅ Environment check script created" -ForegroundColor Green
    
    Write-Host "`n🛡️ Environment protection installed!" -ForegroundColor Green
    Write-Host "   • Protected backup: $protectedBackup" -ForegroundColor Gray
    Write-Host "   • Integrity checksum: $checksumFile" -ForegroundColor Gray
    Write-Host "   • Auto-check script: scripts\powershell\check-environment.ps1" -ForegroundColor Gray
}

# Main execution
if ($Install) {
    Install-Protection
} elseif ($Check) {
    $result = .\scripts\powershell\check-environment.ps1
    if (-not $result) { exit 1 }
} elseif ($Restore) {
    Restore-Environment
} else {
    Write-Host "Usage:"
    Write-Host "  -Install  : Install environment protection"
    Write-Host "  -Check    : Check environment integrity"
    Write-Host "  -Restore  : Restore from protected backup"
}
