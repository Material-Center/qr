@echo off
setlocal

set "PAUSE_FILE=%~dp0phoneworker.pause"
echo pause>"%PAUSE_FILE%"
echo phoneworker pause requested: "%PAUSE_FILE%"
echo The running worker will finish current work and stop creating new tasks.
pause
