@echo off
cd /d %~dp0
echo Starting Bitcoin Sprint Web Application...
node --max-old-space-size=8192 .\node_modules\next\dist\bin\next dev -p 3002
