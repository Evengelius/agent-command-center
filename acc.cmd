@echo off
rem acc starts the Agent Command Center from this repository, its project
rem field prefilled with the folder acc runs in. Arguments go to the program,
rem for example: acc --provider codex. The logic is scripts\acc.ps1 (plan 23).
rem In PowerShell, run it from here as .\acc.
powershell.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -File "%~dp0scripts\acc.ps1" %*
exit /b %ERRORLEVEL%
