@echo off
setlocal

set "PAUSE_FILE=%~dp0phoneworker.pause"

if exist "%PAUSE_FILE%" del "%PAUSE_FILE%"
echo phoneworker resume requested: "%PAUSE_FILE%"
echo If phoneworker is not running, start it with run_phoneworker.bat.
pause
