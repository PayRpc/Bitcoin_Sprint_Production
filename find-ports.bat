@echo off
echo Searching for port 8080...
findstr /r /n "8080" cmd\sprint\*.go
echo.
echo Searching for port 8335...
findstr /r /n "8335" cmd\sprint\*.go
echo.
echo Done.
pause
