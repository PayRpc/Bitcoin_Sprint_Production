Set objShell = CreateObject("Shell.Application")
Set fso = CreateObject("Scripting.FileSystemObject")

' Get the script's directory
scriptDir = fso.GetParentFolderName(WScript.ScriptFullName)

' Build the path to the batch file
batchPath = scriptDir & "\bin\install-service.bat"

' Elevate and execute the batch file
objShell.ShellExecute "cmd.exe", "/c """ & batchPath & """", "", "runas", 1
