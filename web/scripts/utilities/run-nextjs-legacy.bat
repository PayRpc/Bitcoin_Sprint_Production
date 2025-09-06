@echo off
echo Starting Next.js with increased memory...
cd /d %~dp0
node --max-old-space-size=8192 node_modules\next\dist\bin\next dev -p 3002
