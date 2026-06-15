@echo off
setlocal

set "PAUSE_FILE=%~dp0phonecodeworker.pause"
echo pause>"%PAUSE_FILE%"
echo phonecodeworker pause requested: "%PAUSE_FILE%"
echo The running worker will finish active tasks and stop creating new tasks.
pause
