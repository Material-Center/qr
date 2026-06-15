@echo off
setlocal

set "PAUSE_FILE=%~dp0phoneworker.pause"
set "RUN_BAT=%~dp0run_phoneworker.bat"

if exist "%PAUSE_FILE%" del "%PAUSE_FILE%"
echo phoneworker resume requested: "%PAUSE_FILE%"
start "phoneworker" "%RUN_BAT%" %*
