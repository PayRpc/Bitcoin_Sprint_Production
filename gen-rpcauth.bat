@echo off
setlocal enabledelayedexpansion

echo.
echo ===============================================
echo  Bitcoin Core RPC Auth Generator (Windows)
echo ===============================================
echo.

:: Default values
set "username=sprint"
set "password=MyStrongPassw0rd123!"

:: Check if parameters were provided
if not "%1"=="" set "username=%1"
if not "%2"=="" set "password=%2"

echo Username: %username%
echo Password: %password%
echo.

:: Generate random salt (16 characters)
set "chars=ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
set "salt="
for /L %%i in (1,1,16) do (
    set /a "rand=!random! %% 62"
    for %%j in (!rand!) do set "salt=!salt!!chars:~%%j,1!"
)

echo Generated salt: %salt%
echo.

:: Create temporary PowerShell script for HMAC computation
echo $salt = '%salt%' > temp_hmac.ps1
echo $password = '%password%' >> temp_hmac.ps1
echo $hmac = New-Object System.Security.Cryptography.HMACSHA256 >> temp_hmac.ps1
echo $hmac.Key = [Text.Encoding]::UTF8.GetBytes($salt) >> temp_hmac.ps1
echo $hashBytes = $hmac.ComputeHash([Text.Encoding]::UTF8.GetBytes($password)) >> temp_hmac.ps1
echo $hashHex = -join ($hashBytes ^| ForEach-Object { $_.ToString("x2") }) >> temp_hmac.ps1
echo Write-Output $hashHex >> temp_hmac.ps1

:: Execute PowerShell script and capture output
for /f %%i in ('powershell -executionpolicy bypass -file temp_hmac.ps1') do set "hash=%%i"

:: Clean up temp file
del temp_hmac.ps1

:: Output result
echo ===============================================
echo  GENERATED RPC AUTH CONFIGURATION
echo ===============================================
echo.
echo Add this line to your bitcoin.conf:
echo.
echo rpcauth=%username%:%salt%$%hash%
echo.
echo Use these credentials in your .env file:
echo BTC_RPC_USER=%username%
echo BTC_RPC_PASS=%password%
echo.
echo ===============================================
echo  SETUP COMPLETE - Your RPC is now secure!
echo ===============================================
echo.

pause
