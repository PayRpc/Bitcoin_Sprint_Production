@echo off
echo ?? Bitcoin Sprint Self-Contained Startup
echo ========================================
cd /d %~dp0

REM Add libs to PATH
set PATH=%~dp0libs;%PATH%

REM Copy default config if needed
if not exist config.json (
    if exist config\config-production-optimized.json (
        copy config\config-production-optimized.json config.json >nul
        echo ? Config file ready
    )
)

REM Copy default license if needed
if not exist license.json (
    if exist licenses\license-enterprise.json (
        copy licenses\license-enterprise.json license.json >nul
        echo ? License file ready
    )
)

REM Set turbo mode
set TIER=turbo
echo ? Turbo mode enabled

REM Start Bitcoin Sprint
echo ?? Starting Bitcoin Sprint...
bin\bitcoin-sprint.exe
