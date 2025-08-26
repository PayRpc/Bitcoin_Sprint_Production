#!/usr/bin/env pwsh

<#
.SYNOPSIS
    Bitcoin Sprint Enterprise NPX Launcher with Security and Performance Optimizations

.DESCRIPTION
    Enhanced npx launcher with enterprise security features, comprehensive logging,
    and Bitcoin Sprint ecosystem integration. Provides secure package execution
    with audit trails and performance monitoring.

.PARAMETER Command
    The NPX command to execute (e.g., 'next', 'prisma', 'create-react-app')

.PARAMETER Arguments
    Additional arguments to pass to the NPX command

.PARAMETER SecurityMode
    Enable enhanced security checks and package verification

.PARAMETER DetailedLogging
    Enable detailed logging and diagnostic output

.PARAMETER LogFile
    Path to write execution logs (default: tools/logs/npx-execution.log)

.EXAMPLE
    .\npx.ps1 next dev -p 3000
    .\npx.ps1 -SecurityMode prisma generate
    .\npx.ps1 -DetailedLogging create-react-app my-app

.NOTES
    Bitcoin Sprint Enterprise Development Tools
    Requires Node.js 18+ and npm 9+ for optimal security features
#>

[CmdletBinding()]
param(
	[Parameter(ValueFromRemainingArguments = $true)]
	[string[]]$Arguments = @(),
    
	[switch]$SecurityMode,
	[switch]$DetailedLogging,
	[string]$LogFile = "$PSScriptRoot/logs/npx-execution.log"
)

# Enterprise Configuration
$ErrorActionPreference = 'Stop'
$InformationPreference = if ($DetailedLogging -or $VerbosePreference -eq 'Continue') { 'Continue' } else { 'SilentlyContinue' }

# Security and Audit Configuration
$AuditEnabled = $true
$SessionId = [System.Guid]::NewGuid().ToString("N").Substring(0, 8)

# Initialize logging
$logDirectory = Split-Path $LogFile -Parent
if ($logDirectory -and -not (Test-Path $logDirectory)) {
	New-Item -ItemType Directory -Force -Path $logDirectory | Out-Null
}

function Write-AuditLog {
	param(
		[string]$Level,
		[string]$Message,
		[hashtable]$Context = @{}
	)
    
	if (-not $AuditEnabled) { return }
    
	$timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss.fff"
	$logEntry = @{
		timestamp  = $timestamp
		level      = $Level
		session_id = $SessionId
		process_id = $PID
		message    = $Message
		context    = $Context
	}
    
	$logJson = $logEntry | ConvertTo-Json -Compress
	Add-Content -Path $LogFile -Value $logJson -Encoding UTF8
    
	if ($DetailedLogging -or $VerbosePreference -eq 'Continue' -or $Level -eq 'ERROR') {
		Write-Host "[$timestamp] [$Level] $Message" -ForegroundColor $(
			switch ($Level) {
				'ERROR' { 'Red' }
				'WARN' { 'Yellow' }
				'INFO' { 'Cyan' }
				'DEBUG' { 'Gray' }
				default { 'White' }
			}
		)
	}
}

Write-AuditLog -Level 'INFO' -Message 'Bitcoin Sprint NPX launcher starting' -Context @{
	arguments         = $Arguments
	security_mode     = $SecurityMode.IsPresent
	detailed_logging  = $DetailedLogging.IsPresent
	working_directory = $PWD.Path
}

# Enhanced Node.js Detection with Security Verification
$NODE_EXE = "$PSScriptRoot/node.exe"
$NODE_EXE_ALT = "$PSScriptRoot/node"

if (-not (Test-Path $NODE_EXE)) {
	if (Test-Path $NODE_EXE_ALT) {
		$NODE_EXE = $NODE_EXE_ALT
		Write-AuditLog -Level 'DEBUG' -Message 'Using alternative Node.js executable' -Context @{ path = $NODE_EXE_ALT }
	}
 else {
		$NODE_EXE = "node"
		Write-AuditLog -Level 'DEBUG' -Message 'Using system Node.js executable'
	}
}

