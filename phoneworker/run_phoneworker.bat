@echo off
setlocal

set "EXE=%~dp0phoneworker-windows-amd64.exe"
set "DEFAULT_TOKEN=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJVVUlEIjoiZjAyMDY1ODUtZDM1Zi00MmVmLTk0ZTgtZTQzNTQ4NDQ5OTVjIiwiSUQiOjQ4LCJVc2VybmFtZSI6ImppbmppbjAxIiwiTmlja05hbWUiOiLph5Hph5EiLCJBdXRob3JpdHlJZCI6MzAwLCJCdWZmZXJUaW1lIjo4NjQwMCwidG9rZW5UeXBlIjoib3BlbmFwaSIsImlzcyI6InFyIiwiYXVkIjpbIkdWQSJdLCJleHAiOjE3ODc1MDUzNzksIm5iZiI6MTc3OTcyOTM3OX0.ULEgoLVl4evWdNEZDnW7p50GEqOEkI5FqQyZ7T9dR6w"
set "PHONE_URL=http://206.238.179.123:37520/OPenApi/GetOrder?infor=vwZt5p4FmyeupCqMqKsC2ktcpczoBuX23akOGMEPlsw%%3D&project=wb3"
set "TOKEN=%~1"
set "INTERVAL=3s"

if not exist "%EXE%" (
  echo phoneworker executable not found: "%EXE%"
  pause
  exit /b 1
)

if "%TOKEN%"=="" set "TOKEN=%DEFAULT_TOKEN%"

if "%TOKEN%"=="" (
  echo token is required.
  pause
  exit /b 1
)

"%EXE%" ^
  -token "%TOKEN%" ^
  -phone-url "%PHONE_URL%" ^
  -interval "%INTERVAL%"

echo.
echo phoneworker exited with code %ERRORLEVEL%.
pause
