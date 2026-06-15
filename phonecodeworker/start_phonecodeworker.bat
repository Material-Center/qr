@echo off
setlocal

set "PAUSE_FILE=%~dp0phonecodeworker.pause"
set "RUN_BAT=%~dp0run_phonecodeworker.bat"

if exist "%PAUSE_FILE%" del "%PAUSE_FILE%"
echo phonecodeworker resume requested: "%PAUSE_FILE%"
start "phonecodeworker" "%RUN_BAT%" %*