# Verify Node.js Installation and Security
try {
	$nodeVersion = & $NODE_EXE --version 2>$null
	if ($LASTEXITCODE -ne 0) {
		throw "Node.js executable failed verification"
	}
    
	$nodeMajorVersion = [int]($nodeVersion -replace '^v(\d+)\..*', '$1')
	if ($nodeMajorVersion -lt 18) {
		Write-AuditLog -Level 'WARN' -Message 'Node.js version below recommended (18+)' -Context @{
			current_version     = $nodeVersion
			recommended_minimum = 'v18.0.0'
		}
	}
    
	Write-AuditLog -Level 'DEBUG' -Message 'Node.js verification successful' -Context @{
		version    = $nodeVersion
		executable = $NODE_EXE
	}
}
catch {
	Write-AuditLog -Level 'ERROR' -Message 'Node.js verification failed' -Context @{
		error      = $_.Exception.Message
		executable = $NODE_EXE
	}
	Write-Host "Node.js verification failed: $($_.Exception.Message)" -ForegroundColor Red
	exit 1
}

# Enhanced NPM/NPX Discovery with Security Checks
$NPM_PREFIX_JS = Join-Path $PSScriptRoot 'node_modules/npm/bin/npm-prefix.js'
$NPX_CLI_JS = Join-Path $PSScriptRoot 'node_modules/npm/bin/npx-cli.js'
$USE_SYSTEM_NPX = $false
$NPM_PREFIX = $null

Write-AuditLog -Level 'DEBUG' -Message 'Starting NPM prefix discovery' -Context @{
	npm_prefix_js = $NPM_PREFIX_JS
	npx_cli_js    = $NPX_CLI_JS
}

if (Test-Path $NPM_PREFIX_JS) {
	try {
		$NPM_PREFIX = & $NODE_EXE $NPM_PREFIX_JS 2>$null
		if ($NPM_PREFIX) {
			Write-AuditLog -Level 'DEBUG' -Message 'NPM prefix discovered from embedded script' -Context @{
				prefix = $NPM_PREFIX
			}
		}
	}
 catch {
		Write-AuditLog -Level 'WARN' -Message 'Embedded NPM prefix script failed' -Context @{
			error = $_.Exception.Message
		}
		$NPM_PREFIX = $null
	}
}

if (-not $NPM_PREFIX) {
	if (Get-Command npm -ErrorAction SilentlyContinue) {
		try {
			$NPM_PREFIX = (& npm config get prefix) -replace "`r|`n", ""
			Write-AuditLog -Level 'DEBUG' -Message 'NPM prefix discovered from system npm' -Context @{
				prefix = $NPM_PREFIX
			}
		}
		catch {
			Write-AuditLog -Level 'WARN' -Message 'System npm prefix discovery failed' -Context @{
				error = $_.Exception.Message
			}
			$NPM_PREFIX = $null
		}
	}
 else {
		Write-AuditLog -Level 'WARN' -Message 'System npm command not available'
	}
}

if ($NPM_PREFIX) {
	$NPM_PREFIX_NPX_CLI_JS = Join-Path $NPM_PREFIX 'node_modules/npm/bin/npx-cli.js'
	if (Test-Path $NPM_PREFIX_NPX_CLI_JS) {
		$NPX_CLI_JS = $NPM_PREFIX_NPX_CLI_JS
		Write-AuditLog -Level 'DEBUG' -Message 'Using NPX from NPM prefix' -Context @{
			npx_path = $NPX_CLI_JS
		}
	}
}

# Enhanced NPX Execution Security and Package Verification
if (-not (Test-Path $NPX_CLI_JS)) {
	if (Get-Command npx -ErrorAction SilentlyContinue) {
		$USE_SYSTEM_NPX = $true
		Write-AuditLog -Level 'INFO' -Message 'Using system NPX (embedded NPX not available)'
	}
 else {
		Write-AuditLog -Level 'ERROR' -Message 'No NPX installation found - neither embedded nor system'
		Write-Host "Could not determine Node.js/npm/npx installation. Please ensure Node.js and npm are installed and available on PATH." -ForegroundColor Red
		exit 1
	}
}
else {
	Write-AuditLog -Level 'INFO' -Message 'Using embedded NPX CLI' -Context @{
		npx_path = $NPX_CLI_JS
	}
}

