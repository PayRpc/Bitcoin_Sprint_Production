# start-backend.ps1
# Stops any old sprintd process, starts fresh, and tails logs in real time

# Stop any existing sprintd.exe process
Get-Process -Name "sprintd" -ErrorAction SilentlyContinue | Stop-Process -Force

# Start sprintd.exe, logging to sprintd.log (combine stdout and stderr)
cmd /c ".\sprintd.exe > sprintd.log 2>&1"

# Wait a moment for the process to start
Start-Sleep -Seconds 2

# Tail the log file in real time
Get-Content -Path "sprintd.log" -Wait
