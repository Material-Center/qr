@echo off
setlocal

set "EXE=%~dp0phonecodeworker-windows-amd64.exe"
set "DEFAULT_TOKEN=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJVVUlEIjoiOTljYjE5MDctYmVhYi00NmQ1LTgyNTUtMTA3YTA0MzcyYjE2IiwiSUQiOjEyOCwiVXNlcm5hbWUiOiJjaGFveWFuZzEiLCJOaWNrTmFtZSI6IuacnemYsy3mlLYiLCJBdXRob3JpdHlJZCI6MzAwLCJCdWZmZXJUaW1lIjo4NjQwMCwidG9rZW5UeXBlIjoib3BlbmFwaSIsImlzcyI6InFyIiwiYXVkIjpbIkdWQSJdLCJleHAiOjE3ODkwNTgwMTIsIm5iZiI6MTc4MTI4MjAxMn0.DzEajB7jl7kPXk4aos8djjh_nruwTKGH5KexvitrkyU"
set "DEFAULT_INPUT=%~dp0phones.txt"
set "TOKEN=%~1"
set "INPUT_FILE=%~2"
set "INTERVAL=3s"
set "IDLE_THRESHOLD=1"
set "CREATE_DELAY=0s"
set "TASK_SYNC_LIMIT=3"

if not exist "%EXE%" (
  echo phonecodeworker executable not found: "%EXE%"
  pause
  exit /b 1
)

if "%TOKEN%"=="" set "TOKEN=%DEFAULT_TOKEN%"
if "%INPUT_FILE%"=="" set "INPUT_FILE=%DEFAULT_INPUT%"

if "%TOKEN%"=="" (
  echo token is required.
  echo Usage: run_phonecodeworker.bat your-openapi-token phones.txt
  pause
  exit /b 1
)

if not exist "%INPUT_FILE%" (
  echo input file not found: "%INPUT_FILE%"
  echo Put phones.txt next to this bat or pass an input file path as the second argument.
  pause
  exit /b 1
)

"%EXE%" ^
  -token "%TOKEN%" ^
  -input "%INPUT_FILE%" ^
  -interval "%INTERVAL%" ^
  -idle-threshold "%IDLE_THRESHOLD%" ^
  -create-delay "%CREATE_DELAY%" ^
  -task-sync-limit "%TASK_SYNC_LIMIT%"

echo.
echo phonecodeworker exited with code %ERRORLEVEL%.
pause
