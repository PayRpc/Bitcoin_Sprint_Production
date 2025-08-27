@echo off
REM Quick validation of entropy monitoring setup
powershell -ExecutionPolicy Bypass -File "%~dp0validate-entropy-setup.ps1" -QuickTest
pause
