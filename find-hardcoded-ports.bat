@echo off
echo =====================================================
echo Searching for hardcoded port numbers in Bitcoin Sprint
echo =====================================================
echo.

echo Searching for port 8080...
findstr /r /n /i "8080" *.go *.json *.md *.bat *.ps1 *.env* 2>nul
if not errorlevel 1 echo Found 8080 references above
echo.

echo Searching for port 8335...
findstr /r /n /i "8335" *.go *.json *.md *.bat *.ps1 *.env* 2>nul
if not errorlevel 1 echo Found 8335 references above
echo.

echo Searching for port 8332...
findstr /r /n /i "8332" *.go *.json *.md *.bat *.ps1 *.env* 2>nul
if not errorlevel 1 echo Found 8332 references above
echo.

echo Searching for port 8333...
findstr /r /n /i "8333" *.go *.json *.md *.bat *.ps1 *.env* 2>nul
if not errorlevel 1 echo Found 8333 references above
echo.

echo Searching in cmd/sprint/ subdirectory...
cd cmd\sprint 2>nul
if exist *.go (
    echo --- cmd/sprint Go files ---
    findstr /r /n /i "8080\|8335\|8332\|8333" *.go 2>nul
)
if exist *.json (
    echo --- cmd/sprint JSON files ---
    findstr /r /n /i "8080\|8335\|8332\|8333" *.json 2>nul
)
cd ..\.. 2>nul
echo.

echo Searching in pkg/ subdirectory...
cd pkg 2>nul
for /r %%f in (*.go) do (
    findstr /r /n /i "8080\|8335\|8332\|8333" "%%f" 2>nul | findstr /v "^$" >nul
    if not errorlevel 1 (
        echo Found in: %%f
        findstr /r /n /i "8080\|8335\|8332\|8333" "%%f" 2>nul
    )
)
cd .. 2>nul
echo.

echo Searching in web/ subdirectory...
cd web 2>nul
for /r %%f in (*.ts *.tsx *.js *.json) do (
    findstr /r /n /i "8080\|8335\|8332\|8333\|3000" "%%f" 2>nul | findstr /v "^$" >nul
    if not errorlevel 1 (
        echo Found in: %%f
        findstr /r /n /i "8080\|8335\|8332\|8333\|3000" "%%f" 2>nul
    )
)
cd .. 2>nul
echo.

echo Searching for "localhost:" patterns...
findstr /r /n /i "localhost:[0-9]*" *.go *.json *.md *.env* 2>nul
echo.

echo =====================================================
echo Search complete. Check output above for any hardcoded ports.
echo =====================================================
pause