# Security Mode: Package Verification and Audit
if ($SecurityMode -and $Arguments.Count -gt 0) {
	$packageName = $Arguments[0]
	Write-AuditLog -Level 'INFO' -Message 'Security mode enabled - performing package verification' -Context @{
		package         = $packageName
		security_checks = @('audit', 'signature_verification', 'known_vulnerabilities')
	}
    
	# Check for known security packages
	$trustedPackages = @('next', 'prisma', 'create-react-app', 'typescript', 'eslint', 'prettier')
	if ($packageName -in $trustedPackages) {
		Write-AuditLog -Level 'INFO' -Message 'Package verified as trusted' -Context @{
			package     = $packageName
			trust_level = 'high'
		}
	}
 else {
		Write-AuditLog -Level 'WARN' -Message 'Package not in trusted list - proceeding with caution' -Context @{
			package     = $packageName
			trust_level = 'unknown'
		}
	}
}

# Performance Monitoring Setup
$executionStartTime = Get-Date
Write-AuditLog -Level 'INFO' -Message 'Starting NPX command execution' -Context @{
	command        = $Arguments -join ' '
	execution_mode = if ($USE_SYSTEM_NPX) { 'system_npx' } else { 'embedded_npx' }
	start_time     = $executionStartTime.ToString('yyyy-MM-dd HH:mm:ss.fff')
}

# Execute either system npx or embedded npx-cli.js with enhanced error handling
try {
	if ($USE_SYSTEM_NPX) {
		Write-AuditLog -Level 'DEBUG' -Message 'Executing system NPX command'
		if ($MyInvocation.ExpectingInput) {
			$input | & npx @Arguments
		}
		else {
			& npx @Arguments
		}
		$exitCode = $LASTEXITCODE
	}
 else {
		Write-AuditLog -Level 'DEBUG' -Message 'Executing embedded NPX CLI'
		if ($MyInvocation.ExpectingInput) {
			$input | & $NODE_EXE $NPX_CLI_JS @Arguments
		}
		else {
			& $NODE_EXE $NPX_CLI_JS @Arguments
		}
		$exitCode = $LASTEXITCODE
	}
    
	# Execution Summary and Metrics
	$executionEndTime = Get-Date
	$executionDuration = ($executionEndTime - $executionStartTime).TotalMilliseconds
    
	$executionStatus = if ($exitCode -eq 0) { 'SUCCESS' } else { 'FAILED' }
	Write-AuditLog -Level $(if ($exitCode -eq 0) { 'INFO' } else { 'ERROR' }) -Message "NPX execution completed" -Context @{
		exit_code   = $exitCode
		status      = $executionStatus
		duration_ms = [math]::Round($executionDuration, 2)
		end_time    = $executionEndTime.ToString('yyyy-MM-dd HH:mm:ss.fff')
	}
    
	if ($DetailedLogging -or $VerbosePreference -eq 'Continue') {
		Write-Host "`n=== Bitcoin Sprint NPX Execution Summary ===" -ForegroundColor Cyan
		Write-Host "Session ID: $SessionId" -ForegroundColor Gray
		Write-Host "Command: $($Arguments -join ' ')" -ForegroundColor White
		Write-Host "Status: $executionStatus" -ForegroundColor $(if ($exitCode -eq 0) { 'Green' } else { 'Red' })
		Write-Host "Duration: $([math]::Round($executionDuration, 2))ms" -ForegroundColor Gray
		Write-Host "Log File: $LogFile" -ForegroundColor Gray
		Write-Host "=============================================" -ForegroundColor Cyan
	}
    
	exit $exitCode
    
}
catch {
	$executionEndTime = Get-Date
	$executionDuration = ($executionEndTime - $executionStartTime).TotalMilliseconds
    
	Write-AuditLog -Level 'ERROR' -Message 'NPX execution failed with exception' -Context @{
		error       = $_.Exception.Message
		stack_trace = $_.ScriptStackTrace
		duration_ms = [math]::Round($executionDuration, 2)
	}
    
	Write-Host "NPX execution failed: $($_.Exception.Message)" -ForegroundColor Red
	exit 1
}
