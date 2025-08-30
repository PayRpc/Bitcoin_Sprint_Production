@echo off
REM Bitcoin Sprint Tier Switcher
REM Usage: switch-tier.bat [free|pro|business|turbo|enterprise]

if "%1"=="" (
    echo Usage: switch-tier.bat [free^|pro^|business^|turbo^|enterprise]
    echo.
    echo Available tiers:
    echo   free      - Basic tier with limited resources
    echo   pro       - Professional tier with moderate resources
    echo   business  - Business tier with higher performance
    echo   turbo     - High-performance tier with ultra-low latency
    echo   enterprise- Enterprise tier with maximum performance
    goto :eof
)

set TIER=%1

if exist ".env.%TIER%" (
    echo Switching to %TIER% tier...
    copy ".env.%TIER%" ".env" >nul
    echo ✅ Successfully switched to %TIER% tier
    echo.
    echo Current configuration:
    echo TIER=%TIER%
    echo.
    echo To apply changes, restart the application:
    echo   .\start-dev.bat
) else (
    echo ❌ Error: Tier '%TIER%' not found
    echo Available tiers: free, pro, business, turbo, enterprise
)
