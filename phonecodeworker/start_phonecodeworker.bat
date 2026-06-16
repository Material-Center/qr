@echo off
setlocal

set "PAUSE_FILE=%~dp0phonecodeworker.pause"

if exist "%PAUSE_FILE%" del "%PAUSE_FILE%"
echo phonecodeworker resume requested: "%PAUSE_FILE%"
echo If phonecodeworker is not running, start it with run_phonecodeworker.bat.
pause
