#!/usr/bin/env pwsh

# Robust npx launcher: prefer repository-embedded npm's npx-cli.js, fall back to system npx

$NODE_EXE = "$PSScriptRoot/node.exe"
$NODE_EXE_ALT = "$PSScriptRoot/node"
if (-not (Test-Path $NODE_EXE)) {
	if (Test-Path $NODE_EXE_ALT) {
		$NODE_EXE = $NODE_EXE_ALT
	}
 else {
		$NODE_EXE = "node"
	}
}

$NPM_PREFIX_JS = Join-Path $PSScriptRoot 'node_modules/npm/bin/npm-prefix.js'
$NPX_CLI_JS = Join-Path $PSScriptRoot 'node_modules/npm/bin/npx-cli.js'
$USE_SYSTEM_NPX = $false
$NPM_PREFIX = $null

if (Test-Path $NPM_PREFIX_JS) {
	try {
		$NPM_PREFIX = & $NODE_EXE $NPM_PREFIX_JS 2>$null
	}
 catch {
		$NPM_PREFIX = $null
	}
}

if (-not $NPM_PREFIX) {
	if (Get-Command npm -ErrorAction SilentlyContinue) {
		try {
			$NPM_PREFIX = (& npm config get prefix) -replace "`r|`n", ""
		}
		catch {
			$NPM_PREFIX = $null
		}
	}
}

if ($NPM_PREFIX) {
	$NPM_PREFIX_NPX_CLI_JS = Join-Path $NPM_PREFIX 'node_modules/npm/bin/npx-cli.js'
	if (Test-Path $NPM_PREFIX_NPX_CLI_JS) {
		$NPX_CLI_JS = $NPM_PREFIX_NPX_CLI_JS
	}
}

if (-not (Test-Path $NPX_CLI_JS)) {
	if (Get-Command npx -ErrorAction SilentlyContinue) {
		$USE_SYSTEM_NPX = $true
	}
 else {
		Write-Host "Could not determine Node.js/npm/npx installation. Please ensure Node.js and npm are installed and available on PATH."
		exit 1
	}
}

# Execute either system npx or embedded npx-cli.js, preserving pipeline input
if ($USE_SYSTEM_NPX) {
	if ($MyInvocation.ExpectingInput) {
		$input | & npx @args
	}
 else {
		& npx @args
	}
	exit $LASTEXITCODE
}
else {
	if ($MyInvocation.ExpectingInput) {
		$input | & $NODE_EXE $NPX_CLI_JS @args
	}
 else {
		& $NODE_EXE $NPX_CLI_JS @args
	}
	exit $LASTEXITCODE
}
