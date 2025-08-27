@echo off
echo ?? Bitcoin Sprint API Load Test
echo ===============================

if "%~1"=="" (
    echo Usage: %0 [URL] [DURATION] [CONCURRENT_USERS]
    echo Example: %0 http://localhost:8080/health 30 10
    echo.
    echo Using default test...
    powershell -ExecutionPolicy Bypass -File "%~dp0simple-load-test.ps1"
) else (
    powershell -ExecutionPolicy Bypass -File "%~dp0simple-load-test.ps1" -Url "%~1" -DurationSeconds "%~2" -ConcurrentUsers "%~3"
)
